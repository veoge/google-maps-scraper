package export

import (
	"context"
	"io"
	"reflect"

	"github.com/gosom/scrapemate"

	"github.com/gosom/google-maps-scraper/gmaps"
)

var _ scrapemate.ResultWriter = (*xlsxWriter)(nil)

// xlsxWriter collects results and emits a formatted workbook once the run ends.
// A spreadsheet cannot be streamed the way CSV rows can — the summary sheet and
// the column filters both need the complete result set — so entries are held in
// memory until the input channel closes.
type xlsxWriter struct {
	w       io.Writer
	entries []gmaps.Entry
}

// NewXLSXWriter returns a writer that renders results as a multi-sheet XLSX
// report when the scrape finishes.
func NewXLSXWriter(w io.Writer) scrapemate.ResultWriter {
	return &xlsxWriter{w: w}
}

func (x *xlsxWriter) Run(_ context.Context, in <-chan scrapemate.Result) error {
	for result := range in {
		x.entries = append(x.entries, entriesFromResult(result.Data)...)
	}

	return BuildWorkbook(x.entries, x.w)
}

// entriesFromResult unwraps the shapes a job can return: a single entry, a
// pointer to one, or a slice of either.
func entriesFromResult(data any) []gmaps.Entry {
	switch v := data.(type) {
	case nil:
		return nil
	case *gmaps.Entry:
		if v == nil {
			return nil
		}

		return []gmaps.Entry{*v}
	case gmaps.Entry:
		return []gmaps.Entry{v}
	case []*gmaps.Entry:
		out := make([]gmaps.Entry, 0, len(v))

		for _, e := range v {
			if e != nil {
				out = append(out, *e)
			}
		}

		return out
	case []gmaps.Entry:
		return v
	}

	// Fall back to reflection for slice types the switch does not name, matching
	// how the CSV writer accepts arbitrary slices of results.
	rv := reflect.ValueOf(data)
	if rv.Kind() != reflect.Slice {
		return nil
	}

	out := make([]gmaps.Entry, 0, rv.Len())

	for i := range rv.Len() {
		out = append(out, entriesFromResult(rv.Index(i).Interface())...)
	}

	return out
}
