# `Gitea Issue ICS`

Subscribe to open `Gitea` issues with due dates as an `ICS` calendar. Supports multiple accounts and stable subscription URLs that survive restarts.

## Usage

Create a private `.env` file with these entries:

- `GITEA_BASE_URL=https://gitea.example.com`
- `GITEA_ACCESS_TOKENS=your-token`

Grant the access token these permissions:

- Repository and Organization Access: All (public, private, and limited)
- `read:issue`
- `read:user`

For multiple accounts, separate tokens with commas. Build and start:

```sh
docker build -t gitea-issue-ics .
docker run -d --name gitea-issue-ics -p 8080:8080 --env-file .env gitea-issue-ics
docker logs gitea-issue-ics
```

Subscribe using the path printed in the logs, prefixed with your server address, for example `http://localhost:8080/<generated-path>/issue-deadlines.ics`. Use an address reachable by your calendar client.

> [!WARNING] Private subscription URL
> Anyone with the subscription URL can access the feed. Keep it private.

The default cache interval is 10 minutes; set `REFRESH_INTERVAL=5m` to change it. Keep tokens, the base URL, and any configured `FEED_URL_SALT` unchanged to preserve subscription URLs.

Release tags publish images to `ghcr.io/dangjinghao/gitea-issue-ics:<version>`.

## TODO

- [x] `GHCR` publishing workflow
- [x] Optional `HTML` descriptions using `Gitea`'s `Markup API` (experimental), available on [`feat/HTML-render`](https://github.com/dangjinghao/gitea-issue-ics/tree/feat/HTML-render). Kept out of `master` because of limited client support (`macOS Calendar` still shows plain text), incomplete commit-reference rendering in `Gitea`, and extra server requests. Plain-text `Markdown` descriptions remain the default.
- [x] More issue metadata
    - [x] Creation date (`CREATED`)
    - [x] Modification date (`LAST-MODIFIED`)
    - [x] Labels (`CATEGORIES`)
- [ ] Improve logging
- [ ] Incremental cache update
