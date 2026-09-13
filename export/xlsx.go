package export

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"

	"github.com/gosom/google-maps-scraper/gmaps"
)

// Sheet names. Exported so the web layer can describe the download to users.
const (
	SheetLeads   = "Leads"
	SheetReviews = "Reviews"
	SheetHours   = "Opening Hours"
	SheetSummary = "Summary"
)

// excelHyperlinkBudget stays under Excel's hard limit of 65,530 hyperlinks per
// worksheet. Past the budget URLs are still written as text, so no data is lost
// — only the click target is dropped.
const excelHyperlinkBudget = 60000

// excelMaxCellChars is Excel's per-cell character limit. Longer values are
// truncated rather than producing a file Excel refuses to open.
const excelMaxCellChars = 32000

// daysOfWeek is the display order for opening hours. Google returns a map, whose
// iteration order is random, so the order has to be imposed here.
//
//nolint:gochecknoglobals // lookup table, read-only after init.
var daysOfWeek = []string{
	"Monday", "Tuesday", "Wednesday", "Thursday",
	"Friday", "Saturday", "Sunday",
}

// cellFormat selects the number format and alignment applied to a whole column.
type cellFormat int

const (
	fmtText cellFormat = iota
	fmtID              // forced text, so 19-digit CIDs keep every digit
	fmtInt
	fmtRating
	fmtCoord
	fmtURL
	fmtWrap
)

// leadColumn is one column of the Leads sheet. Header, width, format and value
// live together so a column can never be added in one place and forgotten in
// another.
type leadColumn struct {
	header string
	width  float64
	format cellFormat
	value  func(*gmaps.Entry) any
}

