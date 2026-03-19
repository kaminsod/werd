const PATTERNS: { re: RegExp; render: (m: RegExpMatchArray) => React.ReactNode }[] = [
  // New format: "{A} replied to {B} on "{T}""
  {
    re: /^(.+?) replied to (.+?) on "(.+)"$/,
    render: (m) => <><strong>{m[1]}</strong> replied to <strong>{m[2]}</strong> on &ldquo;{m[3]}&rdquo;</>,
  },
  // New format: "{A} replied to {B}"
  {
    re: /^(.+?) replied to (.+)$/,
    render: (m) => <><strong>{m[1]}</strong> replied to <strong>{m[2]}</strong></>,
  },
  // New format: "{A} commented on "{T}""
  {
    re: /^(.+?) commented on "(.+)"$/,
    render: (m) => <><strong>{m[1]}</strong> commented on &ldquo;{m[2]}&rdquo;</>,
  },
  // Legacy: "Comment by {A} on "{T}""
  {
    re: /^Comment by (.+?) on "(.+)"$/,
    render: (m) => <>Comment by <strong>{m[1]}</strong> on &ldquo;{m[2]}&rdquo;</>,
  },
  // Legacy: "Comment by {A}"
  {
    re: /^Comment by (.+)$/,
    render: (m) => <>Comment by <strong>{m[1]}</strong></>,
  },
  // Legacy: "Reply from @{A}"
  {
    re: /^Reply from @(.+)$/,
    render: (m) => <>Reply from <strong>@{m[1]}</strong></>,
  },
  // Legacy: 'Reply by {A} on ""' (empty title)
  {
    re: /^Reply by (.+?) on ""$/,
    render: (m) => <>Reply by <strong>{m[1]}</strong></>,
  },
  // Legacy: "Reply by {A} on your comment"
  {
    re: /^Reply by (.+?) on your comment$/,
    render: (m) => <>Reply by <strong>{m[1]}</strong></>,
  },
  // Legacy: 'Reply by {A} on "{T}"'
  {
    re: /^Reply by (.+?) on "(.+)"$/,
    render: (m) => <>Reply by <strong>{m[1]}</strong> on &ldquo;{m[2]}&rdquo;</>,
  },
];

export default function AlertTitle({ title }: { title: string }) {
  if (!title) return <span className="text-gray-400">(no title)</span>;

  for (const { re, render } of PATTERNS) {
    const m = title.match(re);
    if (m) return <span>{render(m)}</span>;
  }

  return <span>{title}</span>;
}
