# ADR-0001: Single monorepo for backend, frontend and infrastructure

**Status:** Accepted · **Date:** 2026-07-07

## Context

The Internal Developer Portal consists of a Go API, a Next.js frontend, Bicep infrastructure templates and CI/CD workflows. They share one OpenAPI contract (`backend/api/openapi.yaml`) from which both the Go server interfaces and the TypeScript client are generated. The project is built by a single developer on a 2-4 week part-time schedule.

## Decision

Keep everything in a single repository.

## Rationale

- **Shared contract.** The OpenAPI spec is the source of truth for both sides; changing it in one repo and syncing to another invites drift. In one repo, a contract change and both regenerated artifacts land in the same commit.
- **Atomic full-stack changes.** A feature slice (endpoint + UI) is reviewable in one PR/diff.
- **One CI pipeline.** Lint, test and build for both apps in a single workflow with path filters; no cross-repo triggers.
- **Solo developer.** Polyrepo overhead (versioning the contract, coordinating releases, duplicated tooling config) buys nothing here.

## Consequences

- CI must use path filters so a frontend-only change doesn't rebuild the backend image (and vice versa).
- Go module lives at `backend/` rather than the repo root; tooling must be invoked from that directory.
- If the project ever grew separate teams per app, the repo could be split; the `backend/`/`frontend/` boundary keeps that cheap.
