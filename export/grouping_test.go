package export_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gosom/google-maps-scraper/export"
	"github.com/gosom/google-maps-scraper/gmaps"
)

func entryFor(keyword, title string) gmaps.Entry {
	e := sampleEntry()
	e.Keyword = keyword
	e.Title = title
	e.UserReviews = nil
	// Every row built from sampleEntry shares its Facebook and Instagram, and
	// the workbook collapses such rows to one; these tests are about grouping.
	e.Socials = gmaps.Socials{}

	return e
}

// A job running several searches should come back as one tab per search, which
// is the whole point of entering "restaurants, coffee shops, supermarkets".
func TestSheetPerSearchTerm(t *testing.T) {
	t.Parallel()

	f := openWorkbook(t, []gmaps.Entry{
		entryFor("restaurants", "Acme Grill"),
		entryFor("coffee shops", "Bean There"),
		entryFor("restaurants", "Second Grill"),
		entryFor("supermarkets", "Big Mart"),
	})

	require.Equal(t, []string{
		export.SheetLeads,
		"Restaurants",
		"Coffee shops",
		"Supermarkets",
		export.SheetReviews,
		export.SheetHours,
		export.SheetSummary,
	}, f.GetSheetList(), "search tabs sit between All Leads and the shared sheets")

	rows, err := f.GetRows("Restaurants")
	require.NoError(t, err)
	require.Len(t, rows, 3, "header plus the two restaurants")

	all, err := f.GetRows(export.SheetLeads)
	require.NoError(t, err)
	require.Len(t, all, 5, "the combined sheet still holds every result")
}

func TestSearchColumnIsPopulated(t *testing.T) {
	t.Parallel()

	f := openWorkbook(t, []gmaps.Entry{entryFor("coffee shops", "Bean There")})

	require.Equal(t, "coffee shops", cellUnderHeader(t, f, export.SheetLeads, "Search"))
}

// One search needs no split: a second tab identical to All Leads is just noise.
func TestSingleSearchProducesNoExtraSheets(t *testing.T) {
	t.Parallel()

	f := openWorkbook(t, []gmaps.Entry{
		entryFor("restaurants", "Acme Grill"),
		entryFor("restaurants", "Second Grill"),
	})

	require.Equal(t,
		[]string{export.SheetLeads, export.SheetReviews, export.SheetHours, export.SheetSummary},
		f.GetSheetList())
}

// Excel rejects several punctuation marks in a sheet name and caps them at 31
// characters, so a raw search term cannot be used directly.
func TestSheetNamesAreLegalAndUnique(t *testing.T) {
	t.Parallel()

	f := openWorkbook(t, []gmaps.Entry{
		entryFor(`bakeries / patisseries [fresh]`, "One"),
		entryFor("a very long search term that goes past the excel limit", "Two"),
		entryFor("cafes", "Three"),
	})

	for _, name := range f.GetSheetList() {
		require.LessOrEqual(t, len([]rune(name)), 31, name)

		for _, bad := range []string{":", "\\", "/", "?", "*", "[", "]"} {
			require.NotContains(t, name, bad, name)
		}
	}

	require.Len(t, f.GetSheetList(), 7, "3 search tabs plus the 4 standard sheets")
}

// Above the cap the tabs stop helping, so the split is skipped and the Search
// column on All Leads carries the grouping instead.
func TestTooManySearchesFallsBackToOneSheet(t *testing.T) {
	t.Parallel()

	entries := make([]gmaps.Entry, 0, 30)
	for i := range 30 {
		entries = append(entries, entryFor(string(rune('a'+i%26))+"-search-"+itoa(i), "Place"))
	}

	f := openWorkbook(t, entries)

	require.Equal(t,
		[]string{export.SheetLeads, export.SheetReviews, export.SheetHours, export.SheetSummary},
		f.GetSheetList())
}

func TestSummaryBreaksDownBySearch(t *testing.T) {
	t.Parallel()

	f := openWorkbook(t, []gmaps.Entry{
		entryFor("restaurants", "One"),
		entryFor("restaurants", "Two"),
		entryFor("supermarkets", "Three"),
	})

	rows, err := f.GetRows(export.SheetSummary)
	require.NoError(t, err)

	values := map[string]string{}

	for _, row := range rows {
		if len(row) >= 2 {
			values[row[0]] = row[1]
		}
	}

	require.Equal(t, "2", values["restaurants"])
	require.Equal(t, "1", values["supermarkets"])
}

// The search term must survive the CSV that the web download re-reads.
func TestKeywordRoundTripsThroughCSV(t *testing.T) {
	t.Parallel()

	entry := entryFor("coffee shops", "Bean There")

	var buf bytes.Buffer

	buf.WriteString(quoteRow(entry.CsvHeaders()) + "\n")
	buf.WriteString(quoteRow(entry.CsvRow()) + "\n")

	entries, err := export.EntriesFromCSV(&buf)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "coffee shops", entries[0].Keyword)
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}

	var out []byte

	for i > 0 {
		out = append([]byte{byte('0' + i%10)}, out...)
		i /= 10
	}

	return string(out)
}
