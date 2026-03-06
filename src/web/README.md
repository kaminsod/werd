# Werd Dashboard

React + TypeScript SPA for managing the Werd platform.

## Stack

- **React 19** + TypeScript
- **Vite** — build tooling
- **React Router** — client-side routing
- **TanStack Query** — data fetching
- **Zustand** — state management
- **Tailwind CSS** + shadcn/ui — styling and components

## Development

```bash
npm install
npm run dev           # Dev server on http://localhost:3000
npm run build         # Production build
npm run generate-types  # Regenerate API types from OpenAPI spec
```

API requests are proxied to `http://localhost:8090` in dev mode (see `vite.config.ts`).
