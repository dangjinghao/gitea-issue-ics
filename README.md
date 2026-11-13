# Gitea Issue ICS

Subscribe to open Gitea issues with due dates as an ICS calendar. Supports multiple
accounts and stable subscription URLs that survive restarts.

## Usage

Create a private `.env` file:

```dotenv
GITEA_BASE_URL=https://gitea.example.com
GITEA_ACCESS_TOKENS=your-token
```

Access token should be granted those permissions:

- Repository and Organization Access: All (public, private, and limited)
- `read:issue`
- `read:user`


For multiple accounts, separate tokens with commas. Build and start:

```sh
docker build -t gitea-issue-ics .
docker run -d --name gitea-issue-ics -p 8080:8080 --env-file .env gitea-issue-ics
docker logs gitea-issue-ics
```

Subscribe using the path printed in the logs, prefixed with your server address:

```text
http://localhost:8080/<generated-path>/issue-deadlines.ics
```

Use an address reachable by your calendar client. Keep the subscription URL private.
The default cache interval is 10 minutes; set `REFRESH_INTERVAL=5m` to change it.
Keep tokens, the base URL, and any configured `FEED_URL_SALT` unchanged to preserve URLs.

Release tags publish images to `ghcr.io/dangjinghao/gitea-issue-ics:<version>`.

## TODO

- [x] GHCR publishing workflow
- [x] Optional HTML descriptions using Gitea's Markup API — experimental,
  available on [`feat/HTML-render`](https://github.com/dangjinghao/gitea-issue-ics/tree/feat/HTML-render).
  Kept out of `master`: limited client support (macOS Calendar still shows plain
  text), incomplete commit-reference rendering(Gitea server render limitation), and extra requests to server.
  The plain-text Markdown description remains the default.
- [ ] more issue metadata
    - [ ] use issue create date
- [ ] Improve logging
- [ ] Incremental cache update
