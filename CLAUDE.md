# CLAUDE.md - AI Assistant Context

This document provides context for AI assistants (like Claude) working with The Pithy Reader codebase.

## Project Overview

**The Pithy Reader** is a self-hosted RSS/Atom feed aggregator built with Go and React. It allows users to subscribe to feeds, automatically fetches updates, and provides a keyboard-driven interface for reading articles.

**Version**: 0.8.1
**Author**: Jack Christensen
**License**: Copyright Jack Christensen

## Architecture

### High-Level Design

```
┌─────────────┐     HTTP/JSON     ┌──────────────┐
│   React     │ ←──────────────→  │   Go HTTP    │
│   Frontend  │      /api/*       │   Server     │
│   (Vite)    │                   │   (chi)      │
└─────────────┘                   └──────┬───────┘
                                         │
                      ┌──────────────────┼──────────────────┐
                      ▼                  ▼                  ▼
                ┌───────────┐     ┌─────────────┐   ┌──────────┐
                │ PostgreSQL│     │ Feed Updater│   │  SMTP    │
                │  Database │     │ (Background)│   │  Mailer  │
                └───────────┘     └─────────────┘   └──────────┘
```

### Key Components

1. **main.go**: Application entry point with CLI commands
   - `server`: Starts HTTP server
   - `reset-password`: Resets user password

2. **backend/http.go**: HTTP server setup and routing
   - Chi router handling `/api/*` endpoints
   - Environment-based handler pattern for dependency injection
   - Session-based authentication

3. **backend/domain.go**: Core business logic and domain types

4. **backend/feed_updater.go**: Background service
   - Continuously polls feeds for new items
   - Runs in separate goroutine

5. **backend/data/**: Database layer
   - Uses pgx v5 for PostgreSQL
   - Each domain entity has its own file (user.go, feed.go, etc.)
   - Raw SQL queries (no ORM)

6. **frontend/src/**: React application
   - React 18.2.0 with modern hooks
   - React Router v6
   - Functional component architecture with hooks

## Code Patterns and Conventions

### Backend (Go)

#### Handler Pattern

```go
type EnvHandlerFunc func(w http.ResponseWriter, req *http.Request, env *environment)

// Handlers receive environment with user, pool, mailer, logger
func someHandler(w http.ResponseWriter, req *http.Request, env *environment) {
    // env.user - Current authenticated user (may be nil)
    // env.pool - Database connection pool
    // env.mailer - Email sender
    // env.logger - Structured logger
}
```

#### Authentication

- Session-based using `X-Authentication` header
- Sessions stored in PostgreSQL `sessions` table
- `AuthenticatedHandler()` wrapper enforces authentication
- `getUserFromSession()` extracts user from session

#### Database Access

- Direct SQL with pgx
- Context-based queries: `ctx context.Context` first parameter
- Named result types: `data.User`, `data.Feed`, etc.
- Transactions via `pgx.Tx`

Example:
```go
user, err := data.SelectUserByName(ctx, pool, username)
if err != nil {
    return err
}
```

#### Error Handling

- Standard Go error handling
- HTTP status codes set manually
- Logging via `log/slog`: `env.logger.Error("message", "key", value)`

### Frontend (React)

#### Component Structure

```jsx
import React, { useState, useEffect } from 'react'

function ComponentName() {
  const [state, setState] = useState(/* initial state */)

  useEffect(() => {
    // Setup
    return () => {
      // Cleanup
    }
  }, [])

  return <div>...</div>
}
```

Note: This is **React 18.2.0** - uses modern functional components with hooks.

#### Routing

Uses React Router v6:
```jsx
<HashRouter>
  <Routes>
    <Route path="/" element={<App />}>
      <Route index element={<HomePage />} />
      <Route path="login" element={<LoginPage />} />
    </Route>
  </Routes>
