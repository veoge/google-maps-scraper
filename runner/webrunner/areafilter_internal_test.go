package webrunner

import (
	"context"
	"testing"

	"github.com/gosom/scrapemate"
	"github.com/stretchr/testify/require"

	"github.com/gosom/google-maps-scraper/gmaps"
	"github.com/gosom/google-maps-scraper/grid"
)

// collector records what actually reaches the underlying writer.
type collector struct {
	got []scrapemate.Result
}

func (c *collector) Run(_ context.Context, in <-chan scrapemate.Result) error {
	for r := range in {
		c.got = append(c.got, r)
	}

	return nil
}

// aqaba is the bounding box the geo package ships for Aqaba.
func aqaba() grid.BoundingBox {
	return grid.BoundingBox{MinLat: 29.45, MinLon: 34.94, MaxLat: 29.62, MaxLon: 35.07}
}

func runFilter(t *testing.T, entries []*gmaps.Entry) []scrapemate.Result {
	t.Helper()

	sink := &collector{}
	w := newAreaFilterWriter(sink, aqaba())

	in := make(chan scrapemate.Result, len(entries))
	for _, e := range entries {
		in <- scrapemate.Result{Data: e}
	}

	close(in)

	require.NoError(t, w.Run(context.Background(), in))

	return sink.got
}

func entryAt(title string, lat, lon float64) *gmaps.Entry {
	return &gmaps.Entry{Title: title, Latitude: lat, Longtitude: lon}
}

// A search anchored in Aqaba sometimes returns a business in Amman, 300km away.
// It is a valid place, but it is not an answer to "everything in this city".
func TestFilterDropsResultsFromAnotherCity(t *testing.T) {
	t.Parallel()

	got := runFilter(t, []*gmaps.Entry{
		entryAt("Aqaba cafe", 29.53, 35.00),
		entryAt("Amman cafe", 31.95, 35.91),
	})

	require.Len(t, got, 1)

	entry, ok := got[0].Data.(*gmaps.Entry)
	require.True(t, ok)
	require.Equal(t, "Aqaba cafe", entry.Title)
}

// The shipped boxes are approximate outlines, so a business just past the edge
// of one is kept rather than thrown away.
func TestFilterKeepsResultsJustOutsideTheBox(t *testing.T) {
	t.Parallel()

	// 0.02 degrees north of the box; the margin is 25% of a 0.17 degree span.
	got := runFilter(t, []*gmaps.Entry{entryAt("Edge of town", 29.64, 35.00)})

	require.Len(t, got, 1)
}

// Dropping a result because a field is missing would lose data the scrape found.
func TestFilterKeepsEntriesWithoutCoordinates(t *testing.T) {
	t.Parallel()

	got := runFilter(t, []*gmaps.Entry{entryAt("No coords", 0, 0)})

	require.Len(t, got, 1)
}

func TestFilterHandlesSlicesOfEntries(t *testing.T) {
	t.Parallel()

	sink := &collector{}
	w := newAreaFilterWriter(sink, aqaba())

	in := make(chan scrapemate.Result, 2)
	in <- scrapemate.Result{Data: []*gmaps.Entry{
		entryAt("Aqaba one", 29.53, 35.00),
		entryAt("Amman one", 31.95, 35.91),
		entryAt("Aqaba two", 29.55, 35.02),
	}}
	in <- scrapemate.Result{Data: []*gmaps.Entry{entryAt("Amman only", 31.95, 35.91)}}
	close(in)

	require.NoError(t, w.Run(context.Background(), in))

	require.Len(t, sink.got, 1, "the all-out-of-area batch is dropped entirely")

	kept, ok := sink.got[0].Data.([]*gmaps.Entry)
	require.True(t, ok)
	require.Len(t, kept, 2)
	require.Equal(t, "Aqaba one", kept[0].Title)
	require.Equal(t, "Aqaba two", kept[1].Title)
}

// A payload the filter does not understand must pass through rather than vanish.
func TestFilterPassesThroughUnknownPayloads(t *testing.T) {
	t.Parallel()

	sink := &collector{}
	w := newAreaFilterWriter(sink, aqaba())

	in := make(chan scrapemate.Result, 1)
	in <- scrapemate.Result{Data: "something else"}
	close(in)

	require.NoError(t, w.Run(context.Background(), in))
	require.Len(t, sink.got, 1)
}
