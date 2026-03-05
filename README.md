# Werd

**Spreading the word** — an open-source, self-hosted social media automation and monitoring platform.

Werd is a unified stack for managing your online presence across social media platforms, technical blogs, discussion forums, and community channels. It centralizes cross-posting, scheduling, monitoring, sentiment tracking, and notification routing into a single self-hosted deployment — no paid SaaS dependencies.

## What It Does

- **Cross-post everywhere from one place** — schedule and publish to LinkedIn, X, Bluesky, Reddit, Mastodon, Discord, YouTube, and 10+ more platforms from a single dashboard
- **Monitor mentions and keywords** — track brand mentions, competitor activity, and relevant discussions across Reddit, Hacker News, Lobsters, the web, news sites, and RSS feeds
- **Route notifications intelligently** — funnel all alerts into organized team chat channels with push notifications for high-priority items
- **Draft AI-assisted responses** — automatically generate contextual response drafts for human review and approval
- **Syndicate blog content** — publish once on your blog, auto-distribute to Dev.to, Hashnode, and social platforms with canonical URLs preserved
- **Track analytics** — privacy-friendly web analytics without cookie banners
- **Automate workflows** — build custom automation pipelines connecting any combination of the above

## Architecture

```
                        +-------------------+
                        |   Mattermost      |  Notification hub
                        |   (team chat)     |  (channels per concern)
                        +--------^----------+
                                 |
              +------------------+------------------+
              |                  |                  |
     +--------+-------+  +------+--------+  +------+--------+
     | changedetect.  |  |     n8n       |  |    Postiz     |
     | io + RSSHub    |  | (workflow     |  | (cross-post   |
     | (monitoring)   |  |  automation)  |  |  & schedule)  |
     +--------+-------+  +------+--------+  +------+--------+
              |                 |                    |
     +--------v-------+  +-----v-----+       +-----v---------+
     | Reddit (PRAW)  |  | LLM API   |       | LinkedIn, X,  |
     | HN (API/RSS)   |  | (draft    |       | Bluesky,      |
     | Web (scrape)   |  |  assist)  |       | Reddit,       |
     | GitHub (hooks) |  +-----------+       | Mastodon,     |
     +-----------------+                     | YouTube, ...  |
                                             +---------------+
```

## Stack

Every component is open source and self-hosted via Docker. The only external dependency is an LLM API for AI-assisted response drafting (optional).

| Component | Tool | Role | License | GitHub Stars |
|---|---|---|---|---|
| Cross-posting & scheduling | [Postiz](https://github.com/gitroomhq/postiz-app) | Publish to 17+ platforms from one dashboard | AGPL-3.0 | 27K+ |
| Workflow automation | [n8n](https://github.com/n8n-io/n8n) | Connect services, build automation pipelines | Sustainable Use | 60K+ |
| Team chat & notifications | [Mattermost](https://github.com/mattermost/mattermost) | Notification hub with organized channels | AGPL-3.0 | 32K+ |
| Web & page monitoring | [changedetection.io](https://github.com/dgtlmoon/changedetection.io) | Track changes on any web page, keyword alerts | Apache-2.0 | 30K+ |
| RSS feed generation | [RSSHub](https://github.com/DIYgod/RSSHub) | Turn any website into an RSS feed (1000+ routes) | MIT | 41K+ |
| RSS reader / dashboard | [Folo](https://github.com/RSSNext/Folo) | AI-powered feed reader for manual browsing | GPL-3.0 | 37K+ |
| Push notifications | [ntfy](https://github.com/binwiederhier/ntfy) | Instant push alerts to phone/desktop via HTTP | Apache-2.0 | 20K+ |
| Web analytics | [Plausible](https://github.com/plausible/analytics) | Privacy-friendly analytics (no cookies) | AGPL-3.0 | 22K+ |
| Reddit monitoring | Custom bot using [PRAW](https://github.com/praw-dev/praw) | Stream subreddits for keyword matches | BSD | — |
| Hacker News monitoring | Custom poller using [HN API](https://github.com/HackerNews/API) | Poll new stories/comments for keyword matches | Public API | — |
| Blog syndication | [cross-post CLI](https://github.com/shahednasser/cross-post) + GitHub Actions | Auto-publish to Dev.to and Hashnode | MIT | — |

### How the pieces connect

**Monitoring pipeline:**
1. **Reddit** — PRAW bot streams submissions and comments from target subreddits, matches against keyword lists, sends alerts via webhook
2. **Hacker News** — Custom poller checks new stories/comments against keywords via the public HN API; RSSHub generates keyword-filtered HN feeds
3. **Web/News** — changedetection.io watches web pages (competitor blogs, procurement sites, news) for keyword-triggered changes
4. **RSS** — RSSHub turns platforms without native RSS into feeds; changedetection.io monitors feeds for keyword matches
5. **GitHub** — Native webhooks push events (stars, issues, PRs, discussions) directly

**Routing pipeline:**
1. All monitoring sources send webhooks to **n8n**
2. n8n routes alerts to organized **Mattermost** channels (`#mentions`, `#competitors`, `#github`, etc.)
3. High-priority alerts also push to **ntfy** for instant mobile/desktop notifications
4. n8n optionally calls an LLM API to draft contextual responses for human review

**Publishing pipeline:**
1. Content is created and scheduled in **Postiz** (or triggered via n8n)
2. Postiz publishes to all connected social platforms simultaneously
3. Blog posts are syndicated to Dev.to and Hashnode via **cross-post CLI** in CI/CD
4. **Plausible** tracks resulting traffic and referral sources

## Deployment

All services run as Docker containers via a single `docker-compose.yml`:

```yaml
services:
  postiz:          # Cross-posting & scheduling
  n8n:             # Workflow automation
  mattermost:      # Team chat & notifications
  changedetect:    # Web page monitoring
  rsshub:          # RSS feed generation
  ntfy:            # Push notifications
  plausible:       # Web analytics
  reddit-monitor:  # Custom PRAW bot
  hn-monitor:      # Custom HN poller
```

**Estimated requirements:** 4 vCPU, 8GB RAM, 50GB SSD (~$20-40/mo on any VPS, or run on existing infrastructure).

## Status

Early stage — architecture defined, component selection complete. Implementation in progress.

## License

Apache-2.0
