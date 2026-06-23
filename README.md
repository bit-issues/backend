<a id="readme-top"></a>

<!--
*** BitIssues Backend — Lightweight task tracker replacing Bitbucket Issues
*** Built with Go, Fiber, and ❤️
-->

<!-- PROJECT SHIELDS -->
[![Go][Go-shield]][Go-url]
[![License][License-shield]][License-url]
[![Go Report Card][Report-shield]][Report-url]
[![GitHub Release][Release-shield]][Release-url]
[![GHCR][GHCR-shield]][GHCR-url]
[![LinkedIn][LinkedIn-shield]][LinkedIn-url]

<!-- PROJECT LOGO -->
<br />
<div align="center">
  <a href="https://github.com/bit-issues/backend">
    <img src="web/assets/logo.svg" alt="Logo" width="80" height="80">
  </a>

  <h3 align="center">BitIssues Backend</h3>

  <p align="center">
    A lightweight, self-hosted task tracker replacing Bitbucket Issues
    <br />
    <a href="https://github.com/bit-issues/backend"><strong>Explore the docs »</strong></a>
    <br />
    <br />
    <a href="#getting-started">Get Started</a>
    &middot;
    <a href="https://github.com/bit-issues/backend/issues/new?labels=bug">Report Bug</a>
    &middot;
    <a href="https://github.com/bit-issues/backend/issues/new?labels=enhancement">Request Feature</a>
  </p>
</div>

