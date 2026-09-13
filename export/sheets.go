package export

import (
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/gosom/google-maps-scraper/gmaps"
)

// writeReviews gives every review its own row. In the CSV all reviews for a
// business are crammed into a single JSON cell, which is the main reason the
// export was unreadable in a spreadsheet.
func (b *builder) writeReviews(entries []gmaps.Entry) error {
	cols := []struct {
		header string
		width  float64
		format cellFormat
	}{
		{"Business", 32, fmtText},
		{"City", 18, fmtText},
		{"Rating", 8, fmtRating},
		{"Author", 24, fmtText},
		{"When", 16, fmtText},
		{"Published", 20, fmtText},
		{"Language", 10, fmtText},
		{"Review", 90, fmtWrap},
		{"Translated", 90, fmtWrap},
		{"Owner Reply", 60, fmtWrap},
		{"Author Profile", 34, fmtURL},
	}

	names := make([]string, len(cols))
	for i := range cols {
		names[i] = cols[i].header
	}

	if err := b.writeHeaderRow(SheetReviews, names); err != nil {
		return err
	}

	for i := range cols {
		colName, err := excelize.ColumnNumberToName(i + 1)
		if err != nil {
			return err
		}

		if err := b.f.SetColWidth(SheetReviews, colName, colName, cols[i].width); err != nil {
			return err
		}

		if err := b.f.SetColStyle(SheetReviews, colName, b.styles[cols[i].format]); err != nil {
			return err
		}
	}

	row := 1

	for i := range entries {
		e := &entries[i]

		reviews := allReviews(e)

		for j := range reviews {
			rv := &reviews[j]
			row++

			values := []any{
				e.Title,
				e.CompleteAddress.City,
				float64(rv.Rating),
				rv.Name,
				rv.When,
				publishedAt(rv),
				rv.Language,
				reviewText(rv),
				rv.TextTranslated,
				rv.ReplyText,
				rv.AuthorURL,
			}

			for c := range cols {
				cell, err := excelize.CoordinatesToCellName(c+1, row)
				if err != nil {
					return err
				}

				switch cols[c].format {
				case fmtRating:
					if err := b.f.SetCellValue(SheetReviews, cell, values[c]); err != nil {
						return err
					}
				case fmtURL:
					if err := b.writeURL(SheetReviews, cell, asString(values[c])); err != nil {
						return err
					}
				case fmtText, fmtID, fmtInt, fmtCoord, fmtWrap:
					if err := b.f.SetCellStr(SheetReviews, cell, clean(asString(values[c]))); err != nil {
						return err
					}
				}
			}
		}
	}

	return b.finishSheet(SheetReviews, len(cols), row, "B2")
}

// allReviews merges the inline reviews with any collected by -extra-reviews,
// dropping duplicates by review id so the sheet does not repeat itself.
func allReviews(e *gmaps.Entry) []gmaps.Review {
	out := make([]gmaps.Review, 0, len(e.UserReviews)+len(e.UserReviewsExtended))
	seen := map[string]bool{}

	for _, group := range [][]gmaps.Review{e.UserReviews, e.UserReviewsExtended} {
		for j := range group {
			if id := group[j].ReviewID; id != "" {
				if seen[id] {
					continue
				}

				seen[id] = true
			}

			out = append(out, group[j])
		}
	}

	return out
}

// reviewText prefers the original text, falling back to the description field
// used by the older inline parser.
func reviewText(rv *gmaps.Review) string {
	if rv.TextOriginal != "" {
		return rv.TextOriginal
	}

	return rv.Description
}

func publishedAt(rv *gmaps.Review) string {
	if rv.PublishedAt != nil && !rv.PublishedAt.IsZero() {
		return rv.PublishedAt.Format("2006-01-02 15:04")
	}

	if rv.PostedAtUnixMicros > 0 {
		return time.UnixMicro(rv.PostedAtUnixMicros).UTC().Format("2006-01-02 15:04")
	}

	return ""
}

// writeHours turns the opening-hours JSON into one row per business per day.
func (b *builder) writeHours(entries []gmaps.Entry) error {
	names := []string{"Business", "City", "Day", "Hours"}
	widths := []float64{34, 18, 12, 30}

	if err := b.writeHeaderRow(SheetHours, names); err != nil {
		return err
	}

	for i := range names {
		colName, err := excelize.ColumnNumberToName(i + 1)
		if err != nil {
			return err
		}

		if err := b.f.SetColWidth(SheetHours, colName, colName, widths[i]); err != nil {
			return err
		}

		if err := b.f.SetColStyle(SheetHours, colName, b.styles[fmtText]); err != nil {
			return err
		}
	}

	row := 1

	for i := range entries {
		e := &entries[i]

		if len(e.OpenHours) == 0 {
			continue
		}

		for _, day := range daysOfWeek {
			slots, ok := e.OpenHours[day]
			if !ok {
				continue
			}

			row++

			values := []string{e.Title, e.CompleteAddress.City, day, strings.Join(slots, ", ")}

			for c, v := range values {
				cell, err := excelize.CoordinatesToCellName(c+1, row)
				if err != nil {
					return err
				}

				if err := b.f.SetCellStr(SheetHours, cell, clean(v)); err != nil {
					return err
				}
			}
		}
	}

	return b.finishSheet(SheetHours, len(names), row, "B2")
}

