package grid_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gosom/google-maps-scraper/grid"
)

// EstimateCellCount is shown to the user as "this many searches, allow this
// long", so it has to agree with what GenerateCells actually produces.
func TestEstimateMatchesGeneratedCells(t *testing.T) {
	t.Parallel()

	boxes := map[string]grid.BoundingBox{
		"Aqaba":      {MinLat: 29.45, MinLon: 34.94, MaxLat: 29.62, MaxLon: 35.07},
		"Amman":      {MinLat: 31.70, MinLon: 35.73, MaxLat: 32.12, MaxLon: 36.07},
		"Dubai":      {MinLat: 25.00, MinLon: 55.05, MaxLat: 25.35, MaxLon: 55.58},
		"Casablanca": {MinLat: 33.48, MinLon: -7.72, MaxLat: 33.65, MaxLon: -7.47},
		"tiny":       {MinLat: 10.00, MinLon: 20.00, MaxLat: 10.01, MaxLon: 20.01},
		"southern":   {MinLat: -11.76, MinLon: 43.20, MaxLat: -11.67, MaxLon: 43.29},
	}

	for _, cellKm := range []float64{1, 1.5, 3, 6, 10} {
		for name, bbox := range boxes {
			require.Equal(t,
				len(grid.GenerateCells(bbox, cellKm)),
				grid.EstimateCellCount(bbox, cellKm),
				"%s at %.1fkm", name, cellKm)
		}
	}
}

func TestEstimateIsZeroForEmptyBox(t *testing.T) {
	t.Parallel()

	empty := grid.BoundingBox{MinLat: 10, MinLon: 20, MaxLat: 10, MaxLon: 20}

	require.Equal(t, 0, grid.EstimateCellCount(empty, 3))
	require.Empty(t, grid.GenerateCells(empty, 3))
}
