# Changelog

All notable changes to BitIssues are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/) conventions.

## [Unreleased]

### New Features

- **Bitbucket OAuth connection** — connect, view status, and disconnect a Bitbucket workspace from the admin settings page, with encrypted storage of access and refresh tokens at rest

## [0.15.3] - 2026-08-25

### Bug Fixes

- **Webhook signature header fallback** — webhook handler now accepts both `X-Hub-Signature` and `X-Hub-Signature-256` headers, improving compatibility with different Bitbucket webhook configurations
