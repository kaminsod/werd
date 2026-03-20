import { useState, type FormEvent } from "react";
import { useParams, Link, useNavigate } from "react-router";
import { usePost, useUpdatePost, useDeletePost, usePublishPost, useSetPostMonitor } from "@/hooks/use-posts";
import { useConnections } from "@/hooks/use-connections";
import type { PostStatus, PlatformPublishResult } from "@/types/api";

const PLATFORM_LABELS: Record<string, string> = {
  bluesky: "Bluesky",
  reddit: "Reddit",
  hn: "Hacker News",
};

const ALL_PLATFORMS = ["bluesky", "reddit", "hn"];

const STATUS_COLORS: Record<PostStatus, string> = {
  draft: "bg-gray-100 text-gray-700",
  scheduled: "bg-blue-100 text-blue-700",
  publishing: "bg-yellow-100 text-yellow-700",
  published: "bg-green-100 text-green-700",
  failed: "bg-red-100 text-red-700",
};

export default function PostDetailPage() {
  const { id: projectId, postId } = useParams<{ id: string; postId: string }>();
  const navigate = useNavigate();
  const { data: post, isLoading, error } = usePost(projectId!, postId!);
  const { data: connections } = useConnections(projectId!);
  const updatePost = useUpdatePost(projectId!);
  const deletePost = useDeletePost(projectId!);
  const publishPost = usePublishPost(projectId!);
  const setMonitor = useSetPostMonitor(projectId!);

  const [editing, setEditing] = useState(false);
  const [title, setTitle] = useState("");
  const [content, setContent] = useState("");
  const [postURL, setPostURL] = useState("");
  const [postType, setPostType] = useState<"text" | "link">("text");
  const [selectedPlatforms, setSelectedPlatforms] = useState<string[]>([]);
  const [publishResults, setPublishResults] = useState<PlatformPublishResult[] | null>(null);
  const [isReply, setIsReply] = useState(false);
  const [replyToURL, setReplyToURL] = useState("");

  const API_PUB = new Set(["bluesky", "reddit"]);
  const BROWSER_PUB = new Set(["bluesky", "reddit", "hn"]);
  const availablePlatforms = connections
    ?.filter((c) => c.enabled && ((c.method === "api" && API_PUB.has(c.platform)) || (c.method === "browser" && BROWSER_PUB.has(c.platform))))
    .map((c) => c.platform)
    .filter((v, i, a) => a.indexOf(v) === i) ?? [];

  function detectPlatformFromURL(url: string): string | null {
    try {
      const hostname = new URL(url).hostname;
      if (/reddit\.com$/.test(hostname)) return "reddit";
      if (/bsky\.app$/.test(hostname)) return "bluesky";
      if (/news\.ycombinator\.com$/.test(hostname)) return "hn";
    } catch { /* invalid URL */ }
    return null;
  }

  function handleReplyURLChange(url: string) {
    setReplyToURL(url);
    const detected = detectPlatformFromURL(url);
    if (detected) {
      setSelectedPlatforms([detected]);
    }
  }

  function startEdit() {
    if (!post) return;
    setTitle(post.title || "");
    setContent(post.content);
    setPostURL(post.url || "");
    setPostType(post.post_type || "text");
    setSelectedPlatforms(post.platforms);
    setIsReply(!!post.reply_to_url);
    setReplyToURL(post.reply_to_url || "");
    setEditing(true);
  }

  function handleSave(e: FormEvent) {
    e.preventDefault();
    updatePost.mutate(
      { postId: postId!, title, content, url: postURL, post_type: postType, platforms: selectedPlatforms, reply_to_url: isReply ? replyToURL : undefined },
      { onSuccess: () => setEditing(false) },
    );
  }

  function handleDelete() {
    if (confirm("Delete this post?")) {
      deletePost.mutate(postId!, { onSuccess: () => navigate("../posts") });
    }
  }

  function handlePublish() {
    publishPost.mutate(postId!, {
      onSuccess: (res) => setPublishResults(res.results),
    });
  }

  if (isLoading) return <p className="text-gray-500">Loading post...</p>;
  if (error) return <p className="text-red-600">Error: {error.message}</p>;
  if (!post) return <p className="text-gray-500">Post not found.</p>;

  const needsTitle = selectedPlatforms.some((p) => p === "reddit" || p === "hn");

  return (
    <div>
      <Link to="../posts" className="mb-4 inline-block text-sm text-blue-600 hover:underline">&larr; Back to Posts</Link>

      <div className="mb-4 flex items-center gap-3">
        <h2 className="text-xl font-semibold">{post.title || post.content.slice(0, 60) || "(empty)"}</h2>
        <span className={`rounded px-2 py-0.5 text-xs font-medium ${STATUS_COLORS[post.status]}`}>{post.status}</span>
      </div>

      {/* Publish results */}
      {publishResults && (
        <div className="mb-4 rounded border bg-white p-4">
          <h3 className="mb-2 text-sm font-medium">Publish Results</h3>
          <div className="space-y-1">
            {publishResults.map((r) => (
              <div key={r.platform} className="flex items-center gap-2 text-sm">
                <span className={`rounded px-2 py-0.5 text-xs font-medium ${r.success ? "bg-green-100 text-green-700" : "bg-red-100 text-red-700"}`}>
                  {r.platform}
                </span>
                {r.success ? (
                  r.url ? (
                    <a href={r.url} target="_blank" rel="noopener noreferrer" className="text-blue-600 hover:underline">View post</a>
                  ) : (
                    <span className="text-green-600">Published</span>
                  )
                ) : (
                  <span className="text-red-600">{r.error}</span>
                )}
              </div>
            ))}
          </div>
          <button onClick={() => setPublishResults(null)} className="mt-2 text-xs text-gray-500 hover:text-gray-700">Dismiss</button>
        </div>
      )}

      {editing ? (
        <form onSubmit={handleSave} className="space-y-3 rounded border bg-gray-50 p-4">
          <h3 className="text-sm font-medium">Edit Post</h3>
          {updatePost.error && <p className="text-sm text-red-600">{updatePost.error.message}</p>}

          {/* Mode toggle */}
          <div className="flex items-center gap-4">
            <label className="flex items-center gap-1.5">
              <input type="radio" name="post_mode" checked={!isReply} onChange={() => { setIsReply(false); setReplyToURL(""); }} />
              <span className="text-sm">New Post</span>
            </label>
            <label className="flex items-center gap-1.5">
              <input type="radio" name="post_mode" checked={isReply} onChange={() => setIsReply(true)} />
              <span className="text-sm">Reply to Thread</span>
            </label>
          </div>

          {/* Reply URL */}
          {isReply && (
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-600">Reply to URL</label>
              <input
                value={replyToURL}
                onChange={(e) => handleReplyURLChange(e.target.value)}
                type="url"
                required
                placeholder="https://reddit.com/r/... or https://bsky.app/..."
                className="w-full rounded border px-3 py-2 text-sm"
              />
              {replyToURL && !detectPlatformFromURL(replyToURL) && (
                <p className="mt-1 text-xs text-amber-600">Could not detect platform from URL</p>
              )}
            </div>
          )}

          {!isReply && needsTitle && (
            <div className="flex items-center gap-4">
              <label className="text-xs font-medium text-gray-600">Post type:</label>
              <label className="flex items-center gap-1.5">
                <input type="radio" name="post_type" value="text" checked={postType === "text"} onChange={() => setPostType("text")} />
                <span className="text-sm">Text Post</span>
              </label>
              <label className="flex items-center gap-1.5">
                <input type="radio" name="post_type" value="link" checked={postType === "link"} onChange={() => setPostType("link")} />
                <span className="text-sm">Link Post</span>
              </label>
            </div>
          )}

          {!isReply && needsTitle && (
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-600">Title</label>
              <input value={title} onChange={(e) => setTitle(e.target.value)} placeholder="Post title..." className="w-full rounded border px-3 py-2 text-sm" />
            </div>
          )}

          {!isReply && postType === "link" && (
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-600">URL</label>
              <input value={postURL} onChange={(e) => setPostURL(e.target.value)} type="url" placeholder="https://example.com/article" className="w-full rounded border px-3 py-2 text-sm" />
            </div>
          )}

          <div>
            <label className="mb-1 block text-xs font-medium text-gray-600">{needsTitle ? "Body" : "Content"}</label>
            <textarea value={content} onChange={(e) => setContent(e.target.value)} required={postType !== "link"} rows={5} className="w-full rounded border px-3 py-2 text-sm" />
          </div>

          <div>
            <label className="mb-1 block text-xs font-medium text-gray-600">Platforms</label>
            <div className="flex flex-wrap gap-3">
              {ALL_PLATFORMS.map((p) => {
                const available = availablePlatforms.includes(p);
                const target = connections?.find((c) => c.platform === p)?.target;
                const locked = isReply && replyToURL && detectPlatformFromURL(replyToURL) !== null;
                return (
                  <label key={p} className={`flex items-center gap-1.5 ${!available ? "opacity-50" : ""}`}>
                    <input type="checkbox" checked={selectedPlatforms.includes(p)} onChange={() => setSelectedPlatforms((prev) => prev.includes(p) ? prev.filter((x) => x !== p) : [...prev, p])} disabled={!available || !!locked} />
                    <span className="text-sm">
                      {PLATFORM_LABELS[p] ?? p}
                      {target && <span className="ml-1 text-gray-400">({target})</span>}
                      {!available && <span className="ml-1 text-xs text-gray-400">(no connection)</span>}
                    </span>
                  </label>
                );
              })}
            </div>
          </div>

          <div className="flex gap-2">
            <button type="submit" disabled={updatePost.isPending} className="rounded bg-gray-600 px-4 py-2 text-sm font-medium text-white hover:bg-gray-700 disabled:opacity-50">
              Save
            </button>
            <button type="button" onClick={() => setEditing(false)} className="text-sm text-gray-500 hover:text-gray-700">Cancel</button>
          </div>
        </form>
      ) : (
        <div className="space-y-4 rounded border bg-white p-4">
          <div className="grid grid-cols-2 gap-4 text-sm">
            <div>
              <span className="font-medium text-gray-500">Type</span>
              <p className="mt-1">{post.post_type === "link" ? "Link" : "Text"}</p>
            </div>
            <div>
              <span className="font-medium text-gray-500">Platforms</span>
              <div className="mt-1 flex flex-wrap gap-1">
                {post.platforms.map((p) => {
                  const target = connections?.find((c) => c.platform === p)?.target;
                  return (
                    <Link key={p} to="../connections" className="rounded bg-indigo-50 px-2 py-0.5 text-xs text-indigo-600 hover:underline">
                      {PLATFORM_LABELS[p] ?? p}
                      {target && <span className="ml-1 text-indigo-400">{target}</span>}
                    </Link>
                  );
                })}
              </div>
            </div>
          </div>

          {post.reply_to_url && (
            <div>
              <span className="text-sm font-medium text-gray-500">Replying to</span>
              <p className="mt-1">
                <a href={post.reply_to_url} target="_blank" rel="noopener noreferrer" className="text-sm text-blue-600 hover:underline">{post.reply_to_url}</a>
              </p>
            </div>
          )}

          {post.title && (
            <div>
              <span className="text-sm font-medium text-gray-500">Title</span>
              <p className="mt-1 text-sm font-medium">{post.title}</p>
            </div>
          )}

          {post.url && (
            <div>
              <span className="text-sm font-medium text-gray-500">URL</span>
              <p className="mt-1">
                <a href={post.url} target="_blank" rel="noopener noreferrer" className="text-sm text-blue-600 hover:underline">{post.url}</a>
              </p>
            </div>
          )}

          <div>
            <span className="text-sm font-medium text-gray-500">Content</span>
            <p className="mt-1 whitespace-pre-wrap text-sm text-gray-700">{post.content}</p>
          </div>

          <div className="grid grid-cols-3 gap-4 text-sm">
            <div>
              <span className="font-medium text-gray-500">Created</span>
              <p className="mt-1">{new Date(post.created_at).toLocaleString()}</p>
            </div>
            {post.scheduled_at && (
              <div>
                <span className="font-medium text-gray-500">Scheduled</span>
                <p className="mt-1">{new Date(post.scheduled_at).toLocaleString()}</p>
              </div>
            )}
            {post.published_at && (
              <div>
                <span className="font-medium text-gray-500">Published</span>
                <p className="mt-1">{new Date(post.published_at).toLocaleString()}</p>
              </div>
            )}
          </div>

          <div className="flex gap-2 border-t pt-4">
            {post.status === "draft" && (
              <>
                <button onClick={handlePublish} disabled={publishPost.isPending} className="rounded bg-blue-600 px-3 py-1 text-xs font-medium text-white hover:bg-blue-700 disabled:opacity-50">
                  {publishPost.isPending ? "Publishing..." : "Publish Now"}
                </button>
                <button onClick={startEdit} className="rounded px-2 py-1 text-xs text-blue-600 hover:bg-blue-50">Edit</button>
                <button onClick={handleDelete} className="rounded px-2 py-1 text-xs text-red-600 hover:bg-red-50">Delete</button>
              </>
            )}
            {post.status === "failed" && (
              <button onClick={handlePublish} disabled={publishPost.isPending} className="rounded bg-orange-600 px-3 py-1 text-xs font-medium text-white hover:bg-orange-700 disabled:opacity-50">
                Retry
              </button>
            )}
            {post.status === "published" && (
              <button
                onClick={() => setMonitor.mutate({ postId: post.id, enable: true })}
                disabled={setMonitor.isPending}
                className="rounded border border-purple-300 px-3 py-1 text-xs text-purple-600 hover:bg-purple-50 disabled:opacity-50"
              >
                {setMonitor.isPending ? "..." : "Monitor Replies"}
              </button>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