//nolint:funlen // one flat table; splitting it would only hide the schema.
func leadColumns() []leadColumn {
	return []leadColumn{
		{"Business", 34, fmtText, func(e *gmaps.Entry) any { return e.Title }},
		{"Search", 22, fmtText, func(e *gmaps.Entry) any { return e.Keyword }},
		{"Category", 22, fmtText, func(e *gmaps.Entry) any { return e.Category }},
		{"Rating", 8, fmtRating, func(e *gmaps.Entry) any { return e.ReviewRating }},
		{"Reviews", 9, fmtInt, func(e *gmaps.Entry) any { return e.ReviewCount }},
		{"Phone", 18, fmtText, func(e *gmaps.Entry) any { return e.Phone }},
		{"Website", 34, fmtURL, func(e *gmaps.Entry) any { return e.WebSite }},
		{"Email", 28, fmtText, func(e *gmaps.Entry) any { return first(e.Emails) }},
		{"All Emails", 30, fmtWrap, func(e *gmaps.Entry) any { return strings.Join(e.Emails, ", ") }},

		// The combined social column, followed by one column per network so the
		// sheet can be filtered on "has an Instagram" and similar.
		{"Social", 40, fmtWrap, func(e *gmaps.Entry) any { return e.Socials.String() }},
		{"Facebook", 32, fmtURL, func(e *gmaps.Entry) any { return e.Socials.Facebook }},
		{"Instagram", 32, fmtURL, func(e *gmaps.Entry) any { return e.Socials.Instagram }},
		{"LinkedIn", 32, fmtURL, func(e *gmaps.Entry) any { return e.Socials.LinkedIn }},
		{"X (Twitter)", 28, fmtURL, func(e *gmaps.Entry) any { return e.Socials.TwitterX }},
		{"YouTube", 28, fmtURL, func(e *gmaps.Entry) any { return e.Socials.YouTube }},
		{"TikTok", 28, fmtURL, func(e *gmaps.Entry) any { return e.Socials.TikTok }},
		{"WhatsApp", 28, fmtURL, func(e *gmaps.Entry) any { return e.Socials.WhatsApp }},

		{"Address", 40, fmtWrap, func(e *gmaps.Entry) any { return e.Address }},
		{"Street", 26, fmtText, func(e *gmaps.Entry) any { return e.CompleteAddress.Street }},
		{"Area", 20, fmtText, func(e *gmaps.Entry) any { return e.CompleteAddress.Borough }},
		{"City", 18, fmtText, func(e *gmaps.Entry) any { return e.CompleteAddress.City }},
		{"Postcode", 12, fmtID, func(e *gmaps.Entry) any { return e.CompleteAddress.PostalCode }},
		{"State", 16, fmtText, func(e *gmaps.Entry) any { return e.CompleteAddress.State }},
		{"Country", 10, fmtText, func(e *gmaps.Entry) any { return e.CompleteAddress.Country }},

		{"Price Range", 12, fmtText, func(e *gmaps.Entry) any { return e.PriceRange }},
		{"Status", 14, fmtText, func(e *gmaps.Entry) any { return e.Status }},
		{"Timezone", 18, fmtText, func(e *gmaps.Entry) any { return e.Timezone }},

		{"Mon", 20, fmtText, hoursFor("Monday")},
		{"Tue", 20, fmtText, hoursFor("Tuesday")},
		{"Wed", 20, fmtText, hoursFor("Wednesday")},
		{"Thu", 20, fmtText, hoursFor("Thursday")},
		{"Fri", 20, fmtText, hoursFor("Friday")},
		{"Sat", 20, fmtText, hoursFor("Saturday")},
		{"Sun", 20, fmtText, hoursFor("Sunday")},
		{"Busiest Day", 13, fmtText, func(e *gmaps.Entry) any { d, _ := busiest(e); return d }},
		{"Busiest Hour", 13, fmtText, func(e *gmaps.Entry) any { _, h := busiest(e); return h }},

		{"5 Star", 8, fmtInt, ratingCount(5)},
		{"4 Star", 8, fmtInt, ratingCount(4)},
		{"3 Star", 8, fmtInt, ratingCount(3)},
		{"2 Star", 8, fmtInt, ratingCount(2)},
		{"1 Star", 8, fmtInt, ratingCount(1)},

		{"Delivery", 10, fmtText, option("service_options", "Delivery")},
		{"Takeaway", 10, fmtText, option("service_options", "Takeaway")},
		{"Dine In", 10, fmtText, option("service_options", "Dine-in")},
		{"Reservations", 13, fmtText, option("planning", "Accepts reservations")},
		{"Wheelchair", 12, fmtText, option("accessibility", "Wheelchair")},
		{"Cards", 10, fmtText, option("payments", "Credit cards")},
		{"Good For Kids", 14, fmtText, option("children", "Good for kids")},
		{"Cards Accepted", 22, fmtText, func(e *gmaps.Entry) any {
			return strings.Join(e.CreditCardsAccepted, ", ")
		}},
		{"Amenities", 50, fmtWrap, func(e *gmaps.Entry) any { return amenitySummary(e) }},

		{"Owner", 26, fmtText, func(e *gmaps.Entry) any { return e.Owner.Name }},
		{"Menu", 30, fmtURL, func(e *gmaps.Entry) any { return e.Menu.Link }},
		{"Order Online", 30, fmtURL, func(e *gmaps.Entry) any { return firstLink(e.OrderOnline) }},
		{"Reservation Link", 30, fmtURL, func(e *gmaps.Entry) any { return firstLink(e.Reservations) }},

		{"Photos", 8, fmtInt, func(e *gmaps.Entry) any { return len(e.Images) }},
		{"Thumbnail", 30, fmtURL, func(e *gmaps.Entry) any { return e.Thumbnail }},
		{"Maps Link", 30, fmtURL, func(e *gmaps.Entry) any { return e.Link }},
		{"Street View", 30, fmtURL, func(e *gmaps.Entry) any { return e.StreetViewURL }},
		{"Reviews Link", 30, fmtURL, func(e *gmaps.Entry) any { return e.ReviewsLink }},

		{"Latitude", 13, fmtCoord, func(e *gmaps.Entry) any { return e.Latitude }},
		{"Longitude", 13, fmtCoord, func(e *gmaps.Entry) any { return e.Longtitude }},
		{"Plus Code", 18, fmtID, func(e *gmaps.Entry) any { return e.PlusCode }},

		// Identifiers are written as text on purpose: a CID is a 19-digit number
		// and Excel would silently round it to 15 significant digits, turning
		// 18104602341234567890 into 1.81046E+19 with the rest unrecoverable.
		{"CID", 22, fmtID, func(e *gmaps.Entry) any { return e.Cid }},
		{"Place ID", 30, fmtID, func(e *gmaps.Entry) any { return e.PlaceID }},
		{"Data ID", 40, fmtID, func(e *gmaps.Entry) any { return e.DataID }},
		{"Job ID", 38, fmtID, func(e *gmaps.Entry) any { return e.ID }},

		{"Description", 50, fmtWrap, func(e *gmaps.Entry) any { return e.Description }},
	}
}

