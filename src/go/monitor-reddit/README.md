# Reddit Monitor

Streams target subreddits for keyword matches and sends alerts to the Werd API via webhook.

## Configuration

Environment variables:

- `WERD_API_URL` — Werd API server URL for webhook delivery
- `WERD_API_KEY` — Authentication key
- `REDDIT_CLIENT_ID` — Reddit app client ID
- `REDDIT_CLIENT_SECRET` — Reddit app client secret
- `REDDIT_USERNAME` — Reddit account username
- `REDDIT_PASSWORD` — Reddit account password

Keywords and subreddit lists are fetched from the Werd API at startup and refreshed periodically.
