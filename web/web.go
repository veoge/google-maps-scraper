package web

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/gosom/google-maps-scraper/export"
	"github.com/gosom/google-maps-scraper/geo"
)

//go:embed static
var static embed.FS

type Server struct {
	tmpl map[string]*template.Template
	srv  *http.Server
	svc  *Service
}

func New(svc *Service, addr string) (*Server, error) {
	ans := Server{
		svc:  svc,
		tmpl: make(map[string]*template.Template),
		srv: &http.Server{
			Addr:              addr,
			ReadHeaderTimeout: 10 * time.Second,
			ReadTimeout:       60 * time.Second,
			WriteTimeout:      60 * time.Second,
			IdleTimeout:       120 * time.Second,
			MaxHeaderBytes:    1 << 20,
		},
	}

	staticFS, err := fs.Sub(static, "static")
	if err != nil {
		return nil, err
	}

	fileServer := http.FileServer(http.FS(staticFS))
	mux := http.NewServeMux()

	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))
	mux.HandleFunc("/scrape", ans.scrape)
	mux.HandleFunc("/download", func(w http.ResponseWriter, r *http.Request) {
		r = requestWithID(r)

		ans.download(w, r)
	})
	mux.HandleFunc("/delete", func(w http.ResponseWriter, r *http.Request) {
		r = requestWithID(r)

		ans.delete(w, r)
	})
	mux.HandleFunc("/jobs", ans.getJobs)
	mux.HandleFunc("/view", func(w http.ResponseWriter, r *http.Request) {
		r = requestWithID(r)

		ans.viewJob(w, r)
	})
	mux.HandleFunc("/continue", func(w http.ResponseWriter, r *http.Request) {
		r = requestWithID(r)

		ans.resume(w, r)
	})
	mux.HandleFunc("/restart", func(w http.ResponseWriter, r *http.Request) {
		r = requestWithID(r)

		ans.restart(w, r)
	})
	mux.HandleFunc("/api/v1/locations", ans.locations)
	mux.HandleFunc("/results", func(w http.ResponseWriter, r *http.Request) {
		r = requestWithID(r)

		ans.results(w, r)
	})
	mux.HandleFunc("/", ans.index)

	// api routes
	mux.HandleFunc("/api/docs", ans.redocHandler)
	mux.HandleFunc("/api/v1/jobs", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			ans.apiScrape(w, r)
		case http.MethodGet:
			ans.apiGetJobs(w, r)
		default:
			ans := apiError{
				Code:    http.StatusMethodNotAllowed,
				Message: "Method not allowed",
			}

			renderJSON(w, http.StatusMethodNotAllowed, ans)
		}
	})

	mux.HandleFunc("/api/v1/jobs/{id}", func(w http.ResponseWriter, r *http.Request) {
		r = requestWithID(r)

		switch r.Method {
		case http.MethodGet:
			ans.apiGetJob(w, r)
		case http.MethodDelete:
			ans.apiDeleteJob(w, r)
		default:
			ans := apiError{
				Code:    http.StatusMethodNotAllowed,
				Message: "Method not allowed",
			}

			renderJSON(w, http.StatusMethodNotAllowed, ans)
		}
	})

	mux.HandleFunc("/api/v1/jobs/{id}/results", func(w http.ResponseWriter, r *http.Request) {
		r = requestWithID(r)

		ans.results(w, r)
	})

	mux.HandleFunc("/api/v1/jobs/{id}/download", func(w http.ResponseWriter, r *http.Request) {
		r = requestWithID(r)

		if r.Method != http.MethodGet {
			ans := apiError{
				Code:    http.StatusMethodNotAllowed,
				Message: "Method not allowed",
			}

			renderJSON(w, http.StatusMethodNotAllowed, ans)

			return
		}

		ans.download(w, r)
	})

	handler := securityHeaders(mux)
	ans.srv.Handler = handler

	tmplsKeys := []string{
		"static/templates/index.html",
		"static/templates/job_rows.html",
		"static/templates/job_row.html",
		"static/templates/job_view.html",
		"static/templates/redoc.html",
	}

	for _, key := range tmplsKeys {
		tmp, err := template.ParseFS(static, key)
		if err != nil {
			return nil, err
		}

		ans.tmpl[key] = tmp
	}

	return &ans, nil
}