// keywordGroup is one per-search sheet.
type keywordGroup struct {
	keyword string
	sheet   string
	entries []gmaps.Entry
}

// maxKeywordSheets caps how many per-search tabs are produced. Beyond this the
// workbook becomes harder to navigate than the filterable All Leads sheet, and
// Excel starts to struggle, so the split is skipped and the Search column on
// All Leads carries the same information.
const maxKeywordSheets = 25

// groupByKeyword splits entries by the search term that found them, preserving
// the order in which the terms first appear. A single search produces no groups:
// a second tab identical to All Leads would be noise.
func groupByKeyword(entries []gmaps.Entry) []keywordGroup {
	order := []string{}
	byKeyword := map[string][]gmaps.Entry{}

	for i := range entries {
		k := strings.TrimSpace(entries[i].Keyword)
		if k == "" {
			continue
		}

		if _, seen := byKeyword[k]; !seen {
			order = append(order, k)
		}

		byKeyword[k] = append(byKeyword[k], entries[i])
	}

	if len(order) < 2 || len(order) > maxKeywordSheets {
		return nil
	}

	used := map[string]bool{
		strings.ToLower(SheetLeads):   true,
		strings.ToLower(SheetReviews): true,
		strings.ToLower(SheetHours):   true,
		strings.ToLower(SheetSummary): true,
	}

	groups := make([]keywordGroup, 0, len(order))

	for _, k := range order {
		groups = append(groups, keywordGroup{
			keyword: k,
			sheet:   uniqueSheetName(k, used),
			entries: byKeyword[k],
		})
	}

	return groups
}

// invalidSheetChars are rejected by Excel in a worksheet name.
const invalidSheetChars = `:\/?*[]`

// uniqueSheetName turns a search term into a legal, unique worksheet name.
// Excel allows 31 characters, forbids several punctuation marks, and refuses
// duplicates regardless of case.
func uniqueSheetName(raw string, used map[string]bool) string {
	name := strings.Map(func(r rune) rune {
		if strings.ContainsRune(invalidSheetChars, r) {
			return '-'
		}

		return r
	}, strings.TrimSpace(raw))

	name = strings.Trim(name, "'")
	if name == "" {
		name = "Search"
	}

	name = titleCaseFirst(name)

	if len([]rune(name)) > 31 {
		name = string([]rune(name)[:31])
	}

	candidate := name

	for i := 2; used[strings.ToLower(candidate)]; i++ {
		suffix := " (" + strconv.Itoa(i) + ")"
		trimmed := name

		if len([]rune(name))+len(suffix) > 31 {
			trimmed = string([]rune(name)[:31-len(suffix)])
		}

		candidate = trimmed + suffix
	}

	used[strings.ToLower(candidate)] = true

	return candidate
}

func titleCaseFirst(s string) string {
	runes := []rune(s)
	if len(runes) == 0 {
		return s
	}

	return strings.ToUpper(string(runes[0])) + string(runes[1:])
}

