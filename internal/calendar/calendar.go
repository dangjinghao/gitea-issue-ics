package calendar

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	sdk "code.gitea.io/sdk/gitea"
	ics "github.com/arran4/golang-ical"
)

func Build(calName string, issues []*sdk.Issue, refreshInterval time.Duration) string {
	cal := ics.NewCalendar()
	cal.SetMethod(ics.MethodPublish)
	cal.SetProductId("-//github-dangjinghao//gitea-issue-ics//EN")
	cal.SetXWRCalName(calName)
	cal.SetName(calName)
	cal.SetXWRCalDesc("Gitea issues with due dates")
	interval := formatDuration(refreshInterval)
	// RFC 7986 section 5.7: suggested minimum polling interval, not a client mandate.
	// SetRefreshInterval supplies the required VALUE=DURATION parameter.
	// https://www.rfc-editor.org/rfc/rfc7986.html#section-5.7
	cal.SetRefreshInterval(interval)
	// Nonstandard compatibility hint, also emitted alongside REFRESH-INTERVAL by ical-generator.
	// https://github.com/sebbo2002/ical-generator/blob/develop/src/calendar.ts (TTL serialization)
	cal.SetXPublishedTTL(interval)
	// RFC 7986 section 5.9 requires a CSS3 color name; olivedrab approximates Gitea green.
	// https://www.rfc-editor.org/rfc/rfc7986.html#section-5.9
	cal.SetColor("olivedrab")
	// SabreDAV exports this nonstandard field for an exact hex color.
	// This is an interoperability hint; clients may ignore it.
	// https://github.com/sabre-io/dav/blob/master/lib/CalDAV/ICSExportPlugin.php (mergeObjects)
	cal.CalendarProperties = append(cal.CalendarProperties, ics.CalendarProperty{BaseProperty: ics.BaseProperty{
		IANAToken: "X-APPLE-CALENDAR-COLOR",
		Value:     "#609926", // Gitea green.
	}})

	sorted := make([]*sdk.Issue, len(issues))
	copy(sorted, issues)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Deadline.Before(*sorted[j].Deadline)
	})

	for _, issue := range sorted {
		event := cal.AddEvent(fmt.Sprintf("gitea-issue-%d", issue.ID))
		event.SetCreatedTime(issue.Created)
		if !issue.Updated.IsZero() {
			event.SetModifiedAt(issue.Updated)
		}
		for _, label := range issue.Labels {
			if label != nil && label.Name != "" {
				event.AddCategory(label.Name)
			}
		}
		event.SetDtStampTime(time.Now())
		event.SetSummary(summary(issue))
		// Normalize body line endings so ICS text escaping leaves no bare CR.
		body := strings.ReplaceAll(issue.Body, "\r\n", "\n")
		body = strings.ReplaceAll(body, "\r", "\n")
		event.SetDescription(strings.TrimSpace(body))
		event.SetURL(issue.HTMLURL)
		event.SetAllDayStartAt(*issue.Deadline)
	}

	return cal.Serialize()
}

func formatDuration(d time.Duration) string {
	// Configuration guarantees a positive whole-second duration.
	var b strings.Builder
	b.WriteString("PT")
	if h := d / time.Hour; h > 0 {
		fmt.Fprintf(&b, "%dH", h)
	}
	if m := d % time.Hour / time.Minute; m > 0 {
		fmt.Fprintf(&b, "%dM", m)
	}
	if s := d % time.Minute / time.Second; s > 0 {
		fmt.Fprintf(&b, "%dS", s)
	}
	return b.String()
}

func summary(issue *sdk.Issue) string {
	if issue.Repository == nil || issue.Repository.FullName == "" {
		log.Printf("Unexpected issue in repositoy with empty name: %v", issue)
		return fmt.Sprintf("#%d: %s", issue.Index, issue.Title)
	}
	return fmt.Sprintf("%s#%d: %s", issue.Repository.FullName, issue.Index, issue.Title)
}
