package webrunner

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gosom/google-maps-scraper/deduper"
	"github.com/gosom/google-maps-scraper/exiter"
	"github.com/gosom/google-maps-scraper/grid"
	"github.com/gosom/google-maps-scraper/runner"
	"github.com/gosom/google-maps-scraper/tlmt"
	"github.com/gosom/google-maps-scraper/web"
	"github.com/gosom/google-maps-scraper/web/sqlite"
	"github.com/gosom/scrapemate"
	"github.com/gosom/scrapemate/scrapemateapp"
	"golang.org/x/sync/errgroup"
)

// defaultCellSizeKm is the grid resolution used when a city is selected but no
// coverage level is given. 3km keeps a mid-sized city to a few hundred cells.
const defaultCellSizeKm = 3.0

type webrunner struct {
	srv       *web.Server
	svc       *web.Service
	cfg       *runner.Config
	setupMate func(context.Context, *os.File, *web.Job) (mateRunner, error)
}

type mateRunner interface {
	Start(context.Context, ...scrapemate.IJob) error
	Close() error
}

func New(cfg *runner.Config) (runner.Runner, error) {
	if cfg.DataFolder == "" {
		return nil, fmt.Errorf("data folder is required")
	}

	if err := os.MkdirAll(cfg.DataFolder, os.ModePerm); err != nil {
		return nil, err
	}

	const dbfname = "jobs.db"

	dbpath := filepath.Join(cfg.DataFolder, dbfname)

	repo, err := sqlite.New(dbpath)
	if err != nil {
		return nil, err
	}

	svc := web.NewService(repo, cfg.DataFolder)

	srv, err := web.New(svc, cfg.Addr)
	if err != nil {
		return nil, err
	}

	ans := webrunner{
		srv:       srv,
		svc:       svc,
		cfg:       cfg,
		setupMate: defaultSetupMate(cfg),
	}

	return &ans, nil
}

func (w *webrunner) Run(ctx context.Context) error {
	if err := w.recoverStuckJobs(ctx); err != nil {
		log.Printf("could not recover interrupted jobs: %v", err)
	}

	egroup, ctx := errgroup.WithContext(ctx)

	egroup.Go(func() error {
		return w.work(ctx)
	})

	egroup.Go(func() error {
		return w.srv.Start(ctx)
	})

	return egroup.Wait()
}

func (w *webrunner) Close(context.Context) error {
	return nil
}

// recoverStuckJobs closes out jobs left mid-run by a previous process. Nothing is
// running when the runner starts, so a job still marked "working" was interrupted
// — a restart, a crash, or a copied database. Left alone it stays "working" for
// ever: it never shows an outcome, never offers its results, and its elapsed timer
// counts up indefinitely.
func (w *webrunner) recoverStuckJobs(ctx context.Context) error {
	jobs, err := w.svc.All(ctx)
	if err != nil {
		return err
	}

	for i := range jobs {
		// A paused job is waiting for a person, not for a process, so it is left
		// exactly as it is.
		if jobs[i].Status != web.StatusWorking {
			continue
		}

		jobs[i].MarkFinished(web.StatusFailed)

		if err := w.svc.Update(ctx, &jobs[i]); err != nil {
			return err
		}

		log.Printf("job %s was interrupted before it finished; marked failed", jobs[i].ID)
	}

	return nil
}

func (w *webrunner) work(ctx context.Context) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			jobs, err := w.svc.SelectPending(ctx)
			if err != nil {
				return err
			}

			for i := range jobs {
				select {
				case <-ctx.Done():
					return nil
				default:
					t0 := time.Now().UTC()
					if err := w.scrapeJob(ctx, &jobs[i]); err != nil {
						params := map[string]any{
							"job_count": len(jobs[i].Data.Keywords),
							"duration":  time.Now().UTC().Sub(t0).String(),
							"error":     err.Error(),
						}

						evt := tlmt.NewEvent("web_runner", params)

						_ = runner.Telemetry().Send(ctx, evt)

						log.Printf("error scraping job %s: %v", jobs[i].ID, err)
					} else {
						params := map[string]any{
							"job_count": len(jobs[i].Data.Keywords),
							"duration":  time.Now().UTC().Sub(t0).String(),
						}

						_ = runner.Telemetry().Send(ctx, tlmt.NewEvent("web_runner", params))

						log.Printf("job %s scraped successfully", jobs[i].ID)
					}
				}
			}
		}
	}
}

