# Storage backends

TPR supports `--store postgres` (the default), `--store jed`, and `--store dual`.
The equivalent environment variable is `TPR_STORE`. PostgreSQL uses
`--database-url` / `DATABASE_URL`. JED uses `--jed-path` / `TPR_JED_PATH`.
The same flags apply to `server` and `reset-password`.

JED stores **all users, feeds, items, subscriptions, sessions, and password resets
in one database file**. Shared feeds are fetched once; subscriptions and unread
items remain user-scoped. Its embedded schema initializes automatically on first
open and records a schema version. Durability remains enabled. The adapter
serializes access to its database and buffers query results before releasing its
lock. PostgreSQL remains the default and uses the existing connection pool.

```sh
# JED-only; DATABASE_URL is not needed.
tpr server --store jed --jed-path /var/lib/tpr/tpr.jed

tpr reset-password --store jed --jed-path /var/lib/tpr/tpr.jed reader
```

## Migrating an existing installation

1. Stop TPR and any other writers to its PostgreSQL database.
2. Run the normal tern migration against the explicitly selected database.
   Migration `012_uuid_identifiers.sql` converts user, feed, and item IDs and all
   their references to UUIDs. Existing sessions, subscriptions, unread state,
   password-reset associations, and article data are preserved. Legacy integer
   IDs remain in PostgreSQL for the down migration. Apply the migration using a
   database owner; the application role retains its existing grants.
3. Copy to a new or empty JED database:

   ```sh
   tpr copy-to-jed --database-url "$DATABASE_URL" --jed-path /var/lib/tpr/tpr.jed
   ```

   This reads a repeatable-read PostgreSQL snapshot and copies all seven
   application tables in one JED transaction, preserving UUIDs, credentials,
   sessions, and timestamps. It refuses a nonempty destination and verifies the
   result. Keep writers stopped through the following restart.
4. Start `tpr server --store dual --database-url "$DATABASE_URL" --jed-path
   /var/lib/tpr/tpr.jed`, or use `--store jed` for JED alone.

The migration is transactional and has a tested down migration that preserves
relationships and restores original integer IDs; rows created after migration
also receive rollback IDs. Use matching application code when rolling back. IDs
in HTTP JSON and paths are now UUID strings. Reload open browser tabs after the
upgrade so cached numeric feed/item IDs are discarded.

## Dual mode

Dual mode is a validation harness, like FAM and Logger4Life. It runs each SQL
operation against PostgreSQL and JED, compares results, affected row counts, and
SQLSTATE/constraint errors, and returns PostgreSQL's result. IDs and persisted
application timestamps are generated once before SQL execution. Ordered queries
include deterministic tie breakers. Comparisons normalize driver representations
(UUID, numeric, time zone, and IP values), without dropping application fields.
Startup compares every application row; it does not silently initialize or repair
a divergent database.

A mismatch panics and disables subsequent operations on that store instance,
even if the HTTP server recovers the panic. Reconcile the databases before
restarting. Operations are serialized to prevent different write order between
the stores. Use a single TPR writer, with no independent writes to either database.
Transactions roll back both backends on callback errors. There is no distributed
commit protocol: a crash or commit failure between the two commits can leave them
different. A commit failure disables the store; startup verification detects
persisted differences. Dual mode is not replication or automatic failover.

## JED compatibility findings

Updated to `github.com/jackc/jed/impl/go`
`v0.2.1-0.20261113022601-b48cba26e8a6` (upstream HEAD when checked).
The previously verified fixes for parameterized `LIMIT`, table-alias `row_to_json`, and the
`timestamp with time zone` type spelling still pass. The archive uses `LIMIT $2`
with argument `250`. API JSON assembly remains in Go; a regression test also
exercises the original nested `json_agg(row_to_json(t))` query shape. The valid
`timestamptz` spelling remains in the JED schema.

The remaining confirmed gaps are:

| SQL | JED result | TPR workaround |
| --- | --- | --- |
| `create table addresses(ip inet)` | `42704: type does not exist: inet` | Store canonical IP strings in JED; PostgreSQL keeps `inet`. |
| `select 'reader' ~ '\A[a-zA-Z0-9]+\Z'` | `2201B: invalid escape \Z` | Use `^[a-zA-Z0-9]+$` for JED's username constraint. |
| `create function answer() returns integer language sql as 'select 42'` | `42601: expected keyword 'table'` | Implement subscription creation as transactional SQL in Go; remove the PostgreSQL stored function. |

UUID inference is now fixed (upstream commit `bc54713c`). TPR no longer
needs explicit `$1::uuid` casts in these `INSERT ... SELECT` projections:

```sql
-- $1 inherits UUID from subscriptions.user_id; $2 is text.
insert into subscriptions(user_id, feed_id)
select $1, id from feeds where url=$2;

-- $1 inherits UUID from unread_items.item_id; $2 is UUID.
insert into unread_items(user_id, feed_id, item_id)
select user_id, feed_id, $1 from subscriptions where feed_id=$2;
```

Both now execute on JED, matching PostgreSQL 18.6's inferred parameter types
`[uuid, text]` and `[uuid, uuid]`. The storage conformance suite exercises the
uncast production statements in JED and dual mode.

The same upstream change adds parameter binding across data-modifying CTEs.
The conformance suite now tests inserting an article in a CTE and inserting its
unread marker in the primary statement, with different parameter subsets in each
part and an uncast UUID parameter in the primary SELECT projection.

These are upstream compatibility opportunities; the workarounds preserve TPR's
application features. The PostgreSQL-specific `pgundolog` browser test endpoints
remain PostgreSQL-only and are rejected when enabled with JED/dual mode.

## Validation

The Go suite exercises JED and dual mode for shared feeds, unread isolation,
authentication, uniqueness errors, account changes, session revocation, password
reset IPs, feed polling and updates, transaction rollback, concurrent writes,
reopening files, snapshot copying, and fail-stop divergence detection. The
PostgreSQL migration test exercises populated integer → UUID → integer → UUID
transitions. Dual tests use the disposable `tpr_test_8` database, created by the
normal test setup. All 11 existing browser tests exercise UUIDs through PostgreSQL.

Run the normal test setup and `scripts/dev-exec go test ./...`; JED needs permission
to open local Unix sockets for its database coordination. Tests do not disable
fsync. Run `scripts/dev-exec npm run test:e2e` for browser coverage.
