package calendar

import (
	"net/url"
	"strings"
	"testing"
	"time"

	sdk "code.gitea.io/sdk/gitea"
	ics "github.com/arran4/golang-ical"
)

func TestHTMLDescription(t *testing.T) {
	deadline := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	plain := "**bold**\n#10"
	html := "<p><strong>bold</strong></p>\n<p><a href=\"https://git.example/alice/project/issues/10\">#10</a>; 中文, a+b & 100% \\</p>"
	data := Build("test", []*sdk.Issue{{
		ID: 1, Body: plain, Deadline: &deadline,
		Repository: &sdk.RepositoryMeta{FullName: "alice/project"},
	}}, map[int64]string{1: strings.ReplaceAll(html, "\n", "\r\n")})
	if strings.Contains(strings.ReplaceAll(data, "\r\n", ""), "\r") {
		t.Fatal("HTML description contains a bare carriage return")
	}
	cal, err := ics.ParseCalendar(strings.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	events := cal.Events()
	if len(events) != 1 {
		t.Fatalf("want 1 event, got %d", len(events))
	}
	description := events[0].GetProperty(ics.ComponentPropertyDescription)
	if description == nil || description.Value != plain {
		t.Fatalf("plain description changed: %#v", description)
	}
	altrep := description.ICalParameters["ALTREP"]
	// Match Thunderbird's prefix check and decodeURIComponent behavior.
	const prefix = "data:text/html,"
	if len(altrep) != 1 || !strings.HasPrefix(altrep[0], prefix) {
		t.Fatalf("unexpected HTML alternative URI: %v", altrep)
	}
	decoded, err := url.PathUnescape(strings.TrimPrefix(altrep[0], prefix))
	if err != nil || decoded != html {
		t.Fatalf("HTML alternative did not decode correctly: %q, %v", decoded, err)
	}
	alternate := events[0].GetProperty(ics.ComponentProperty("X-ALT-DESC"))
	if alternate == nil || alternate.Value != html || strings.Join(alternate.ICalParameters["FMTTYPE"], "") != "text/html" {
		t.Fatalf("HTML did not round-trip: %#v", alternate)
	}
}