func (w *webrunner) scrapeJob(ctx context.Context, job *web.Job) error {
	job.MarkStarted()

	err := w.svc.Update(ctx, job)
	if err != nil {
		return err
	}

	if len(job.Data.Keywords) == 0 {
		job.MarkFinished(web.StatusFailed)

		return w.svc.Update(ctx, job)
	}

	outpath := filepath.Join(w.cfg.DataFolder, job.ID+".csv")

	// A continued run appends: the results collected before the pause stay.
	resuming := job.Data.Progress > 0

	outfile, err := openResultFile(outpath, resuming)
	if err != nil {
		return err
	}

	defer func() {
		_ = outfile.Close()
	}()

	setupMate := w.setupMate
	if setupMate == nil {
		setupMate = defaultSetupMate(w.cfg)
	}

	mate, err := setupMate(ctx, outfile, job)
	if err != nil {
		job.MarkFinished(web.StatusFailed)

		err2 := w.svc.Update(ctx, job)
		if err2 != nil {
			log.Printf("failed to update job status: %v", err2)
		}

		return err
	}

	defer mate.Close()

	var coords string
	if job.Data.Lat != "" && job.Data.Lon != "" {
		coords = job.Data.Lat + "," + job.Data.Lon
	}

	dedup := deduper.New()
	exitMonitor := exiter.New()

	seedJobs, err := w.buildSeedJobs(job, dedup, exitMonitor, coords)
	if err != nil {
		job.MarkFinished(web.StatusFailed)

		if err2 := w.svc.Update(ctx, job); err2 != nil {
			log.Printf("failed to update job status: %v", err2)
		}

		return err
	}

	if resuming {
		seeded, derr := seedDeduper(outpath, dedup)
		if derr != nil {
			log.Printf("job %s: could not seed deduper: %v", job.ID, derr)
		}

		if job.Data.Progress < len(seedJobs) {
			log.Printf("job %s continuing from search %d of %d (%d results already held)",
				job.ID, job.Data.Progress, len(seedJobs), seeded)

			seedJobs = seedJobs[job.Data.Progress:]
		} else {
			seedJobs = nil
		}
	}

	done := job.Data.Progress

	if len(seedJobs) > 0 {
		exitMonitor.SetSeedCount(len(seedJobs))

		allowedSeconds := max(60, len(seedJobs)*10*job.Data.Depth/50+120)

		if job.Data.MaxTime > 0 {
			if job.Data.MaxTime.Seconds() < 180 {
				allowedSeconds = 180
			} else {
				allowedSeconds = int(job.Data.MaxTime.Seconds())
			}
		}

		log.Printf("running job %s with %d seed jobs and %d allowed seconds", job.ID, len(seedJobs), allowedSeconds)

		mateCtx, cancel := context.WithTimeout(ctx, time.Duration(allowedSeconds)*time.Second)
		defer cancel()

		exitMonitor.SetCancelFunc(cancel)

		go exitMonitor.Run(mateCtx)

		// The connection is watched for the whole run. If it stays down the run is
		// stopped rather than left to burn every retry against a dead network and
		// silently skip the areas it was working on.
		outage := false

		netCtx, stopWatch := context.WithCancel(mateCtx)
		defer stopWatch()

		go watchNetwork(netCtx, cancel, &outage)

		err = mate.Start(mateCtx, seedJobs...)

		stopWatch()

		if outage {
			cancel()

			// Resume from the start of this batch: redoing searches is cheap and
			// safe, while skipping any is not. De-duplication stops repeats.
			job.MarkPaused("The internet connection dropped, so the scan stopped to "+
				"avoid skipping areas. Press Continue when you are back online.",
				job.Data.Progress)

			if err2 := w.svc.Update(ctx, job); err2 != nil {
				log.Printf("failed to update job status: %v", err2)
			}

			log.Printf("job %s paused: network unreachable", job.ID)

			return nil
		}

		if err != nil && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
			cancel()

			job.MarkFinished(web.StatusFailed)

			err2 := w.svc.Update(ctx, job)
			if err2 != nil {
				log.Printf("failed to update job status: %v", err2)
			}

			return err
		}

		done += len(seedJobs)

		cancel()
	}

	job.Data.Progress = done
	job.Data.PausedReason = ""

	job.MarkFinished(web.StatusOK)

	return w.svc.Update(ctx, job)
}