func (s *Server) Start(ctx context.Context) error {
	go func() {
		<-ctx.Done()

		err := s.srv.Shutdown(context.Background())
		if err != nil {
			log.Println(err)

			return
		}

		log.Println("server stopped")
	}()

	fmt.Fprintf(os.Stderr, "visit http://localhost%s\n", s.srv.Addr)

	err := s.srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

type formData struct {
	Name     string
	MaxTime  string
	Keywords []string
	Language string
	Zoom     int
	FastMode bool
	Radius   int
	Lat      string
	Lon      string
	Depth    int
	Email    bool
	Proxies  []string
}

type ctxKey string

const idCtxKey ctxKey = "id"

func requestWithID(r *http.Request) *http.Request {
	id := r.PathValue("id")
	if id == "" {
		id = r.URL.Query().Get("id")
	}

	parsed, err := uuid.Parse(id)
	if err == nil {
		r = r.WithContext(context.WithValue(r.Context(), idCtxKey, parsed))
	}

	return r
}

func getIDFromRequest(r *http.Request) (uuid.UUID, bool) {
	id, ok := r.Context().Value(idCtxKey).(uuid.UUID)

	return id, ok
}

//nolint:gocritic // this is used in template
func (f formData) ProxiesString() string {
	return strings.Join(f.Proxies, "\n")
}

//nolint:gocritic // this is used in template
func (f formData) KeywordsString() string {
	return strings.Join(f.Keywords, "\n")
}

// renderTemplate writes a template to the response, turning a failure into a
// logged 500 instead of a silently blank page. Executing straight into the
// ResponseWriter cannot do that: by the time it fails, a partial body is already
// on the wire and the status is fixed.
func renderTemplate(w http.ResponseWriter, tmpl *template.Template, name string, data any) {
	var buf bytes.Buffer

	if err := tmpl.Execute(&buf, data); err != nil {
		log.Printf("render %s: %v", name, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)

		return
	}

	_, _ = buf.WriteTo(w)
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

		return
	}

	tmpl, ok := s.tmpl["static/templates/index.html"]
	if !ok {
		http.Error(w, "missing tpl", http.StatusInternalServerError)

		return
	}

	data := formData{
		Name:     "",
		MaxTime:  "10m",
		Keywords: []string{},
		Language: "en",
		Zoom:     15,
		FastMode: false,
		Radius:   10000,
		Lat:      "0",
		Lon:      "0",
		Depth:    10,
		// Website enrichment is on by default: it is the only pass that visits a
		// business's own site, and therefore the only source of email addresses
		// and social profiles. It can be switched off per job in the form.
		Email: true,
	}

	renderTemplate(w, tmpl, "index", data)
}

// splitKeywords accepts searches separated by newlines or commas, because both
// are natural ways to type "restaurants, coffee shops, supermarkets". The place
// name no longer needs to appear in the query now that country and city are
// chosen separately, so a comma is a separator rather than part of a search.
func splitKeywords(raw string) []string {
	out := []string{}

	for _, line := range strings.Split(raw, "\n") {
		for _, part := range strings.Split(line, ",") {
			if k := strings.TrimSpace(part); k != "" {
				out = append(out, k)
			}
		}
	}

	return out
}

