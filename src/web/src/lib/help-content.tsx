// Centralized help content for info icons throughout the dashboard.
// Each entry has a short `tooltip` (hover) and detailed `modal` content (click).

import type { ReactNode } from "react";

interface HelpEntry {
  tooltip: string;
  modal: ReactNode;
}

// ── Sidebar / Navigation ──

export const navAlerts: HelpEntry = {
  tooltip: "Incoming monitoring events",
  modal: (
    <>
      <h3 className="mb-2 font-semibold">Alerts</h3>
      <p className="mb-2">
        Alerts are events detected by your monitoring sources that matched your keywords.
        They arrive via webhook from services like changedetection.io, Reddit monitors,
        or HN pollers.
      </p>
      <p className="mb-2">
        Each alert has a <strong>severity</strong> (low, medium, high, critical) and a
        <strong> status</strong> for triage: new → seen → triaged → dismissed/responded.
      </p>
      <p>Click an alert to expand it and change its status.</p>
    </>
  ),
};

export const navKeywords: HelpEntry = {
  tooltip: "Patterns that trigger alerts",
  modal: (
    <>
      <h3 className="mb-2 font-semibold">Keywords</h3>
      <p className="mb-2">
        Keywords define what text patterns to look for in incoming content.
        When an alert arrives, its title and content are checked against all your keywords.
      </p>
      <p className="mb-2"><strong>Match types:</strong></p>
      <ul className="mb-2 list-inside list-disc space-y-1">
        <li><strong>Substring</strong> — matches if the keyword appears anywhere (case-insensitive)</li>
        <li><strong>Exact</strong> — matches only if the entire title or content equals the keyword</li>
        <li><strong>Regex</strong> — matches using a regular expression pattern</li>
      </ul>
      <p>Matched keywords are shown on each alert for quick reference.</p>
    </>
  ),
};

export const navSources: HelpEntry = {
  tooltip: "Configure what to monitor",
  modal: (
    <>
      <h3 className="mb-2 font-semibold">Monitor Sources</h3>
      <p className="mb-2">
        Sources define what external services and feeds to monitor. Each source has a type
        and a JSON configuration specific to that type.
      </p>
      <p className="mb-2"><strong>Source types:</strong></p>
      <ul className="mb-2 list-inside list-disc space-y-1">
        <li><strong>reddit</strong> — subreddits, threads, or account activity</li>
        <li><strong>hn</strong> — Hacker News stories, threads, or accounts</li>
        <li><strong>bluesky</strong> — account notifications or user feeds</li>
        <li><strong>web</strong> — web pages via changedetection.io</li>
        <li><strong>rss</strong> — RSS/Atom feeds via RSSHub</li>
        <li><strong>github</strong> — repositories and organizations</li>
      </ul>
      <p>Sources can be enabled/disabled without deleting them.</p>
    </>
  ),
};

export const navProcessing: HelpEntry = {
  tooltip: "Filter and classify monitored items",
  modal: (
    <>
      <h3 className="mb-2 font-semibold">Processing Rules</h3>
      <p className="mb-2">
        Processing rules sit in the pipeline between monitor polling and alert ingestion.
        They control which items become alerts and how those alerts are classified.
      </p>
      <p className="mb-2"><strong>Two phases:</strong></p>
      <ul className="mb-2 list-inside list-disc space-y-1">
        <li><strong>Filter</strong> — include or exclude items based on keywords or regex patterns</li>
        <li><strong>Classify</strong> — assign severity, tags, and classification reasons to items</li>
      </ul>
      <p className="mb-2"><strong>Rule types:</strong></p>
      <ul className="mb-2 list-inside list-disc space-y-1">
        <li><strong>Keyword</strong> — match against keywords with exact, substring, or regex matching</li>
        <li><strong>Regex</strong> — match using a regular expression pattern</li>
        <li><strong>LLM</strong> — use an AI model for semantic analysis (requires LLM endpoint configured)</li>
      </ul>
      <p>Rules can be scoped to a specific source or apply to all sources in the project.</p>
    </>
  ),
};

