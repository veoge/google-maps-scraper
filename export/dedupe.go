package export

import "github.com/gosom/google-maps-scraper/gmaps"

// DedupeBySocial collapses businesses that share a Facebook or Instagram
// account down to one row. Chains list every branch on Google Maps, but all the
// branches point at the same social profile, so an outreach campaign that
// messages each profile would otherwise contact the same account several times.
// The first occurrence wins; entries without either profile are always kept.
// The second return value is how many rows were dropped.
func DedupeBySocial(entries []gmaps.Entry) ([]gmaps.Entry, int) {
	seen := map[string]bool{}
	kept := make([]gmaps.Entry, 0, len(entries))

	for i := range entries {
		handles := entries[i].Socials.MessagingHandles()

		dup := false

		for _, h := range handles {
			if seen[h] {
				dup = true

				break
			}
		}

		if dup {
			continue
		}

		for _, h := range handles {
			seen[h] = true
		}

		kept = append(kept, entries[i])
	}

	return kept, len(entries) - len(kept)
}