func (s *Server) scrape(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	newJob := Job{
		ID:     uuid.New().String(),
		Name:   r.Form.Get("name"),
		Date:   time.Now().UTC(),
		Status: StatusPending,
		Data:   JobData{},
	}

	maxTimeStr := r.Form.Get("maxtime")

	maxTime, err := time.ParseDuration(maxTimeStr)
	if err != nil {
		http.Error(w, "invalid max time", http.StatusUnprocessableEntity)

		return
	}

	if maxTime < time.Minute*3 {
		http.Error(w, "max time must be more than 3m", http.StatusUnprocessableEntity)

		return
	}

	newJob.Data.MaxTime = maxTime

	keywordsStr, ok := r.Form["keywords"]
	if !ok {
		http.Error(w, "missing keywords", http.StatusUnprocessableEntity)

		return
	}

	newJob.Data.Keywords = splitKeywords(keywordsStr[0])

	newJob.Data.Lang = r.Form.Get("lang")

	newJob.Data.Zoom, err = strconv.Atoi(r.Form.Get("zoom"))
	if err != nil {
		http.Error(w, "invalid zoom", http.StatusUnprocessableEntity)

		return
	}

	if r.Form.Get("fastmode") == "on" {
		newJob.Data.FastMode = true
	}

	newJob.Data.Radius, err = strconv.Atoi(r.Form.Get("radius"))
	if err != nil {
		http.Error(w, "invalid radius", http.StatusUnprocessableEntity)

		return
	}

	newJob.Data.Lat = r.Form.Get("latitude")
	newJob.Data.Lon = r.Form.Get("longitude")

	newJob.Data.Depth, err = strconv.Atoi(r.Form.Get("depth"))
	if err != nil {
		http.Error(w, "invalid depth", http.StatusUnprocessableEntity)

		return
	}

	newJob.Data.Email = r.Form.Get("email") == "on"

	if err := applyLocation(&newJob.Data, r.Form.Get("country"), r.Form.Get("city"), r.Form.Get("coverage")); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)

		return
	}

	proxies := strings.Split(r.Form.Get("proxies"), "\n")
	if len(proxies) > 0 {
		for _, p := range proxies {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}

			newJob.Data.Proxies = append(newJob.Data.Proxies, p)
		}
	}

	err = newJob.Validate()
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)

		return
	}

	err = s.svc.Create(r.Context(), &newJob)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	tmpl, ok := s.tmpl["static/templates/job_row.html"]
	if !ok {
		http.Error(w, "missing tpl", http.StatusInternalServerError)

		return
	}

	renderTemplate(w, tmpl, "job_row", newJob)
}

func (s *Server) getJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

		return
	}

	tmpl, ok := s.tmpl["static/templates/job_rows.html"]
	if !ok {
		http.Error(w, "missing tpl", http.StatusInternalServerError)
		return
	}

	jobs, err := s.svc.All(context.Background())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	renderTemplate(w, tmpl, "job_rows", jobs)
}

func (s *Server) download(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

		return
	}

	ctx := r.Context()

	id, ok := getIDFromRequest(r)
	if !ok {
		http.Error(w, "Invalid ID", http.StatusUnprocessableEntity)

		return
	}

	switch strings.ToLower(r.URL.Query().Get("format")) {
	case "xlsx", "excel":
		s.downloadXLSX(w, r, id.String())
	case "json":
		s.downloadJSON(w, r, id.String())
	default:
		s.downloadCSV(ctx, w, id.String())
	}
}

// downloadCSV streams the job's CSV with a UTF-8 byte order mark. Without the
// BOM, Excel on Windows opens the file using the system ANSI code page, which
// renders every non-Latin name and every typographic dash as mojibake
// ("Ù…Ø·Ø¹Ù…" instead of Arabic). The bytes on disk were always correct; only
// Excel's guess was wrong.
func (s *Server) downloadCSV(ctx context.Context, w http.ResponseWriter, id string) {
	filePath, err := s.svc.GetCSV(ctx, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)

		return
	}

	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "Failed to open file", http.StatusInternalServerError)

		return
	}

	defer func() {
		_ = file.Close()
	}()

	fileName := filepath.Base(filePath)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")

	if _, err := w.Write([]byte(export.BOM)); err != nil {
		return
	}

	if _, err = io.Copy(w, file); err != nil {
		log.Printf("download csv %s: %v", id, err)
	}
}