// writeSummary is the sheet a person opens first: how many leads, how reachable
// they are, and what the result set is made of.
//
//nolint:funlen // a report layout is inherently a long sequence of writes.
func (b *builder) writeSummary(entries []gmaps.Entry) error {
	if err := b.f.SetColWidth(SheetSummary, "A", "A", 34); err != nil {
		return err
	}

	if err := b.f.SetColWidth(SheetSummary, "B", "B", 18); err != nil {
		return err
	}

	stats := summarize(entries)

	row := 1
	put := func(label string, value any, bold bool) error {
		cellA, _ := excelize.CoordinatesToCellName(1, row)
		cellB, _ := excelize.CoordinatesToCellName(2, row)

		if err := b.f.SetCellStr(SheetSummary, cellA, label); err != nil {
			return err
		}

		if bold {
			if err := b.f.SetCellStyle(SheetSummary, cellA, cellA, b.label); err != nil {
				return err
			}
		}

		if value != nil {
			if err := b.f.SetCellValue(SheetSummary, cellB, value); err != nil {
				return err
			}
		}

		row++

		return nil
	}

	if err := b.f.SetCellStr(SheetSummary, "A1", "Google Maps Scrape Report"); err != nil {
		return err
	}

	if err := b.f.SetCellStyle(SheetSummary, "A1", "A1", b.title); err != nil {
		return err
	}

	row = 2

	if err := put("Generated", time.Now().UTC().Format("2006-01-02 15:04 UTC"), false); err != nil {
		return err
	}

	row++

	sections := []struct {
		label string
		value any
		bold  bool
	}{
		{"Results", nil, true},
		{"Businesses found", stats.total, false},
		{"Duplicate branches removed", b.dupesRemoved, false},
		{"Average rating", roundTo(stats.avgRating, 2), false},
		{"Total reviews", stats.totalReviews, false},
		{"", nil, false},
		{"Contactability", nil, true},
		{"With phone", stats.withPhone, false},
		{"With website", stats.withWebsite, false},
		{"With email", stats.withEmail, false},
		{"With social profile", stats.withSocial, false},
		{"With any contact method", stats.withAnyContact, false},
	}

	for _, s := range sections {
		if err := put(s.label, s.value, s.bold); err != nil {
			return err
		}
	}

	row++

	if err := put("Social profiles by network", nil, true); err != nil {
		return err
	}

	for _, pair := range sortedByCount(stats.byNetwork, 20) {
		if err := put(asString(pair[0]), pair[1], false); err != nil {
			return err
		}
	}

	row++

	// Only meaningful when the job ran more than one search.
	if len(stats.byKeyword) > 1 {
		if err := put("Results by search", nil, true); err != nil {
			return err
		}

		for _, pair := range sortedByCount(stats.byKeyword, maxKeywordSheets) {
			if err := put(asString(pair[0]), pair[1], false); err != nil {
				return err
			}
		}

		row++
	}

	if err := put("Top categories", nil, true); err != nil {
		return err
	}

	for _, pair := range sortedByCount(stats.byCategory, 15) {
		if err := put(asString(pair[0]), pair[1], false); err != nil {
			return err
		}
	}

	row++

	if err := put("Top cities", nil, true); err != nil {
		return err
	}

	for _, pair := range sortedByCount(stats.byCity, 15) {
		if err := put(asString(pair[0]), pair[1], false); err != nil {
			return err
		}
	}

	return nil
}

type stats struct {
	total          int
	totalReviews   int
	avgRating      float64
	withPhone      int
	withWebsite    int
	withEmail      int
	withSocial     int
	withAnyContact int
	byCategory     map[string]int
	byKeyword      map[string]int
	byCity         map[string]int
	byNetwork      map[string]int
}

func summarize(entries []gmaps.Entry) stats {
	s := stats{
		byCategory: map[string]int{},
		byKeyword:  map[string]int{},
		byCity:     map[string]int{},
		byNetwork:  map[string]int{},
	}

	var ratingSum float64

	var rated int

	for i := range entries {
		e := &entries[i]
		s.total++
		s.totalReviews += e.ReviewCount

		if e.ReviewRating > 0 {
			ratingSum += e.ReviewRating
			rated++
		}

		hasPhone := e.Phone != ""
		hasWebsite := e.WebSite != ""
		hasEmail := len(e.Emails) > 0
		hasSocial := !e.Socials.IsEmpty()

		if hasPhone {
			s.withPhone++
		}

		if hasWebsite {
			s.withWebsite++
		}

		if hasEmail {
			s.withEmail++
		}

		if hasSocial {
			s.withSocial++
		}

		if hasPhone || hasWebsite || hasEmail || hasSocial {
			s.withAnyContact++
		}

		if e.Category != "" {
			s.byCategory[e.Category]++
		}

		if e.Keyword != "" {
			s.byKeyword[e.Keyword]++
		}

		if city := e.CompleteAddress.City; city != "" {
			s.byCity[city]++
		}

		countNetworks(&e.Socials, s.byNetwork)
	}

	if rated > 0 {
		s.avgRating = ratingSum / float64(rated)
	}

	return s
}

func countNetworks(socials *gmaps.Socials, into map[string]int) {
	pairs := map[string]string{
		"Facebook":    socials.Facebook,
		"Instagram":   socials.Instagram,
		"LinkedIn":    socials.LinkedIn,
		"X (Twitter)": socials.TwitterX,
		"YouTube":     socials.YouTube,
		"TikTok":      socials.TikTok,
		"WhatsApp":    socials.WhatsApp,
		"Telegram":    socials.Telegram,
		"Pinterest":   socials.Pinterest,
	}

	for name, v := range pairs {
		if v != "" {
			into[name]++
		}
	}
}

func roundTo(v float64, places int) float64 {
	pow := 1.0
	for range places {
		pow *= 10
	}

	return float64(int(v*pow+0.5)) / pow
}
