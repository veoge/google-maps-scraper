package gmaps

import (
	"net/url"
	"strings"
)

// Socials holds the social media profiles discovered for a business. Each field
// holds at most one URL: the profile page itself, not a deep link into it.
type Socials struct {
	Facebook  string `json:"facebook,omitempty"`
	Instagram string `json:"instagram,omitempty"`
	LinkedIn  string `json:"linkedin,omitempty"`
	TwitterX  string `json:"twitter_x,omitempty"`
	YouTube   string `json:"youtube,omitempty"`
	TikTok    string `json:"tiktok,omitempty"`
	WhatsApp  string `json:"whatsapp,omitempty"`
	Telegram  string `json:"telegram,omitempty"`
	Pinterest string `json:"pinterest,omitempty"`
}

// socialNetwork describes one network: how to recognise its URLs and where to
// store the result. Keeping recognition and storage in one table means the
// extractor and the CSV/XLSX columns can never drift apart.
type socialNetwork struct {
	name string
	// hosts matched after stripping a leading "www."; a host also matches when
	// it is a subdomain of one of these (e.g. "m.facebook.com").
	hosts []string
	field func(*Socials) *string
}

//nolint:gochecknoglobals // lookup table, read-only after init.
var socialNetworks = []socialNetwork{
	{"facebook", []string{"facebook.com", "fb.com", "fb.me"}, func(s *Socials) *string { return &s.Facebook }},
	{"instagram", []string{"instagram.com", "instagr.am"}, func(s *Socials) *string { return &s.Instagram }},
	{"linkedin", []string{"linkedin.com", "lnkd.in"}, func(s *Socials) *string { return &s.LinkedIn }},
	{"twitter_x", []string{"twitter.com", "x.com"}, func(s *Socials) *string { return &s.TwitterX }},
	{"youtube", []string{"youtube.com", "youtu.be"}, func(s *Socials) *string { return &s.YouTube }},
	{"tiktok", []string{"tiktok.com"}, func(s *Socials) *string { return &s.TikTok }},
	{"whatsapp", []string{"wa.me", "whatsapp.com"}, func(s *Socials) *string { return &s.WhatsApp }},
	{"telegram", []string{"t.me", "telegram.me", "telegram.org"}, func(s *Socials) *string { return &s.Telegram }},
	{"pinterest", []string{"pinterest.com", "pin.it"}, func(s *Socials) *string { return &s.Pinterest }},
}

// rejectedSocialPaths are share widgets, login walls and generic destinations.
// They live on the right hosts but are not the business profile, and they appear
// on a large share of websites, so without this filter almost every result would
// come back with a bogus Facebook "profile".
//
//nolint:gochecknoglobals // lookup table, read-only after init.
var rejectedSocialPaths = []string{
	"/sharer", "/share", "/share_channel", "/intent",
	"/plugins/", "/dialog/", "/home.php", "/login", "/signup", "/register",
	"/help", "/about", "/privacy", "/policy", "/terms", "/legal",
	"/settings", "/widgets/", "/embed", "/oauth", "/pixel",
	"/hashtag/", "/explore/", "/search", "/results", "/watch",
	"/pages/create", "/developers", "/apps/",
}

// reservedSocialHandles are first path segments that never identify a business.
//
//nolint:gochecknoglobals // lookup table, read-only after init.
var reservedSocialHandles = map[string]bool{
	"p": true, "reel": true, "reels": true, "stories": true, "tv": true,
	"posts": true, "photo": true, "photos": true, "video": true, "videos": true,
	"events": true, "event": true, "groups": true, "group": true,
	"marketplace": true, "watch": true, "gaming": true, "jobs": true,
	"feed": true, "pulse": true, "learning": true, "status": true,
	"i": true, "home": true, "profile.php": true, "sharer.php": true,
}

// All returns every discovered profile URL in a stable order, so the combined
// "social" column is deterministic across runs.
func (s *Socials) All() []string {
	out := make([]string, 0, len(socialNetworks))

	cp := *s

	for i := range socialNetworks {
		if v := *socialNetworks[i].field(&cp); v != "" {
			out = append(out, v)
		}
	}

	return out
}

// IsEmpty reports whether no profile was found at all.
func (s *Socials) IsEmpty() bool {
	return len(s.All()) == 0
}

// String renders the profiles for the combined "social" column.
func (s *Socials) String() string {
	return strings.Join(s.All(), ", ")
}

// Merge fills empty fields of s from other, preferring what s already holds.
func (s *Socials) Merge(other *Socials) {
	for i := range socialNetworks {
		dst := socialNetworks[i].field(s)
		if *dst != "" {
			continue
		}

		src := socialNetworks[i].field(other)
		*dst = *src
	}
}

