# CLAUDE.md - AI Assistant Context

This document provides context for AI assistants (like Claude) working with The Pithy Reader codebase.

## Project Overview

**The Pithy Reader** is a self-hosted RSS/Atom feed aggregator built with Go and Svelte. It allows users to subscribe to feeds, automatically fetches updates, and provides a keyboard-driven interface for reading articles.

**Version**: 0.8.1
**Author**: Jack Christensen
**License**: Copyright Jack Christensen

## Architecture

### High-Level Design

```
┌─────────────┐     HTTP/JSON     ┌──────────────┐
│   Svelte    │ ←──────────────→  │   Go HTTP    │
│   Frontend  │      /api/*       │   Server     │
│  (SvelteKit)│                   │   (chi)      │
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

2. **backend/http_registry.go** and **backend/handlers.go**: HTTP transport
   - Manifest HTTP registry declares every route and its authentication policy
   - Named stages handle session loading, request IDs, logging and authentication
   - Handlers decode requests, call typed core actions, and serialize responses
   - Existing browser URLs and lenient JSON decoding remain supported

3. **backend/core/**: Manifest application action catalog
   - Typed parameters/results and business logic independent of HTTP
   - `Core` owns dependencies and the Manifest dispatcher
   - `ActionMetadata{Public, Mutating}` declares action policies
   - Core middleware requires an identity on all private calls
   - `core.WithUser` supplies a trusted identity; never accept identity from JSON
   - `Mutating` is descriptive, not automatic auditing or transactions
   - `tpr inventory actions|http` prints declarations without database access

4. **backend/feed_updater.go**: Background service
   - Continuously polls feeds for new items
   - Runs in separate goroutine

5. **backend/data/**: Database layer
   - Uses pgx v5 for PostgreSQL
   - Each domain entity has its own file (user.go, feed.go, etc.)
   - Raw SQL queries (no ORM)

6. **src/**: Svelte SPA (Single Page Application)
   - Svelte 5 with SvelteKit 3 (static adapter)
   - SvelteKit file-based routing
   - Svelte stores for state management

## Code Patterns and Conventions

### Backend (Go)

#### Action and Handler Pattern

Define application operations in `backend/core` using `Define(ActionDef[P, R])`.
Pass `context.Context` through database calls. Transport adapters in
`backend/handlers.go` call `Action.Call` via the shared `call` helper, which maps
core errors to HTTP responses. Keep HTTP request/response objects out of core.

#### Authentication

- PostgreSQL sessions arrive through `X-Authentication` or the OPML `session` form value
- The registry compiles private route metadata into `require-session-user`
- The core dispatcher independently enforces private action authentication
- `core.WithUser` is for trusted in-process callers and the session loader
- Session deletion is scoped to the authenticated user's ID

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

### Frontend (Svelte)

#### Component Structure

```svelte
<script>
  import { onMount } from 'svelte';

  let items = [];
  let selected = null;

  onMount(() => {
    // Setup: fetch data, add event listeners
    fetchData();

    return () => {
      // Cleanup
    };
  });
</script>