export const navRules: HelpEntry = {
  tooltip: "Where and when alerts trigger notifications",
  modal: (
    <>
      <h3 className="mb-2 font-semibold">Notification Rules</h3>
      <p className="mb-2">
        Rules control which alerts trigger notifications and where those notifications are sent.
        Each rule filters by source type and minimum severity.
      </p>
      <p className="mb-2"><strong>Destinations:</strong></p>
      <ul className="mb-2 list-inside list-disc space-y-1">
        <li><strong>ntfy</strong> — push notification via ntfy.sh topic (works on phone/desktop)</li>
        <li><strong>webhook</strong> — HTTP POST to a custom URL with alert details</li>
        <li><strong>email</strong> — not yet implemented</li>
      </ul>
      <p>A rule with source type "all" and min severity "low" will fire for every alert.</p>
    </>
  ),
};

export const navConnections: HelpEntry = {
  tooltip: "Social media account credentials",
  modal: (
    <>
      <h3 className="mb-2 font-semibold">Platform Connections</h3>
      <p className="mb-2">
        Connections store your credentials for social media platforms. These are used when
        publishing posts to those platforms.
      </p>
      <p className="mb-2"><strong>Currently supported:</strong></p>
      <ul className="mb-2 list-inside list-disc space-y-1">
        <li>
          <strong>Bluesky</strong> — uses an app password (not your main password).
          Generate one at Settings → App Passwords in Bluesky.
        </li>
        <li>
          <strong>Reddit</strong> — uses a "script" app for OAuth2.
          Create one at reddit.com/prefs/apps. Each connection targets a single subreddit.
        </li>
        <li>
          <strong>Hacker News</strong> — monitoring only (no posting API).
          No credentials needed. Alerts are ingested by the HN monitor.
        </li>
      </ul>
      <p>Credentials are stored securely and never shown after creation.</p>
    </>
  ),
};

export const navPosts: HelpEntry = {
  tooltip: "Compose and publish to social media",
  modal: (
    <>
      <h3 className="mb-2 font-semibold">Posts</h3>
      <p className="mb-2">
        Create posts and publish them to one or more connected platforms simultaneously.
      </p>
      <p className="mb-2"><strong>Workflow:</strong></p>
      <ol className="mb-2 list-inside list-decimal space-y-1">
        <li>Compose your post content</li>
        <li>Select target platforms (from your connections)</li>
        <li>Save as draft or publish immediately</li>
      </ol>
      <p className="mb-2"><strong>Post statuses:</strong></p>
      <ul className="list-inside list-disc space-y-1">
        <li><strong>Draft</strong> — saved, can be edited or deleted</li>
        <li><strong>Published</strong> — successfully posted to all platforms</li>
        <li><strong>Failed</strong> — one or more platforms failed (can retry)</li>
      </ul>
    </>
  ),
};

export const navMembers: HelpEntry = {
  tooltip: "Team access and roles",
  modal: (
    <>
      <h3 className="mb-2 font-semibold">Members</h3>
      <p className="mb-2">Manage who has access to this project and what they can do.</p>
      <p className="mb-2"><strong>Roles:</strong></p>
      <ul className="mb-2 list-inside list-disc space-y-1">
        <li><strong>Owner</strong> — full control, can delete the project</li>
        <li><strong>Admin</strong> — can manage settings, members, and all content</li>
        <li><strong>Member</strong> — can create keywords, rules, sources, and posts</li>
        <li><strong>Viewer</strong> — read-only access to all project data</li>
      </ul>
      <p>Any member can leave a project, except the owner (transfer ownership first).</p>
    </>
  ),
};

export const navSettings: HelpEntry = {
  tooltip: "Project name, slug, and deletion",
  modal: (
    <>
      <h3 className="mb-2 font-semibold">Settings</h3>
      <p className="mb-2">Update your project name and slug, or delete the project entirely.</p>
      <p className="mb-2">
        The <strong>slug</strong> is a URL-safe identifier (lowercase letters, numbers, and hyphens).
        It must be unique across all projects.
      </p>
      <p><strong>Danger zone:</strong> Deleting a project permanently removes all its data — alerts, keywords, rules, connections, posts, and members.</p>
    </>
  ),
};