// downloadXLSX renders the job's results as a formatted workbook. The file is
// built on demand from the stored CSV so that jobs scraped before this format
// existed can still be exported.
func (s *Server) downloadXLSX(w http.ResponseWriter, r *http.Request, id string) {
	entries, err := s.svc.LoadEntries(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)

		return
	}

	var buf bytes.Buffer

	if err := export.BuildWorkbook(entries, &buf); err != nil {
		log.Printf("download xlsx %s: %v", id, err)
		http.Error(w, "failed to build workbook", http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.xlsx", id))
	w.Header().Set("Content-Type",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))

	_, _ = buf.WriteTo(w)
}

func (s *Server) downloadJSON(w http.ResponseWriter, r *http.Request, id string) {
	entries, err := s.svc.LoadEntries(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)

		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.json", id))
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	_ = enc.Encode(entries)
}

// coverageCellKm maps the coverage choice offered in the UI onto a grid cell
// size. Smaller cells mean more searches and better coverage of dense areas.
//
//nolint:gochecknoglobals // lookup table, read-only after init.
var coverageCellKm = map[string]float64{
	"quick":    6.0,
	"balanced": 3.0,
	"thorough": 1.5,
}

// coverageSingle runs one search per term anchored on the city instead of a grid.
// A grid over a whole city multiplied by a dozen categories runs for days, so this
// is the sweep to start with: far faster, at the cost of only seeing what Google
// returns for a single search rather than every side street.
const coverageSingle = "single"

// citySweepZoom is the map zoom used for a single sweep. The form default (15) is
// street level and would only cover a neighbourhood; 12 frames the whole city.
const citySweepZoom = 12

// applyLocation turns the country and city choices into a bounding box, so the
// job covers the whole selected city rather than a radius around its centre.
// A country with no city selected leaves the box unset and falls back to a plain
// keyword search, because grid-scraping an entire country is rarely what anyone
// means and would run for days.
func applyLocation(data *JobData, countryCode, cityName, coverage string) error {
	data.Country = countryCode
	data.City = cityName
	data.Coverage = strings.ToLower(strings.TrimSpace(coverage))

	if countryCode == "" {
		return nil
	}

	country, city, ok := geo.Lookup(countryCode, cityName)
	if !ok {
		return fmt.Errorf("unknown location: %s / %s", countryCode, cityName)
	}

	data.Country = country.Code

	cell, known := coverageCellKm[strings.ToLower(coverage)]
	if !known {
		cell = coverageCellKm["balanced"]
		data.Coverage = "balanced"
	}

	// No city means every city in the country, each gridded in turn.
	if cityName == "" {
		if strings.EqualFold(coverage, coverageSingle) {
			return nil
		}

		data.BBoxes = make([]string, 0, len(country.Cities))
		for _, c := range country.Cities {
			data.BBoxes = append(data.BBoxes, c.BBoxString())
		}

		data.CellSizeKm = cell

		return nil
	}

	data.City = city.Name
	data.Lat = strconv.FormatFloat(city.Lat, 'f', 6, 64)
	data.Lon = strconv.FormatFloat(city.Lon, 'f', 6, 64)

	if strings.EqualFold(coverage, coverageSingle) {
		// No bounding box means no grid: the job falls back to one search per term,
		// anchored on the city centre at a zoom that frames the city.
		data.BBox = ""
		data.CellSizeKm = 0
		data.Zoom = citySweepZoom

		return nil
	}

	data.BBox = city.BBoxString()
	data.CellSizeKm = cell

	return nil
}

// locations serves the country and city list that populates the dropdowns.
func (s *Server) locations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

		return
	}

	renderJSON(w, http.StatusOK, map[string]any{
		"default_country": geo.DefaultCountry,
		"countries":       geo.Countries(),
	})
}

