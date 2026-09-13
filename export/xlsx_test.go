package export_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"

	"github.com/gosom/google-maps-scraper/export"
	"github.com/gosom/google-maps-scraper/gmaps"
)

// sampleEntry mirrors the shape of a real scraped row, including the two things
// that were destroyed on the way into Excel: a 19-digit CID and a non-Latin name.
func sampleEntry() gmaps.Entry {
	return gmaps.Entry{
		Title:        "مطعم خبزة وجبنة",
		Category:     "Pizza restaurant",
		Address:      "No. 11, Naltshek St., Amman",
		Phone:        "07 9963 4146",
		WebSite:      "https://acme.example",
		Cid:          "1538490000000000000",
		PlaceID:      "ChIJNYmuCACnHBURMVvAH65RgtU",
		ReviewCount:  28,
		ReviewRating: 4.4,
		Latitude:     31.910156,
		Longtitude:   35.849771,
		PriceRange:   "JOD 1–5",
		OpenHours: map[string][]string{
			"Monday": {"8 AM–12 AM"},
			"Sunday": {"8 AM–12 AM"},
		},
		PopularTimes: map[string]map[int]int{
			"Thursday": {21: 92, 22: 100},
			"Monday":   {17: 88},
		},
		ReviewsPerRating: map[int]int{1: 2, 3: 3, 4: 2, 5: 21},
		CompleteAddress: gmaps.Address{
			Borough: "Marj Al Hamam", City: "Amman", Country: "JO",
		},
		About: []gmaps.About{
			{
				ID:   "service_options",
				Name: "Service options",
				Options: []gmaps.Option{
					{Name: "Delivery", Enabled: true},
					{Name: "Dine-in", Enabled: true},
					{Name: "Kerbside pickup", Enabled: false},
				},
			},
		},
		Emails: []string{"hi@acme.example"},
		Socials: gmaps.Socials{
			Facebook:  "https://www.facebook.com/acmecafe",
			Instagram: "https://instagram.com/acme.cafe",
		},
		UserReviews: []gmaps.Review{
			{
				Name: "Mohammed Salman", Rating: 5, When: "2 years ago",
				ReviewID: "r1", TextOriginal: "Friendly and highly trained staff",
			},
		},
	}
}

func openWorkbook(t *testing.T, entries []gmaps.Entry) *excelize.File {
	t.Helper()

	var buf bytes.Buffer

	require.NoError(t, export.BuildWorkbook(entries, &buf))

	f, err := excelize.OpenReader(bytes.NewReader(buf.Bytes()))
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = f.Close()
	})

	return f
}

func TestWorkbookHasAllSheets(t *testing.T) {
	t.Parallel()

	f := openWorkbook(t, []gmaps.Entry{sampleEntry()})

	require.Equal(t,
		[]string{export.SheetLeads, export.SheetReviews, export.SheetHours, export.SheetSummary},
		f.GetSheetList())
}

// A CID is a 19-digit integer. Written as a number, Excel keeps only 15
// significant digits and shows 1.53849E+18, losing the identifier for good.
func TestIdentifiersSurviveAsText(t *testing.T) {
	t.Parallel()

	f := openWorkbook(t, []gmaps.Entry{sampleEntry()})

	cell := cellUnderHeader(t, f, export.SheetLeads, "CID")
	require.Equal(t, "1538490000000000000", cell)
	require.NotContains(t, cell, "E+")

	require.Equal(t, "ChIJNYmuCACnHBURMVvAH65RgtU",
		cellUnderHeader(t, f, export.SheetLeads, "Place ID"))
}

func TestNonLatinTextIsPreserved(t *testing.T) {
	t.Parallel()

	f := openWorkbook(t, []gmaps.Entry{sampleEntry()})

	require.Equal(t, "مطعم خبزة وجبنة",
		cellUnderHeader(t, f, export.SheetLeads, "Business"))
}

func TestSocialColumnsArePopulated(t *testing.T) {
	t.Parallel()

	f := openWorkbook(t, []gmaps.Entry{sampleEntry()})

	combined := cellUnderHeader(t, f, export.SheetLeads, "Social")
	require.Equal(t,
		"https://www.facebook.com/acmecafe, https://instagram.com/acme.cafe", combined)

	require.Equal(t, "https://www.facebook.com/acmecafe",
		cellUnderHeader(t, f, export.SheetLeads, "Facebook"))
	require.Equal(t, "https://instagram.com/acme.cafe",
		cellUnderHeader(t, f, export.SheetLeads, "Instagram"))
	require.Empty(t, cellUnderHeader(t, f, export.SheetLeads, "LinkedIn"))
}

// The JSON blobs in the CSV are the reason the sheet was unreadable. They must
// come back as plain columns a person can filter on.
func TestNestedDataIsFlattened(t *testing.T) {
	t.Parallel()

	f := openWorkbook(t, []gmaps.Entry{sampleEntry()})

	require.Equal(t, "8 AM–12 AM", cellUnderHeader(t, f, export.SheetLeads, "Mon"))
	require.Equal(t, "21", cellUnderHeader(t, f, export.SheetLeads, "5 Star"))
	require.Equal(t, "Amman", cellUnderHeader(t, f, export.SheetLeads, "City"))
	require.Equal(t, "Yes", cellUnderHeader(t, f, export.SheetLeads, "Delivery"))
	require.Empty(t, cellUnderHeader(t, f, export.SheetLeads, "Reservations"))

	amenities := cellUnderHeader(t, f, export.SheetLeads, "Amenities")
	require.Contains(t, amenities, "Delivery")
	require.NotContains(t, amenities, "Kerbside pickup", "disabled options must be left out")
	require.NotContains(t, amenities, `{"`, "no raw JSON should reach the sheet")

	// Popular times collapses from a 168-value matrix to the one fact that fits
	// in a spreadsheet.
	require.Equal(t, "Thursday", cellUnderHeader(t, f, export.SheetLeads, "Busiest Day"))
	require.Equal(t, "10 PM", cellUnderHeader(t, f, export.SheetLeads, "Busiest Hour"))
}

