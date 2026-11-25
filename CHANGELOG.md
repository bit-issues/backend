# Changelog

All notable changes to BitIssues are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/) conventions.

## [Unreleased]

### New Features

- **Live comment stream** — open task pages subscribe to the authenticated `/api/v1/events/stream` Server-Sent Events feed and refetch affected resources when comment, task, or attachment changes arrive; the heartbeat and dead-connection interval is configurable via `EVENTS__HEARTBEAT_INTERVAL` (default `15s`)

## [0.19.0] - 2026-11-19

### New Features

- **Task change history** — a per-task activity feed at `/api/v1/tasks/:slug/:number/feed` merges field-level changes, comments, and attachment events into one newest-first timeline

## [0.18.1] - 2026-11-08

### New Features

- **URL-persisted task filters** — filter state on dashboards and project task lists is stored in the URL, so filtered views survive reloads and can be shared

## [0.18.0] - 2026-11-06

### New Features

- **Per-project Bitbucket webhook management** — admins can register and unregister Bitbucket push hooks from a project's admin settings page; requires the Bitbucket OAuth connection, `BITBUCKET__CALLBACK_URL`, and `WEBHOOKS__SECRET`

## [0.17.0] - 2026-11-01

### New Features

- **Project-qualified commit references** — commit messages can target a task in another project with `fixes <project-slug>#<number>` (for example `fixes resto-web#13`) in addition to bare `#13` references

## [0.16.0] - 2026-09-21

### New Features

- **Bitbucket OAuth connection** — connect, view status, and disconnect a Bitbucket workspace from the admin settings page, with encrypted storage of access and refresh tokens at rest

## [0.15.3] - 2026-08-25

### Bug Fixes

- **Webhook signature header fallback** — webhook handler now accepts both `X-Hub-Signature` and `X-Hub-Signature-256` headers, improving compatibility with different Bitbucket webhook configurations
