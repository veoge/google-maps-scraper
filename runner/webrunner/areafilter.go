package webrunner

import (
	"context"
	"reflect"

	"github.com/gosom/scrapemate"

	"github.com/gosom/google-maps-scraper/gmaps"
	"github.com/gosom/google-maps-scraper/grid"
)

// areaMarginFraction widens the bounding box before filtering. The boxes in the
// geo package are approximate metropolitan outlines, so a business on the edge of
// town can legitimately sit slightly outside one. A quarter of the span on each
// side keeps those while still discarding results from another city entirely.
const areaMarginFraction = 0.25

var _ scrapemate.ResultWriter = (*areaFilterWriter)(nil)

// areaFilterWriter drops results that fall outside the area the job asked for.
//
// Google Maps answers a search anchored at a point in Aqaba with the occasional
// business 300km away in Amman. Those are not wrong answers to Google's question,
// but they are wrong answers to "everything in this city", and they are noise in a
// lead list, so they are filtered out before anything is written.
type box struct {
	minLat float64
	minLon float64
	maxLat float64
	maxLon float64
}

type areaFilterWriter struct {
	inner   scrapemate.ResultWriter
	boxes   []box
	dropped int
}

// newAreaFilterWriter wraps w so only results inside one of the boxes (plus a
// margin) reach it. A whole-country run passes one box per city.
func newAreaFilterWriter(w scrapemate.ResultWriter, bboxes ...grid.BoundingBox) *areaFilterWriter {
	boxes := make([]box, 0, len(bboxes))

	for _, b := range bboxes {
		latMargin := (b.MaxLat - b.MinLat) * areaMarginFraction
		lonMargin := (b.MaxLon - b.MinLon) * areaMarginFraction

		boxes = append(boxes, box{
			minLat: b.MinLat - latMargin,
			minLon: b.MinLon - lonMargin,
			maxLat: b.MaxLat + latMargin,
			maxLon: b.MaxLon + lonMargin,
		})
	}

	return &areaFilterWriter{inner: w, boxes: boxes}
}

func (w *areaFilterWriter) Run(ctx context.Context, in <-chan scrapemate.Result) error {
	out := make(chan scrapemate.Result)

	go func() {
		defer close(out)

		for result := range in {
			filtered, kept := w.filter(result)
			if !kept {
				continue
			}

			select {
			case out <- filtered:
			case <-ctx.Done():
				return
			}
		}
	}()

	return w.inner.Run(ctx, out)
}

// filter removes out-of-area entries from a result, reporting whether anything
// is left to write.
func (w *areaFilterWriter) filter(result scrapemate.Result) (scrapemate.Result, bool) {
	switch data := result.Data.(type) {
	case *gmaps.Entry:
		if data == nil || !w.contains(data) {
			return result, false
		}

		return result, true
	case []*gmaps.Entry:
		kept := make([]*gmaps.Entry, 0, len(data))

		for _, e := range data {
			if e != nil && w.contains(e) {
				kept = append(kept, e)
			}
		}

		result.Data = kept

		return result, len(kept) > 0
	}

	// Anything else (a slice of another type, or a custom payload) passes through
	// untouched rather than being silently discarded.
	if rv := reflect.ValueOf(result.Data); rv.Kind() == reflect.Slice && rv.Len() == 0 {
		return result, false
	}

	return result, true
}

// contains reports whether the entry belongs to the requested area. An entry with
// no coordinates is kept: dropping a result because a field is missing would lose
// data the scrape did find.
func (w *areaFilterWriter) contains(e *gmaps.Entry) bool {
	if e.Latitude == 0 && e.Longtitude == 0 {
		return true
	}

	for _, b := range w.boxes {
		if e.Latitude >= b.minLat && e.Latitude <= b.maxLat &&
			e.Longtitude >= b.minLon && e.Longtitude <= b.maxLon {
			return true
		}
	}

	w.dropped++

	return false
}
