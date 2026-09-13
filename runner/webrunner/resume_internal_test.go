package webrunner

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gosom/scrapemate"
	"github.com/stretchr/testify/require"

	"github.com/gosom/google-maps-scraper/deduper"
	"github.com/gosom/google-maps-scraper/export"
	"github.com/gosom/google-maps-scraper/gmaps"
)

func write(t *testing.T, path string, entries ...*gmaps.Entry) {
	t.Helper()

	f, err := openResultFile(path, false)
	require.NoError(t, err)

	w, err := newResultWriter(f)
	require.NoError(t, err)

	in := make(chan scrapemate.Result, len(entries))
	for _, e := range entries {
		in <- scrapemate.Result{Data: e}
	}

	close(in)

	require.NoError(t, w.Run(context.Background(), in))
	require.NoError(t, f.Close())
}

func appendRun(t *testing.T, path string, entries ...*gmaps.Entry) {
	t.Helper()

	f, err := openResultFile(path, true)
	require.NoError(t, err)

	w, err := newResultWriter(f)
	require.NoError(t, err)

	in := make(chan scrapemate.Result, len(entries))
	for _, e := range entries {
		in <- scrapemate.Result{Data: e}
	}

	close(in)

	require.NoError(t, w.Run(context.Background(), in))
	require.NoError(t, f.Close())
}

func entry(title, link string) *gmaps.Entry {
	return &gmaps.Entry{Title: title, Category: "Cafe", Link: link}
}

// Continuing a run must add to the file, not replace it, and must not plant a
// second header in the middle — which would make every row after it unreadable.
func TestContinuingAppendsWithoutRepeatingTheHeader(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "job.csv")

	write(t, path, entry("First", "https://maps.google.com/1"))
	appendRun(t, path, entry("Second", "https://maps.google.com/2"))

	raw, err := os.ReadFile(path)
	require.NoError(t, err)

	require.Equal(t, 1, strings.Count(string(raw), "input_id"),
		"exactly one header row")

	f, err := os.Open(path)
	require.NoError(t, err)

	defer func() { _ = f.Close() }()

	entries, err := export.EntriesFromCSV(f)
	require.NoError(t, err)
	require.Len(t, entries, 2)
	require.Equal(t, "First", entries[0].Title)
	require.Equal(t, "Second", entries[1].Title)
}

// Starting fresh must truncate, so a re-run does not inherit old rows.
func TestFreshRunTruncates(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "job.csv")

	write(t, path, entry("Old", "https://maps.google.com/1"))
	write(t, path, entry("New", "https://maps.google.com/2"))

	f, err := os.Open(path)
	require.NoError(t, err)

	defer func() { _ = f.Close() }()

	entries, err := export.EntriesFromCSV(f)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "New", entries[0].Title)
}

// A continued run redoes the batch it was interrupted in, so the de-duplication
// set has to know what was already written or those businesses appear twice.
func TestDeduperIsSeededFromExistingResults(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "job.csv")

	write(t, path,
		entry("One", "https://maps.google.com/place/1"),
		entry("Two", "https://maps.google.com/place/2"),
	)

	dedup := deduper.New()

	seeded, err := seedDeduper(path, dedup)
	require.NoError(t, err)
	require.Equal(t, 2, seeded)

	require.False(t, dedup.AddIfNotExists(context.Background(), "https://maps.google.com/place/1"),
		"an already-collected place must not be scraped again")
	require.True(t, dedup.AddIfNotExists(context.Background(), "https://maps.google.com/place/3"),
		"a new place is still allowed through")
}

func TestSeedDeduperOnMissingFileIsHarmless(t *testing.T) {
	t.Parallel()

	seeded, err := seedDeduper(filepath.Join(t.TempDir(), "absent.csv"), deduper.New())
	require.NoError(t, err)
	require.Zero(t, seeded)
}
