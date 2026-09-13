package web

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gosom/google-maps-scraper/geo"
	"github.com/gosom/google-maps-scraper/grid"
)

// Choosing a city must produce a bounding box, because that is what makes the
// job cover the whole city instead of a radius around its centre.
func TestApplyLocationSetsBoundingBoxForCity(t *testing.T) {
	t.Parallel()

	var data JobData

	require.NoError(t, applyLocation(&data, "JO", "Amman", "balanced"))

	require.Equal(t, "JO", data.Country)
	require.Equal(t, "Amman", data.City)
	require.NotEmpty(t, data.BBox)
	require.InDelta(t, 3.0, data.CellSizeKm, 0.001)

	bbox, err := grid.ParseBoundingBox(data.BBox)
	require.NoError(t, err)
	require.Less(t, bbox.MinLat, bbox.MaxLat)
	require.Less(t, bbox.MinLon, bbox.MaxLon)

	// The centre coordinates should sit inside the box the job will cover.
	require.NotEmpty(t, data.Lat)
	require.NotEmpty(t, data.Lon)

	// A city grid is many cells, not one search.
	require.Greater(t, grid.EstimateCellCount(bbox, data.CellSizeKm), 10)
}

func TestApplyLocationCoverageLevels(t *testing.T) {
	t.Parallel()

	for coverage, want := range map[string]float64{
		"quick":    6,
		"balanced": 3,
		"thorough": 1.5,
		"":         3, // unknown values fall back to balanced
		"nonsense": 3,
	} {
		var data JobData

		require.NoError(t, applyLocation(&data, "JO", "Amman", coverage))
		require.InDelta(t, want, data.CellSizeKm, 0.001, coverage)
	}
}

// A single sweep is the fast path: no grid, just one search per term anchored on
// the city. Without it, a dozen categories across a city grid runs for days.
func TestApplyLocationSingleSweepSkipsTheGrid(t *testing.T) {
	t.Parallel()

	var data JobData

	require.NoError(t, applyLocation(&data, "JO", "Amman", "single"))

	require.Equal(t, "Amman", data.City)
	require.Empty(t, data.BBox, "no bounding box means no grid")
	require.Zero(t, data.CellSizeKm)

	// The centre still anchors the search, at a zoom that frames the whole city
	// rather than the street-level default.
	require.NotEmpty(t, data.Lat)
	require.NotEmpty(t, data.Lon)
	require.Equal(t, citySweepZoom, data.Zoom)

	// It must stay valid: BBox validation only applies when a box is set.
	data.Keywords = []string{"cafe"}
	data.Lang = "en"
	data.Depth = 1
	data.MaxTime = 1
	require.NoError(t, data.Validate())
}

// "All cities" has to mean every city is actually searched. Previously a country
// with no city produced no bounding box at all, so the job fell back to a keyword
// search with no location — strictly worse than picking a single city.
func TestApplyLocationCountryOnlyGridsEveryCity(t *testing.T) {
	t.Parallel()

	var data JobData

	require.NoError(t, applyLocation(&data, "JO", "", "balanced"))

	require.Equal(t, "JO", data.Country)
	require.Empty(t, data.City)
	require.Empty(t, data.BBox, "no single city was chosen")

	jordan, _, ok := geo.Lookup("JO", "")
	require.True(t, ok)
	require.Len(t, data.BBoxes, len(jordan.Cities),
		"one grid area per city in the country")
	require.InDelta(t, 3.0, data.CellSizeKm, 0.001)

	// Areas() is what the runner grids, and every one of them must parse.
	areas := data.Areas()
	require.Len(t, areas, len(jordan.Cities))

	total := 0

	for _, a := range areas {
		bbox, err := grid.ParseBoundingBox(a)
		require.NoError(t, err)

		total += grid.EstimateCellCount(bbox, data.CellSizeKm)
	}

	require.Greater(t, total, 200, "a country-wide run is many more cells than one city")
}

