# gitea issue ics

Convert accessible gitea issue with due date to calendar event in an ics file.

Serve this ics file over HTTP with a stable, token-derived subscription path.
Subscribe at `/<token-derived-hash>/issue-deadlines.ics`. The readable filename
lets Thunderbird suggest a useful default name instead of the hash, since it
does not automatically use the ICS calendar name. The original
`/<token-derived-hash>` URL remains available for existing subscriptions.

The calendar name is `<username>@<gitea-instance-domain>/issue-deadlines`, for example
`alice@gitea.example.com/issue-deadlines`. The domain comes from `GITEA_BASE_URL`, without
the scheme, port, or subpath.

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
- generate one stable path per access token using HMAC-SHA256, with the token as
  the secret key and the Gitea base URL and optional salt as context
- optionally set `FEED_URL_SALT` to a fixed, deployment-specific secret; preserve
  its value across restarts (leaving it unset also produces stable URLs)
- dev with devcontainer

Subscription URLs are printed at startup. Restarting or reordering tokens keeps
the same URLs as long as the token, `GITEA_BASE_URL`, and `FEED_URL_SALT` stay the
same. Changing any of these changes the URL and requires resubscribing. Trailing
slashes in the base URL do not affect the path. Treat subscription URLs as secrets:
anyone with the URL can read the feed.

Existing random URLs cannot be recovered after restart; subscribe once to the new
stable URL after upgrading. Some calendar clients keep the name chosen when first
subscribing; rename or resubscribe if the old name persists.

## TODO
- [ ] markdown -> rich text event content
- [ ] ghcr.io