// builder carries the workbook and the styles shared by every sheet.
type builder struct {
	f          *excelize.File
	styles     map[cellFormat]int
	header     int
	title      int
	label      int
	linksSpent int
	// dupesRemoved counts the multi-branch rows dropped by DedupeBySocial, so
	// the Summary sheet can say why the workbook holds fewer rows than the job.
	dupesRemoved int
}

// BuildWorkbook writes a multi-sheet XLSX report for entries to w. Businesses
// sharing a Facebook or Instagram account are collapsed to one row first, so
// every social profile appears exactly once across the workbook.
func BuildWorkbook(entries []gmaps.Entry, w io.Writer) error {
	entries, removed := DedupeBySocial(entries)

	b := &builder{f: excelize.NewFile(), dupesRemoved: removed}

	defer func() {
		_ = b.f.Close()
	}()

	if err := b.initStyles(); err != nil {
		return err
	}

	if err := b.f.SetSheetName(b.f.GetSheetName(0), SheetLeads); err != nil {
		return err
	}

	// One sheet per search term, so a job covering "restaurants", "coffee shops"
	// and "supermarkets" comes back as three readable tabs instead of one mixed
	// list. The All Leads sheet is kept as the complete, filterable view.
	groups := groupByKeyword(entries)

	for _, g := range groups {
		if _, err := b.f.NewSheet(g.sheet); err != nil {
			return err
		}
	}

	for _, name := range []string{SheetReviews, SheetHours, SheetSummary} {
		if _, err := b.f.NewSheet(name); err != nil {
			return err
		}
	}

	if err := b.writeLeads(SheetLeads, entries); err != nil {
		return err
	}

	for _, g := range groups {
		if err := b.writeLeads(g.sheet, g.entries); err != nil {
			return err
		}
	}

	if err := b.writeReviews(entries); err != nil {
		return err
	}

	if err := b.writeHours(entries); err != nil {
		return err
	}

	if err := b.writeSummary(entries); err != nil {
		return err
	}

	b.f.SetActiveSheet(0)

	return b.f.Write(w)
}

func (b *builder) initStyles() error {
	border := []excelize.Border{
		{Type: "bottom", Color: "D9D9D9", Style: 1},
	}

	header, err := b.f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF", Size: 11},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"1F4E79"}, Pattern: 1},
		Alignment: &excelize.Alignment{
			Horizontal: "left", Vertical: "center", WrapText: false,
		},
	})
	if err != nil {
		return err
	}

	b.header = header

	if b.title, err = b.f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 16, Color: "1F4E79"},
	}); err != nil {
		return err
	}

	if b.label, err = b.f.NewStyle(&excelize.Style{
		Font:   &excelize.Font{Bold: true},
		Border: border,
	}); err != nil {
		return err
	}

	b.styles = map[cellFormat]int{}

	specs := map[cellFormat]*excelize.Style{
		fmtText: {Alignment: &excelize.Alignment{Vertical: "center"}},
		fmtID: {
			NumFmt:    49, // "@" text, keeps long identifiers exact
			Alignment: &excelize.Alignment{Vertical: "center", Horizontal: "left"},
		},
		fmtInt: {
			CustomNumFmt: strPtr("#,##0"),
			Alignment:    &excelize.Alignment{Vertical: "center", Horizontal: "right"},
		},
		fmtRating: {
			CustomNumFmt: strPtr("0.0"),
			Alignment:    &excelize.Alignment{Vertical: "center", Horizontal: "right"},
		},
		fmtCoord: {
			CustomNumFmt: strPtr("0.000000"),
			Alignment:    &excelize.Alignment{Vertical: "center", Horizontal: "right"},
		},
		fmtURL: {
			Font:      &excelize.Font{Color: "0563C1", Underline: "single"},
			Alignment: &excelize.Alignment{Vertical: "center"},
		},
		fmtWrap: {
			Alignment: &excelize.Alignment{Vertical: "top", WrapText: true},
		},
	}

	for k, spec := range specs {
		id, err := b.f.NewStyle(spec)
		if err != nil {
			return err
		}

		b.styles[k] = id
	}

	return nil
}