// results returns a job's scraped rows as JSON so the browser can show them in a
// table instead of forcing a download to see anything at all.
func (s *Server) results(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

		return
	}

	id, ok := getIDFromRequest(r)
	if !ok {
		http.Error(w, "Invalid ID", http.StatusUnprocessableEntity)

		return
	}

	res, err := s.svc.GetResults(r.Context(), id.String())
	if err != nil {
		if errors.Is(err, ErrPlacesNotFound) {
			renderJSON(w, http.StatusOK, ResultsResponse{Rows: []ResultRow{}})

			return
		}

		log.Printf("results %s: %v", id, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)

		return
	}

	renderJSON(w, http.StatusOK, res)
}

// resume puts a paused job back in the queue. Its progress marker is kept, so
// the worker carries on from where the outage stopped it rather than starting
// the grid again.
func (s *Server) resume(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

		return
	}

	id, ok := getIDFromRequest(r)
	if !ok {
		http.Error(w, "Invalid ID", http.StatusUnprocessableEntity)

		return
	}

	job, err := s.svc.Get(r.Context(), id.String())
	if err != nil {
		http.Error(w, "job not found", http.StatusNotFound)

		return
	}

	if job.Status != StatusPaused {
		http.Error(w, "only a paused scan can be continued", http.StatusConflict)

		return
	}

	job.Status = StatusPending
	job.Data.PausedReason = ""

	if err := s.svc.Update(r.Context(), &job); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	tmpl, ok := s.tmpl["static/templates/job_row.html"]
	if !ok {
		http.Error(w, "missing tpl", http.StatusInternalServerError)

		return
	}

	renderTemplate(w, tmpl, "job_row", job)
}

// restart queues a fresh job with the same settings as an existing one. The
// original is left untouched: its results stay downloadable while the repeat
// run collects a new set.
func (s *Server) restart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

		return
	}

	id, ok := getIDFromRequest(r)
	if !ok {
		http.Error(w, "Invalid ID", http.StatusUnprocessableEntity)

		return
	}

	original, err := s.svc.Get(r.Context(), id.String())
	if err != nil {
		http.Error(w, "job not found", http.StatusNotFound)

		return
	}

	repeat := Job{
		ID:     uuid.New().String(),
		Name:   original.Name,
		Date:   time.Now().UTC(),
		Status: StatusPending,
		Data:   original.Data,
	}

	// Timings belong to the run that produced them, not to the repeat.
	repeat.Data.StartedAt = nil
	repeat.Data.FinishedAt = nil

	if err := repeat.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)

		return
	}

	if err := s.svc.Create(r.Context(), &repeat); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	tmpl, ok := s.tmpl["static/templates/job_row.html"]
	if !ok {
		http.Error(w, "missing tpl", http.StatusInternalServerError)

		return
	}

	renderTemplate(w, tmpl, "job_row", repeat)
}

func (s *Server) delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

		return
	}

	deleteID, ok := getIDFromRequest(r)
	if !ok {
		http.Error(w, "Invalid ID", http.StatusUnprocessableEntity)

		return
	}

	err := s.svc.Delete(r.Context(), deleteID.String())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusOK)
}

type apiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type apiScrapeRequest struct {
	Name string
	JobData
}

type apiScrapeResponse struct {
	ID string `json:"id"`
}

func (s *Server) redocHandler(w http.ResponseWriter, _ *http.Request) {
	tmpl, ok := s.tmpl["static/templates/redoc.html"]
	if !ok {
		http.Error(w, "missing tpl", http.StatusInternalServerError)

		return
	}

	renderTemplate(w, tmpl, "redoc", nil)
}

func (s *Server) apiScrape(w http.ResponseWriter, r *http.Request) {
	var req apiScrapeRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		ans := apiError{
			Code:    http.StatusUnprocessableEntity,
			Message: err.Error(),
		}

		renderJSON(w, http.StatusUnprocessableEntity, ans)

		return
	}

	newJob := Job{
		ID:     uuid.New().String(),
		Name:   req.Name,
		Date:   time.Now().UTC(),
		Status: StatusPending,
		Data:   req.JobData,
	}

	// convert to seconds
	newJob.Data.MaxTime *= time.Second

	err = newJob.Validate()
	if err != nil {
		ans := apiError{
			Code:    http.StatusUnprocessableEntity,
			Message: err.Error(),
		}

		renderJSON(w, http.StatusUnprocessableEntity, ans)

		return
	}

	err = s.svc.Create(r.Context(), &newJob)
	if err != nil {
		ans := apiError{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		}

		renderJSON(w, http.StatusInternalServerError, ans)

		return
	}

	ans := apiScrapeResponse{
		ID: newJob.ID,
	}

	renderJSON(w, http.StatusCreated, ans)
}

