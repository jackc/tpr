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
- **Language**: Go 1.25
- **Web Framework**: [chi v5](https://github.com/go-chi/chi) for HTTP routing
- **Database**: PostgreSQL with [pgx v5](https://github.com/jackc/pgx) driver
- **Migrations**: [tern](https://github.com/jackc/tern) for database migrations
- **Logging**: log15
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

The application follows a traditional client-server architecture:

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

### Prerequisites

- Go 1.25+
- Node.js 24+
- PostgreSQL 14+
- Ruby 4+ (for Rake build tasks)

### Setup

The preferred development environment is the provided devcontainer, which includes all dependencies pre-configured.

The following two servers need to run in development:

- `rake rerun` - run Go server on port 4000, automatically rebuilds on change.
- `npm run dev` - run Vite dev server on port 5173, proxies `/api` to Go server.

If using VS Code, they automatically run via VS Code tasks. When you first open the devcontainer, VS Code tasks will
auto-start but may fail. Manually restart them:
- Restart the "Go HTTP Server" task
- Restart the "Vite Dev" task

### Running Tests

Run all tests (backend + E2E):
```bash
rake
```

Run backend tests only:
```bash
rake test:backend
```

The rake task automatically sets up test databases before running Go tests.

Run E2E tests:
```bash
npm run test:e2e          # Headless mode
npm run test:e2e:headed   # With browser
npm run test:e2e:ui       # Interactive UI mode
```

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
├── svelte.config.js     # SvelteKit configuration
├── vite.config.js       # Vite configuration
└── Rakefile             # Build tasks
```

## Building for Production

Build a deployment tarball:
```bash
rake build/linux_amd64.tar.gz
```

This creates a tarball containing:
- `tpr` - Linux AMD64 binary
- `assets/` - Frontend static assets
- `VERSION` - Git commit hash

Other targets: `linux_arm64`, `darwin_amd64`, `darwin_arm64`.

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
   rake build/linux_amd64.tar.gz && verna app deploy build/linux_amd64.tar.gz
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