</HashRouter>
```

#### State Management

- Component local state via `this.state`
- Signal library for cross-component communication
- No Redux or modern state management

## Database Schema

### Core Tables

- **users**: User accounts
  - `id`, `name`, `password_digest`, `password_salt`, `email`

- **feeds**: RSS/Atom sources
  - `id`, `name`, `url`, `last_fetch_time`, `etag`, `last_modified`

- **subscriptions**: Links users to feeds
  - `id`, `user_id`, `feed_id`, `name`

- **items**: Feed articles
  - `id`, `feed_id`, `url`, `title`, `publication_time`, `content`

- **unread_items**: Tracks unread articles per user
  - `user_id`, `feed_id`, `item_id`

- **sessions**: Authentication sessions
  - `id`, `user_id`, `created_at`

- **password_resets**: Password reset tokens
  - `id`, `user_id`, `token`, `created_at`

### Migrations

- Located in `postgresql/migrations/`
- Managed with [tern](https://github.com/jackc/tern)
- Numbered sequentially: `001_`, `002_`, etc.
- Run with: `cd postgresql && tern migrate`

## Common Tasks

### Adding a New API Endpoint

1. Add route in `backend/http.go` (in `NewAPIHandler`):
   ```go
   r.Get("/new-endpoint", EnvHandler(pool, mailer, logger, newEndpointHandler))
   ```

2. Create handler function:
   ```go
   func newEndpointHandler(w http.ResponseWriter, req *http.Request, env *environment) {
       // Implementation
   }
   ```

3. Add authentication if needed:
   ```go
   r.Get("/protected", EnvHandler(pool, mailer, logger, AuthenticatedHandler(protectedHandler)))
   ```

### Adding a Database Migration

1. Create new file: `postgresql/migrations/XXX_description.sql`
   - Increment number from last migration
   - Use descriptive name

2. Write SQL:
   ```sql
   -- Add up migration
   CREATE TABLE new_table (
     id serial PRIMARY KEY,
     name text NOT NULL
   );

   ---- create above / drop below ----

   -- Add down migration
   DROP TABLE new_table;
   ```

3. Run migration:
   ```bash
   cd postgresql
   tern migrate
   ```

### Adding a React Component

1. Create file: `frontend/src/components/ComponentName.jsx`

2. Use functional component with hooks:
   ```jsx
   import React, { useState, useEffect } from 'react'

   export default function ComponentName() {
     const [state, setState] = useState(initialValue)

     useEffect(() => {
       // Side effects
       return () => {
         // Cleanup
       }
     }, [dependencies])

     return <div>Component content</div>
   }
   ```

3. Import where needed:
   ```jsx
   import ComponentName from './components/ComponentName.jsx';
   ```

### Running Tests

Backend:
```bash
rake                    # Run all Go tests
go test ./backend/...  # Test specific package
```

E2E:
```bash
npm run test:e2e         # Headless
npm run test:e2e:headed  # With browser
npm run test:e2e:ui      # Interactive mode
```

## Important Files

### Configuration
- `.mise.toml` - Development environment variables (TPR_* settings)
- `.mise.local.toml` - Local overrides (gitignored)
- Configuration via CLI flags or environment variables (see CLI Usage below)

### Build
- `Rakefile` - Build tasks (build, test, deploy)
- `package.json` - Frontend dependencies and scripts
- `go.mod` - Go dependencies

### Development
- `.devcontainer/` - VS Code devcontainer configuration
- `.vscode/tasks.json` - VS Code tasks for auto-start
- `frontend/vite.config.js` - Vite configuration

## CLI Usage

### Server Command

```bash
# Using environment variables (set in .mise.toml for development)
go run . server

# Using CLI flags
go run . server --db-host localhost --db-name tpr_dev --db-user tpr --db-password secret
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `BIND_ADDRESS` | Listen address | `127.0.0.1` |
| `PORT` | Listen port | `8080` |
| `DATABASE_URL` | PostgreSQL connection URL | *required* |
| `LOG_LEVEL` | Log level (debug/info/warn/error/crit/none) | `warn` |
| `LOG_FORMAT` | Log format (text/json/journal) | `text` |
| `PGX_LOG_LEVEL` | PGX driver log level | `warn` |
| `SMTP_SERVER` | SMTP server hostname | |
| `SMTP_PORT` | SMTP port | `587` |
| `SMTP_USERNAME` | SMTP username | |
| `SMTP_PASSWORD` | SMTP password | |
| `SMTP_FROM` | From email address | |
| `ROOT_URL` | Root URL for email links | |

CLI flags take the same names with `--` prefix (e.g., `--db-host`, `--log-level`).

## Gotchas and Notes

### Backend

1. **Session Authentication**: Uses custom `X-Authentication` header, not cookies. Frontend must send this header.

2. **pgx Context**: Always pass `context.Context` as first parameter to database functions.

3. **Error Responses**: Must manually set HTTP status codes. No automatic error handling.

4. **Feed Updater**: Runs in background goroutine. Be careful with shared state.

5. **SQL Injection**: Uses pgx's prepared statements. Always use `$1, $2` placeholders.

### Frontend

1. **Modern React 18**: Uses functional components with hooks. Always use useState, useEffect, useCallback, etc.