<!-- TABLE OF CONTENTS -->
## Table of Contents
- [Table of Contents](#table-of-contents)
- [About The Project](#about-the-project)
  - [Features](#features)
  - [Architecture](#architecture)
  - [Built With](#built-with)
- [Getting Started](#getting-started)
  - [Download](#download)
  - [Prerequisites](#prerequisites)
  - [Build from Source](#build-from-source)
- [Usage](#usage)
  - [CLI Commands](#cli-commands)
  - [API Overview](#api-overview)
  - [Configuration](#configuration)
  - [Development Commands](#development-commands)
  - [Deployment](#deployment)
- [Roadmap](#roadmap)
- [Contributing](#contributing)
- [License](#license)
- [Contact](#contact)
- [Acknowledgments](#acknowledgments)


<!-- ABOUT THE PROJECT -->
## About The Project

[Bitbucket Issues](https://community.atlassian.com/forums/Bitbucket-articles/Announcing-sunset-of-Bitbucket-Issues-and-Wikis/ba-p/3193882) sunsets in August 2026. BitIssues is a self-hosted replacement for teams who need simple task tracking without the overhead of full-featured tools like Jira.

Here's why BitIssues exists:

- **Drop-in migration** — import your existing Bitbucket Issues JSON export with a single CLI command
- **Lightweight** — single Go binary with an embedded SPA frontend, no heavy infrastructure
- **Self-hosted** — full control over your data, deploy anywhere (bare metal, Docker, Kubernetes)

<p align="right">(<a href="#readme-top">back to top</a>)</p>

### Features

- **Projects** — create and manage projects with auto-generated slug-based routing
- **Tasks** — full lifecycle with priority (trivial → blocker), status (new → closed), kind (bug, enhancement, task, proposal), assignees, and due dates
- **Comments** — per-task discussion with markdown support
- **File Attachments** — two-phase upload via S3 presigned URLs (client uploads directly to storage, bypassing the server)
- **JWT Authentication** — HS256 tokens with Argon2id password hashing
- **WebAuthn/Passkey Authentication** — passwordless login via platform authenticators (fingerprint, Face ID, security keys)
- **Role-Based Access** — admin and user roles with pending activation flow
- **Rich Filtering & Sorting** — tasks filterable by project, author, assignee, status, priority, date range
- **Dashboard Queries** — quick access to tasks assigned to or created by the current user
- **Swagger/OpenAPI** — auto-generated API docs at `/api/v1/docs`

<p align="right">(<a href="#readme-top">back to top</a>)</p>

### Architecture

The project follows a clean/layered architecture organized as Uber Fx modules:

```
main.go
  └── app.go
        ├── serve command       # HTTP server
        │     └── Fx container
        │           ├── Core (logger, db, storage, validator, health)
        │           ├── Server (handlers, middlewares, routes, swagger)
        │           └── Domains (users, projects, tasks, comments, attachments, webauthn)
        └── import command      # Bitbucket import CLI
```

Each domain module follows the same layout:

- `domain.go` — entities, value objects, enums, validation
- `models.go` — ORM database models with domain converters
- `repository.go` — data access layer
- `service.go` — business logic
- `module.go` — Fx provider wiring
- `errors.go` — domain-specific errors

The HTTP layer lives in `internal/server/` with separate handler/DTO files per domain and a shared JWT middleware for auth enforcement.

**Attachment upload flow:**

```
Client                  Server                  S3 Storage
  │                       │                        │
  │── POST /attachments ──▶  Create pending record │
  │                       │── Generate presigned   │
  │                       │   PUT URL              │
  │◀── Presigned URL ─────│                        │
  │── PUT file ───────────┼──────────────────────▶│
  │── PUT /confirm ──────▶│  Set status=uploaded   │
  │                       │── Delete from S3       │
  │── DELETE ────────────▶│  Soft-delete record    │
```

<p align="right">(<a href="#readme-top">back to top</a>)</p>

### Built With

* [![Go][Go-badge]][Go-url]
* [![Fiber][Fiber-badge]][Fiber-url]
* [![MySQL][MySQL-badge]][MySQL-url]
* [![Swagger][Swagger-badge]][Swagger-url]
* [![Svelte][Svelte-badge]][Svelte-url]
* [![Tailwind CSS][Tailwind-badge]][Tailwind-url]
* [![Docker][Docker-badge]][Docker-url]
* [![Prometheus][Prometheus-badge]][Prometheus-url]

<p align="right">(<a href="#readme-top">back to top</a>)</p>

<!-- GETTING STARTED -->
## Getting Started

### Download

Get the latest pre-built binary or Docker image:

**Option 1 — Binary (GitHub Releases)**
```sh
# Download the latest release for your platform
curl -LO https://github.com/bit-issues/backend/releases/latest/download/backend_Linux_x86_64.tar.gz
tar xzf backend_Linux_x86_64.tar.gz
./backend serve
```

Or download manually from the [Releases page](https://github.com/bit-issues/backend/releases). Binaries are available for Linux, macOS, and Windows.

**Option 2 — Docker (GHCR)**
```sh
docker pull ghcr.io/bit-issues/backend:latest

docker run -d \
  -p 3000:3000 \
  -e DATABASE__URL="mariadb://..." \
  -e STORAGE__URL="s3://..." \
  ghcr.io/bit-issues/backend:latest
```

### Prerequisites

* MySQL 8+ or MariaDB 10.5+
* S3-compatible storage (use [MinIO](https://min.io) for local development)

### Build from Source

For development or custom builds:

1. Clone the repo
   ```sh
   git clone https://github.com/bit-issues/backend.git
   cd backend
   ```
2. Configure environment
   ```sh
   cp .env.example .env
   # Edit .env with your database and S3 credentials
   ```
3. Install Go dependencies
   ```sh
   make deps
   ```
4. Generate Swagger documentation
   ```sh
   make gen
   ```
5. Start the development server
   ```sh
   make air
   ```
   Database migrations run automatically on startup.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

<!-- USAGE -->
## Usage

### CLI Commands

```sh
# Start HTTP server (default)
backend serve

# Import Bitbucket Issues JSON export
backend import --project=<slug> --file=<path> --default-user=<id> [--dry-run]

# Show version
backend --version
```

The `import` command maps Bitbucket issues and comments into the BitIssues schema. Use `--dry-run` to preview without writing.

### API Overview

All endpoints are under `/api/v1`. Full documentation is available via Swagger UI at `/api/v1/docs`.

| Area        | Base Path                    | Auth                          |
| ----------- | ---------------------------- | ----------------------------- |
| Auth        | `/auth`                      | Public (login, register)      |
| Users       | `/users`                     | JWT (admin for management)    |
| Projects    | `/projects`                  | JWT                           |
| Tasks       | `/tasks`                     | JWT                           |
| Comments    | `/tasks/:taskId/comments`    | JWT                           |
| Attachments | `/tasks/:taskId/attachments` | JWT                           |
| Passkeys    | `/auth/passkey`              | Public (login) / JWT (manage) |
| Docs        | `/docs`                      | Public                        |

A complete API reference with request/response examples is available in [`requests.http`](requests.http).

### Configuration

Configuration is loaded from environment variables with optional YAML override via `CONFIG_PATH`.

| Variable                    | Default                     | Description                            |
| --------------------------- | --------------------------- | -------------------------------------- |
| `DATABASE__URL`             | `mariadb://bit-issues:...`  | Database connection string             |
| `JWT__SECRET`               | `secret`                    | JWT signing key                        |
| `JWT__ACCESS_TTL`           | `15m`                       | Access token lifetime                  |
| `STORAGE__URL`              | `s3://bucket/prefix?...`    | S3 storage URL                         |
| `STORAGE__LINKS_TTL`        | `15m`                       | Presigned URL lifetime                 |
| `ATTACHMENTS__MAX_SIZE`     | `10485760`                  | Max file size in bytes (10 MB)         |
| `HTTP__ADDRESS`             | `127.0.0.1:3000`            | Server listen address                  |
| `AWS_ACCESS_KEY_ID`         | —                           | S3 access key                          |
| `AWS_SECRET_ACCESS_KEY`     | —                           | S3 secret key                          |
| `AWS_REGION`                | —                           | S3 region                              |
| `WEBAUTHN__RP_DISPLAY_NAME` | `BitIssues`                 | Display name shown during registration |
| `WEBAUTHN__RP_ID`           | `localhost`                 | Relying Party ID (domain)              |
| `WEBAUTHN__RP_ORIGINS`      | `["http://localhost:5173"]` | Allowed origins JSON array             |

### Development Commands

| Command          | Description                               |
| ---------------- | ----------------------------------------- |
| `make fmt`       | Format code via golangci-lint             |
| `make lint`      | Run linter                                |
| `make test`      | Run tests with race detector and coverage |
| `make coverage`  | Generate HTML coverage report             |
| `make benchmark` | Run benchmarks                            |
| `make build`     | Compile binary to `bin/`                  |
| `make air`       | Start dev server with live reload         |
| `make gen`       | Run `go generate` (Swagger docs)          |
| `make deps`      | Download Go module dependencies           |
| `make clean`     | Remove build artifacts                    |

### Deployment

Pre-built multi-arch Docker images are published to **GHCR** (`ghcr.io/bit-issues/backend`) and binaries to **GitHub Releases** on every version tag.

- **Docker** — `docker pull ghcr.io/bit-issues/backend:latest` (linux/amd64, linux/arm64)
- **Binary** — download platform-specific tarballs from the [Releases page](https://github.com/bit-issues/backend/releases)
- **CI/CD** — GitHub Actions: lint + test (push/PR), release + publish (tag `v*`), PR snapshot builds

<p align="right">(<a href="#readme-top">back to top</a>)</p>

<!-- ROADMAP -->
## Roadmap

- [x] Bitbucket Issues JSON import
- [x] JWT authentication with Argon2id
- [x] WebAuthn/passkey authentication
- [x] File attachments via S3 presigned URLs
- [ ] Email notifications
- [ ] Webhook integration
- [ ] Multi-language support
- [ ] Kanban board view

See the [open issues](https://github.com/bit-issues/backend/issues) for a full list of proposed features and known issues.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

<!-- CONTRIBUTING -->
## Contributing

Contributions are what make the open source community such an amazing place to learn, inspire, and create. Any contributions you make are **greatly appreciated**.

If you have a suggestion that would make this better, please fork the repo and create a pull request. You can also simply open an issue with the tag "enhancement".

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit your Changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the Branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

<p align="right">(<a href="#readme-top">back to top</a>)</p>

<!-- LICENSE -->
## License

Distributed under the Apache License 2.0. See `LICENSE` for more information.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

<!-- CONTACT -->
## Contact

Project Link: [https://github.com/bit-issues/backend](https://github.com/bit-issues/backend)

<p align="right">(<a href="#readme-top">back to top</a>)</p>

<!-- ACKNOWLEDGMENTS -->
## Acknowledgments

* [Fiber](https://gofiber.io)
* [Uber Fx](https://github.com/uber-go/fx)
* [uptrace/bun](https://github.com/uptrace/bun)
* [pressly/goose](https://github.com/pressly/goose)
* [MinIO](https://min.io)
* [Svelte](https://svelte.dev)
* [Tailwind CSS](https://tailwindcss.com)
* [Swaggo](https://github.com/swaggo/swag)
* [GoReleaser](https://goreleaser.com)
* [Best-README-Template](https://github.com/othneildrew/Best-README-Template)

<p align="right">(<a href="#readme-top">back to top</a>)</p>

<!-- MARKDOWN LINKS & IMAGES -->
[Go-shield]: https://img.shields.io/badge/Go-1.25+-00ADD8?style=for-the-badge&logo=go&logoColor=white
[Go-url]: https://go.dev
[License-shield]: https://img.shields.io/badge/License-Apache%202.0-blue?style=for-the-badge&logo=apache
[License-url]: https://github.com/bit-issues/backend/blob/main/LICENSE
[Report-shield]: https://goreportcard.com/badge/github.com/bit-issues/backend?style=for-the-badge
[Report-url]: https://goreportcard.com/report/github.com/bit-issues/backend
[Release-shield]: https://img.shields.io/github/v/release/bit-issues/backend?style=for-the-badge&logo=github
[Release-url]: https://github.com/bit-issues/backend/releases
[GHCR-shield]: https://img.shields.io/badge/GHCR-latest-blue?style=for-the-badge&logo=docker&logoColor=white
[GHCR-url]: https://github.com/bit-issues/backend/pkgs/container/backend
[LinkedIn-shield]: https://img.shields.io/badge/-LinkedIn-black.svg?style=for-the-badge&logo=linkedin&colorB=555
[LinkedIn-url]: https://linkedin.com/in/capcom6

[Go-badge]: https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white
[Fiber-badge]: https://img.shields.io/badge/Fiber-00B4D8?style=for-the-badge&logo=go&logoColor=white
[Fiber-url]: https://gofiber.io
[MySQL-badge]: https://img.shields.io/badge/MySQL-4479A1?style=for-the-badge&logo=mysql&logoColor=white
[MySQL-url]: https://www.mysql.com
[Swagger-badge]: https://img.shields.io/badge/Swagger-85EA2D?style=for-the-badge&logo=swagger&logoColor=black
[Swagger-url]: https://swagger.io
[Svelte-badge]: https://img.shields.io/badge/Svelte-FF3E00?style=for-the-badge&logo=svelte&logoColor=white
[Svelte-url]: https://svelte.dev
[Tailwind-badge]: https://img.shields.io/badge/Tailwind_CSS-06B6D4?style=for-the-badge&logo=tailwindcss&logoColor=white
[Tailwind-url]: https://tailwindcss.com
[Docker-badge]: https://img.shields.io/badge/Docker-2496ED?style=for-the-badge&logo=docker&logoColor=white
[Docker-url]: https://docker.com
[Prometheus-badge]: https://img.shields.io/badge/Prometheus-E6522C?style=for-the-badge&logo=prometheus&logoColor=white
[Prometheus-url]: https://prometheus.io