// buildSeedJobs turns the job's location settings into the searches to run: one
// per grid cell per term when areas are selected, otherwise one per term.
func (w *webrunner) buildSeedJobs(
	job *web.Job,
	dedup deduper.Deduper,
	exitMonitor exiter.Exiter,
	coords string,
) ([]scrapemate.IJob, error) {
	queryText := strings.Join(job.Data.Keywords, "\n")
	areas := job.Data.Areas()

	if len(areas) == 0 {
		radius := float64(job.Data.Radius)
		if job.Data.Radius <= 0 {
			radius = 10000 // 10 km
		}

		return runner.CreateSeedJobs(
			job.Data.FastMode,
			job.Data.Lang,
			strings.NewReader(queryText),
			job.Data.Depth,
			job.Data.Email,
			coords,
			job.Data.Zoom,
			radius,
			dedup,
			exitMonitor,
			w.cfg.ExtraReviews || job.Data.ExtraReviews,
		)
	}

	cellSize := job.Data.CellSizeKm
	if cellSize <= 0 {
		cellSize = defaultCellSizeKm
	}

	var (
		seedJobs   []scrapemate.IJob
		totalCells int
	)

	for _, area := range areas {
		bbox, err := grid.ParseBoundingBox(area)
		if err != nil {
			return nil, err
		}

		// A fresh reader per area: the previous one is already consumed.
		areaJobs, err := runner.CreateGridSeedJobs(
			job.Data.Lang,
			strings.NewReader(queryText),
			job.Data.Depth,
			job.Data.Email,
			bbox,
			cellSize,
			job.Data.Zoom,
			dedup,
			exitMonitor,
			w.cfg.ExtraReviews || job.Data.ExtraReviews,
		)
		if err != nil {
			return nil, err
		}

		totalCells += grid.EstimateCellCount(bbox, cellSize)

		seedJobs = append(seedJobs, areaJobs...)
	}

	log.Printf("job %s covers %d area(s) as %d cells of %.1fkm",
		job.ID, len(areas), totalCells, cellSize)

	return seedJobs, nil
}

func defaultSetupMate(cfg *runner.Config) func(context.Context, *os.File, *web.Job) (mateRunner, error) {
	return func(_ context.Context, writer *os.File, job *web.Job) (mateRunner, error) {
		opts := []func(*scrapemateapp.Config) error{
			scrapemateapp.WithConcurrency(cfg.Concurrency),
			scrapemateapp.WithExitOnInactivity(time.Minute * 3),
		}

		if !job.Data.FastMode {
			opts = append(opts,
				scrapemateapp.WithJS(scrapemateapp.DisableImages()),
			)
		} else {
			opts = append(opts,
				scrapemateapp.WithStealth("firefox"),
			)
		}

		opts = runner.AppendBrowserCapacityOptions(opts, cfg)

		hasProxy := false

		if len(cfg.Proxies) > 0 {
			opts = append(opts, scrapemateapp.WithProxies(cfg.Proxies))
			hasProxy = true
		} else if len(job.Data.Proxies) > 0 {
			opts = append(opts,
				scrapemateapp.WithProxies(job.Data.Proxies),
			)
			hasProxy = true
		}

		if !cfg.DisablePageReuse {
			opts = append(opts,
				scrapemateapp.WithPageReuseLimit(2),
				scrapemateapp.WithBrowserReuseLimit(200),
			)
		}

		log.Printf("job %s has proxy: %v", job.ID, hasProxy)

		resultWriter, err := newResultWriter(writer)
		if err != nil {
			return nil, err
		}

		// When the job covers specific areas, keep the output to those areas.
		if boxes := job.Data.Areas(); len(boxes) > 0 {
			parsed := make([]grid.BoundingBox, 0, len(boxes))

			for _, b := range boxes {
				if bbox, err := grid.ParseBoundingBox(b); err == nil {
					parsed = append(parsed, bbox)
				}
			}

			if len(parsed) > 0 {
				resultWriter = newAreaFilterWriter(resultWriter, parsed...)
			}
		}

		writers := []scrapemate.ResultWriter{resultWriter}

		matecfg, err := scrapemateapp.NewConfig(
			writers,
			opts...,
		)
		if err != nil {
			return nil, err
		}

		return scrapemateapp.NewScrapeMateApp(matecfg)
	}
}
