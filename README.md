# Google Maps Scraper

<p align="center">
  <a href="https://github.com/gosom/google-maps-scraper/stargazers"><img src="https://img.shields.io/github/stars/gosom/google-maps-scraper?style=social" alt="GitHub Stars"></a>
  <a href="https://github.com/gosom/google-maps-scraper/network/members"><img src="https://img.shields.io/github/forks/gosom/google-maps-scraper?style=social" alt="GitHub Forks"></a>
  <a href="https://twitter.com/intent/tweet?text=Powerful%20open-source%20Google%20Maps%20scraper%20-%20extract%20business%20data%20at%20scale%20with%20CLI%2C%20Web%20UI%2C%20or%20REST%20API&url=https%3A%2F%2Fgithub.com%2Fgosom%2Fgoogle-maps-scraper&hashtags=golang,webscraping,googlemaps,opensource"><img src="https://img.shields.io/twitter/url/http/shields.io.svg?style=social" alt="Tweet"></a>
</p>

[![Build Status](https://github.com/gosom/google-maps-scraper/actions/workflows/build.yml/badge.svg)](https://github.com/gosom/google-maps-scraper/actions/workflows/build.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/gosom/google-maps-scraper)](https://goreportcard.com/report/github.com/gosom/google-maps-scraper)
[![GoDoc](https://godoc.org/github.com/gosom/google-maps-scraper?status.svg)](https://godoc.org/github.com/gosom/google-maps-scraper)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Discord](https://img.shields.io/badge/Discord-Join%20Chat-7289DA?logo=discord&logoColor=white)](https://discord.gg/fpaAVhNCCu)

Extract Google Maps business leads, emails, reviews, phone numbers, websites, ratings, coordinates, and more with a free open-source CLI, Web UI, REST API, and optional self-hosted platform.

Use it for lead generation, local business research, sales prospecting, data enrichment, or developer automation.

## Ask an AI Agent to Get Leads

The easiest way to use Google Maps Scraper is with an AI coding agent such as [Claude Code](https://claude.com/claude-code), Codex, Cursor, GitHub Copilot, or any [Agent Skills-compatible tool](https://agentskills.io). You describe the leads you want; the agent plans the searches, runs a small validation, starts the full local scrape, monitors it, and helps you work with the results.

Install the skill:

```bash
npx skills add gosom/google-maps-scraper
```

Then ask your agent in plain language:

> Find dentists in Berlin and include their websites and email addresses.

The agent automatically checks for the latest skill and Docker image, then asks only for details it still needs. For larger crawls, you can provide your own proxy, continue without one, or review three randomly selected proxy sponsors. Proxy credentials are entered through a masked local terminal prompt and are never pasted into the agent chat.

Requires Docker and Node.js on macOS, Linux, or Windows through WSL. See [how the agent workflow works](#ai-agent-skill).

| Goal | Start here |
|---|---|
| Get leads into CSV/JSON | [Command Line](#command-line) |
| Ask an AI coding agent to run a scrape | [AI Agent Skill](#ai-agent-skill) |
| Run a browser UI locally | [Web UI](#web-ui) |
| Automate scraping from your app | [REST API](#rest-api) |
| Run a multi-user scraping platform | [SaaS Edition](docs/saas.md) |
| Follow common workflows | [Recipes](docs/recipes.md) |

![Example GIF](img/example.gif)

If this project is useful to you, a GitHub star helps others discover it. Sponsorships help fund maintenance and new work.

---

## Sponsored By

<p align="center"><i>This project is made possible by our amazing sponsors</i></p>

### [Coreclaw](https://www.coreclaw.com/?utm_source=github&utm_medium=referral&utm_campaign=gosom&utm_term=&utm_id=gosom) - Full-stack web scraping and data extraction platform

[![Coreclaw - Full-stack web scraping and data extraction platform](./img/coreclaw.png)](https://www.coreclaw.com/?utm_source=github&utm_medium=referral&utm_campaign=gosom&utm_term=&utm_id=gosom)

Find ready-made workers for public websites, run them instantly, and get structured data you can export or connect anywhere. [**Get free test for $3 →**](https://www.coreclaw.com/?utm_source=github&utm_medium=referral&utm_campaign=gosom&utm_term=&utm_id=gosom)

---

### [G Maps Extractor](https://gmapsextractor.com?utm_source=github&utm_medium=banner&utm_campaign=gosom) - No-code Google Maps scraper

[![G Maps Extractor](./img/gmaps-extractor-banner.png)](https://gmapsextractor.com?utm_source=github&utm_medium=banner&utm_campaign=gosom)

Chrome extension that extracts emails, social profiles, phone numbers, reviews & more. [**Get 1,000 free leads →**](https://gmapsextractor.com?utm_source=github&utm_medium=banner&utm_campaign=gosom)

---

### [Scrap.io](https://scrap.io?utm_medium=ads&utm_source=github_gosom_gmap_scraper) - Extract ALL Google Maps listings at country-scale

[![Scrap.io - Extract ALL Google Maps Listings](./img/premium_scrap_io.png)](https://scrap.io?utm_medium=ads&utm_source=github_gosom_gmap_scraper)

No keywords needed. No limits. Export millions of businesses in 2 clicks. [**Try it free →**](https://scrap.io?utm_medium=ads&utm_source=github_gosom_gmap_scraper)

---

### [SerpApi](https://serpapi.com/?utm_source=google-maps-scraper) - Google Maps API and 30+ search engine APIs

[![SerpApi](./img/SerpApi-banner.png)](https://serpapi.com/?utm_source=google-maps-scraper)

Fast, reliable, and scalable. Used by Fortune 500 companies. [**View all APIs →**](https://serpapi.com/search-api)

---

### [SearchApi](https://www.searchapi.io/google-maps?via=gosom&utm_source=github&utm_medium=sponsorship&utm_campaign=gosom) - Google Maps API for SERP scraping

[![SearchApi](./img/searchapi_google_maps.png)](https://www.searchapi.io/google-maps?via=gosom&utm_source=github&utm_medium=sponsorship&utm_campaign=gosom)

Real-time Google Maps data with a simple integration. [**Explore the API →**](https://www.searchapi.io/google-maps?via=gosom&utm_source=github&utm_medium=sponsorship&utm_campaign=gosom)

---

### [Evomi](https://evomi.com?utm_source=github&utm_medium=banner&utm_campaign=gosom-maps) - Swiss quality proxies for scraping

[![Evomi](https://my.evomi.com/images/brand/cta.png)](https://evomi.com?utm_source=github&utm_medium=banner&utm_campaign=gosom-maps)

Swiss quality proxies from $0.49/GB across 150+ countries, with 24/7 support and 99.9% uptime. [**Visit Evomi →**](https://evomi.com?utm_source=github&utm_medium=banner&utm_campaign=gosom-maps)

---

### [HasData](https://hasdata.com/scrapers/google-maps?utm_source=github&utm_medium=sponsorship&utm_campaign=gosom) - No-code Google Maps Scraper & Email Extraction

[![HasData Google Maps Scraper](./img/hd-gm-banner.png)](https://hasdata.com/scrapers/google-maps?utm_source=github&utm_medium=sponsorship&utm_campaign=gosom)

Extract business leads, emails, addresses, phones, reviews and more. [**Get 1,000 free credits →**](https://hasdata.com/scrapers/google-maps?utm_source=github&utm_medium=sponsorship&utm_campaign=gosom)

---

### [RapidProxy](https://www.rapidproxy.io/?ref=gosom) - High-Performance Proxy Solution

[![RapidProxy](./img/rapidproxy-banner.png)](https://www.rapidproxy.io/?ref=gosom)

Unlock global access with consistent, high-speed connections from $0.65/GB, 90M+ real residential IPs worldwide, and traffic that never expires. [**Try it free →**](https://www.rapidproxy.io/?ref=gosom)

---

### [TalorData](https://talordata.com/?campaignid=f01u8cHondg2qA47&utm_source=github&utm_term=googlemaps) - Fast SERP API for Google Maps and Search Data

[![TalorData](./img/talordata.png)](https://talordata.com/?campaignid=f01u8cHondg2qA47&utm_source=github&utm_term=googlemaps)

Real-time SERP data APIs for Google Maps and search results, with structured JSON / HTML responses and 1,000 free API responses to start. [**Start using TalorData →**](https://talordata.com/?campaignid=f01u8cHondg2qA47&utm_source=github&utm_term=googlemaps) | [Learn more](talordata.md)

---

### [Webshare](https://www.webshare.io/?referral_code=0q3l81eet8mp) - Premium proxies for scraping at scale

[![Webshare](./img/webshare-banner.png)](https://www.webshare.io/?referral_code=0q3l81eet8mp)

The most affordable premium proxies across 195 countries & 80+ million IPs, plus a FREE plan for new users. [Learn more](webshare.md)

---

### [BirdProxies](https://birdproxies.com/?utm_source=github&utm_medium=sponsorship&utm_campaign=gosom-google-maps-scraper) - Residential and ISP proxies

[![BirdProxies](./img/birdproxies.png)](https://birdproxies.com/?utm_source=github&utm_medium=sponsorship&utm_campaign=gosom-google-maps-scraper)

Hey, we built BirdProxies because proxies shouldn't be complicated or overpriced. Fast residential and ISP proxies in 195+ locations, fair pricing, and real support. Try our FlappyBird game on the landing page for free data!

[**Visit BirdProxies →**](https://birdproxies.com/?utm_source=github&utm_medium=sponsorship&utm_campaign=gosom-google-maps-scraper) | [Join Discord](https://discord.com/invite/birdproxies)

---

### [Proxidize](https://proxidize.com/?utm_source=github&utm_medium=sponsorship&utm_campaign=google_maps_scraper&utm_content=gosom) - Proxies for Google Maps Scraping

[![Proxidize | Proxies for Google Maps Scraping](https://imagedelivery.net/r4caA8hJ3Ww3j8uyC_NNCA/23ee92b0-9fae-4c55-6865-9ca35387fb00/public)](https://proxidize.com/?utm_source=github&utm_medium=sponsorship&utm_campaign=google_maps_scraper&utm_content=gosom)

Mobile and residential proxies for Google Maps scraping, local SEO, lead generation, and data collection. Use code `gmaps20` for 20% off. [**Visit Proxidize →**](https://proxidize.com/?utm_source=github&utm_medium=sponsorship&utm_campaign=google_maps_scraper&utm_content=gosom)

---

### [NodeMaven](https://go.nodemaven.com/GoogleMapsScrapperaugust)

[![NodeMaven - The most efficient proxy provider for Web Scraping and Automation](./img/nodemaven.png)](https://go.nodemaven.com/GoogleMapsScrapperaugust)

[**NodeMaven**](https://go.nodemaven.com/GoogleMapsScrapperaugust): The most efficient proxy provider for Web Scraping and Automation with the Highest Quality IP on the market.

Why [**NodeMaven**](https://go.nodemaven.com/GoogleMapsScrapperaugust)?

- ZIP targeting
- 99.9% uptime
- IP filtering: all proxies have fraud score <97%
- No KYC required
- Unique free tools: Proxy Bandwidth Checker, Meta Tag Checker, IP Lookup and others!

**Special codes for Google Maps Scraper users:**

- `MAPS35` - 35% off to Mobile and Residential Proxies
- `MAPS40` - 40% off to ISP (Static) Proxies

[**Visit NodeMaven →**](https://go.nodemaven.com/GoogleMapsScrapperaugust)

---

<p align="center">
  <a href="#sponsored-by">View all sponsors</a> | <a href="https://github.com/sponsors/gosom">Become a sponsor</a>
</p>

---

## Why Use This Scraper?

| | |
|---|---|
| **Completely Free & Open Source** | MIT licensed, no hidden costs or usage limits |
| **Multiple Interfaces** | CLI, Web UI, REST API - use what fits your workflow |
| **High Performance** | ~120 places/minute with optimized concurrency |
| **33+ Data Points** | Business details, reviews, emails, coordinates, and more |
| **Production Ready** | Scale from a single machine to Kubernetes clusters |
| **Flexible Output** | CSV, JSON, PostgreSQL, S3, LeadsDB, or custom plugins |
| **Proxy Support** | Built-in SOCKS5/HTTP/HTTPS proxy rotation |

---

## What's Next After Scraping?

Once you've collected your data, you'll need to manage, deduplicate, and work with your leads. **[LeadsDB](https://getleadsdb.com/)** is a companion tool designed exactly for this:

- **Automatic Deduplication** - Import from multiple scrapes without worrying about duplicates
- **AI Agent Ready** - Query and manage leads with natural language via MCP
- **Advanced Filtering** - Combine filters with AND/OR logic on any field
- **Export Anywhere** - CSV, JSON, or use the REST API

The scraper has [built-in LeadsDB integration](#export-to-leadsdb) - just add your API key and leads flow directly into your database.

**[Start free with 500 leads](https://getleadsdb.com/)**

---

## Table of Contents

- [Quick Start](#quick-start)
  - [Command Line](#command-line)
  - [Web UI](#web-ui)
  - [REST API](#rest-api)
  - [SaaS Edition](#saas-edition)
- [AI Agent Skill](#ai-agent-skill)
- [Recipes](docs/recipes.md)
- [Proxy Sponsors](docs/proxies.md)
- [Installation](#installation)
- [Features](#features)
- [Extracted Data Points](#extracted-data-points)
- [Configuration](#configuration)
  - [Command Line Options](#command-line-options)
  - [Using Proxies](#using-proxies)
  - [Email Extraction](#email-extraction)
  - [Fast Mode](#fast-mode)
- [Export to LeadsDB](#export-to-leadsdb)
- [Advanced Usage](#advanced-usage)
  - [PostgreSQL Database Provider](#postgresql-database-provider)
  - [Kubernetes Deployment](#kubernetes-deployment)
  - [Custom Writer Plugins](#custom-writer-plugins)
- [Performance](#performance)
- [Support the Project](#support-the-project)
- [Community](#community)
- [Contributing](#contributing)
- [License](#license)

---

## Quick Start

### Command Line

```bash
mkdir -p gmaps-output

docker run \
  -v gmaps-playwright-cache:/opt \
  -v "$PWD/example-queries.txt:/queries.txt:ro" \
  -v "$PWD/gmaps-output:/out" \
  gosom/google-maps-scraper \
  -input /queries.txt \
  -results /out/results.csv \
  -depth 1 \
  -exit-on-inactivity 3m
```

Useful options:

| Need | Flag |
|---|---|
| Extract emails from business websites | `-email` |
| Write JSON instead of CSV | `-json -results /out/results.json` |
| Collect extra reviews | `-extra-reviews -json -results /out/results.json` |
| Increase concurrency | `-c 4`, `-c 8`, or `-c 16` |
| Run multiple pages per browser | `-pages-per-browser 4` |
| Limit browser processes | `-browser-pool-size 2` |
| Use proxies | `-proxies "http://user:pass@host:port,socks5://host:port"` |
| Read proxies from a credentials file | `-proxies-file /path/to/proxies.txt` |

`-c` controls how many scrape jobs run in parallel. Higher concurrency can finish large input files faster, but it also uses more CPU/RAM and can increase blocking or failures, especially without proxies. Start with the default for a first run. For larger jobs on a capable machine, try `-c 4`, `-c 8`, or `-c 16` and measure the result.

**Want to skip CSV files?** Send leads directly to [LeadsDB](https://getleadsdb.com/):

```bash
docker run \
  -v gmaps-playwright-cache:/opt \
  -v "$PWD/example-queries.txt:/queries.txt:ro" \
  gosom/google-maps-scraper \
  -input /queries.txt \
  -depth 1 \
  -leadsdb-api-key "your-api-key" \
  -exit-on-inactivity 3m
```

### Web UI

Start the web interface with a single command:

```bash
mkdir -p gmapsdata

docker run \
  -v "$PWD/gmapsdata:/gmapsdata" \
  -p 8080:8080 \
  gosom/google-maps-scraper \
  -data-folder /gmapsdata
```

Then open http://localhost:8080 in your browser.

The browser interface shows results directly, so you do not have to download a file to see what a
job found:

- **Country and city pickers** covering the Arab League markets, Jordan first. Choosing a city
  scrapes the **whole city**, not a radius around its centre
- **Results table** with search, sorting, and filters for search term, rating, category, city, and
  whether a business has a website, phone, email, or social profile
- **Detail panel** for any row: contact details, social profiles, rating breakdown, opening hours,
  amenities, and review text
- **Map view** of the current filter selection
- **Export** as a formatted Excel workbook, CSV, or JSON

Or download the [binary release](https://github.com/gosom/google-maps-scraper/releases) for your platform.

> **Note:** Results take at least 3 minutes to appear (minimum configured runtime).
> 
> **macOS Users:** Docker command may not work. See [MacOS Instructions](MacOS%20instructions.md).

### REST API

When running the web server, a full REST API is available:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/jobs` | POST | Create a new scraping job |
| `/api/v1/jobs` | GET | List all jobs |
| `/api/v1/jobs/{id}` | GET | Get job details |
| `/api/v1/jobs/{id}` | DELETE | Delete a job |
| `/api/v1/jobs/{id}/download` | GET | Download results as CSV |

Full OpenAPI 3.0.3 documentation available at http://localhost:8080/api/docs

### SaaS Edition

Need a multi-user platform with API keys, admin UI, job queue, workers, and cloud provisioning? Use the optional self-hosted SaaS edition:

```bash
curl -fsSL https://raw.githubusercontent.com/gosom/google-maps-scraper/main/PROVISION | sh
```

See [SaaS documentation](docs/saas.md) for deployment and operations details.
There is also a [5-minute deployment walkthrough](https://gosom.dev/deploy-your-own-maps-scraping-api-in-5-minutes/) and a [YouTube video walkthrough](https://www.youtube.com/watch?v=STG9mZw_nac).

More examples are available in [Recipes](docs/recipes.md). If you need proxies for larger jobs, see [Proxy Sponsors](docs/proxies.md).

---

## AI Agent Skill

The AI Agent Skill turns a natural-language lead request into a guided local scraping workflow. It is designed for nontechnical users as well as developers and keeps you in control of the search scope, proxy choice, and output.

If you have not installed it yet:

```bash
npx skills add gosom/google-maps-scraper
```

Then just ask your agent:

> Find me all dentists in Berlin with their emails

The agent will:

1. Noninteractively check for the latest skill and Docker image.
2. Infer sensible search defaults and ask only for missing essentials.
3. Let you use your own proxy, continue without one, or choose from three randomly selected proxy sponsors.
4. Collect proxy credentials through a masked local terminal prompt instead of chat.
5. Run a small validation scrape before the full job.
6. Start and monitor the Docker crawl in the background.
7. Present the results with options to save, filter, analyze, or export.

Proxy sponsor recommendations are clearly disclosed and shown with equal placement. Any discount or offer is displayed only when it is configured in the skill's [active sponsor registry](skills/google-maps-scraper/references/proxy-sponsors.json). You can always use another provider or no proxy.

Requires Docker and Node.js on macOS, Linux, or Windows through WSL. See the [skill definition](skills/google-maps-scraper/SKILL.md) for details.

---

## Installation

### Using Docker (Recommended)

The published Docker image uses Playwright:

```bash
docker pull gosom/google-maps-scraper
```

### Build from Source

Requirements: Go 1.26.6+

```bash
git clone https://github.com/gosom/google-maps-scraper.git
cd google-maps-scraper
go mod download

go build
./google-maps-scraper -input example-queries.txt -results results.csv -exit-on-inactivity 3m
```

> First run downloads required browser libraries for Playwright.

---

## Features

| Feature | Description |
|---------|-------------|
| **33+ Data Points** | Business name, address, phone, website, reviews, coordinates, and more |
| **Email Extraction** | Optional crawling of business websites for email addresses |
| **Multiple Output Formats** | CSV, JSON, PostgreSQL, S3, LeadsDB, or custom plugins |
| **Proxy Support** | SOCKS5, HTTP, HTTPS with authentication |
| **Scalable Architecture** | Single machine to Kubernetes cluster |
| **REST API** | Programmatic control for automation |
| **Web UI** | User-friendly browser interface |
| **Fast Mode (Beta)** | Quick extraction of up to 21 results per query |
| **AWS Lambda** | Serverless execution support (experimental) |

---

## Extracted Data Points

<details>
<summary><strong>Click to expand all 45 data points</strong></summary>

| # | Field | Description |
|---|-------|-------------|
| 1 | `input_id` | Internal identifier for the input query |
| 2 | `search_term` | The search that produced this result; groups the Excel export |
| 3 | `link` | Direct URL to the Google Maps listing |
| 4 | `title` | Business name |
| 5 | `category` | Business type (e.g., Restaurant, Hotel) |
| 6 | `address` | Street address |
| 7 | `open_hours` | Operating hours |
| 8 | `popular_times` | Visitor traffic patterns |
| 9 | `website` | Official business website |
| 10 | `phone` | Contact phone number |
| 11 | `plus_code` | Location shortcode |
| 12 | `review_count` | Total number of reviews |
| 13 | `review_rating` | Average star rating |
| 14 | `reviews_per_rating` | Breakdown by star rating |
| 15 | `latitude` | GPS latitude |
| 16 | `longitude` | GPS longitude |
| 17 | `cid` | Google's unique Customer ID |
| 18 | `status` | Business status (open/closed/temporary) |
| 19 | `descriptions` | Business description |
| 20 | `reviews_link` | Direct link to reviews |
| 21 | `thumbnail` | Thumbnail image URL |
| 22 | `timezone` | Business timezone |
| 23 | `price_range` | Price level ($, $$, $$$) |
| 24 | `data_id` | Internal Google Maps identifier |
| 25 | `street_view_url` | Street View URL |
| 26 | `place_id` | Google's unique place id |
| 27 | `images` | Associated image URLs |
| 28 | `reservations` | Reservation booking link |
| 29 | `order_online` | Online ordering link |
| 30 | `menu` | Menu link |
| 31 | `owner` | Owner-claimed status |
| 32 | `complete_address` | Full formatted address |
| 33 | `credit_cards_accepted` | Accepted credit card networks |
| 34 | `about` | Additional business info |
| 35 | `user_reviews` | Customer reviews (text, rating, timestamp) |
| 36 | `user_reviews_extended` | Extended reviews up to ~300 (requires `-extra-reviews`) |
| 37 | `emails` | Extracted email addresses (requires `-email` flag) |
| 38 | `social` | All social profiles found, comma separated (requires `-email` flag) |
| 39 | `facebook` | Facebook page |
| 40 | `instagram` | Instagram profile |
| 41 | `linkedin` | LinkedIn page |
| 42 | `twitter_x` | X (Twitter) profile |
| 43 | `youtube` | YouTube channel |
| 44 | `tiktok` | TikTok profile |
| 45 | `whatsapp` | WhatsApp contact link |

</details>

**Custom Input IDs:** Define your own IDs in the input file:
```
Matsuhisa Athens #!#MyCustomID
```

**Direct Google Maps URLs:** Input lines can be regular search queries or direct Google Maps URLs. Supported URL formats include:

```text
https://www.google.com/maps/search/pizza
https://www.google.com/maps/place/Empire+State+Building/@40.7484405,-73.9856632
https://maps.google.com/maps?z=16&q=Empire+State+Building
maps.app.goo.gl/abc123
```

URLs on `google.com` subdomains must include a scheme (`http://` or `https://`) and a `/maps` path. Short `maps.app.goo.gl` links are also supported without a scheme.

---

## Configuration

### Command Line Options

```
Usage: google-maps-scraper [options]

Core Options:
  -input string       Path to input file with queries (one per line)
  -results string     Output file path (default: stdout)
  -json              Output JSON instead of CSV
  -xlsx              Output a formatted Excel workbook (implied by a .xlsx -results path)
  -depth int         Max scroll depth in results (default: 10)
  -c int             Concurrency level (default: half of CPU cores)

Website Enrichment & Reviews:
  -email             Visit each business website to extract emails and social profiles
  -extra-reviews     Collect extended reviews (up to ~300)

Location Settings:
  -lang string       Language code, e.g., 'de' for German (default: "en")
  -geo string        Coordinates for search, e.g., '37.7749,-122.4194'
  -zoom int          Zoom level 0-21 (default: 15)
  -radius float      Search radius in meters (default: 10000)
  -grid-bbox string  Bounding box for grid scraping, format: "minLat,minLon,maxLat,maxLon"
  -grid-cell float   Grid cell size in km (default: 1.0, used with -grid-bbox)

Web Server:
  -web               Run web server mode
  -addr string       Server address (default: ":8080")
  -data-folder       Data folder for web runner (default: "webdata")

Database:
  -dsn string        PostgreSQL connection string
  -produce           Produce seed jobs only (requires -dsn)

Proxy:
  -proxies string    Comma-separated proxy list
                     Format: protocol://user:pass@host:port
  -proxies-file      Path to a file containing one proxy URL per line

Export:
  -leadsdb-api-key   Export directly to LeadsDB (get key at getleadsdb.com)

Advanced:
  -exit-on-inactivity duration    Exit after inactivity (e.g., '5m')
  -fast-mode                      Quick mode with reduced data
  -debug                          Show browser window
  -writer string                  Custom writer plugin (format: 'dir:pluginName')
  -browser-pool-size int          Number of browser processes to launch (default: 0, derived from -c and -pages-per-browser)
  -pages-per-browser int          Max concurrent pages per browser process (default: 1)

Notes:
  -grid-bbox requires a valid zoom level (1-21)
  -fast-mode cannot be used together with -grid-bbox
```

Run `./google-maps-scraper -h` for the complete list.

### Choosing Where To Search

The web form has a country picker (Arab League markets, defaulting to Jordan) and a city picker that
follows from it. Picking a city does **not** search outwards from the city centre with a radius.
Each city carries a bounding box, and the job splits that box into a grid, searching every cell, so
the whole built-up area is covered — outskirts included.

The coverage setting controls the grid resolution:

| Coverage | Cell size | Amman, for example |
|---|---|---|
| Quick | 6 km | ~48 cells |
| Balanced (default) | 3 km | ~176 cells |
| Thorough | 1.5 km | ~700 cells |

The form shows the resulting number of searches and a rough duration before you start, and warns
when the job would outlast its maximum run time (a job that runs out of time stops part-way through
and returns only part of the city).

Selecting a country without a city leaves the grid off and falls back to a plain keyword search —
grid-scraping an entire country would run for days.

The market list lives in `geo/geo.go`; adding a country or a city means adding one entry with its
bounding box.

### Searching For Several Things At Once

Searches can be entered one per line or separated by commas:

```
restaurants, coffee shops, supermarkets
```

Every result records the search term that found it, in the `search_term` column. That drives:

- **In the browser** — a "Search" column, a filter for one term, and a *Group by search* toggle that
  splits the table into labelled sections
- **In Excel** — one worksheet per search term, so the example above returns `Restaurants`,
  `Coffee shops` and `Supermarkets` tabs alongside the combined `Leads` sheet
- **In the Summary sheet** — a results-per-search breakdown

Above 25 distinct searches the per-term tabs are skipped, since the filterable `Leads` sheet is
easier to navigate at that point than 25 worksheets.

### Excel Export

Passing a `.xlsx` output path (or the `-xlsx` flag) produces a formatted workbook rather than a
single flat sheet:

```bash
docker run   -v gmaps-playwright-cache:/opt   -v "$PWD/example-queries.txt:/queries.txt:ro"   -v "$PWD/gmaps-output:/out"   gosom/google-maps-scraper   -input /queries.txt   -results /out/leads.xlsx   -depth 1   -email   -exit-on-inactivity 3m
```

The workbook contains four sheets:

| Sheet | Contents |
|---|---|
| **Leads** | One row per business, with frozen headers, filters, and clickable links |
| **Reviews** | One row per review: author, rating, date, language, and full text |
| **Opening Hours** | One row per business per day |
| **Summary** | Totals, average rating, contactability, and category/city breakdowns |

The Leads sheet flattens the fields that are JSON blobs in the CSV — opening hours become seven day
columns, the rating breakdown becomes five star columns, the address is split into street/city/
postcode/country, and `about` becomes filterable Yes/No columns (delivery, dine-in, reservations,
wheelchair access, card payments) plus a readable amenities summary. Popular times collapse to
`Busiest Day` and `Busiest Hour`.

**Why the workbook rather than the CSV:** identifiers such as `cid` are 19-digit numbers, and Excel
silently rounds those to 15 significant digits when it opens a CSV, turning `18104602341234567890`
into `1.81046E+19` with the remainder unrecoverable. The workbook writes them as text so they stay
exact. CSV downloads from the web UI now also carry a UTF-8 BOM, which is what stops Excel decoding
non-Latin names with the system ANSI code page and showing mojibake.

### Social Profiles

The `-email` flag visits each business's own website. Social profiles are collected during that same
page fetch, so they cost no extra requests:

- A combined `social` column with every profile found
- Individual `facebook`, `instagram`, `linkedin`, `twitter_x`, `youtube`, `tiktok` and `whatsapp`
  columns
- Businesses whose Maps "website" is itself a Facebook or Instagram page are captured too, even
  without `-email`

Share widgets and login pages (`facebook.com/sharer`, `twitter.com/intent`, bare domains) are
filtered out, and where a network appears more than once the profile root wins over a deep link.

### Using Proxies

For larger scraping jobs, proxies help avoid rate limiting. Here's how to configure them:

```bash
./google-maps-scraper \
  -input queries.txt \
  -results results.csv \
  -proxies 'socks5://user:pass@host:port,http://host2:port2' \
  -depth 1 -c 2
```

**Supported protocols:** `socks5`, `socks5h`, `http`, `https`

Current proxy sponsors are listed in [Proxy Sponsors](docs/proxies.md). Using those links helps fund project maintenance.

### Email Extraction

Email extraction is **disabled by default**. When enabled, the scraper visits each business website to find email addresses.

```bash
./google-maps-scraper -input queries.txt -results results.csv -email
```

> **Note:** Email extraction increases processing time significantly.

### Fast Mode

Fast mode returns up to 21 results per query, ordered by distance. Useful for quick data collection with basic fields.

```bash
./google-maps-scraper \
  -input queries.txt \
  -results results.csv \
  -fast-mode \
  -zoom 15 \
  -radius 5000 \
  -geo '37.7749,-122.4194'
```

> **Warning:** Fast mode is in Beta. You may experience blocking.

### Grid Scraping (BBox)

Grid mode splits a bounding box into cells and runs one search per cell. This is useful when a single search does not return enough places.

`queries.txt` example:

```text
cafes in Peristeri, Greece
```

Command example:

```bash
./google-maps-scraper \
  -input queries.txt \
  -results peristeri-cafes.csv \
  -grid-bbox "38.0077,23.6719,38.0257,23.6947" \
  -grid-cell 0.5 \
  -zoom 16 \
  -depth 1 \
  -c 4
```

Notes:
- `-grid-bbox` guides where searches are launched from, but results are not strictly clipped to the box.
- For strict distance filtering, use `-fast-mode` with `-geo` + `-radius` (or post-filter by latitude/longitude).

---

### Browser Page Concurrency

With the default `-pages-per-browser 1`, each concurrent job (`-c`) effectively uses its own browser process with a single page tab. This can be inefficient because browser pages can be CPU- and memory-heavy, and each browser process adds overhead.

The `-pages-per-browser` flag lets you run multiple page tabs inside the same browser process, reducing overhead. Leave `-browser-pool-size` at `0` to derive the browser count from concurrency and pages per browser, or set it explicitly to cap the number of browser processes independently of concurrency.

**How the flags interact:**

| Flag | What it controls |
|------|------------------|
| `-c` | Total number of concurrent scrape jobs |
| `-browser-pool-size` | Number of browser processes to launch (default: `0`, derived from `-c` and `-pages-per-browser`) |
| `-pages-per-browser` | Number of page tabs per browser process (default: 1) |

When `-pages-per-browser` is greater than 1, the scraper opens multiple tabs within each browser process and routes jobs through a shared page pool. This can significantly increase job throughput on the same hardware.

**Example — 8 concurrent jobs across 2 browsers with 4 tabs each:**

```bash
./google-maps-scraper \
  -c 8 \
  -browser-pool-size 2 \
  -pages-per-browser 4 \
  -input queries.txt \
  -results results.csv \
  -depth 1
```

**Tuning guidance:**

- Start with `-c 4 -browser-pool-size 1 -pages-per-browser 4` for a 4:1 page-to-browser ratio.
- Monitor CPU and RAM usage with `htop` or `docker stats`; browser resource use varies significantly by workload, Chromium version, and page count.
- If memory is the bottleneck, keep `-browser-pool-size` low and reduce `-c` or `-pages-per-browser`.
- If CPU is the bottleneck, reduce the total number of active pages by lowering `-c` or `-pages-per-browser`.
- The product `-browser-pool-size × -pages-per-browser` should roughly equal or exceed `-c` to keep all jobs busy.
- Setting an explicit `-browser-pool-size` is most useful in containerized environments (Docker, Kubernetes) where you want predictable resource usage.

---

## Export to LeadsDB

Skip the CSV files and send leads directly to a managed database. [LeadsDB](https://getleadsdb.com/) handles deduplication, filtering, and provides an API for your applications.

**Using Docker:**
```bash
docker run \
  -v gmaps-playwright-cache:/opt \
  -v "$PWD/example-queries.txt:/queries.txt:ro" \
  gosom/google-maps-scraper \
  -input /queries.txt \
  -depth 1 \
  -leadsdb-api-key "your-api-key" \
  -exit-on-inactivity 3m
```

**Using binary:**
```bash
./google-maps-scraper \
  -input queries.txt \
  -leadsdb-api-key "your-api-key" \
  -exit-on-inactivity 3m
```

Or via environment variable:
```bash
export LEADSDB_API_KEY="your-api-key"
./google-maps-scraper -input queries.txt -exit-on-inactivity 3m
```

<details>
<summary><strong>Field Mapping</strong></summary>

| Google Maps | LeadsDB |
|-------------|---------|
| Title | Name |
| Category | Category |
| Categories | Tags |
| Phone | Phone |
| Website | Website |
| Address | Address, City, State, Country, PostalCode |
| Latitude/Longitude | Coordinates |
| Review Rating | Rating |
| Review Count | ReviewCount |
| Emails | Email |
| Thumbnail | LogoURL |
| CID | SourceID |

Additional fields (Google Maps link, plus code, price range, etc.) are stored as custom attributes.

</details>

Get your API key at [getleadsdb.com/settings](https://getleadsdb.com/settings) after signing up.

---

## Advanced Usage

### PostgreSQL Database Provider

For distributed scraping across multiple machines:

**1. Start PostgreSQL:**
```bash
docker-compose -f docker-compose.dev.yaml up -d
```

**2. Seed the jobs:**
```bash
./google-maps-scraper \
  -dsn "postgres://postgres:postgres@localhost:5432/postgres" \
  -produce \
  -input example-queries.txt \
  -lang en
```

**3. Run scrapers (on multiple machines):**
```bash
./google-maps-scraper \
  -c 2 \
  -depth 1 \
  -dsn "postgres://postgres:postgres@localhost:5432/postgres"
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: google-maps-scraper
spec:
  replicas: 3  # Adjust based on needs
  selector:
    matchLabels:
      app: google-maps-scraper
  template:
    metadata:
      labels:
        app: google-maps-scraper
    spec:
      containers:
      - name: google-maps-scraper
        image: gosom/google-maps-scraper:latest
        args: ["-c", "1", "-depth", "10", "-dsn", "postgres://user:pass@host:5432/db"]
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
```

> **Note:** The headless browser requires significant CPU/memory resources.

### Custom Writer Plugins

Create custom output handlers using Go plugins:

**1. Write the plugin** (see `examples/plugins/example_writer.go`)

**2. Build:**
```bash
go build -buildmode=plugin -tags=plugin -o myplugin.so myplugin.go
```

**3. Run:**
```bash
./google-maps-scraper -writer ~/plugins:MyWriter -input queries.txt
```

---

## Performance

**Expected throughput:** ~120 places/minute (with `-c 8 -depth 1`)

| Keywords | Results/Keyword | Total Jobs | Estimated Time |
|----------|-----------------|------------|----------------|
| 100 | 16 | 1,600 | ~13 minutes |
| 1,000 | 16 | 16,000 | ~2.5 hours |
| 10,000 | 16 | 160,000 | ~22 hours |

For large-scale scraping, use the PostgreSQL provider with Kubernetes.

### Telemetry

Anonymous usage statistics are collected for improvement purposes. Opt out:
```bash
export DISABLE_TELEMETRY=1
```

---

## Support the Project

This project is free and open source. Stars, sponsorships, and sponsor referrals help fund maintenance.

- Star the repository: [github.com/gosom/google-maps-scraper](https://github.com/gosom/google-maps-scraper)
- Sponsor development: [GitHub Sponsors](https://github.com/sponsors/gosom)
- Need proxies? See [Proxy Sponsors](docs/proxies.md)
- Deploying the self-hosted platform? See [SaaS deployment options](docs/saas.md)
- Managing scraped leads? See [LeadsDB](https://getleadsdb.com/)

---

## Community

[![Discord](https://img.shields.io/badge/Discord-Join%20Our%20Server-7289DA?logo=discord&logoColor=white&style=for-the-badge)](https://discord.gg/fpaAVhNCCu)

Join our Discord to:
- Get help with setup and configuration
- Share your use cases and success stories
- Request features and report bugs
- Connect with other users

---

## Contributing

Contributions are welcome! Please:

1. Open an issue to discuss your idea
2. Fork the repository
3. Create a pull request

See [AGENTS.md](AGENTS.md) for development guidelines.

---

## References

- [How to Extract Data from Google Maps Using Golang](https://blog.gkomninos.com/how-to-extract-data-from-google-maps-using-golang)
- [Distributed Google Maps Scraping](https://blog.gkomninos.com/distributed-google-maps-scraping)
- [Deploy your own Maps scraping API in 5 minutes (includes video walkthrough)](https://gosom.dev/deploy-your-own-maps-scraping-api-in-5-minutes/)
- [Video walkthrough (YouTube)](https://www.youtube.com/watch?v=STG9mZw_nac)
- [scrapemate](https://github.com/gosom/scrapemate) - The underlying web crawling framework
- [omkarcloud/google-maps-scraper](https://github.com/omkarcloud/google-maps-scraper) - Inspiration for JS data extraction

---

## License

This project is licensed under the [MIT License](LICENSE).

---

## Star History

<a href="https://www.star-history.com/?repos=gosom%2Fgoogle-maps-scraper&type=date&legend=top-left">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/chart?repos=gosom/google-maps-scraper&type=date&theme=dark&legend=top-left&sealed_token=Yr-6oa5yTlmDhKxqKB2hmq1BmacSXwVftAnHxVCYmn7c4GiYaBtbLGZIZufr71ZfE4jkDpnt75NiAK4fBXP9FPuusVh3L0ofCigowkfI_OxlShvmkys9esorL4X5DRFPCCEVfdt0P0PV-52Qq61fiZSVkUKRjWYT8lhKUkwcXeCH8H8aIbc9NiLItTXz" />
   <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/chart?repos=gosom/google-maps-scraper&type=date&legend=top-left&sealed_token=Yr-6oa5yTlmDhKxqKB2hmq1BmacSXwVftAnHxVCYmn7c4GiYaBtbLGZIZufr71ZfE4jkDpnt75NiAK4fBXP9FPuusVh3L0ofCigowkfI_OxlShvmkys9esorL4X5DRFPCCEVfdt0P0PV-52Qq61fiZSVkUKRjWYT8lhKUkwcXeCH8H8aIbc9NiLItTXz" />
   <img alt="Star History Chart" src="https://api.star-history.com/chart?repos=gosom/google-maps-scraper&type=date&legend=top-left&sealed_token=Yr-6oa5yTlmDhKxqKB2hmq1BmacSXwVftAnHxVCYmn7c4GiYaBtbLGZIZufr71ZfE4jkDpnt75NiAK4fBXP9FPuusVh3L0ofCigowkfI_OxlShvmkys9esorL4X5DRFPCCEVfdt0P0PV-52Qq61fiZSVkUKRjWYT8lhKUkwcXeCH8H8aIbc9NiLItTXz" />
 </picture>
</a>

---

## Legal Notice

Please use this scraper responsibly and in accordance with applicable laws and regulations. Unauthorized scraping may violate terms of service.

---

<p align="center">
  <sub>Banner generated using OpenAI's DALL-E</sub>
</p>

<p align="center">
  <sub><strong>SPONSOR DISCLAIMER:</strong> Sponsor listings and referral links do not constitute an endorsement. This project and its maintainers are not responsible for sponsors' products, services, representations, conduct, or any resulting loss or damage. Users engage with sponsors at their own risk.</sub>
</p>
