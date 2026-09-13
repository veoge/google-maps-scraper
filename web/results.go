package web

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/gosom/google-maps-scraper/export"
	"github.com/gosom/google-maps-scraper/gmaps"
)

// maxDetailReviews caps how many reviews travel with each row. The browser only
// shows a preview in the detail panel, and sending every review of every result
// would turn a modest job into a multi-megabyte response.
const maxDetailReviews = 6

// ResultRow is one scraped business, shaped for the results table in the browser.
// It deliberately flattens the nested JSON that makes the raw CSV unreadable.
type ResultRow struct {
	Title      string   `json:"title"`
	Keyword    string   `json:"keyword"`
	Category   string   `json:"category"`
	Rating     float64  `json:"rating"`
	Reviews    int      `json:"reviews"`
	Phone      string   `json:"phone"`
	Website    string   `json:"website"`
	Emails     []string `json:"emails"`
	Social     []string `json:"social"`
	Facebook   string   `json:"facebook"`
	Instagram  string   `json:"instagram"`
	LinkedIn   string   `json:"linkedin"`
	TwitterX   string   `json:"twitter_x"`
	YouTube    string   `json:"youtube"`
	TikTok     string   `json:"tiktok"`
	WhatsApp   string   `json:"whatsapp"`
	Address    string   `json:"address"`
	City       string   `json:"city"`
	Country    string   `json:"country"`
	Latitude   float64  `json:"latitude"`
	Longitude  float64  `json:"longitude"`
	Link       string   `json:"link"`
	Thumbnail  string   `json:"thumbnail"`
	PriceRange string   `json:"price_range"`
	Status     string   `json:"status"`
	PlaceID    string   `json:"place_id"`
	CID        string   `json:"cid"`

	Hours       [][2]string `json:"hours"`
	Breakdown   [5]int      `json:"breakdown"`
	Amenities   []string    `json:"amenities"`
	TopReviews  []ReviewRow `json:"top_reviews"`
	Description string      `json:"description"`
}

// ReviewRow is a single review as shown in the detail panel.
type ReviewRow struct {
	Author string `json:"author"`
	Rating int    `json:"rating"`
	When   string `json:"when"`
	Text   string `json:"text"`
}

// ResultsResponse is what the results endpoint returns: the rows plus the
// headline numbers the UI shows above the table.
type ResultsResponse struct {
	Total      int         `json:"total"`
	WithPhone  int         `json:"with_phone"`
	WithEmail  int         `json:"with_email"`
	WithSocial int         `json:"with_social"`
	AvgRating  float64     `json:"avg_rating"`
	Keywords   []string    `json:"keywords"`
	Categories []string    `json:"categories"`
	Cities     []string    `json:"cities"`
	Rows       []ResultRow `json:"rows"`
}

// GetResults loads a finished job's output and shapes it for the browser.
func (s *Service) GetResults(_ context.Context, id string) (ResultsResponse, error) {
	path, err := s.csvPath(id)
	if err != nil {
		return ResultsResponse{}, err
	}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ResultsResponse{}, fmt.Errorf("no results for job %s: %w", id, ErrPlacesNotFound)
		}

		return ResultsResponse{}, err
	}

	defer func() {
		_ = f.Close()
	}()

	entries, err := export.EntriesFromCSV(f)
	if err != nil {
		return ResultsResponse{}, err
	}

	return buildResults(entries), nil
}

// LoadEntries returns a job's raw entries, used by the download endpoint to
// re-render the results in another format.
func (s *Service) LoadEntries(_ context.Context, id string) ([]gmaps.Entry, error) {
	path, err := s.csvPath(id)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no results for job %s: %w", id, ErrPlacesNotFound)
		}

		return nil, err
	}

	defer func() {
		_ = f.Close()
	}()

	return export.EntriesFromCSV(f)
}