// extractSocials picks the best profile URL per network out of raw hrefs. When a
// network appears more than once the shallowest URL wins, because a profile root
// ("/acmecafe") is what we want rather than a deep link into it
// ("/acmecafe/posts/12345").
func extractSocials(hrefs []string) Socials {
	type candidate struct {
		url   string
		depth int
	}

	best := map[string]candidate{}

	for _, href := range hrefs {
		normalized, network, depth, ok := classifySocialURL(href)
		if !ok {
			continue
		}

		if cur, seen := best[network]; seen && cur.depth <= depth {
			continue
		}

		best[network] = candidate{url: normalized, depth: depth}
	}

	var out Socials

	for i := range socialNetworks {
		if c, ok := best[socialNetworks[i].name]; ok {
			*socialNetworks[i].field(&out) = c.url
		}
	}

	return out
}

// classifySocialURL normalizes a href and reports which network it belongs to.
// The returned depth is the number of path segments, used to prefer profile
// roots over deep links.
func classifySocialURL(href string) (normalized, network string, depth int, ok bool) {
	raw := strings.TrimSpace(href)
	if raw == "" {
		return "", "", 0, false
	}

	// Protocol-relative links ("//facebook.com/x") are common in footers.
	if strings.HasPrefix(raw, "//") {
		raw = "https:" + raw
	}

	if !strings.Contains(raw, "://") {
		return "", "", 0, false
	}

	u, err := url.Parse(raw)
	if err != nil {
		return "", "", 0, false
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return "", "", 0, false
	}

	host := strings.ToLower(u.Hostname())
	host = strings.TrimPrefix(host, "www.")

	network, ok = matchSocialHost(host)
	if !ok {
		return "", "", 0, false
	}

	path := strings.TrimSuffix(u.EscapedPath(), "/")

	lowerPath := strings.ToLower(path)
	for _, bad := range rejectedSocialPaths {
		if strings.HasPrefix(lowerPath, bad) {
			return "", "", 0, false
		}
	}

	segments := splitPathSegments(path)

	// A bare domain is a link to the network itself, not to a business profile.
	if len(segments) == 0 {
		return "", "", 0, false
	}

	if reservedSocialHandles[strings.ToLower(segments[0])] {
		return "", "", 0, false
	}

	// Rebuild without query and fragment: those carry tracking parameters
	// (utm_*, fbclid, ref) that differ per page and would defeat de-duplication.
	// WhatsApp links are the exception, since the phone number lives in the query.
	cleaned := url.URL{
		Scheme: "https",
		Host:   u.Hostname(),
		Path:   path,
	}

	if keepsSocialQuery(host) {
		cleaned.RawQuery = u.RawQuery
	}

	return cleaned.String(), network, len(segments), true
}

// isSocialWebsite reports whether rawURL points at a social network rather than
// at a site of the business's own. Such a URL is worth recording as a profile but
// is not worth crawling for contact details.
func isSocialWebsite(rawURL string) bool {
	raw := strings.TrimSpace(rawURL)
	if strings.HasPrefix(raw, "//") {
		raw = "https:" + raw
	}

	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}

	u, err := url.Parse(raw)
	if err != nil {
		return false
	}

	host := strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")

	_, ok := matchSocialHost(host)

	return ok
}

// matchSocialHost reports the network owning host, allowing subdomains such as
// "m.facebook.com" or "uk.linkedin.com".
func matchSocialHost(host string) (string, bool) {
	for i := range socialNetworks {
		for _, h := range socialNetworks[i].hosts {
			if host == h || strings.HasSuffix(host, "."+h) {
				return socialNetworks[i].name, true
			}
		}
	}

	return "", false
}

// keepsSocialQuery reports whether the query string carries the identity of the
// target rather than tracking noise.
func keepsSocialQuery(host string) bool {
	return strings.HasSuffix(host, "whatsapp.com")
}

func splitPathSegments(path string) []string {
	parts := strings.Split(strings.Trim(path, "/"), "/")

	out := make([]string, 0, len(parts))

	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}

	return out
}

// MessagingHandles returns a normalized key for each network a business can be
// messaged on directly (Facebook and Instagram), such as "instagram:acme.cafe".
// Two profiles that differ only in scheme, "www."/"m." prefix, letter case or a
// trailing slash produce the same key, so the keys are what to compare when
// collapsing multi-branch chains down to one contactable account.
func (s *Socials) MessagingHandles() []string {
	out := make([]string, 0, 2)

	for _, p := range []struct{ network, rawURL string }{
		{"facebook", s.Facebook},
		{"instagram", s.Instagram},
	} {
		if key := socialHandleKey(p.network, p.rawURL); key != "" {
			out = append(out, key)
		}
	}

	return out
}

// socialHandleKey reduces a profile URL to "network:path" with the host and any
// tracking noise removed. It returns "" when rawURL is empty or unparseable.
func socialHandleKey(network, rawURL string) string {
	raw := strings.TrimSpace(rawURL)
	if raw == "" {
		return ""
	}

	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}

	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}

	path := strings.ToLower(strings.Trim(u.EscapedPath(), "/"))
	if path == "" {
		return ""
	}

	return network + ":" + path
}
