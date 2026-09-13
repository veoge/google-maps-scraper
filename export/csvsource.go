// Package export turns scraped entries into shareable office formats. It exists
// so the CLI, the web download endpoint and the REST API all produce byte
// identical workbooks from one implementation.
package export

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"strconv"
	"strings"

	"github.com/gosom/google-maps-scraper/gmaps"
)

// BOM is the UTF-8 byte order mark. Excel on Windows needs it at the start of a
// CSV, otherwise it decodes the file with the system ANSI code page and turns
// every non-Latin character into mojibake. It is also emitted by Excel when it
// saves a CSV, so it must be stripped before the first header name is compared,
// or every column lookup fails.
const BOM = utf8BOM

const utf8BOM = "\uFEFF"

// EntriesFromCSV rebuilds entries from the CSV produced by gmaps.Entry.CsvRow.
// Columns are resolved by header name, so a file written by an older version
// (without the social columns, say) still loads: missing columns read as empty
// rather than shifting every later value by one.
func EntriesFromCSV(r io.Reader) ([]gmaps.Entry, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true

	header, err := reader.Read()
	if err != nil {
		if err == io.EOF {
			return []gmaps.Entry{}, nil
		}

		return nil, err
	}

	col := make(map[string]int, len(header))

	for i, name := range header {
		col[strings.TrimPrefix(name, utf8BOM)] = i
	}

	entries := []gmaps.Entry{}

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, err
		}

		entries = append(entries, entryFromRow(row, col))
	}

	return entries, nil
}

//nolint:funlen // a flat field-by-field mapping is clearer than splitting it up.
func entryFromRow(row []string, col map[string]int) gmaps.Entry {
	get := func(name string) string {
		idx, ok := col[name]
		if !ok || idx >= len(row) {
			return ""
		}

		return row[idx]
	}

	entry := gmaps.Entry{
		ID:            get("input_id"),
		Keyword:       get("search_term"),
		Link:          get("link"),
		Title:         get("title"),
		Category:      get("category"),
		Address:       get("address"),
		WebSite:       get("website"),
		Phone:         get("phone"),
		PlusCode:      get("plus_code"),
		ReviewCount:   atoi(get("review_count")),
		ReviewRating:  atof(get("review_rating")),
		Latitude:      atof(get("latitude")),
		Longtitude:    atof(get("longitude")),
		Cid:           get("cid"),
		Status:        get("status"),
		Description:   get("descriptions"),
		ReviewsLink:   get("reviews_link"),
		Thumbnail:     get("thumbnail"),
		Timezone:      get("timezone"),
		PriceRange:    get("price_range"),
		DataID:        get("data_id"),
		StreetViewURL: get("street_view_url"),
		PlaceID:       get("place_id"),
		Emails:        splitList(get("emails")),
		Socials: gmaps.Socials{
			Facebook:  get("facebook"),
			Instagram: get("instagram"),
			LinkedIn:  get("linkedin"),
			TwitterX:  get("twitter_x"),
			YouTube:   get("youtube"),
			TikTok:    get("tiktok"),
			WhatsApp:  get("whatsapp"),
		},
		CreditCardsAccepted: splitList(get("credit_cards_accepted")),
	}

	decodeJSON(get("open_hours"), &entry.OpenHours)
	decodeJSON(get("popular_times"), &entry.PopularTimes)
	decodeJSON(get("reviews_per_rating"), &entry.ReviewsPerRating)
	decodeJSON(get("images"), &entry.Images)
	decodeJSON(get("reservations"), &entry.Reservations)
	decodeJSON(get("order_online"), &entry.OrderOnline)
	decodeJSON(get("menu"), &entry.Menu)
	decodeJSON(get("owner"), &entry.Owner)
	decodeJSON(get("complete_address"), &entry.CompleteAddress)
	decodeJSON(get("about"), &entry.About)
	decodeJSON(get("user_reviews"), &entry.UserReviews)
	decodeJSON(get("user_reviews_extended"), &entry.UserReviewsExtended)

	return entry
}

// decodeJSON fills dst from a JSON-encoded cell, leaving dst untouched when the
// cell is empty or malformed. A single unparseable cell must not fail the export
// of an otherwise good file.
func decodeJSON(s string, dst any) {
	s = strings.TrimSpace(s)
	if s == "" || s == "null" {
		return
	}

	_ = json.Unmarshal([]byte(s), dst)
}

func splitList(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}

	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))

	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}

	return out
}

func atoi(s string) int {
	v, _ := strconv.Atoi(strings.TrimSpace(s))

	return v
}

func atof(s string) float64 {
	v, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)

	return v
}
