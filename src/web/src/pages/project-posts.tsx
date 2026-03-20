import { useState, type FormEvent } from "react";
import { useParams, Link } from "react-router";
import { usePosts, useCreatePost, useUpdatePost, useDeletePost, usePublishPost, useSetPostMonitor } from "@/hooks/use-posts";
import { useConnections } from "@/hooks/use-connections";
import InfoIcon from "@/components/info-icon";
import { publishPost as publishHelp } from "@/lib/help-content";
import type { Post, PostStatus, PlatformPublishResult } from "@/types/api";

const STATUS_COLORS: Record<PostStatus, string> = {
  draft: "bg-gray-100 text-gray-700",
  scheduled: "bg-blue-100 text-blue-700",
  publishing: "bg-yellow-100 text-yellow-700",
  published: "bg-green-100 text-green-700",
  failed: "bg-red-100 text-red-700",
};

const PLATFORM_LABELS: Record<string, string> = {
  bluesky: "Bluesky",
  reddit: "Reddit",
  hn: "Hacker News",
};

const ALL_PLATFORMS = ["bluesky", "reddit", "hn"];

const PAGE_SIZE = 20;

export default function PostsPage() {
  const { id: projectId } = useParams<{ id: string }>();
  const [statusFilter, setStatusFilter] = useState("");
  const [page, setPage] = useState(0);

  const { data, isLoading, error } = usePosts(projectId!, {
    status: statusFilter || undefined,
    limit: PAGE_SIZE,
    offset: page * PAGE_SIZE,
  });
  const { data: connections } = useConnections(projectId!);

  const createPost = useCreatePost(projectId!);
  const updatePost = useUpdatePost(projectId!);
  const deletePost = useDeletePost(projectId!);
  const publishPost = usePublishPost(projectId!);
  const setMonitor = useSetPostMonitor(projectId!);

  const [showCompose, setShowCompose] = useState(false);
  const [editId, setEditId] = useState<string | null>(null);
  const [title, setTitle] = useState("");
  const [content, setContent] = useState("");
  const [postURL, setPostURL] = useState("");
  const [postType, setPostType] = useState<"text" | "link">("text");
  const [selectedPlatforms, setSelectedPlatforms] = useState<string[]>([]);
  const [publishResults, setPublishResults] = useState<PlatformPublishResult[] | null>(null);
  const [isReply, setIsReply] = useState(false);
  const [replyToURL, setReplyToURL] = useState("");

  // Show platforms that can publish — API-publishable or browser-publishable connections.
  const API_PUB = new Set(["bluesky", "reddit"]);
  const BROWSER_PUB = new Set(["bluesky", "reddit", "hn"]);
  const availablePlatforms = connections
    ?.filter((c) => c.enabled && ((c.method === "api" && API_PUB.has(c.platform)) || (c.method === "browser" && BROWSER_PUB.has(c.platform))))
    .map((c) => c.platform)
    .filter((v, i, a) => a.indexOf(v) === i) ?? [];

  // Build platform → target lookup from connections.
  const platformTargets: Record<string, string> = {};
  connections?.forEach((c) => {
    if (c.target && !platformTargets[c.platform]) platformTargets[c.platform] = c.target;
  });

  // Determine if any selected platform needs a title (Reddit, HN).
  const needsTitle = selectedPlatforms.some((p) => p === "reddit" || p === "hn");
  const needsPostType = needsTitle;

  function resetForm() {
    setTitle("");
    setContent("");
    setPostURL("");
    setPostType("text");
    setSelectedPlatforms([]);
    setShowCompose(false);
    setEditId(null);
    setPublishResults(null);
    setIsReply(false);
    setReplyToURL("");
  }

  function startEdit(post: Post) {
    setEditId(post.id);
    setTitle(post.title || "");
    setContent(post.content);
    setPostURL(post.url || "");
    setPostType(post.post_type || "text");
    setSelectedPlatforms(post.platforms);
    setShowCompose(true);
    setPublishResults(null);
    setIsReply(!!post.reply_to_url);
    setReplyToURL(post.reply_to_url || "");
  }

  function togglePlatform(platform: string) {
    setSelectedPlatforms((prev) =>
      prev.includes(platform) ? prev.filter((p) => p !== platform) : [...prev, platform],
    );
  }

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

  function handleSave(e: FormEvent) {
    e.preventDefault();
    const postData = { title, content, url: postURL, post_type: postType, platforms: selectedPlatforms, reply_to_url: isReply ? replyToURL : undefined };
    if (editId) {
      updatePost.mutate(
        { postId: editId, ...postData },
        { onSuccess: resetForm },
      );
    } else {
      createPost.mutate(postData, { onSuccess: resetForm });
    }
  }

  function handlePublish(postId: string) {
    publishPost.mutate(postId, {
      onSuccess: (res) => setPublishResults(res.results),
      onError: () => setPublishResults(null),
    });
  }

  function handleDelete(postId: string) {
    if (confirm("Delete this draft?")) {
      deletePost.mutate(postId);
    }
  }

  if (isLoading) return <p className="text-gray-500">Loading posts...</p>;
  if (error) return <p className="text-red-600">Error: {error.message}</p>;

  const { posts, total } = data!;
  const totalPages = Math.ceil(total / PAGE_SIZE);

  return (
    <div>
      <div className="mb-4 flex items-center justify-between">
        <h2 className="text-xl font-semibold">Posts</h2>
        <button
          onClick={() => { resetForm(); setShowCompose(!showCompose); }}
          className="rounded bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
        >
          {showCompose && !editId ? "Cancel" : "Compose"}
        </button>
      </div>

      {/* Compose / edit form */}
      {showCompose && (
        <form onSubmit={handleSave} className="mb-6 space-y-3 rounded border bg-gray-50 p-4">
          <h3 className="text-sm font-medium">{editId ? "Edit Draft" : "New Post"}</h3>

          {(createPost.error || updatePost.error) && (
            <p className="text-sm text-red-600">
              {(createPost.error || updatePost.error)?.message}
            </p>
          )}

          {/* Mode toggle: New Post vs Reply */}
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

          {/* Post type selector (for Reddit, HN) */}
          {!isReply && needsPostType && (
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

          {/* Title field (for Reddit, HN) — hidden in reply mode */}
          {!isReply && needsTitle && (
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-600">Title</label>
              <input
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder="Post title..."
                className="w-full rounded border px-3 py-2 text-sm"
              />
            </div>
          )}

          {/* URL field (for link posts) — hidden in reply mode */}
          {!isReply && postType === "link" && (
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-600">URL</label>
              <input
                value={postURL}
                onChange={(e) => setPostURL(e.target.value)}
                type="url"
                placeholder="https://example.com/article"
                className="w-full rounded border px-3 py-2 text-sm"
              />
            </div>
          )}

          {/* Body/content */}
          <div>
            <label className="mb-1 block text-xs font-medium text-gray-600">
              {needsTitle ? "Body" : "Content"}
              {postType === "link" && <span className="font-normal text-gray-400"> (optional for link posts)</span>}
            </label>
            <textarea
              value={content}
              onChange={(e) => setContent(e.target.value)}
              required={postType !== "link"}
              rows={5}
              placeholder={needsTitle ? "Post body (optional for link posts)..." : "Write your post..."}
              className="w-full rounded border px-3 py-2 text-sm"
            />
          </div>

          <div>
            <label className="mb-1 block text-xs font-medium text-gray-600">Platforms</label>
            <div className="flex flex-wrap gap-3">
              {ALL_PLATFORMS.map((p) => {
                const available = availablePlatforms.includes(p);
                const locked = isReply && replyToURL && detectPlatformFromURL(replyToURL) !== null;
                return (
                  <label key={p} className={`flex items-center gap-1.5 ${!available ? "opacity-50" : ""}`}>
                    <input
                      type="checkbox"
                      checked={selectedPlatforms.includes(p)}
                      onChange={() => togglePlatform(p)}
                      disabled={!available || !!locked}
                    />
                    <span className="text-sm">
                      {PLATFORM_LABELS[p] ?? p}
                      {platformTargets[p] && <span className="ml-1 text-gray-400">({platformTargets[p]})</span>}
                      {!available && <span className="ml-1 text-xs text-gray-400">(no connection)</span>}
                    </span>
                  </label>
                );
              })}
            </div>
          </div>

          <div className="flex gap-2">
            <button
              type="submit"
              disabled={createPost.isPending || updatePost.isPending || selectedPlatforms.length === 0}
              className="rounded bg-gray-600 px-4 py-2 text-sm font-medium text-white hover:bg-gray-700 disabled:opacity-50"
            >
              {editId ? "Update Draft" : "Save Draft"}
            </button>
            {editId && (
              <button type="button" onClick={resetForm} className="text-sm text-gray-500 hover:text-gray-700">
                Cancel
              </button>
            )}
          </div>
        </form>
      )}

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
                    <a href={r.url} target="_blank" rel="noopener noreferrer" className="text-blue-600 hover:underline">
                      View post
                    </a>
                  ) : (
                    <span className="text-green-600">Published</span>
                  )
                ) : (
                  <span className="text-red-600">{r.error}</span>
                )}
              </div>
            ))}
          </div>
          <button onClick={() => setPublishResults(null)} className="mt-2 text-xs text-gray-500 hover:text-gray-700">
            Dismiss
          </button>
        </div>
      )}

      {/* Filter */}
      <div className="mb-4 flex gap-3">
        <select
          value={statusFilter}
          onChange={(e) => { setStatusFilter(e.target.value); setPage(0); }}
          className="rounded border px-3 py-1.5 text-sm"
        >
          <option value="">All statuses</option>
          <option value="draft">Draft</option>
          <option value="published">Published</option>
          <option value="failed">Failed</option>
        </select>
        <span className="ml-auto text-sm text-gray-500">{total} post{total !== 1 ? "s" : ""}</span>
      </div>

      {/* Post list */}
      {posts.length === 0 ? (
        <p className="text-gray-500">No posts yet. Compose one to get started.</p>
      ) : (
        <div className="space-y-2">
          {posts.map((post) => (
            <div key={post.id} className="rounded border bg-white p-3">
              <div className="mb-2 flex items-center gap-2">
                <span className={`rounded px-2 py-0.5 text-xs font-medium ${STATUS_COLORS[post.status]}`}>
                  {post.status}
                </span>
                {post.platforms.map((p) => (
                  <span key={p} className="rounded bg-indigo-50 px-2 py-0.5 text-xs text-indigo-600">
                    {PLATFORM_LABELS[p] ?? p}
                    {platformTargets[p] && <span className="ml-1 text-indigo-400">{platformTargets[p]}</span>}
                  </span>
                ))}
                {post.post_type === "link" && (
                  <span className="rounded bg-sky-50 px-2 py-0.5 text-xs text-sky-600">link</span>
                )}
                {post.reply_to_url && (
                  <span className="rounded bg-amber-50 px-2 py-0.5 text-xs text-amber-700" title={post.reply_to_url}>reply</span>
                )}
                <span className="ml-auto text-xs text-gray-400">
                  {post.published_at
                    ? `Published ${new Date(post.published_at).toLocaleString()}`
                    : new Date(post.created_at).toLocaleString()}
                </span>
              </div>
              <Link to={post.id} className="block hover:underline">
                {post.title && (
                  <p className="mb-1 text-sm font-medium text-gray-900">{post.title}</p>
                )}
                {post.url && (
                  <p className="mb-1 text-xs text-blue-600 truncate">{post.url}</p>
                )}
                <p className="mb-2 whitespace-pre-wrap text-sm text-gray-700">
                  {post.content.length > 200 ? post.content.slice(0, 200) + "..." : post.content}
                </p>
              </Link>
              <div className="flex gap-2">
                {post.status === "draft" && (
                  <>
                    <span className="inline-flex items-center gap-1">
                      <button
                        onClick={() => handlePublish(post.id)}
                        disabled={publishPost.isPending}
                        className="rounded bg-blue-600 px-3 py-1 text-xs font-medium text-white hover:bg-blue-700 disabled:opacity-50"
                      >
                        {publishPost.isPending ? "Publishing..." : "Publish Now"}
                      </button>
                      <InfoIcon tooltip={publishHelp.tooltip}>{publishHelp.modal}</InfoIcon>
                    </span>
                    <button onClick={() => startEdit(post)} className="rounded px-2 py-1 text-xs text-blue-600 hover:bg-blue-50">
                      Edit
                    </button>
                    <button onClick={() => handleDelete(post.id)} className="rounded px-2 py-1 text-xs text-red-600 hover:bg-red-50">
                      Delete
                    </button>
                  </>
                )}
                {post.status === "failed" && (
                  <button
                    onClick={() => handlePublish(post.id)}
                    disabled={publishPost.isPending}
                    className="rounded bg-orange-600 px-3 py-1 text-xs font-medium text-white hover:bg-orange-700 disabled:opacity-50"
                  >
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
          ))}
        </div>
      )}

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="mt-4 flex items-center justify-between">
          <button disabled={page === 0} onClick={() => setPage(page - 1)} className="rounded border px-3 py-1 text-sm disabled:opacity-40">
            Previous
          </button>
          <span className="text-sm text-gray-500">Page {page + 1} of {totalPages}</span>
          <button disabled={page >= totalPages - 1} onClick={() => setPage(page + 1)} className="rounded border px-3 py-1 text-sm disabled:opacity-40">
            Next
          </button>
        </div>
      )}
    </div>
  );
}