func TestReviewsGetTheirOwnRows(t *testing.T) {
	t.Parallel()

	f := openWorkbook(t, []gmaps.Entry{sampleEntry()})

	require.Equal(t, "مطعم خبزة وجبنة",
		cellUnderHeader(t, f, export.SheetReviews, "Business"))
	require.Equal(t, "Friendly and highly trained staff",
		cellUnderHeader(t, f, export.SheetReviews, "Review"))
	require.Equal(t, "Mohammed Salman",
		cellUnderHeader(t, f, export.SheetReviews, "Author"))
}

func TestHoursSheetHasOneRowPerDay(t *testing.T) {
	t.Parallel()

	f := openWorkbook(t, []gmaps.Entry{sampleEntry()})

	rows, err := f.GetRows(export.SheetHours)
	require.NoError(t, err)
	require.Len(t, rows, 3, "header plus Monday and Sunday")
	require.Equal(t, "Monday", rows[1][2])
	require.Equal(t, "Sunday", rows[2][2])
}

func TestSummaryCountsContactability(t *testing.T) {
	t.Parallel()

	noContact := sampleEntry()
	noContact.Phone = ""
	noContact.WebSite = ""
	noContact.Emails = nil
	noContact.Socials = gmaps.Socials{}

	f := openWorkbook(t, []gmaps.Entry{sampleEntry(), noContact})

	rows, err := f.GetRows(export.SheetSummary)
	require.NoError(t, err)

	values := map[string]string{}

	for _, row := range rows {
		if len(row) >= 2 {
			values[row[0]] = row[1]
		}
	}

	require.Equal(t, "2", values["Businesses found"])
	require.Equal(t, "1", values["With social profile"])
	require.Equal(t, "1", values["With email"])
	require.Equal(t, "1", values["With any contact method"])
}

func TestEmptyResultSetStillProducesAValidWorkbook(t *testing.T) {
	t.Parallel()

	f := openWorkbook(t, nil)

	rows, err := f.GetRows(export.SheetLeads)
	require.NoError(t, err)
	require.Len(t, rows, 1, "headers only")
}

// The round trip is what the web download relies on: results are stored as CSV
// and re-rendered as a workbook on demand.
func TestCSVRoundTrip(t *testing.T) {
	t.Parallel()

	entry := sampleEntry()

	var csvBuf bytes.Buffer

	csvBuf.WriteString(strings.Join(entry.CsvHeaders(), ",") + "\n")
	csvBuf.WriteString(quoteRow(entry.CsvRow()) + "\n")

	entries, err := export.EntriesFromCSV(&csvBuf)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	got := entries[0]
	require.Equal(t, entry.Title, got.Title)
	require.Equal(t, entry.Cid, got.Cid)
	require.Equal(t, entry.Socials.Facebook, got.Socials.Facebook)
	require.Equal(t, entry.CompleteAddress.City, got.CompleteAddress.City)
	require.Equal(t, entry.OpenHours["Monday"], got.OpenHours["Monday"])
	require.Equal(t, 21, got.ReviewsPerRating[5])
	require.Len(t, got.UserReviews, 1)
}

// A file written before the social columns existed must still load, with the new
// columns simply empty rather than shifting every value.
func TestCSVWithoutSocialColumnsStillLoads(t *testing.T) {
	t.Parallel()

	csv := "title,category,phone,cid\n" +
		"Acme Cafe,Cafe,0790000000,1538490000000000000\n"

	entries, err := export.EntriesFromCSV(strings.NewReader(csv))
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "Acme Cafe", entries[0].Title)
	require.Equal(t, "1538490000000000000", entries[0].Cid)
	require.True(t, entries[0].Socials.IsEmpty())
}

// cellUnderHeader looks a value up by column name so the tests do not break every
// time a column is inserted.
func cellUnderHeader(t *testing.T, f *excelize.File, sheet, header string) string {
	t.Helper()

	// Header is row 1, so the first data row is row 2.
	const row = 2

	rows, err := f.GetRows(sheet)
	require.NoError(t, err)
	require.NotEmpty(t, rows)

	idx := -1

	for i, name := range rows[0] {
		if name == header {
			idx = i

			break
		}
	}

	require.GreaterOrEqual(t, idx, 0, "column %q not found in %s", header, sheet)
	require.GreaterOrEqual(t, len(rows), row, "sheet %s has no row %d", sheet, row)

	values := rows[row-1]
	if idx >= len(values) {
		return ""
	}

	return values[idx]
}

func quoteRow(values []string) string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = `"` + strings.ReplaceAll(v, `"`, `""`) + `"`
	}

	return strings.Join(out, ",")
}
