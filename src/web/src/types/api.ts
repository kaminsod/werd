// API response types — manually typed to match Go handler structs.

export interface User {
  id: string;
  email: string;
  name: string;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface LoginResponse {
  token: string;
  user: User;
}

export interface MessageResponse {
  message: string;
}

export interface Project {
  id: string;
  name: string;
  slug: string;
  settings: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface Member {
  user_id: string;
  email: string;
  name: string;
  role: ProjectRole;
  created_at: string;
}

export type ProjectRole = "owner" | "admin" | "member" | "viewer";

export interface Alert {
  id: string;
  project_id: string;
  source_type: MonitorType;
  source_id: string;
  title: string;
  content: string;
  url: string;
  matched_keywords: string[];
  severity: AlertSeverity;
  status: AlertStatus;
  created_at: string;
  updated_at: string;
}

export interface AlertListResponse {
  alerts: Alert[];
  total: number;
}

export type AlertSeverity = "low" | "medium" | "high" | "critical";
export type AlertStatus = "new" | "seen" | "triaged" | "dismissed" | "responded";
export type MonitorType = "reddit" | "hn" | "web" | "rss" | "github";

export interface Keyword {
  id: string;
  project_id: string;
  keyword: string;
  match_type: KeywordMatchType;
  created_at: string;
}

export type KeywordMatchType = "exact" | "substring" | "regex";

export interface Rule {
  id: string;
  project_id: string;
  source_type: NotificationSourceType;
  min_severity: AlertSeverity;
  destination: NotificationDestination;
  config: Record<string, unknown>;
  enabled: boolean;
  created_at: string;
}

export type NotificationSourceType = MonitorType | "all";
export type NotificationDestination = "ntfy" | "email" | "webhook";

export interface Source {
  id: string;
  project_id: string;
  type: MonitorType;
  config: Record<string, unknown>;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export type ConnectionMethod = "api" | "browser";

export interface Connection {
  id: string;
  project_id: string;
  platform: string;
  method: ConnectionMethod;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export type PostType = "text" | "link";

export interface Post {
  id: string;
  project_id: string;
  title: string;
  content: string;
  url: string;
  post_type: PostType;
  platforms: string[];
  scheduled_at: string | null;
  published_at: string | null;
  status: PostStatus;
  created_at: string;
  updated_at: string;
}

export interface PostListResponse {
  posts: Post[];
  total: number;
}

export type PostStatus = "draft" | "scheduled" | "publishing" | "published" | "failed";

export interface PlatformPublishResult {
  platform: string;
  success: boolean;
  post_id?: string;
  url?: string;
  error?: string;
}

export interface PublishResponse {
  post: Post;
  results: PlatformPublishResult[];
}
