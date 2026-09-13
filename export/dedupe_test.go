package export_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gosom/google-maps-scraper/export"
	"github.com/gosom/google-maps-scraper/gmaps"
)

func TestDedupeBySocialKeepsOneRowPerAccount(t *testing.T) {
	t.Parallel()

	withSocials := func(title, fb, ig string) gmaps.Entry {
		return gmaps.Entry{Title: title, Socials: gmaps.Socials{Facebook: fb, Instagram: ig}}
	}

	entries := []gmaps.Entry{
		withSocials("Marouf Coffee - Abdoun", "https://www.facebook.com/MaroufCoffee", "https://www.instagram.com/maroufcoffeeofficial"),
		withSocials("Marouf Coffee - Sweifieh", "https://m.facebook.com/maroufcoffee/", ""),
		withSocials("Marouf Coffee - Khalda", "", "https://instagram.com/MaroufCoffeeOfficial"),
		withSocials("Blue Fig", "https://www.facebook.com/BlueFigJo", "https://www.instagram.com/bluefigjo"),
		withSocials("No socials A", "", ""),
		withSocials("No socials B", "", ""),
		withSocials("Blue Fig - Airport", "https://www.facebook.com/BlueFigJo", "https://www.instagram.com/bluefig.airport"),
	}

	kept, removed := export.DedupeBySocial(entries)

	require.Equal(t, 3, removed)

	titles := make([]string, 0, len(kept))
	for i := range kept {
		titles = append(titles, kept[i].Title)
	}

	require.Equal(t, []string{
		"Marouf Coffee - Abdoun",
		"Blue Fig",
		"No socials A",
		"No socials B",
	}, titles, "the first branch wins; scheme, www/m prefix, case and trailing slash must not matter; rows without socials are kept")
}

func TestDedupeBySocialEmpty(t *testing.T) {
	t.Parallel()

	kept, removed := export.DedupeBySocial(nil)
	require.Empty(t, kept)
	require.Zero(t, removed)
}
