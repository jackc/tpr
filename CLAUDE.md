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
   - Legacy React 0.14.7 (pre-hooks)
   - React Router v2
   - Component-based architecture

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
- Logging via log15: `env.logger.Error("message", "key", value)`

### Frontend (React)

#### Component Structure

```jsx
var ComponentName = React.createClass({
  getInitialState: function() {
    return {/* initial state */};
  },
  componentDidMount: function() {
    // Setup
  },
  render: function() {
    return <div>...</div>;
  }
});
```

Note: This is **React 0.14.7** - no hooks, no functional components with state.

#### Routing

Uses React Router v2:
```jsx
<Router history={browserHistory}>
  <Route path="/" component={App}>
    <IndexRoute component={HomePage} />
    <Route path="login" component={LoginPage} />
  </Route>
</Router>
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

2. Use React.createClass pattern:
   ```jsx
   var ComponentName = React.createClass({
     render: function() {
       return <div>Component content</div>;
     }
   });

   export default ComponentName;
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
- `tpr.conf` - Main configuration (gitignored)
- `tpr.example.conf` - Example configuration
- `tpr.test.conf` - Test database configuration

### Build
- `Rakefile` - Build tasks (build, test, deploy)
- `package.json` - Frontend dependencies and scripts
- `go.mod` - Go dependencies

### Development
- `.devcontainer/` - VS Code devcontainer configuration
- `.vscode/tasks.json` - VS Code tasks for auto-start
- `frontend/vite.config.js` - Vite configuration

## Gotchas and Notes

### Backend

1. **Old React Version**: Frontend uses React 0.14.7 (2016). No hooks, fragments, or modern features.

2. **Session Authentication**: Uses custom `X-Authentication` header, not cookies. Frontend must send this header.

3. **pgx Context**: Always pass `context.Context` as first parameter to database functions.

4. **Error Responses**: Must manually set HTTP status codes. No automatic error handling.

5. **Feed Updater**: Runs in background goroutine. Be careful with shared state.

6. **SQL Injection**: Uses pgx's prepared statements. Always use `$1, $2` placeholders.

### Frontend

1. **No JSX Transform**: Using old JSX syntax. May need pragma comments.

2. **React Router v2**: Old API. Use `browserHistory`, not `BrowserRouter`.

3. **No Module Bundler State**: Vite is added later. Some old patterns may exist.

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

1. **Build**: `rake build` creates production assets
2. **Deploy**: `bin/deploy production` deploys to configured host
3. **Config**: Set `PRODUCTION_HOST` in `.mise.local.toml`
4. **Systemd**: Sample service file in `config/systemd/tpr.service`
5. **Nginx**: Sample reverse proxy config in `config/nginx/tpr`

## Resources

- [pgx Documentation](https://github.com/jackc/pgx)
- [chi Router](https://github.com/go-chi/chi)
- [React 0.14 Docs](https://legacy.reactjs.org/docs/react-api.html)
- [Tern Migrations](https://github.com/jackc/tern)
- [Playwright](https://playwright.dev)

## Code Style

### Go
- Follow standard Go conventions
- Use `gofmt`
- Error handling: explicit checks, no panic in library code
- Logging: structured logging with log15
- Comments: godoc-style for exported functions

### JavaScript/React
- Use semicolons
- Single quotes for strings
- 2-space indentation
- Follow existing patterns (pre-ES6 style due to old React)

## Contributing Guidelines

When modifying this codebase:

1. **Maintain compatibility**: Keep using React 0.14.7 patterns unless upgrading
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
- Gzip/zopfli compression on static assets
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