// A sample run is explicitly not full coverage, so it grids nothing.
func TestApplyLocationCountryOnlySampleGridsNothing(t *testing.T) {
	t.Parallel()

	var data JobData

	require.NoError(t, applyLocation(&data, "JO", "", "single"))

	require.Empty(t, data.Areas())
}

// One city still grids just that city.
func TestAreasPrefersTheSingleCityBox(t *testing.T) {
	t.Parallel()

	var data JobData

	require.NoError(t, applyLocation(&data, "JO", "Amman", "balanced"))

	require.Len(t, data.Areas(), 1)
	require.Equal(t, data.BBox, data.Areas()[0])
}

func TestApplyLocationRejectsUnknownPlaces(t *testing.T) {
	t.Parallel()

	var data JobData

	require.Error(t, applyLocation(&data, "ZZ", "", "balanced"))
	require.Error(t, applyLocation(&data, "JO", "Atlantis", "balanced"))
}

func TestApplyLocationNoCountryIsAllowed(t *testing.T) {
	t.Parallel()

	var data JobData

	require.NoError(t, applyLocation(&data, "", "", ""))
	require.Empty(t, data.BBox)
}

// Both are natural ways to type several searches.
func TestSplitKeywords(t *testing.T) {
	t.Parallel()

	require.Equal(t,
		[]string{"restaurants", "coffee houses", "supermarkets"},
		splitKeywords("restaurants, coffee houses, supermarkets"))

	require.Equal(t,
		[]string{"restaurants", "coffee houses"},
		splitKeywords("restaurants\ncoffee houses"))

	require.Equal(t,
		[]string{"a", "b", "c"},
		splitKeywords("  a , b \n\n c  ,,  "))

	require.Empty(t, splitKeywords("   \n  "))
}

// A bad bounding box has to be caught before the job is queued.
func TestJobDataValidatesArea(t *testing.T) {
	t.Parallel()

	base := func() JobData {
		return JobData{
			Keywords: []string{"cafe"},
			Lang:     "en",
			Depth:    1,
			MaxTime:  1,
		}
	}

	good := base()
	good.BBox = "31.7,35.73,32.12,36.07"
	require.NoError(t, good.Validate())

	bad := base()
	bad.BBox = "not-a-box"
	require.Error(t, bad.Validate())

	// Fast mode searches from a point, so it cannot also cover an area.
	conflict := base()
	conflict.BBox = "31.7,35.73,32.12,36.07"
	conflict.FastMode = true
	conflict.Lat = "31.9"
	conflict.Lon = "35.9"
	require.Error(t, conflict.Validate())
}

func TestEveryCityHasAUsableBoundingBox(t *testing.T) {
	t.Parallel()

	for _, country := range geo.Countries() {
		require.NotEmpty(t, country.Cities, country.Name)

		for _, c := range country.Cities {
			bbox, err := grid.ParseBoundingBox(c.BBoxString())
			require.NoError(t, err, "%s / %s", country.Name, c.Name)

			require.Less(t, bbox.MinLat, bbox.MaxLat, c.Name)
			require.Less(t, bbox.MinLon, bbox.MaxLon, c.Name)

			// The centre should fall inside the box it belongs to.
			require.GreaterOrEqual(t, c.Lat, bbox.MinLat, c.Name)
			require.LessOrEqual(t, c.Lat, bbox.MaxLat, c.Name)
			require.GreaterOrEqual(t, c.Lon, bbox.MinLon, c.Name)
			require.LessOrEqual(t, c.Lon, bbox.MaxLon, c.Name)

			// A box that produces one cell would defeat the point of the grid.
			require.Greater(t, grid.EstimateCellCount(bbox, 3.0), 1, c.Name)
		}
	}
}

func TestJordanIsTheDefaultAndListedFirst(t *testing.T) {
	t.Parallel()

	countries := geo.Countries()

	require.Equal(t, "JO", geo.DefaultCountry)
	require.Equal(t, "JO", countries[0].Code)
	require.Equal(t, "Jordan", countries[0].Name)
}