// ── Page-level help ──

export const alertSeverity: HelpEntry = {
  tooltip: "How urgent this alert is",
  modal: (
    <>
      <h3 className="mb-2 font-semibold">Alert Severity</h3>
      <ul className="list-inside list-disc space-y-1">
        <li><strong className="text-red-700">Critical</strong> — requires immediate attention</li>
        <li><strong className="text-orange-700">High</strong> — important, should be reviewed soon</li>
        <li><strong className="text-yellow-700">Medium</strong> — moderate relevance</li>
        <li><strong className="text-gray-600">Low</strong> — informational, no urgency</li>
      </ul>
    </>
  ),
};

export const alertStatus: HelpEntry = {
  tooltip: "Triage workflow for alerts",
  modal: (
    <>
      <h3 className="mb-2 font-semibold">Alert Status</h3>
      <p className="mb-2">Use status to track how you've handled each alert:</p>
      <ul className="list-inside list-disc space-y-1">
        <li><strong>New</strong> — just arrived, not reviewed yet</li>
        <li><strong>Seen</strong> — you've looked at it</li>
        <li><strong>Triaged</strong> — confirmed important, needs action</li>
        <li><strong>Dismissed</strong> — not relevant, ignored</li>
        <li><strong>Responded</strong> — you've taken action (replied, published, etc.)</li>
      </ul>
    </>
  ),
};

export const matchType: HelpEntry = {
  tooltip: "How the keyword is compared to content",
  modal: (
    <>
      <h3 className="mb-2 font-semibold">Match Types</h3>
      <ul className="list-inside list-disc space-y-1">
        <li><strong>Substring</strong> — the keyword appears anywhere in the title or content (case-insensitive). Best for general monitoring.</li>
        <li><strong>Exact</strong> — the entire title or content must equal the keyword exactly. Use for very specific terms.</li>
        <li><strong>Regex</strong> — a regular expression pattern. Powerful but must be valid syntax. Example: <code className="bg-gray-100 px-1 rounded">competitor\s+(launch|release)</code></li>
      </ul>
    </>
  ),
};

export const sourceConfig: HelpEntry = {
  tooltip: "JSON configuration for this source type",
  modal: (
    <>
      <h3 className="mb-2 font-semibold">Source Configuration</h3>
      <p className="mb-2">Each source type uses a JSON object for its configuration. Examples:</p>
      <pre className="mb-2 rounded bg-gray-100 p-2 text-xs overflow-x-auto">
{`// Reddit
{"subreddits": ["golang", "selfhosted"]}

// Web (changedetection.io)
{"urls": ["https://example.com/blog"]}

// RSS
{"feeds": ["https://example.com/feed.xml"]}

// GitHub
{"repos": ["owner/repo"]}`}
      </pre>
      <p>The exact fields depend on how your monitors are configured.</p>
    </>
  ),
};

export const ntfyTopic: HelpEntry = {
  tooltip: "ntfy.sh topic name for push notifications",
  modal: (
    <>
      <h3 className="mb-2 font-semibold">ntfy Topic</h3>
      <p className="mb-2">
        A topic is a channel name on ntfy.sh where notifications are sent.
        Choose a unique name for your project, e.g. <code className="bg-gray-100 px-1 rounded">werd-myproject-alerts</code>.
      </p>
      <p className="mb-2">
        Subscribe to this topic in the ntfy app (Android/iOS/desktop) to receive
        push notifications when matching alerts arrive.
      </p>
      <p>If you're running the self-hosted ntfy instance, topics are created automatically.</p>
    </>
  ),
};

export const webhookUrl: HelpEntry = {
  tooltip: "URL that receives alert notifications via HTTP POST",
  modal: (
    <>
      <h3 className="mb-2 font-semibold">Webhook URL</h3>
      <p className="mb-2">
        When a matching alert arrives, Werd sends an HTTP POST request to this URL with
        the alert details as JSON in the request body.
      </p>
      <p className="mb-2">
        The payload includes: alert ID, title, content, URL, severity, matched keywords,
        and source type.
      </p>
      <p>Use this to integrate with Slack, Discord, Zapier, or any custom automation.</p>
    </>
  ),
};

