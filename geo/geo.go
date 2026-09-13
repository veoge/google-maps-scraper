// Package geo holds the country and city list offered by the web UI.
//
// The scope is the Arab League: membership is an objective criterion, it matches
// what the tool is used for in the region, and it keeps the list short enough to
// stay useful. Adding a market means adding one entry to countries below.
//
// Every city carries a bounding box rather than only a centre point. Choosing a
// city runs a grid scrape across that box, so the whole built-up area is covered
// instead of a fixed radius around the middle of town.
package geo

import (
	"fmt"
	"sort"
	"strings"
)

// DefaultCountry is pre-selected in the UI.
const DefaultCountry = "JO"

// City is a searchable place. BBox is min latitude, min longitude, max latitude,
// max longitude, covering the metropolitan area.
type City struct {
	Name string     `json:"name"`
	Lat  float64    `json:"lat"`
	Lon  float64    `json:"lon"`
	BBox [4]float64 `json:"bbox"`
}

// BBoxString renders the box in the form the grid scraper parses.
func (c City) BBoxString() string {
	return fmt.Sprintf("%.4f,%.4f,%.4f,%.4f", c.BBox[0], c.BBox[1], c.BBox[2], c.BBox[3])
}

// AreaKm2 approximates the covered area, used to warn when a selection is large.
func (c City) AreaKm2() float64 {
	const kmPerDegree = 111.0

	latSpan := (c.BBox[2] - c.BBox[0]) * kmPerDegree
	lonSpan := (c.BBox[3] - c.BBox[1]) * kmPerDegree * cosApprox(c.Lat)

	return latSpan * lonSpan
}

// Country groups cities under one market.
type Country struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Lang   string `json:"lang"`
	Cities []City `json:"cities"`
}

// Countries returns every supported market, sorted by name with the default
// country first so it lands at the top of the dropdown.
func Countries() []Country {
	out := make([]Country, len(countries))
	copy(out, countries)

	sort.SliceStable(out, func(i, j int) bool {
		if (out[i].Code == DefaultCountry) != (out[j].Code == DefaultCountry) {
			return out[i].Code == DefaultCountry
		}

		return out[i].Name < out[j].Name
	})

	return out
}

// Lookup resolves a country code and city name to their entries. A blank city
// name returns the country with ok set, so callers can offer "every city".
func Lookup(countryCode, cityName string) (Country, City, bool) {
	for i := range countries {
		if !strings.EqualFold(countries[i].Code, countryCode) {
			continue
		}

		if cityName == "" {
			return countries[i], City{}, true
		}

		for _, c := range countries[i].Cities {
			if strings.EqualFold(c.Name, cityName) {
				return countries[i], c, true
			}
		}

		return countries[i], City{}, false
	}

	return Country{}, City{}, false
}

// cosApprox is a small-table cosine good enough for area estimates, avoiding a
// math import for what is only ever a rough figure shown in the UI.
func cosApprox(latDeg float64) float64 {
	if latDeg < 0 {
		latDeg = -latDeg
	}

	switch {
	case latDeg < 15:
		return 0.97
	case latDeg < 25:
		return 0.91
	case latDeg < 35:
		return 0.82
	default:
		return 0.72
	}
}

func city(name string, lat, lon, minLat, minLon, maxLat, maxLon float64) City {
	return City{Name: name, Lat: lat, Lon: lon, BBox: [4]float64{minLat, minLon, maxLat, maxLon}}
}

