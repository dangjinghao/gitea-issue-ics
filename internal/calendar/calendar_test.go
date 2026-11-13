package calendar

import (
	"fmt"
	"strings"
	"testing"
	"time"

	sdk "code.gitea.io/sdk/gitea"
	ics "github.com/arran4/golang-ical"
)

func TestBuildCalendar(t *testing.T) {
	first := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	second := first.Add(24 * time.Hour)
	body := "**Markdown** 中文, semicolon; backslash \\\nEND:VEVENT\nBEGIN:VEVENT\nSUMMARY:injected"
	issues := []*sdk.Issue{
		{ID: 2, Index: 7, Title: "later", Deadline: &second, Repository: &sdk.RepositoryMeta{FullName: "alice/project"}},
		{ID: 1, Index: 3, Title: "earlier", Deadline: &first, Created: first.Add(-time.Hour), Body: " \n" + body + "\n ", HTMLURL: "https://git.example/alice/project/issues/3", Repository: &sdk.RepositoryMeta{FullName: "alice/project"}},
	}
	data := Build("alice@git.example/issue-deadlines", issues)
	cal, err := ics.ParseCalendar(strings.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"NAME:", "X-WR-CALNAME:"} {
		if !strings.Contains(data, name+"alice@git.example/issue-deadlines") {
			t.Fatalf("missing calendar name %s", name)
		}
	}
	events := cal.Events()
	if len(events) != 2 {
		t.Fatalf("want 2 events after parsing, got %d", len(events))
	}
	for _, check := range []struct {
		property ics.ComponentProperty
		value    string
	}{
		{ics.ComponentPropertyUniqueId, "gitea-issue-1"},
		{ics.ComponentPropertySummary, "alice/project#3: earlier"},
		{ics.ComponentPropertyDescription, body},
		{ics.ComponentPropertyUrl, issues[1].HTMLURL},
	} {
		property := events[0].GetProperty(check.property)
		if property == nil || property.Value != check.value {
			t.Errorf("property %s: got %#v, want %q", check.property, property, check.value)
		}
	}
	start := events[0].GetProperty(ics.ComponentPropertyDtStart)
	if start == nil || start.Value != "20261120" || strings.Join(start.ICalParameters["VALUE"], "") != "DATE" {
		t.Fatalf("expected an all-day deadline event, got %#v", start)
	}
	if events[0].GetProperty(ics.ComponentProperty("X-ALT-DESC")) != nil || len(events[0].GetProperty(ics.ComponentPropertyDescription).ICalParameters["ALTREP"]) != 0 {
		t.Fatal("plain Markdown description must not include HTML alternatives")
	}
	if issues[0].ID != 2 {
		t.Fatal("Build must not reorder the caller's issue slice")
	}
}

func TestBuildDescriptionLineEndings(t *testing.T) {
	deadline := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	for _, newline := range []string{"\n", "\r\n", "\r"} {
		t.Run(fmt.Sprintf("%q", newline), func(t *testing.T) {
			body := strings.Join([]string{"- [x] first", "  - [x] nested", "", "- [ ] last"}, newline)
			data := Build("test", []*sdk.Issue{{
				ID: 1, Body: body, Deadline: &deadline,
				Repository: &sdk.RepositoryMeta{FullName: "alice/project"},
			}})
			// Only ICS physical line endings may contain carriage returns.
			if strings.Contains(strings.ReplaceAll(data, "\r\n", ""), "\r") {
				t.Fatal("serialized calendar contains a bare carriage return")
			}
			cal, err := ics.ParseCalendar(strings.NewReader(data))
			if err != nil {
				t.Fatal(err)
			}
			if len(cal.Events()) != 1 {
				t.Fatalf("want 1 event, got %d", len(cal.Events()))
			}
			description := cal.Events()[0].GetProperty(ics.ComponentPropertyDescription)
			want := "- [x] first\n  - [x] nested\n\n- [ ] last"
			if description == nil || description.Value != want {
				t.Fatalf("description: got %#v, want %q", description, want)
			}
		})
	}
}

func TestSummaryFallback(t *testing.T) {
	for _, repo := range []*sdk.RepositoryMeta{nil, {}} {
		if got := summary(&sdk.Issue{Index: 5, Title: "task", Repository: repo}); got != "#5: task" {
			t.Fatalf("unexpected fallback summary: %q", got)
		}
	}
}

func TestBuildEmptyCalendar(t *testing.T) {
	cal, err := ics.ParseCalendar(strings.NewReader(Build("empty", nil)))
	if err != nil || len(cal.Events()) != 0 {
		t.Fatalf("expected valid empty calendar, got error %v", err)
	}
}
