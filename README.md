# The Pithy Reader

A lightweight, self-hosted RSS/Atom feed aggregator written in Go with a React frontend.

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
- `shift+a` - Mark all items as read

## Tech Stack

### Backend
- **Language**: Go 1.25
- **Web Framework**: [chi v5](https://github.com/go-chi/chi) for HTTP routing
- **Database**: PostgreSQL with [pgx v5](https://github.com/jackc/pgx) driver
- **Migrations**: [tern](https://github.com/jackc/tern) for database migrations
- **Logging**: log15
- **Email**: SMTP for password reset

### Frontend
- **Framework**: React 0.14.7
- **Router**: React Router v2
- **Build Tool**: Vite
- **Styling**: Sass

### Testing
- **Backend**: Go standard testing + [testify](https://github.com/stretchr/testify)
- **E2E**: [Playwright](https://playwright.dev/)

## Architecture

The application follows a traditional client-server architecture:

- **HTTP API** (`/api/*`) handles all client requests
- **Session-based authentication** using PostgreSQL-backed sessions
- **Background feed updater** runs continuously to fetch new feed items
- **Static asset serving** via Vite dev server (development) or reverse proxy (production)

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
- Node.js 18+
- PostgreSQL 14+
- Ruby (for Rake build tasks)

### Setup

The preferred development environment is the provided devcontainer, which includes all dependencies pre-configured.

The following two servers need to run in development to use the system.

- `rake rerun` - run Go server, automatically rebuilds on change.
- `npx vite serve frontend` - run Vite server for asset building.

If using VS Code, they automatically run via VS Code tasks. When you first open the devcontainer, VS Code tasks will
auto-start but may fail. Manually restart them:
- Restart the "Go HTTP Server" task
- Restart the "Vite Dev Server" task

### Running Tests

Run all backend tests:
```bash
rake
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
├── backend/           # Go backend code
│   ├── data/         # Database models and queries
│   ├── http.go       # HTTP server and routing
│   ├── domain.go     # Domain types
│   └── feed_updater.go  # Background feed fetcher
├── frontend/         # React frontend
│   └── src/
│       ├── components/  # React components
│       └── index.jsx    # Entry point
├── postgresql/       # Database migrations
│   └── migrations/
├── e2e/             # Playwright E2E tests
├── test/            # Test utilities and fixtures
├── config/          # systemd and nginx configs
├── main.go          # Application entry point
└── Rakefile         # Build tasks
```

## Building for Production

Build all assets and binaries:
```bash
rake build
```

This creates:
- `build/assets/` - Minified and compressed frontend assets
- `build/tpr` - Native binary
- `build/tpr-linux` - Linux AMD64 binary

## Deployment

1. Configure production host in `.mise.local.toml`:
   ```toml
   [env]
   PRODUCTION_HOST = "your.server.com"
   ```

2. Deploy:
   ```bash
   bin/deploy production
   ```

### Server Configuration

Sample systemd service and nginx configurations are in the `config/` directory.

## CLI Usage

### Start the server

```bash
tpr server --address 127.0.0.1 --port 8080 --config tpr.conf
```

### Reset a user's password

```bash
tpr reset-password <username>
```

Generates a random password and updates the user account.

## Configuration

Configuration is stored in INI format (default: `tpr.conf`):

```ini
[database]
host = localhost
port = 5432
database = tpr
user = tpr
password = secret

[server]
address = 127.0.0.1
port = 8080

[log]
level = info
pgx_level = warn

[mail]
smtp_server = smtp.example.com
port = 587
from_address = noreply@example.com
username = user
password = pass
root_url = https://example.com
```

## License

MIT
