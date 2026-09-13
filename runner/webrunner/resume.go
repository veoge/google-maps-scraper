package webrunner

import (
	"context"
	"encoding/csv"
	"os"
	"reflect"

	"github.com/gosom/scrapemate"

	"github.com/gosom/google-maps-scraper/deduper"
	"github.com/gosom/google-maps-scraper/export"
)

// openResultFile opens a job's CSV for writing, appending when the run is being
// continued so the results already collected are not thrown away.
func openResultFile(path string, resuming bool) (*os.File, error) {
	if !resuming {
		return os.Create(path)
	}

	return os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o600)
}

var _ scrapemate.ResultWriter = (*resumableCSVWriter)(nil)

// resumableCSVWriter writes results as CSV, emitting the header row only when the
// file is still empty.
//
// The stock writer always writes a header the first time it sees a result. On a
// continued run that plants a header in the middle of the file, and every row
// after it is read back as data with the wrong column names.
type resumableCSVWriter struct {
	w           *csv.Writer
	needsHeader bool
}

func newResultWriter(f *os.File) (scrapemate.ResultWriter, error) {
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}

	return &resumableCSVWriter{
		w:           csv.NewWriter(f),
		needsHeader: info.Size() == 0,
	}, nil
}

func (c *resumableCSVWriter) Run(_ context.Context, in <-chan scrapemate.Result) error {
	for result := range in {
		for _, element := range csvCapable(result.Data) {
			if c.needsHeader {
				if err := c.w.Write(element.CsvHeaders()); err != nil {
					return err
				}

				c.needsHeader = false
			}

			if err := c.w.Write(element.CsvRow()); err != nil {
				return err
			}
		}

		c.w.Flush()
	}

	c.w.Flush()

	return c.w.Error()
}

// csvCapable unwraps the shapes a job can return into writable rows.
func csvCapable(data any) []scrapemate.CsvCapable {
	if data == nil {
		return nil
	}

	if one, ok := data.(scrapemate.CsvCapable); ok {
		return []scrapemate.CsvCapable{one}
	}

	rv := reflect.ValueOf(data)
	if rv.Kind() != reflect.Slice {
		return nil
	}

	out := make([]scrapemate.CsvCapable, 0, rv.Len())

	for i := range rv.Len() {
		if item, ok := rv.Index(i).Interface().(scrapemate.CsvCapable); ok {
			out = append(out, item)
		}
	}

	return out
}

// seedDeduper primes the de-duplication set from results already on disk, so a
// continued run does not re-add businesses the earlier attempt already wrote.
func seedDeduper(path string, dedup deduper.Deduper) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}

		return 0, err
	}

	defer func() {
		_ = f.Close()
	}()

	entries, err := export.EntriesFromCSV(f)
	if err != nil {
		return 0, err
	}

	seeded := 0

	for i := range entries {
		if link := entries[i].Link; link != "" {
			dedup.AddIfNotExists(context.Background(), link)

			seeded++
		}
	}

	return seeded, nil
}