<div>
  {#each items as item (item.id)}
    <li class:selected={item === selected}>{item.title}</li>
  {/each}
</div>
```

Note: This is **Svelte 5** with **SvelteKit 3** — uses `.svelte` components with reactive `let` declarations and `onMount` lifecycle.

#### Routing

Uses SvelteKit file-based routing with route groups:
```
src/routes/
  +layout.svelte              # Root layout (header, footer)
  +page.svelte                # Root redirect to /home
  (auth)/
    login/+page.svelte        # Login form
    register/+page.svelte     # Registration
    lostPassword/+page.svelte # Password reset request
    resetPassword/+page.svelte
  (protected)/
    +layout.svelte            # Auth guard (redirects if not logged in)
    home/+page.svelte         # Main feed reader
    archive/+page.svelte      # Archived items
    feeds/+page.svelte        # Feed management
    account/+page.svelte      # Account settings
```

Navigation via `goto()` from `$app/navigation`.

#### State Management

- **Svelte stores** (`writable`) for shared state (session, ajax pending indicator)
- **Local `let` variables** for component state (reactive by default in Svelte)
- **Model classes** (`UnreadItems`, `ArchivedItems`, `Item`) with internal `writable` stores for change notification
- Session persisted to `localStorage` via custom store wrapper (`src/lib/session.js`)

#### API Client

`src/lib/api.js` — async/await Fetch API client:
- Auth via `X-Authentication` header (reads session store)
- Tracks pending requests via `ajaxPending` store
- Auto-clears session on 403 responses

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

1. Define the typed operation in `backend/core`, with public/mutation metadata.
2. Add a thin adapter in `backend/handlers.go`; preserve the existing HTTP JSON
   conventions rather than using strict Manifest bindings implicitly.
3. Declare its route in `backend/http_registry.go` with an explicit public policy.
4. Test the core behavior and transport policy. Update inventory assertions when
   adding routes or actions. Test-only routes belong behind the named condition.

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

### Adding a Svelte Component

1. Create file in `src/lib/components/ComponentName.svelte`

2. Use standard Svelte component structure:
   ```svelte
   <script>
     import { onMount } from 'svelte';

     export let propName = 'default';
     let localState = [];

     onMount(() => {
       // Side effects
       return () => {
         // Cleanup
       };
     });
   </script>

   <div>{propName}</div>
   ```

3. Import where needed:
   ```svelte
   <script>
     import ComponentName from '#lib/components/ComponentName.svelte';
   </script>

   <ComponentName propName="value" />
   ```

### Adding a New Page/Route

1. Create directory and page file: `src/routes/(protected)/pagename/+page.svelte`
   - Use `(auth)` group for unauthenticated pages
   - Use `(protected)` group for authenticated pages

2. The route is automatically available at `/pagename`

### Running Tests

Backend:
```bash
mise run dev -- -D      # Start this worktree's services
mise run dev:wait
mise run test:backend   # Prepare databases and run all Go tests
go test ./backend/...  # Test specific package
```

E2E:
```bash
mise run test:browser         # Headless
mise run test:browser:headed  # With browser
mise run test:browser:ui      # Interactive mode
```

## Important Files

### Configuration
- `.mise.toml` - Development tools, tasks, and worktree environment
- `.mise.local.toml` - Local overrides (gitignored)
- Configuration via CLI flags or environment variables (see CLI Usage below)

### Build
- `.mise.toml` - Tool versions and build, run, and test tasks
- `package.json` - Frontend dependencies and scripts
- `go.mod` - Go dependencies

### Development
- `.devcontainer/` - VS Code devcontainer configuration
- `.vscode/tasks.json` - VS Code tasks for auto-start
- `vite.config.js` - Vite configuration

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

1. **Svelte 5 + SvelteKit 3**: Uses `.svelte` components with reactive `let` declarations. Lifecycle via `onMount()`. Always clean up subscriptions and event listeners in `onMount` return functions.

2. **SvelteKit File-Based Routing**: Routes in `src/routes/`. Route groups `(auth)` and `(protected)` for organization. Navigation via `goto()` from `$app/navigation`.

3. **Svelte Stores**: `writable` stores for cross-component state (session, ajax pending). Model classes use internal stores for change notification. Use `$store` syntax for auto-subscription in templates.

4. **Vite Build Tool**: `mise run dev` starts Vite on this worktree's assigned port and proxies `/api/*` to its backend.

5. **Source Location**: Frontend source is in `src/`, NOT `frontend/src/`.

### Database

1. **Tern Migrations**: Two-way migrations. Write both up and down SQL.

2. **Test Database**: Tests use separate database (`tpr_test`). Set up via `test/setup_test_databases.sql`.

3. **Permissions**: Migrations include permission grants. Don't forget these.

### Development

1. **First Run**: Use `scripts/setup-host` once per native host, then `mise trust`, `mise install`, and `mise run dev:init` in each checkout/worktree. The devcontainer runs initialization automatically.

2. **Service Lifecycle**: `mise run dev` owns PostgreSQL, migrations, the backend, and Vite through process-compose. Use `dev:wait`, `dev:urls`, and `dev:down` to manage it; tests require the running stack.

3. **Worktree Isolation**: Ports live in `.port-tamer.env`; PostgreSQL data lives in `.dev/<os>-<architecture>/postgres/data`. Never copy these between worktrees. Vite proxies API requests to `BACKEND_URL`.

4. **Task Runner**: Use `mise run <task>`; `mise tasks` lists available tasks.

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
- Reports use the worktree's `PLAYWRIGHT_REPORT_PORT` on `127.0.0.1` (forward it in a devcontainer)

## Deployment

Deployment uses [verna](https://github.com/jackc/verna) with Caddy as the reverse proxy.

1. **Build**: `mise run build:linux-amd64` creates a deployment tarball
2. **Deploy**: `verna app deploy build/linux_amd64.tar.gz`
3. **Config**: Set `VERNA_SSH_HOST` and `VERNA_APP` in `.mise.local.toml`
4. **Caddy**: Reverse proxy template in `deploy/caddy-handle-template.json`

## Resources

- [pgx Documentation](https://github.com/jackc/pgx)
- [chi Router](https://github.com/go-chi/chi)
- [Svelte 5 Docs](https://svelte.dev/docs)
- [SvelteKit Docs](https://svelte.dev/docs/kit)
- [Tern Migrations](https://github.com/jackc/tern)
- [Playwright](https://playwright.dev)

## Code Style

### Go
- Follow standard Go conventions
- Use `gofmt`
- Error handling: explicit checks, no panic in library code
- Logging: structured logging with `log/slog`. Supports `text`, `json`, and `journal` (systemd journald) output formats via `LOG_FORMAT`
- Comments: godoc-style for exported functions

### JavaScript/Svelte
- Use semicolons
- Single quotes for strings
- Tab indentation in `.svelte` files (Svelte default)
- 2-space indentation in `.js` files
- Use `#lib/` alias for imports from `src/lib/`

## Contributing Guidelines

When modifying this codebase:

1. **Use Svelte patterns**: Use `.svelte` components with reactive `let` declarations, `onMount` lifecycle, and Svelte stores for shared state
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

- Passwords: scrypt hashed with random salt
- Sessions: Random 32-byte tokens, hex-encoded
- CSRF: Not currently implemented (session tokens in header)
- SQL Injection: Prevented via prepared statements
- XSS: Svelte auto-escapes by default
- Email: SMTP with optional TLS
- Input Validation: Check in handlers before database calls

## Future Improvements

Potential areas for enhancement:

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

Last updated: 2026-03-21