//nolint:funlen // sheet assembly reads better as one sequence.
func (b *builder) writeLeads(sheet string, entries []gmaps.Entry) error {
	cols := leadColumns()

	if err := b.writeHeaderRow(sheet, headers(cols)); err != nil {
		return err
	}

	for i := range cols {
		colName, err := excelize.ColumnNumberToName(i + 1)
		if err != nil {
			return err
		}

		if err := b.f.SetColWidth(sheet, colName, colName, cols[i].width); err != nil {
			return err
		}

		if err := b.f.SetColStyle(sheet, colName, b.styles[cols[i].format]); err != nil {
			return err
		}
	}

	// Column styles cover the header cells too, so the header style is reapplied
	// afterwards to win.
	if err := b.styleHeaderRow(sheet, len(cols)); err != nil {
		return err
	}

	for r := range entries {
		entry := &entries[r]
		row := r + 2

		for c := range cols {
			cell, err := excelize.CoordinatesToCellName(c+1, row)
			if err != nil {
				return err
			}

			if err := b.writeValue(sheet, cell, cols[c], entry); err != nil {
				return err
			}
		}
	}

	return b.finishSheet(sheet, len(cols), len(entries)+1, "B2")
}

// writeValue writes one cell, choosing between a numeric value, a forced string
// and a hyperlink according to the column format.
func (b *builder) writeValue(sheet, cell string, col leadColumn, e *gmaps.Entry) error {
	v := col.value(e)

	switch col.format {
	case fmtInt, fmtRating, fmtCoord:
		return b.f.SetCellValue(sheet, cell, v)
	case fmtID:
		// SetCellStr stores the value as a string, which is what stops Excel
		// re-interpreting a long identifier as a floating point number.
		return b.f.SetCellStr(sheet, cell, clean(asString(v)))
	case fmtURL:
		return b.writeURL(sheet, cell, asString(v))
	case fmtText, fmtWrap:
		return b.f.SetCellStr(sheet, cell, clean(asString(v)))
	}

	return b.f.SetCellStr(sheet, cell, clean(asString(v)))
}

func (b *builder) writeURL(sheet, cell, raw string) error {
	value := clean(raw)
	if value == "" {
		return nil
	}

	if err := b.f.SetCellStr(sheet, cell, value); err != nil {
		return err
	}

	if b.linksSpent >= excelHyperlinkBudget || !strings.HasPrefix(value, "http") {
		return nil
	}

	b.linksSpent++

	return b.f.SetCellHyperLink(sheet, cell, value, "External")
}

func (b *builder) writeHeaderRow(sheet string, names []string) error {
	for i, name := range names {
		cell, err := excelize.CoordinatesToCellName(i+1, 1)
		if err != nil {
			return err
		}

		if err := b.f.SetCellStr(sheet, cell, name); err != nil {
			return err
		}
	}

	return b.f.SetRowHeight(sheet, 1, 22)
}

func (b *builder) styleHeaderRow(sheet string, width int) error {
	last, err := excelize.CoordinatesToCellName(width, 1)
	if err != nil {
		return err
	}

	return b.f.SetCellStyle(sheet, "A1", last, b.header)
}

// finishSheet freezes the header (and any leading columns), and turns the header
// into a filter row so the sheet is usable the moment it opens.
func (b *builder) finishSheet(sheet string, width, rows int, freezeAt string) error {
	if err := b.styleHeaderRow(sheet, width); err != nil {
		return err
	}

	xSplit := 0
	if freezeAt != "A2" {
		xSplit = 1
	}

	if err := b.f.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		XSplit:      xSplit,
		YSplit:      1,
		TopLeftCell: freezeAt,
		ActivePane:  "bottomRight",
	}); err != nil {
		return err
	}

	if rows < 1 {
		rows = 1
	}

	bottom, err := excelize.CoordinatesToCellName(width, rows)
	if err != nil {
		return err
	}

	return b.f.AutoFilter(sheet, "A1:"+bottom, []excelize.AutoFilterOptions{})
}

