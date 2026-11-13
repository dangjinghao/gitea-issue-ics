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

func Build(calName string, issues []*sdk.Issue) string {
	cal := ics.NewCalendar()
	cal.SetMethod(ics.MethodPublish)
	cal.SetProductId("-//github-dangjinghao//gitea-issue-ics//EN")
	cal.SetXWRCalName(calName)
	cal.SetName(calName)
	cal.SetXWRCalDesc("Gitea issues with due dates")

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

func summary(issue *sdk.Issue) string {
	if issue.Repository == nil || issue.Repository.FullName == "" {
		log.Printf("Unexpected issue in repositoy with empty name: %v", issue)
		return fmt.Sprintf("#%d: %s", issue.Index, issue.Title)
	}
	return fmt.Sprintf("%s#%d: %s", issue.Repository.FullName, issue.Index, issue.Title)
}