2. **React Router v6**: Uses `<HashRouter>`, `<Routes>`, and `element` prop. Navigation via `useNavigate()` hook.

3. **Signals Library**: Used for pub/sub between components (WorkingNotice, HomePage, ArchivePage). Always clean up signal listeners in useEffect return functions.

4. **Vite Build Tool**: Modern build tool with Fast Refresh. Run `npm run dev` for development server.

4. **Signals Library**: Used for pub/sub between components. Check before adding new state management.

### Database

1. **Tern Migrations**: Two-way migrations. Write both up and down SQL.

2. **Test Database**: Tests use separate database (`tpr_test`). Set up via `test/setup_test_databases.sql`.

3. **Permissions**: Migrations include permission grants. Don't forget these.

### Development

1. **First Run**: Devcontainer tasks fail on first run. Manually restart them.

2. **Two Servers**: Must run both Go server (8080) and Vite (5173) in development.

3. **Static URL**: Go server proxies to Vite in dev via `--static-url` flag.

4. **Ruby Dependency**: Rake requires Ruby. Bundler manages gems.

## Testing Strategy

### Backend Tests
- Unit tests in `*_test.go` files
- Integration tests use real PostgreSQL (via testdb)
- Test fixtures in `test/testdata/`
- Helper utilities in `test/testutil/`

### E2E Tests
- Playwright tests in `e2e/` directory
- Test against full stack (backend + frontend)
- Use TypeScript for test definitions
- Reports served on `0.0.0.0` for devcontainer access

## Deployment

Deployment uses [verna](https://github.com/jackc/verna) with Caddy as the reverse proxy.

1. **Build**: `rake build/linux_amd64.tar.gz` creates a deployment tarball
2. **Deploy**: `verna app deploy build/linux_amd64.tar.gz`
3. **Config**: Set `VERNA_SSH_HOST` and `VERNA_APP` in `.mise.local.toml`
4. **Caddy**: Reverse proxy template in `deploy/caddy-handle-template.json`

## Resources

- [pgx Documentation](https://github.com/jackc/pgx)
- [chi Router](https://github.com/go-chi/chi)
- [React 18 Docs](https://react.dev/)
- [React Router v6 Docs](https://reactrouter.com/)
- [Tern Migrations](https://github.com/jackc/tern)
- [Playwright](https://playwright.dev)

## Code Style

### Go
- Follow standard Go conventions
- Use `gofmt`
- Error handling: explicit checks, no panic in library code
- Logging: structured logging with `log/slog`. Supports `text`, `json`, and `journal` (systemd journald) output formats via `LOG_FORMAT`
- Comments: godoc-style for exported functions

### JavaScript/React
- Use semicolons
- Single quotes for strings
- 2-space indentation
- Follow existing patterns (pre-ES6 style due to old React)

## Contributing Guidelines

When modifying this codebase:

1. **Use modern patterns**: Use functional components with hooks (useState, useEffect, useCallback, etc.)
2. **Write tests**: Add tests for new backend functionality
3. **Migrations**: Always include both up and down migrations
4. **Security**: Validate input, use prepared statements, hash passwords
5. **Logging**: Add appropriate logging for debugging
6. **Documentation**: Update README.md and this file for significant changes

## Questions to Ask

When working on new features, consider:

1. Does this need authentication?
2. Should this be a new database table or extend existing?
3. Will this affect the feed updater background process?
4. Do we need email notifications?
5. What keyboard shortcuts make sense?
6. How does this work with existing subscriptions?
7. What happens on error/failure?
8. Is this testable with the current test setup?

## Performance Considerations

- Database connection pooling via pgxpool (max 10 connections)
- Feed updates run continuously in background
- Frontend is mostly static after build
- Caddy serves precompressed static assets
- PostgreSQL indexes on frequently queried columns
- Consider N+1 queries when adding new features

## Security Notes

- Passwords: bcrypt hashed with random salt
- Sessions: Random 32-byte tokens, hex-encoded
- CSRF: Not currently implemented (session tokens in header)
- SQL Injection: Prevented via prepared statements
- XSS: React auto-escapes by default
- Email: SMTP with optional TLS
- Input Validation: Check in handlers before database calls

## Future Improvements

Potential areas for enhancement:

- Upgrade React to modern version
- Add real-time updates (WebSocket/SSE)
- Implement full-text search
- Add feed categories/tags
- Mobile-responsive design
- API rate limiting
- CSRF protection
- Read/unread sync between devices
- Import/export user data
- Feed favicon fetching
- Article content extraction/readability

---

Last updated: 2026-02-01
