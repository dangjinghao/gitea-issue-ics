# AGENTS.md

## Build / verify

Go is NOT installed on the host. All go commands must run in the running devcontainer:

```sh
docker exec -u vscode <container-id> sh -c "cd /workspaces/gitea-issue-ics && go build ./... && go vet ./..."
```

Find the container with `docker ps` (image name starts with `vsc-gitea-issue-ics`). There is no test suite or lint config yet.

## Architecture

Single Go binary converting a Gitea account's open issues with due dates into an ics feed.

Flow: `main.go` → `internal/config` (envvars: `GITEA_ACCESS_TOKEN[S]`, `GITEA_BASE_URL`, `LISTEN_ADDR`, `REFRESH_INTERVAL`) → `internal/server` creates one feed per token with a random 16-byte base64url URL path, serves cached ics refreshed per `REFRESH_INTERVAL` (stale fallback on fetch error) → `internal/gitea` (paged `ListIssues` search, filters `Deadline != nil`, `Type: issues` to exclude PRs) → `internal/calendar` (golang-ical).

## Library API gotchas (v0.25.1 sdk / v0.3.8 golang-ical)

- `gitea.Issue`: due date is `Deadline`, created is `Created` (no `DueDate`/`CreatedAt`); user field is `UserName`, not `Login`.
- `sdk.ListIssues(opt)` takes no repo arg — it searches all issues the account can access; use `ListIssueOption` (not `ListIssuesOptions`).
- golang-ical: `Calendar.Serialize()` returns only `string`; use `SetXWRCalName`/`SetXWRCalDesc` and `event.SetAllDayStartAt`/`SetCreatedTime` (no `AddProperty`/`SetAllDay`/`SetCreated`).
