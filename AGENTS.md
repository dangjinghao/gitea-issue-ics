# AGENTS.md

## Build and verification

Run local Go commands in the running devcontainer. Find it with `docker ps`;
the image starts with `vsc-gitea-issue-ics`.

```sh
docker exec -u vscode <container-id> sh -c "cd /workspaces/gitea-issue-ics && go build ./... && go vet ./... && go test -race ./..."
```

Use `gofmt` for Go formatting. Tests use the standard `testing` package and
`httptest`, live alongside their packages as `*_test.go`, and do not require a
real Gitea instance. Live Gitea and calendar-client checks are separate manual
integration tests; do not claim compatibility based only on unit tests.

The devcontainer uses the Go image and the
`ghcr.io/devcontainers-extra/features/pre-commit:2` Feature. Its
`postCreateCommand` runs `pre-commit install`. Rebuild the container after changing
its configuration, and run commits inside it. `.pre-commit-config.yaml` pins
TekWizely/pre-commit-golang hooks for formatting, vet, and module tests. Run
`pre-commit run --all-files` inside the container to invoke them manually.

## Architecture

Single Go binary converting accessible, open Gitea issues with due dates into
all-day calendar events. Keep changes small and independently reviewable.

- `main.go`: load configuration, initialize the server, handle graceful shutdown,
  and expose `-version` (defaults to `dev`).
- `internal/config`: read environment variables and parse tokens and durations.
- `internal/server`: create one feed per distinct token, resolve the username at
  startup, register subscription routes, and manage cached ICS responses.
- `internal/gitea`: use the SDK's paged `ListIssues` search with `StateOpen` and
  `Type: issues`, retaining results whose `Deadline` is not nil.
- `internal/calendar`: sort issues by deadline and serialize ICS with golang-ical.
  Each event uses UID `gitea-issue-<ID>`, repository/index/title as its summary,
  the trimmed Markdown body as plain-text `DESCRIPTION`, and the original issue URL.

There is no separate Feed package, incremental issue cache, background refresh,
or webhook processing. These approaches were deferred for the MVP.

## Branch workflow

Keep the optional HTML rendering work on `feat/HTML-render`. Keep `master`
aligned with the upstream baseline (currently tracked as `origin/master`). Do
not merge the feature into `master` or rewrite remote history unless requested.

## Configuration

| Variable | Default | Purpose |
|---|---|---|
| `GITEA_ACCESS_TOKENS` | Required | Comma-separated tokens; whitespace and empty entries are ignored |
| `GITEA_BASE_URL` | `https://gitea.com` | Instance URL, including any deployment subpath |
| `LISTEN_ADDR` | `:8080` | HTTP listen address |
| `REFRESH_INTERVAL` | `10m` | Go duration controlling ICS cache freshness |
| `FEED_URL_SALT` | Empty | Optional fixed context for deriving subscription paths |
| `RENDER_HTML` | `false` | Enable Gitea rendering of issue descriptions |

Do not add undocumented token-variable aliases. Configuration currently accepts
any syntactically valid duration; stricter validation is a separate change.

## Subscription and cache behavior

The route is `GET /<hash>/issue-deadlines.ics`. The hash is the full hex-encoded
HMAC-SHA256 result, using the token as key and
`gitea-issue-ics/feed/v1\x00<baseURL>\x00<salt>` as input. Trailing `/` characters
are removed from the base URL for hashing. Token ordering does not affect URLs;
changing a token, normalized base URL, or salt does. The old bare-hash route is
not registered. Subscription URLs grant access to the feed and appear in startup logs.

The calendar's `NAME` and `X-WR-CALNAME` are
`<username>@<instance-hostname>/issue-deadlines`; the hostname excludes port and
subpath. The URL's readable filename helps Thunderbird's default subscription
name because it does not automatically adopt these metadata fields.

Each feed has a mutex protecting its ICS string and fetch time. A request serves
fresh cache directly; after expiry it fetches the complete current issue list
and regenerates ICS while holding the lock. Concurrent requests share a successful
refresh. Fetch failures retain the old calendar; without one, HTTP returns 503.
Failures do not advance the fetch time, so subsequent requests retry. Caches are
in memory and disappear on restart. Responses use `text/calendar; charset=utf-8`
and attachment filename `issue-deadlines.ics`.

