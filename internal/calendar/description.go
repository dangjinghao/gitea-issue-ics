package calendar

import (
	"net/url"
	"strings"

	ics "github.com/arran4/golang-ical"
)

func normalizeDescription(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	return strings.TrimSpace(strings.ReplaceAll(text, "\r", "\n"))
}

func setDescription(event *ics.VEvent, markdown, html string) {
	plain := normalizeDescription(markdown)
	html = normalizeDescription(html)
	if html == "" {
		event.SetDescription(plain)
		return
	}

	// Preserve readable Markdown for clients that do not support HTML.
	// Thunderbird only recognizes the exact data:text/html, prefix and decodes
	// its contents with decodeURIComponent.
	alternative := &url.URL{
		Scheme: "data",
		Opaque: "text/html," + url.PathEscape(html),
	}
	event.SetDescription(plain, ics.WithAlternativeRepresentation(alternative))
	// Explicit TEXT typing makes the library escape HTML as an ICS text value.
	event.SetProperty(ics.ComponentProperty("X-ALT-DESC"), html,
		ics.WithFmtType("text/html"), ics.WithValue("TEXT"))
}