func buildResults(entries []gmaps.Entry) ResultsResponse {
	out := ResultsResponse{
		Rows:  make([]ResultRow, 0, len(entries)),
		Total: len(entries),
	}

	categories := map[string]bool{}
	cities := map[string]bool{}
	keywords := map[string]bool{}

	var ratingSum float64

	var rated int

	for i := range entries {
		e := &entries[i]
		row := rowFromEntry(e)

		if len(row.Emails) > 0 {
			out.WithEmail++
		}

		if len(row.Social) > 0 {
			out.WithSocial++
		}

		if row.Phone != "" {
			out.WithPhone++
		}

		if row.Rating > 0 {
			ratingSum += row.Rating
			rated++
		}

		if row.Category != "" {
			categories[row.Category] = true
		}

		if row.Keyword != "" {
			keywords[row.Keyword] = true
		}

		if row.City != "" {
			cities[row.City] = true
		}

		out.Rows = append(out.Rows, row)
	}

	if rated > 0 {
		out.AvgRating = ratingSum / float64(rated)
	}

	out.Keywords = sortedKeys(keywords)
	out.Categories = sortedKeys(categories)
	out.Cities = sortedKeys(cities)

	return out
}

//nolint:funlen // a flat field mapping; splitting it would not make it clearer.
func rowFromEntry(e *gmaps.Entry) ResultRow {
	row := ResultRow{
		Title:       e.Title,
		Keyword:     e.Keyword,
		Category:    e.Category,
		Rating:      e.ReviewRating,
		Reviews:     e.ReviewCount,
		Phone:       e.Phone,
		Website:     e.WebSite,
		Emails:      e.Emails,
		Social:      e.Socials.All(),
		Facebook:    e.Socials.Facebook,
		Instagram:   e.Socials.Instagram,
		LinkedIn:    e.Socials.LinkedIn,
		TwitterX:    e.Socials.TwitterX,
		YouTube:     e.Socials.YouTube,
		TikTok:      e.Socials.TikTok,
		WhatsApp:    e.Socials.WhatsApp,
		Address:     e.Address,
		City:        e.CompleteAddress.City,
		Country:     e.CompleteAddress.Country,
		Latitude:    e.Latitude,
		Longitude:   e.Longtitude,
		Link:        e.Link,
		Thumbnail:   e.Thumbnail,
		PriceRange:  e.PriceRange,
		Status:      e.Status,
		PlaceID:     e.PlaceID,
		CID:         e.Cid,
		Description: e.Description,
	}

	for star := 1; star <= 5; star++ {
		row.Breakdown[star-1] = e.ReviewsPerRating[star]
	}

	for _, day := range []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"} {
		if slots, ok := e.OpenHours[day]; ok {
			row.Hours = append(row.Hours, [2]string{day, strings.Join(slots, ", ")})
		}
	}

	for i := range e.About {
		var enabled []string

		for _, opt := range e.About[i].Options {
			if opt.Enabled {
				enabled = append(enabled, opt.Name)
			}
		}

		if len(enabled) > 0 {
			row.Amenities = append(row.Amenities, e.About[i].Name+": "+strings.Join(enabled, ", "))
		}
	}

	row.TopReviews = topReviews(e)

	return row
}

// topReviews returns the most substantial reviews first, so the detail panel
// leads with text rather than with bare star ratings.
func topReviews(e *gmaps.Entry) []ReviewRow {
	all := append(append([]gmaps.Review{}, e.UserReviews...), e.UserReviewsExtended...)

	rows := make([]ReviewRow, 0, len(all))
	seen := map[string]bool{}

	for i := range all {
		rv := &all[i]

		if rv.ReviewID != "" {
			if seen[rv.ReviewID] {
				continue
			}

			seen[rv.ReviewID] = true
		}

		text := rv.TextOriginal
		if text == "" {
			text = rv.Description
		}

		rows = append(rows, ReviewRow{
			Author: rv.Name,
			Rating: rv.Rating,
			When:   rv.When,
			Text:   text,
		})
	}

	sort.SliceStable(rows, func(i, j int) bool {
		return len(rows[i].Text) > len(rows[j].Text)
	})

	if len(rows) > maxDetailReviews {
		rows = rows[:maxDetailReviews]
	}

	return rows
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}

	sort.Strings(out)

	return out
}
