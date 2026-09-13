package web

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gosom/google-maps-scraper/grid"
)

var jobs []Job

const (
	StatusPending = "pending"
	StatusWorking = "working"
	StatusOK      = "ok"
	StatusFailed  = "failed"
	// StatusPaused means the run stopped on purpose and can be continued. It is
	// used when the network drops: retrying into a dead connection just burns
	// through the retry budget and silently skips work.
	StatusPaused = "paused"
)

type SelectParams struct {
	Status string
	Limit  int
}

type JobRepository interface {
	Get(context.Context, string) (Job, error)
	Create(context.Context, *Job) error
	Delete(context.Context, string) error
	Select(context.Context, SelectParams) ([]Job, error)
	Update(context.Context, *Job) error
}

type Job struct {
	ID     string
	Name   string
	Date   time.Time
	Status string
	Data   JobData

	// UpdatedAt is the last time the row was written. For a job that finished
	// before run timings were recorded it is the best available end time.
	UpdatedAt time.Time
}

// MarkStarted records when the worker actually picked the job up. This is not the
// same as when it was created: the worker runs one job at a time, so a job can sit
// pending for hours, and counting that wait as run time would be wrong.
func (j *Job) MarkStarted() {
	now := time.Now().UTC()
	j.Status = StatusWorking
	j.Data.StartedAt = &now
}

// MarkPaused stops the run in a state that can be picked up again, keeping the
// results collected so far and remembering where to continue from.
func (j *Job) MarkPaused(reason string, progress int) {
	j.Status = StatusPaused
	j.Data.PausedReason = reason
	j.Data.Progress = progress
	j.Data.FinishedAt = nil
}

// MarkFinished records the end of a run, whatever the outcome.
func (j *Job) MarkFinished(status string) {
	now := time.Now().UTC()
	j.Status = status
	j.Data.FinishedAt = &now
}

// Duration reports how long the job ran, and whether that is known. A running job
// reports the time so far.
func (j *Job) Duration() (time.Duration, bool) {
	start := j.startedAt()
	if start.IsZero() {
		return 0, false
	}

	if j.Data.FinishedAt != nil {
		return j.Data.FinishedAt.Sub(start), true
	}

	if j.Status == StatusOK || j.Status == StatusFailed || j.Status == StatusPaused {
		// Finished before timings were recorded: the last write is the end.
		if j.UpdatedAt.After(start) {
			return j.UpdatedAt.Sub(start), true
		}

		return 0, false
	}

	return time.Since(start), true
}

func (j *Job) startedAt() time.Time {
	if j.Data.StartedAt != nil {
		return *j.Data.StartedAt
	}

	// Jobs from before StartedAt existed fall back to their creation time.
	return j.Date
}

// StartedUnix and DurationSeconds feed the job table, which renders the elapsed
// time in the browser so a running job can tick without reloading the page.
//
//nolint:gocritic // used from templates.
func (j Job) StartedUnix() int64 {
	start := j.startedAt()
	if start.IsZero() {
		return 0
	}

	return start.Unix()
}

//nolint:gocritic // used from templates.
func (j Job) DurationSeconds() int64 {
	if j.Status != StatusOK && j.Status != StatusFailed && j.Status != StatusPaused {
		return 0
	}

	d, ok := j.Duration()
	if !ok {
		return 0
	}

	return int64(d.Seconds())
}

func (j *Job) Validate() error {
	if j.ID == "" {
		return errors.New("missing id")
	}

	if j.Name == "" {
		return errors.New("missing name")
	}

	if j.Status == "" {
		return errors.New("missing status")
	}

	if j.Date.IsZero() {
		return errors.New("missing date")
	}

	if err := j.Data.Validate(); err != nil {
		return err
	}

	return nil
}

type JobData struct {
	Keywords     []string      `json:"keywords"`
	Lang         string        `json:"lang"`
	Country      string        `json:"country"`
	City         string        `json:"city"`
	// Coverage records which option produced BBox/CellSizeKm, so the form can be
	// restored exactly rather than guessed at from the cell size.
	Coverage string `json:"coverage"`
	// BBox is "minLat,minLon,maxLat,maxLon". When set, the job grid-scrapes the
	// whole area instead of searching outwards from a single point, which is
	// what makes "everything in Amman" mean the city rather than a radius.
	BBox string `json:"bbox"`
	// BBoxes carries one box per city when a whole country is selected. "All
	// cities" has to mean every city actually gets searched, not a single search
	// with no location at all.
	BBoxes     []string `json:"bboxes,omitempty"`
	CellSizeKm float64  `json:"cell_size_km"`
	Zoom         int           `json:"zoom"`
	Lat          string        `json:"lat"`
	Lon          string        `json:"lon"`
	FastMode     bool          `json:"fast_mode"`
	Radius       int           `json:"radius"`
	Depth        int           `json:"depth"`
	Email        bool          `json:"email"`
	ExtraReviews bool          `json:"extra_reviews"`
	MaxTime      time.Duration `json:"max_time"`
	Proxies      []string      `json:"proxies"`

	// Run timings live alongside the configuration because the jobs table stores
	// this struct as a single JSON blob; adding them here avoids a schema change
	// while still recording when a run actually began and ended.
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`

	// Progress is how many seed searches have been dispatched. A continued run
	// resumes from here instead of starting the grid again.
	Progress int `json:"progress,omitempty"`
	// PausedReason explains the stop to whoever has to decide about continuing.
	PausedReason string `json:"paused_reason,omitempty"`
}

// Areas returns every bounding box the job should grid, which is one per city
// for a whole-country run and a single box for one city. An empty result means
// no grid: the job falls back to a plain keyword search.
func (d *JobData) Areas() []string {
	if len(d.BBoxes) > 0 {
		return d.BBoxes
	}

	if d.BBox != "" {
		return []string{d.BBox}
	}

	return nil
}

func (d *JobData) Validate() error {
	if len(d.Keywords) == 0 {
		return errors.New("missing keywords")
	}

	if d.Lang == "" {
		return errors.New("missing lang")
	}

	if len(d.Lang) != 2 {
		return errors.New("invalid lang")
	}

	if d.Depth == 0 {
		return errors.New("missing depth")
	}

	if d.MaxTime == 0 {
		return errors.New("missing max time")
	}

	if d.FastMode && (d.Lat == "" || d.Lon == "") {
		return errors.New("missing geo coordinates")
	}

	areas := d.Areas()

	if len(areas) > 0 {
		if d.FastMode {
			return errors.New("fast mode cannot cover a whole city; turn one of them off")
		}

		for _, a := range areas {
			if _, err := grid.ParseBoundingBox(a); err != nil {
				return fmt.Errorf("invalid area: %w", err)
			}
		}
	}

	return nil
}
