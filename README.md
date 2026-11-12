# gitea issue ics

Convert accessible gitea issue with due date to calendar event in an ics file.

Serve this ics file by http server with a random genreated path.

## Why not use gitea action?

- Hard to subscribe with `access_token` by client
- too many action call per day. E.g. every 10 mins to scan and generate file

## Impl details

- go
- gitea API sdk
- `arran4/golang-ical`
- `http`

### config
- pass `access_token` (one or more, comma-separated) by `GITEA_ACCESS_TOKENS` envvar
- generate one random path for one access_token(let's mark one access_token as one account)
- dev with devcontainer