func (s *Server) apiGetJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := s.svc.All(r.Context())
	if err != nil {
		apiError := apiError{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		}

		renderJSON(w, http.StatusInternalServerError, apiError)

		return
	}

	renderJSON(w, http.StatusOK, jobs)
}

func (s *Server) apiGetJob(w http.ResponseWriter, r *http.Request) {
	id, ok := getIDFromRequest(r)
	if !ok {
		apiError := apiError{
			Code:    http.StatusUnprocessableEntity,
			Message: "Invalid ID",
		}

		renderJSON(w, http.StatusUnprocessableEntity, apiError)

		return
	}

	job, err := s.svc.Get(r.Context(), id.String())
	if err != nil {
		apiError := apiError{
			Code:    http.StatusNotFound,
			Message: http.StatusText(http.StatusNotFound),
		}

		renderJSON(w, http.StatusNotFound, apiError)

		return
	}

	renderJSON(w, http.StatusOK, job)
}

// viewJob renders the map modal fragment for a job, embedding the job's places
// directly so the client needs no separate data request.
func (s *Server) viewJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

		return
	}

	id, ok := getIDFromRequest(r)
	if !ok {
		http.Error(w, "Invalid ID", http.StatusUnprocessableEntity)

		return
	}

	places, err := s.svc.GetPlaces(r.Context(), id.String())

	if err != nil {
		if !errors.Is(err, ErrPlacesNotFound) {
			log.Printf("view job %s: %v", id, err)
			http.Error(w, "internal server error", http.StatusInternalServerError)

			return
		}

		// No CSV yet: render the modal with an empty state rather than an error.
		places = []Place{}
	}

	tmpl, ok := s.tmpl["static/templates/job_view.html"]
	if !ok {
		http.Error(w, "missing tpl", http.StatusInternalServerError)

		return
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, places); err != nil {
		log.Printf("view job %s: render: %v", id, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)

		return
	}

	_, _ = buf.WriteTo(w)
}

func (s *Server) apiDeleteJob(w http.ResponseWriter, r *http.Request) {
	id, ok := getIDFromRequest(r)
	if !ok {
		apiError := apiError{
			Code:    http.StatusUnprocessableEntity,
			Message: "Invalid ID",
		}

		renderJSON(w, http.StatusUnprocessableEntity, apiError)

		return
	}

	err := s.svc.Delete(r.Context(), id.String())
	if err != nil {
		apiError := apiError{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		}

		renderJSON(w, http.StatusInternalServerError, apiError)

		return
	}

	w.WriteHeader(http.StatusOK)
}

func renderJSON(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	_ = json.NewEncoder(w).Encode(data)
}

func formatDate(t time.Time) string {
	return t.Format("Jan 02, 2006 15:04:05")
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' cdn.redoc.ly cdnjs.cloudflare.com 'unsafe-inline' 'unsafe-eval'; "+
				"worker-src 'self' blob:; "+
				"style-src 'self' 'unsafe-inline' fonts.googleapis.com cdnjs.cloudflare.com; "+
				// Business photos and Street View thumbnails are served from Google's
				// image hosts. Without them the detail panel shows broken images.
				"img-src 'self' data: cdn.redoc.ly cdnjs.cloudflare.com "+
				"*.tile.openstreetmap.org *.googleusercontent.com "+
				"streetviewpixels-pa.googleapis.com maps.gstatic.com; "+
				"font-src 'self' fonts.gstatic.com; "+
				"connect-src 'self'")

		next.ServeHTTP(w, r)
	})
}