export const minSeverity: HelpEntry = {
  tooltip: "Only trigger for alerts at or above this level",
  modal: (
    <>
      <h3 className="mb-2 font-semibold">Minimum Severity</h3>
      <p className="mb-2">
        This rule will only fire for alerts with a severity at or above the selected level.
      </p>
      <ul className="list-inside list-disc space-y-1">
        <li><strong>Low</strong> — fires for all alerts (low, medium, high, critical)</li>
        <li><strong>Medium</strong> — fires for medium, high, and critical only</li>
        <li><strong>High</strong> — fires for high and critical only</li>
        <li><strong>Critical</strong> — fires for critical alerts only</li>
      </ul>
    </>
  ),
};

export const platformCredentials: HelpEntry = {
  tooltip: "Authentication credentials as JSON",
  modal: (
    <>
      <h3 className="mb-2 font-semibold">Platform Credentials</h3>
      <p className="mb-2">Enter your platform credentials as a JSON object.</p>
      <p className="mb-2"><strong>Bluesky:</strong></p>
      <pre className="mb-2 rounded bg-gray-100 p-2 text-xs overflow-x-auto">
{`{
  "identifier": "yourname.bsky.social",
  "app_password": "xxxx-xxxx-xxxx-xxxx"
}`}
      </pre>
      <p className="mb-2">
        Generate an app password in Bluesky: Settings → App Passwords → Add App Password.
        Do not use your main account password.
      </p>
      <p className="mb-2"><strong>Reddit:</strong></p>
      <pre className="mb-2 rounded bg-gray-100 p-2 text-xs overflow-x-auto">
{`{
  "client_id": "your_app_client_id",
  "client_secret": "your_app_client_secret",
  "username": "your_reddit_username",
  "password": "your_reddit_password",
  "user_agent": "werd/1.0 by u/yourname",
  "subreddit": "target_subreddit"
}`}
      </pre>
      <p className="mb-2">
        Create a Reddit "script" app at reddit.com/prefs/apps.
        The first line of your post content becomes the Reddit title.
      </p>
      <p className="mb-2"><strong>Hacker News:</strong></p>
      <p className="mb-2">
        HN is monitoring-only — no credentials needed. Add an HN connection to receive
        alerts from the HN monitor. Cross-posting to HN is not supported (no posting API).
      </p>
      <p>Credentials are validated on save and never shown again after creation.</p>
    </>
  ),
};

export const publishPost: HelpEntry = {
  tooltip: "Send this post to the selected platforms now",
  modal: (
    <>
      <h3 className="mb-2 font-semibold">Publishing</h3>
      <p className="mb-2">
        Publishing sends your post content to all selected platforms simultaneously.
        Results are shown per-platform after completion.
      </p>
      <p className="mb-2">
        If some platforms succeed and others fail, the post is marked as "failed"
        and you can retry. Only the failed platforms will be retried.
      </p>
      <p>Published posts cannot be edited or deleted from Werd (manage them on each platform directly).</p>
    </>
  ),
};

export const projectSlug: HelpEntry = {
  tooltip: "URL-safe project identifier",
  modal: (
    <>
      <h3 className="mb-2 font-semibold">Project Slug</h3>
      <p className="mb-2">
        The slug is a short, URL-safe identifier for your project. It must be unique and can only
        contain lowercase letters, numbers, and hyphens.
      </p>
      <p>Examples: <code className="bg-gray-100 px-1 rounded">my-project</code>, <code className="bg-gray-100 px-1 rounded">acme-monitoring</code>, <code className="bg-gray-100 px-1 rounded">blog-2026</code></p>
    </>
  ),
};

export const memberUserId: HelpEntry = {
  tooltip: "The UUID of the user to add",
  modal: (
    <>
      <h3 className="mb-2 font-semibold">User ID</h3>
      <p className="mb-2">
        Enter the UUID of the user you want to add to this project.
        You can find a user's ID by asking them to check their profile
        or by looking at the API response.
      </p>
      <p>Format: <code className="bg-gray-100 px-1 rounded">xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx</code></p>
    </>
  ),
};