## Issue descriptions

`internal/calendar/description.go` normalizes CRLF and bare CR in descriptions to
LF before ICS escaping. Plain-text `DESCRIPTION` always preserves the Markdown.
With `RENDER_HTML=true`, the feed calls the SDK's `RenderMarkup` (`POST /markup`)
for each nonempty body during cache refresh. `Mode: comment` enables issue-style
references; `Context` is the repository path including the instance deployment
subpath, such as `/gitea/alice/project`. Do not use the default `markdown` mode
for contextual rendering. Gitea handles rendering and supported link resolution;
the public API does not necessarily match repository-scoped web rendering.

HTML is included through `X-ALT-DESC;FMTTYPE=text/html;VALUE=TEXT` and a
percent-encoded `data:text/html,` URI in `DESCRIPTION;ALTREP`. Thunderbird checks
that exact prefix and uses `decodeURIComponent`; do not add charset or base64
parameters to this URI. Display depends on the calendar client.
Rendering failures log an error and fall back to plain Markdown for that issue.
Fresh ICS cache avoids rendering calls; expired cache rerenders all nonempty
bodies. There is no separate HTML cache or updated-timestamp comparison yet.

### HTML rendering assessment

Server-side HTML rendering has limited value for the MVP and remains optional
on the feature branch. Prefer readable Markdown and the original issue URL as
the baseline. Every expired feed refresh adds one rendering request per nonempty
body. Verify client and API support before expanding this implementation.

- Manual macOS Calendar checks showed plain Markdown for both base64 and
  percent-encoded HTML ALTREP, with `X-ALT-DESC` also present. Versions were not
  recorded; this is an observed limitation, not a claim about all Apple clients.
- Thunderbird's HTML path requires the ALTREP encoding described above.
- Nextcloud Calendar's inspected web UI displays plain text with link detection.
- Public Gitea API checks linked issue references but not bare commit hashes;
  repository-scoped web rendering linked both. `Context` supplies a path, not a
  loaded, permission-checked repository object. Behavior may vary by version.

These limitations do not authorize removing the feature. Keep examples generic;
never document real tokens, subscription hashes, private instance domains, or
private issue content.

## Images and releases

The root Dockerfile is for the application, independently of the devcontainer.
It builds with Go 1.27.3 and `CGO_ENABLED=0`, then copies the static binary into
Alpine 3.24. `VERSION` is a build argument injected into `main.version` via linker
flags. `.dockerignore` excludes Git metadata, development configuration, `.env`
files, local binaries, and ICS files. Never bake access tokens into an image.

The GitHub Actions workflow checks formatting, build, vet, and race tests on PRs
targeting `master`, pushes to `master`, and `v*` tags. Only tag pushes publish,
and only after checks pass. Docker metadata generates semantic-version tags and
OCI labels; stable versions also update `latest`, while prereleases do not.
The current workflow publishes `linux/amd64` images to
`ghcr.io/<repository-owner>/<repository-name>`. Provenance attestations are
disabled to avoid GHCR displaying an additional `unknown/unknown` platform.
Feature-branch pushes alone do not trigger this workflow; PRs targeting `master`
do trigger checks.

```sh
git tag v0.1.0
git push origin v0.1.0
```

Publishing uses `GITHUB_TOKEN` with `contents: read` and `packages: write`, without
a separate PAT. For anonymous pulls, make the package public after its first
publication. Do not push tags or publish as part of unrelated implementation work.

## Library API gotchas (v0.25.1 SDK / v0.3.8 golang-ical)

- `gitea.Issue`: due date is `Deadline`, created is `Created` (no `DueDate`/`CreatedAt`); user field is `UserName`, not `Login`.
- `sdk.ListIssues(opt)` takes no repo arg — it searches all issues the account can access; use `ListIssueOption` (not `ListIssuesOptions`).
- golang-ical: `Calendar.Serialize()` returns only `string`; use `SetXWRCalName`/`SetXWRCalDesc`, `event.SetAllDayStartAt`/`SetCreatedTime`, and `SetProperty` for extension fields.
