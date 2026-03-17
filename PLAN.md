# Werd — Development Plan

Current status and implementation milestones. See [README.md](README.md) for the full milestone table.

## Current Status

MVP complete. Core infrastructure, API server, dashboard, and monitoring pipeline are functional. Processing rules pipeline with LLM classification is implemented. Dual method support (API/browser) for all platforms.

## Phase Overview

| Phase | Name | Status |
|---|---|---|
| 1 | Core Infrastructure | Done |
| 2 | Werd API Server (Go Backend) | Mostly done |
| 3 | Lightweight Service Deployment | Done |
| 4 | Werd Dashboard (React Frontend) | Done |
| 5 | Monitoring Pipeline | Done (Reddit, HN monitors; processing rules) |
| 6 | Notification & Routing | Done (ntfy, webhook, routing engine) |
| 7 | Publishing Pipeline | In progress |
| 8 | Network Access & Tunnel | Not started |
| 9 | Kubernetes Deployment | Not started |
| 10 | Hardening & Documentation | Not started |

See `plan/phases/` for detailed per-phase plans.