func headers(cols []leadColumn) []string {
	out := make([]string, len(cols))
	for i := range cols {
		out[i] = cols[i].header
	}

	return out
}

// --- value helpers -------------------------------------------------------

func hoursFor(day string) func(*gmaps.Entry) any {
	return func(e *gmaps.Entry) any {
		return strings.Join(e.OpenHours[day], ", ")
	}
}

func ratingCount(star int) func(*gmaps.Entry) any {
	return func(e *gmaps.Entry) any {
		return e.ReviewsPerRating[star]
	}
}

// option reports whether a Google "About" attribute is enabled, matching by name
// prefix so grouped variants ("Wheelchair-accessible entrance", "... seating")
// all satisfy a single "Wheelchair" column.
func option(groupID, namePrefix string) func(*gmaps.Entry) any {
	return func(e *gmaps.Entry) any {
		for i := range e.About {
			if e.About[i].ID != groupID {
				continue
			}

			for _, opt := range e.About[i].Options {
				if opt.Enabled && strings.HasPrefix(opt.Name, namePrefix) {
					return "Yes"
				}
			}
		}

		return ""
	}
}

// busiest returns the day and hour with the highest popular-times score, which
// is far more useful in a spreadsheet than the raw 168-value matrix.
func busiest(e *gmaps.Entry) (day, hour string) {
	best := -1

	for _, d := range daysOfWeek {
		for h, score := range e.PopularTimes[d] {
			if score > best {
				best = score
				day = d
				hour = formatHour(h)
			}
		}
	}

	if best <= 0 {
		return "", ""
	}

	return day, hour
}

func formatHour(h int) string {
	switch {
	case h == 0:
		return "12 AM"
	case h < 12:
		return fmt.Sprintf("%d AM", h)
	case h == 12:
		return "12 PM"
	default:
		return fmt.Sprintf("%d PM", h-12)
	}
}

// amenitySummary flattens the About tree into a readable list of the attributes
// a business actually has, replacing a multi-kilobyte JSON blob.
func amenitySummary(e *gmaps.Entry) string {
	var parts []string

	for i := range e.About {
		var enabled []string

		for _, opt := range e.About[i].Options {
			if opt.Enabled {
				enabled = append(enabled, opt.Name)
			}
		}

		if len(enabled) > 0 {
			parts = append(parts, e.About[i].Name+": "+strings.Join(enabled, ", "))
		}
	}

	return strings.Join(parts, " | ")
}

func firstLink(links []gmaps.LinkSource) string {
	for i := range links {
		if links[i].Link != "" {
			return links[i].Link
		}
	}

	return ""
}

func first(values []string) string {
	if len(values) == 0 {
		return ""
	}

	return values[0]
}

func asString(v any) string {
	if v == nil {
		return ""
	}

	if s, ok := v.(string); ok {
		return s
	}

	return fmt.Sprintf("%v", v)
}

// clean removes control characters that would make the workbook unopenable and
// enforces Excel's per-cell length limit.
func clean(s string) string {
	s = strings.Map(func(r rune) rune {
		if r < 0x20 && r != '\n' && r != '\t' {
			return -1
		}

		return r
	}, s)

	if len(s) > excelMaxCellChars {
		s = s[:excelMaxCellChars]
	}

	return s
}

func strPtr(s string) *string {
	return &s
}

func sortedByCount(counts map[string]int, limit int) [][2]any {
	type kv struct {
		key string
		n   int
	}

	pairs := make([]kv, 0, len(counts))
	for k, v := range counts {
		pairs = append(pairs, kv{k, v})
	}

	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].n != pairs[j].n {
			return pairs[i].n > pairs[j].n
		}

		return pairs[i].key < pairs[j].key
	})

	if len(pairs) > limit {
		pairs = pairs[:limit]
	}

	out := make([][2]any, len(pairs))
	for i, p := range pairs {
		out[i] = [2]any{p.key, p.n}
	}

	return out
}
