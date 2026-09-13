package gmaps

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractSocialsPicksProfilesAndIgnoresWidgets(t *testing.T) {
	t.Parallel()

	hrefs := []string{
		"https://www.facebook.com/sharer/sharer.php?u=https://acme.example",
		"https://www.facebook.com/acmecafe/posts/123456",
		"https://www.facebook.com/acmecafe",
		"https://twitter.com/intent/tweet?text=hi",
		"https://x.com/acmecafe",
		"//instagram.com/acme.cafe/?hl=en",
		"https://www.linkedin.com/company/acme-cafe/",
		"https://youtube.com/@acmecafe",
		"https://wa.me/962790000000",
		"https://www.tiktok.com/@acmecafe",
		"mailto:hi@acme.example",
		"/about-us",
		"https://acme.example/contact",
	}

	got := extractSocials(hrefs)

	require.Equal(t, "https://www.facebook.com/acmecafe", got.Facebook,
		"the profile root should win over a deep link, and the share widget must be ignored")
	require.Equal(t, "https://x.com/acmecafe", got.TwitterX)
	require.Equal(t, "https://instagram.com/acme.cafe", got.Instagram,
		"tracking query parameters should be stripped")
	require.Equal(t, "https://www.linkedin.com/company/acme-cafe", got.LinkedIn)
	require.Equal(t, "https://youtube.com/@acmecafe", got.YouTube)
	require.Equal(t, "https://wa.me/962790000000", got.WhatsApp)
	require.Equal(t, "https://www.tiktok.com/@acmecafe", got.TikTok)
}

func TestExtractSocialsRejectsBareDomainsAndReservedPaths(t *testing.T) {
	t.Parallel()

	got := extractSocials([]string{
		"https://www.facebook.com",
		"https://www.facebook.com/",
		"https://instagram.com/p/Cx1234567",
		"https://www.linkedin.com/feed/",
		"https://twitter.com/home",
	})

	require.True(t, got.IsEmpty(), "none of these identify a business, got %v", got.All())
}

func TestSocialsStringIsStableAndOrdered(t *testing.T) {
	t.Parallel()

	s := Socials{
		Instagram: "https://instagram.com/b",
		Facebook:  "https://facebook.com/a",
		WhatsApp:  "https://wa.me/1",
	}

	require.Equal(t,
		"https://facebook.com/a, https://instagram.com/b, https://wa.me/1",
		s.String())
}

func TestSocialsMergeKeepsExistingValues(t *testing.T) {
	t.Parallel()

	s := Socials{Facebook: "https://facebook.com/from-maps"}
	found := Socials{
		Facebook:  "https://facebook.com/from-footer",
		Instagram: "https://instagram.com/from-footer",
	}

	s.Merge(&found)

	require.Equal(t, "https://facebook.com/from-maps", s.Facebook,
		"the value taken from the Maps listing is more authoritative")
	require.Equal(t, "https://instagram.com/from-footer", s.Instagram)
}

func TestIsWebsiteValidForEmail(t *testing.T) {
	t.Parallel()

	// The old check tested for "instragram" (sic), so Instagram-only businesses
	// were crawled for emails pointlessly.
	for _, website := range []string{
		"https://www.facebook.com/acmecafe",
		"https://instagram.com/acmecafe",
		"https://www.tiktok.com/@acmecafe",
	} {
		e := Entry{WebSite: website}
		require.False(t, e.IsWebsiteValidForEmail(), website)
	}

	e := Entry{WebSite: "https://acme.example"}
	require.True(t, e.IsWebsiteValidForEmail())
}

func TestEntryAdoptsSocialWebsite(t *testing.T) {
	t.Parallel()

	e := Entry{WebSite: "https://www.facebook.com/acmecafe"}
	e.adoptWebsiteAsSocial()

	require.Equal(t, "https://www.facebook.com/acmecafe", e.Socials.Facebook)
}

func TestCsvHeadersAndRowStayInSync(t *testing.T) {
	t.Parallel()

	e := Entry{
		Title:   "Acme",
		Socials: Socials{Facebook: "https://facebook.com/acme"},
	}

	require.Equal(t, len(e.CsvHeaders()), len(e.CsvRow()))
	require.Contains(t, e.CsvHeaders(), "social")
}

func TestMessagingHandlesNormalize(t *testing.T) {
	t.Parallel()

	s := Socials{
		Facebook:  "https://m.facebook.com/AcmeCafe/",
		Instagram: "instagram.com/Acme.Cafe",
		LinkedIn:  "https://www.linkedin.com/company/acme-cafe",
	}

	require.Equal(t, []string{"facebook:acmecafe", "instagram:acme.cafe"}, s.MessagingHandles(),
		"only Facebook and Instagram are messaging channels; host, case and slashes are normalized away")

	require.Empty(t, (&Socials{}).MessagingHandles())
}
