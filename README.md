# The Pithy Reader

A lightweight, self-hosted RSS/Atom feed aggregator written in Go with a SvelteKit frontend.

## Features

- Subscribe to RSS and Atom feeds
- Automatic feed updates in the background
- Mark items as read/unread
- Feed management with OPML import/export support
- User authentication and session management
- Password reset via email (SMTP)
- Keyboard-driven interface for efficient navigation

## Keyboard Shortcuts

- `j` - Select next item
- `k` - Select previous item
- `v` - Open item in new tab
- `shift+a` - Mark all items as read (home page)

## Tech Stack

### Backend
- **Language**: Go 1.27.4
- **Application foundation**: Manifest typed actions and HTTP registry (backed by chi)
- **Database**: PostgreSQL with [pgx v5](https://github.com/jackc/pgx) driver
- **Migrations**: [tern](https://github.com/jackc/tern) for database migrations
- **Logging**: `log/slog`
- **Email**: SMTP for password reset

### Frontend
- **Framework**: [Svelte 5](https://svelte.dev/) with [SvelteKit](https://svelte.dev/docs/kit)
- **Routing**: SvelteKit file-based routing
- **Build Tool**: [Vite 8](https://vite.dev/)
- **Styling**: Sass
- **Static Adapter**: [@sveltejs/adapter-static](https://svelte.dev/docs/kit/adapter-static)

### Testing
- **Backend**: Go standard testing + [testify](https://github.com/stretchr/testify)
- **E2E**: [Playwright](https://playwright.dev/)

## Architecture

Application operations live in `backend/core`, using the same Manifest action
catalog and HTTP registry foundation as Logger4Life and FAM:

- `backend/core`: typed parameters/results, business rules, and a shared dispatcher.
  Action metadata declares public access and mutations; the dispatcher requires a
  user for every private action, including calls made without HTTP. Mutation
  metadata is descriptive; it does not introduce an audit log or transaction layer.
- `backend/handlers.go`: HTTP decoding, error/status mapping, JSON and OPML output.
  The existing URLs and lenient `encoding/json` request decoding remain in place.
  Direct `Core.InvokeJSON` calls use Manifest's strict decoder.
- `backend/http_registry.go`: the serving route declarations and named middleware
  stages, including session loading, authentication, request IDs and logging.
  Test routes have an explicit `test-endpoints` condition.
- `backend/data`: PostgreSQL queries shared by the core and feed updater.

Inspect the declarations without starting a server or connecting to PostgreSQL:

```sh
go run . inventory actions
go run . inventory http
TEST_ENDPOINTS=true go run . inventory http
```

The HTTP inventory describes custom adapters and explicitly reports that their
internal action calls are not inspected. TPR has no MCP transport; this migration
does not introduce one. Session revocation is now scoped to the authenticated
reader's sessions.

Runtime components:

- **HTTP API** (`/api/*`) handles all client requests
- **Session-based authentication** using PostgreSQL-backed sessions
- **Background feed updater** runs continuously to fetch new feed items
- **Vite dev server** proxies `/api` requests to the Go backend in development
- **Caddy** reverse proxy serves the app in production

### Database Schema

Key tables:
- `users` - User accounts with bcrypt-hashed passwords
- `feeds` - RSS/Atom feed sources
- `subscriptions` - User feed subscriptions
- `items` - Feed items/articles
- `unread_items` - Tracks which items users haven't read
- `sessions` - User authentication sessions
- `password_resets` - Password reset tokens

## Development

### Host setup

On native macOS (with Homebrew installed) or Ubuntu, run once per machine as your
development user (the script requests sudo for system packages when needed):

```sh
scripts/setup-host
```

This installs PostgreSQL 18 server/client binaries and mise if missing. On Ubuntu it
also installs Chromium's system libraries. It does not start a machine-wide database
or install this checkout's project dependencies. Existing system databases are left alone.
Go, Node.js, process-compose, port-tamer, tern, and verna are managed by `.mise.toml`.
If mise was newly installed, add `$HOME/.local/bin` to your shell's `PATH` before
running the commands below.

### Initialize a checkout or worktree

Run these commands in each new checkout, including each `git worktree`:

```sh
mise trust
mise install
mise run dev:init
mise run dev
```

`dev:init` allocates stable ports, installs npm dependencies and Playwright Chromium,
and creates `postgresql/tern.conf` from its example if missing. Run it again after
pulling dependency changes. Do not copy `.port-tamer.env`, `.dev/`, or `node_modules/`
from another worktree.

`dev` starts PostgreSQL, creates and migrates the development/test databases, then
starts the Go backend and Vite under process-compose. The backend restarts when Go
sources change; Vite handles frontend updates. `mise run dev` is the service launcher;
tests and database tasks require the running stack. The earlier `run` and `rerun`
task names are aliases for `dev`.

Each worktree gets its own port group and PostgreSQL cluster under
`.dev/<os>-<architecture>/postgres/data`. The cluster is initialized on first startup
and persists across stops. Local connections use TCP on `127.0.0.1` with trust
authentication; the application connects as the `tpr` role and administration uses
`postgres`. These settings are for local development only.

For a detached stack (useful for agents and automation):

```sh
mise run dev -- -D
mise run dev:wait
mise run test
mise run dev:down
```

| Command | Purpose |
|---------|---------|
| `mise run dev:urls` | Show this worktree's frontend, backend, database, supervisor, and test ports |
| `mise run dev:browser` | Open the frontend in your browser |
| `mise run dev:wait` | Wait for startup; report failed services and their logs |
| `mise run dev:down` | Stop this worktree's stack, keeping its database files |
| `mise run db:psql` | Open a PostgreSQL session in the development database |
| `mise run db:migrate` | Apply development database migrations |
| `mise run db:reset` | Drop and rebuild development/test databases; invalidate test preparation |
| `mise run dev:ports:reset` | Allocate a new port group after stopping the stack |

Use `mise exec -- scripts/dev-exec process-compose process list` to inspect services,
and `mise exec -- scripts/dev-exec process-compose process logs backend` for backend
logs. Logs are also saved in `.dev/logs/`. Database repair commands require only
PostgreSQL to be healthy, so they remain usable if migrations prevent the rest of
the stack from starting. To discard the entire local cluster, stop the stack first,
then remove its `.dev/<os>-<architecture>/postgres` directory.

### Devcontainer

The container installs PostgreSQL binaries and mise, then runs the same `dev:init`
workflow. It has no separate database service. VS Code's **Development Stack** task
starts process-compose; if it runs before initial setup finishes, restart that task.
Ports vary by worktree, so VS Code forwards detected listeners instead of fixed ports.
Use `mise run dev:urls` to find them. Native and container environments use separate
cluster directories when their operating system or architecture differs.

### Running tests

Start the stack first, then run:

```sh
mise run test                 # Backend followed by browser tests
mise run test:backend         # Go tests
mise run test:browser         # Headless Playwright tests
mise run test:browser:headed  # Visible browser
mise run test:browser:ui      # Interactive Playwright UI
mise run test:browser:report  # View the HTML report on the assigned report port
```

Test preparation builds `tpr_test` and its parallel copies in this worktree's cluster.
Backend tests use `TEST_DATABASE_URL`; browser tests start their own backend and Vite
on reserved test ports, separate from the development servers. Test preparation is
repeated when SQL, database configuration, or allocated ports change.

### Project Structure

```
tpr-a/
├── backend/             # Go backend code
│   ├── data/           # Database models and queries
│   ├── http.go         # HTTP server and routing
│   ├── domain.go       # Domain types
│   └── feed_updater.go # Background feed fetcher
├── src/                 # SvelteKit frontend
│   ├── routes/         # SvelteKit file-based routes
│   ├── lib/            # Shared components, models, and utilities
│   └── styles/         # Sass stylesheets
├── postgresql/          # Database migrations
│   └── migrations/
├── e2e/                 # Playwright E2E tests
├── test/                # Test utilities and fixtures
├── deploy/              # Verna deployment configuration
├── main.go              # Application entry point
├── vite.config.js       # Vite configuration
└── .mise.toml           # Tool versions, environment, and tasks
```

## Building for Production

Build a deployment tarball:
```bash
mise run build:linux-amd64
```

This creates a tarball containing:
- `tpr` - Linux AMD64 binary
- `assets/` - Frontend static assets
- `VERSION` - Git commit hash

Other release tasks: `build:linux-arm64`, `build:darwin-amd64`, `build:darwin-arm64`.

Use `mise run build` for a local binary and assets, `mise run clean` to remove build artifacts,
and `mise tasks` to list all available tasks.

## Deployment

Deployment uses [verna](https://github.com/jackc/verna) with Caddy as the reverse proxy.

1. Configure verna in `.mise.local.toml`:
   ```toml
   [env]
   VERNA_SSH_HOST = "your.server.com"
   VERNA_APP = "tpr"
   ```

2. Build and deploy:
   ```bash
   mise run build:linux-amd64 && verna app deploy build/linux_amd64.tar.gz
   ```

The Caddy reverse proxy template is in `deploy/caddy-handle-template.json`.

## CLI Usage

### Start the server

```bash
# Using environment variables
tpr server

# Using CLI flags
tpr server --database-url postgres://user:pass@localhost:5432/tpr --bind-address 127.0.0.1 --port 8080
```

### Reset a user's password

```bash
tpr reset-password --database-url postgres://user:pass@localhost:5432/tpr <username>
```

Generates a random password and updates the user account. Use `--password` to set a specific password.

## Configuration

Configuration is via CLI flags or environment variables. CLI flags take precedence over environment variables.

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DATABASE_URL` | PostgreSQL connection URL | *required* |
| `BIND_ADDRESS` | Listen address | `127.0.0.1` |
| `PORT` | Listen port | `8080` |
| `LOG_LEVEL` | Log level (debug, info, warn, error, crit, none) | `warn` |
| `PGX_LOG_LEVEL` | PGX driver log level | `warn` |
| `SMTP_SERVER` | SMTP server hostname | |
| `SMTP_PORT` | SMTP port | `587` |
| `SMTP_USERNAME` | SMTP username | |
| `SMTP_PASSWORD` | SMTP password | |
| `SMTP_FROM` | From email address | |
| `ROOT_URL` | Root URL for email links | |

### Development Configuration

For development, environment variables are set in `.mise.toml`. Create `.mise.local.toml` for local overrides.

## License

MIT