//nolint:gochecknoglobals,lll // static reference data, read-only after init.
var countries = []Country{
	{Code: "JO", Name: "Jordan", Lang: "en", Cities: []City{
		city("Amman", 31.9539, 35.9106, 31.7000, 35.7300, 32.1200, 36.0700),
		city("Zarqa", 32.0728, 36.0880, 32.0000, 36.0200, 32.1500, 36.1700),
		city("Irbid", 32.5556, 35.8500, 32.4700, 35.7700, 32.6400, 35.9400),
		city("Aqaba", 29.5321, 35.0063, 29.4500, 34.9400, 29.6200, 35.0700),
		city("Russeifa", 32.0181, 36.0464, 31.9700, 36.0000, 32.0700, 36.0900),
		city("Salt", 32.0392, 35.7272, 31.9800, 35.6600, 32.1000, 35.7900),
		city("Madaba", 31.7160, 35.7958, 31.6600, 35.7300, 31.7700, 35.8600),
		city("Jerash", 32.2808, 35.8990, 32.2200, 35.8300, 32.3400, 35.9600),
		city("Mafraq", 32.3430, 36.2080, 32.2800, 36.1400, 32.4000, 36.2700),
		city("Ajloun", 32.3326, 35.7519, 32.2800, 35.6900, 32.3900, 35.8100),
		city("Karak", 31.1850, 35.7047, 31.1200, 35.6400, 31.2400, 35.7700),
		city("Ma'an", 30.1962, 35.7340, 30.1400, 35.6700, 30.2500, 35.8000),
		city("Tafilah", 30.8376, 35.6042, 30.7800, 35.5400, 30.8900, 35.6600),
		city("Ramtha", 32.5610, 36.0080, 32.5100, 35.9500, 32.6100, 36.0600),
		city("Zai / Deir Alla", 32.1900, 35.6200, 32.1300, 35.5600, 32.2500, 35.6900),
	}},
	{Code: "AE", Name: "United Arab Emirates", Lang: "en", Cities: []City{
		city("Dubai", 25.2048, 55.2708, 24.9800, 55.0300, 25.3500, 55.5800),
		city("Abu Dhabi", 24.4539, 54.3773, 24.3000, 54.2600, 24.5800, 54.6600),
		city("Sharjah", 25.3463, 55.4209, 25.2600, 55.3300, 25.4400, 55.5800),
		city("Al Ain", 24.2075, 55.7447, 24.1000, 55.6400, 24.3200, 55.8600),
		city("Ajman", 25.4052, 55.5136, 25.3400, 55.4300, 25.4700, 55.5800),
		city("Ras Al Khaimah", 25.7895, 55.9432, 25.6600, 55.8300, 25.9000, 56.0700),
		city("Fujairah", 25.1288, 56.3265, 25.0300, 56.2500, 25.2400, 56.3900),
		city("Umm Al Quwain", 25.5647, 55.5532, 25.4900, 55.4800, 25.6300, 55.6500),
	}},
	{Code: "SA", Name: "Saudi Arabia", Lang: "en", Cities: []City{
		city("Riyadh", 24.7136, 46.6753, 24.4700, 46.4300, 24.9500, 47.0000),
		city("Jeddah", 21.4858, 39.1925, 21.2700, 39.0500, 21.8000, 39.3200),
		city("Mecca", 21.3891, 39.8579, 21.2800, 39.7300, 21.5000, 39.9700),
		city("Medina", 24.5247, 39.5692, 24.3900, 39.4400, 24.6600, 39.7000),
		city("Dammam", 26.4207, 50.0888, 26.3000, 49.9500, 26.5500, 50.2200),
		city("Al Khobar", 26.2794, 50.2083, 26.1800, 50.1000, 26.3800, 50.2900),
		city("Taif", 21.2854, 40.4183, 21.1900, 40.3200, 21.3900, 40.5200),
		city("Tabuk", 28.3835, 36.5662, 28.2900, 36.4700, 28.4800, 36.6600),
		city("Abha", 18.2465, 42.5117, 18.1500, 42.4200, 18.3400, 42.6100),
		city("Buraidah", 26.3260, 43.9750, 26.2300, 43.8800, 26.4200, 44.0700),
	}},
	{Code: "EG", Name: "Egypt", Lang: "en", Cities: []City{
		city("Cairo", 30.0444, 31.2357, 29.8700, 31.1000, 30.2200, 31.4200),
		city("Giza", 30.0131, 31.2089, 29.9000, 31.0700, 30.1100, 31.2700),
		city("Alexandria", 31.2001, 29.9187, 31.0900, 29.7500, 31.3300, 30.1100),
		city("Port Said", 31.2653, 32.3019, 31.1900, 32.2200, 31.3300, 32.3700),
		city("Mansoura", 31.0409, 31.3785, 30.9700, 31.3000, 31.1000, 31.4500),
		city("Sharm El Sheikh", 27.9158, 34.3300, 27.8300, 34.2500, 28.0100, 34.4200),
		city("Hurghada", 27.2579, 33.8116, 27.1300, 33.7300, 27.3900, 33.8800),
		city("Luxor", 25.6872, 32.6396, 25.6100, 32.5700, 25.7600, 32.7100),
		city("Aswan", 24.0889, 32.8998, 24.0100, 32.8300, 24.1700, 32.9600),
	}},
	{Code: "QA", Name: "Qatar", Lang: "en", Cities: []City{
		city("Doha", 25.2854, 51.5310, 25.1700, 51.4300, 25.4000, 51.6200),
		city("Al Rayyan", 25.2919, 51.4244, 25.2000, 51.3300, 25.3800, 51.4900),
		city("Al Wakrah", 25.1659, 51.5983, 25.1000, 51.5400, 25.2200, 51.6500),
		city("Lusail", 25.4300, 51.4900, 25.3700, 51.4300, 25.4900, 51.5400),
	}},
	{Code: "KW", Name: "Kuwait", Lang: "en", Cities: []City{
		city("Kuwait City", 29.3759, 47.9774, 29.2700, 47.8800, 29.4500, 48.0800),
		city("Hawalli", 29.3326, 48.0289, 29.2800, 47.9700, 29.3800, 48.0800),
		city("Salmiya", 29.3394, 48.0758, 29.2900, 48.0300, 29.3800, 48.1200),
		city("Al Ahmadi", 29.0769, 48.0838, 29.0000, 48.0100, 29.1500, 48.1600),
		city("Jahra", 29.3375, 47.6581, 29.2700, 47.5900, 29.4000, 47.7300),
	}},
	{Code: "BH", Name: "Bahrain", Lang: "en", Cities: []City{
		city("Manama", 26.2285, 50.5860, 26.1700, 50.5200, 26.2800, 50.6400),
		city("Muharraq", 26.2572, 50.6119, 26.2100, 50.5600, 26.3000, 50.6700),
		city("Riffa", 26.1300, 50.5550, 26.0600, 50.4900, 26.1900, 50.6200),
		city("Isa Town", 26.1736, 50.5478, 26.1300, 50.5000, 26.2100, 50.5900),
	}},
	{Code: "OM", Name: "Oman", Lang: "en", Cities: []City{
		city("Muscat", 23.5880, 58.3829, 23.5000, 58.1500, 23.6800, 58.6000),
		city("Salalah", 17.0151, 54.0924, 16.9300, 54.0000, 17.1000, 54.1800),
		city("Sohar", 24.3643, 56.7468, 24.2900, 56.6700, 24.4400, 56.8200),
		city("Nizwa", 22.9333, 57.5333, 22.8700, 57.4700, 23.0000, 57.6000),
	}},
	{Code: "LB", Name: "Lebanon", Lang: "en", Cities: []City{
		city("Beirut", 33.8938, 35.5018, 33.8200, 35.4400, 33.9400, 35.5700),
		city("Tripoli", 34.4367, 35.8497, 34.3800, 35.7900, 34.4900, 35.9100),
		city("Sidon", 33.5571, 35.3729, 33.5000, 35.3200, 33.6100, 35.4200),
		city("Tyre", 33.2704, 35.2038, 33.2200, 35.1500, 33.3200, 35.2500),
		city("Jounieh", 33.9808, 35.6178, 33.9300, 35.5700, 34.0300, 35.6700),
		city("Zahle", 33.8463, 35.9020, 33.7900, 35.8400, 33.9000, 35.9600),
	}},
	{Code: "PS", Name: "Palestine", Lang: "en", Cities: []City{
		city("Gaza", 31.5017, 34.4668, 31.4400, 34.4100, 31.5600, 34.5200),
		city("Ramallah", 31.9038, 35.2034, 31.8500, 35.1500, 31.9500, 35.2600),
		city("Hebron", 31.5326, 35.0998, 31.4700, 35.0400, 31.5900, 35.1600),
		city("Nablus", 32.2211, 35.2544, 32.1700, 35.2000, 32.2700, 35.3100),
		city("Bethlehem", 31.7054, 35.2024, 31.6600, 35.1600, 31.7500, 35.2500),
		city("Jenin", 32.4597, 35.2956, 32.4100, 35.2400, 32.5100, 35.3500),
		city("Jericho", 31.8667, 35.4500, 31.8200, 35.4000, 31.9100, 35.5000),
	}},
	{Code: "SY", Name: "Syria", Lang: "en", Cities: []City{
		city("Damascus", 33.5138, 36.2765, 33.4400, 36.2000, 33.5900, 36.3600),
		city("Aleppo", 36.2021, 37.1343, 36.1300, 37.0500, 36.2800, 37.2300),
		city("Homs", 34.7308, 36.7090, 34.6700, 36.6400, 34.7900, 36.7700),
		city("Latakia", 35.5138, 35.7713, 35.4600, 35.7100, 35.5700, 35.8300),
		city("Hama", 35.1318, 36.7578, 35.0800, 36.7000, 35.1900, 36.8200),
		city("Tartus", 34.8890, 35.8866, 34.8400, 35.8300, 34.9400, 35.9400),
	}},
	{Code: "IQ", Name: "Iraq", Lang: "en", Cities: []City{
		city("Baghdad", 33.3152, 44.3661, 33.2000, 44.2200, 33.4400, 44.5300),
		city("Basra", 30.5085, 47.7804, 30.4300, 47.7100, 30.5900, 47.8600),
		city("Erbil", 36.1911, 44.0092, 36.1100, 43.9200, 36.2700, 44.1000),
		city("Mosul", 36.3350, 43.1189, 36.2600, 43.0400, 36.4100, 43.2000),
		city("Sulaymaniyah", 35.5556, 45.4351, 35.4900, 45.3600, 35.6200, 45.5100),
		city("Najaf", 32.0000, 44.3300, 31.9400, 44.2700, 32.0600, 44.4000),
		city("Karbala", 32.6160, 44.0242, 32.5500, 43.9600, 32.6700, 44.0900),
	}},
	{Code: "YE", Name: "Yemen", Lang: "en", Cities: []City{
		city("Sana'a", 15.3694, 44.1910, 15.2900, 44.1200, 15.4400, 44.2600),
		city("Aden", 12.7855, 45.0187, 12.7200, 44.9400, 12.8600, 45.1000),
		city("Taiz", 13.5795, 44.0209, 13.5200, 43.9600, 13.6400, 44.0800),
		city("Hodeidah", 14.7978, 42.9545, 14.7400, 42.8900, 14.8600, 43.0100),
	}},
	{Code: "MA", Name: "Morocco", Lang: "en", Cities: []City{
		city("Casablanca", 33.5731, -7.5898, 33.4800, -7.7200, 33.6500, -7.4700),
		city("Rabat", 34.0209, -6.8416, 33.9500, -6.9200, 34.0800, -6.7500),
		city("Marrakesh", 31.6295, -7.9811, 31.5600, -8.0700, 31.7000, -7.9000),
		city("Fes", 34.0181, -5.0078, 33.9500, -5.0900, 34.0900, -4.9300),
		city("Tangier", 35.7595, -5.8340, 35.7000, -5.9200, 35.8200, -5.7500),
		city("Agadir", 30.4278, -9.5981, 30.3600, -9.6700, 30.4900, -9.5300),
	}},
	{Code: "DZ", Name: "Algeria", Lang: "en", Cities: []City{
		city("Algiers", 36.7538, 3.0588, 36.6700, 2.9500, 36.8200, 3.2200),
		city("Oran", 35.6969, -0.6331, 35.6300, -0.7200, 35.7600, -0.5500),
		city("Constantine", 36.3650, 6.6147, 36.3000, 6.5400, 36.4200, 6.6900),
		city("Annaba", 36.9000, 7.7667, 36.8400, 7.7000, 36.9600, 7.8300),
	}},
	{Code: "TN", Name: "Tunisia", Lang: "en", Cities: []City{
		city("Tunis", 36.8065, 10.1815, 36.7300, 10.0800, 36.8900, 10.2800),
		city("Sfax", 34.7406, 10.7603, 34.6800, 10.6900, 34.8000, 10.8300),
		city("Sousse", 35.8256, 10.6084, 35.7600, 10.5400, 35.8900, 10.6700),
	}},
	{Code: "LY", Name: "Libya", Lang: "en", Cities: []City{
		city("Tripoli", 32.8872, 13.1913, 32.8200, 13.0900, 32.9400, 13.3000),
		city("Benghazi", 32.1167, 20.0667, 32.0500, 19.9800, 32.1800, 20.1400),
		city("Misrata", 32.3754, 15.0925, 32.3100, 15.0200, 32.4300, 15.1600),
	}},
	{Code: "SD", Name: "Sudan", Lang: "en", Cities: []City{
		city("Khartoum", 15.5007, 32.5599, 15.4200, 32.4800, 15.6200, 32.6400),
		city("Omdurman", 15.6445, 32.4777, 15.5700, 32.4100, 15.7100, 32.5400),
		city("Port Sudan", 19.6158, 37.2164, 19.5500, 37.1600, 19.6700, 37.2700),
	}},
	{Code: "SO", Name: "Somalia", Lang: "en", Cities: []City{
		city("Mogadishu", 2.0469, 45.3182, 1.9900, 45.2500, 2.1000, 45.3900),
		city("Hargeisa", 9.5600, 44.0650, 9.5000, 44.0000, 9.6200, 44.1300),
	}},
	{Code: "MR", Name: "Mauritania", Lang: "en", Cities: []City{
		city("Nouakchott", 18.0735, -15.9582, 18.0000, -16.0300, 18.1500, -15.8800),
		city("Nouadhibou", 20.9310, -17.0347, 20.8700, -17.0900, 21.0000, -16.9700),
	}},
	{Code: "DJ", Name: "Djibouti", Lang: "en", Cities: []City{
		city("Djibouti City", 11.5721, 43.1456, 11.5100, 43.0800, 11.6300, 43.2100),
	}},
	{Code: "KM", Name: "Comoros", Lang: "en", Cities: []City{
		city("Moroni", -11.7172, 43.2473, -11.7600, 43.2000, -11.6700, 43.2900),
	}},
}
