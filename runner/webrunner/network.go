package webrunner

import (
	"context"
	"net/http"
	"time"
)

// Connectivity is checked against hosts that are cheap to reach and unlikely to
// be down together. Google is included because it is what the scraper actually
// depends on: reaching the wider internet is not enough if Maps is unreachable.
//
//nolint:gochecknoglobals // lookup table, read-only after init.
var connectivityProbes = []string{
	"https://www.google.com/generate_204",
	"https://connectivitycheck.gstatic.com/generate_204",
	"https://cloudflare.com/cdn-cgi/trace",
}

const (
	probeTimeout = 6 * time.Second
	// How long the connection has to stay down before a run is paused. Short
	// enough to stop wasting the retry budget, long enough to ride out a blip.
	outageGrace = 25 * time.Second
	// How often the watchdog looks while a scrape is running.
	probeInterval = 10 * time.Second
)

// networkUp reports whether any probe host answers.
func networkUp(ctx context.Context) bool {
	client := &http.Client{Timeout: probeTimeout}

	for _, url := range connectivityProbes {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
		if err != nil {
			continue
		}

		resp, err := client.Do(req)
		if err != nil {
			continue
		}

		_ = resp.Body.Close()

		if resp.StatusCode < 500 {
			return true
		}
	}

	return false
}

// watchNetwork cancels onOutage when connectivity has been gone for longer than
// outageGrace, and reports whether it fired.
//
// Without this, losing the connection mid-run is silently destructive: every
// in-flight search burns its three retries within seconds, is marked failed, and
// is never queued again — so the run finishes looking successful while missing
// whole areas of the map.
func watchNetwork(ctx context.Context, onOutage context.CancelFunc, fired *bool) {
	ticker := time.NewTicker(probeInterval)
	defer ticker.Stop()

	var downSince time.Time

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		if networkUp(ctx) {
			downSince = time.Time{}

			continue
		}

		if downSince.IsZero() {
			downSince = time.Now()

			continue
		}

		if time.Since(downSince) >= outageGrace {
			*fired = true

			onOutage()

			return
		}
	}
}
