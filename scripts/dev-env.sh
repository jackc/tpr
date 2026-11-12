# shellcheck shell=bash
#
# Sourced by the other scripts in this directory, never run directly. Assumes
# the caller has already changed to the worktree root and set -e.
#
# This is the one place where a worktree's environment is assembled: the
# ports persisted by port-tamer, the project-specific values derived from
# them, the paths of this worktree's PostgreSQL cluster, the PostgreSQL
# binaries to run it with, and the locale every process runs under.

set -a

port-tamer allocate
. .port-tamer.env

# port-tamer deliberately owns only named port assignments. URLs, database
# names, and other application settings belong to this project.
BIND_ADDRESS="127.0.0.1"
BACKEND_URL="http://127.0.0.1:$BACKEND_PORT"
VITE_URL="http://127.0.0.1:$VITE_PORT"
TEST_BACKEND_URL="http://127.0.0.1:$TEST_BACKEND_PORT"
TEST_VITE_URL="http://127.0.0.1:$TEST_VITE_PORT"
PGHOST="127.0.0.1"
DATABASE_URL="postgres://tpr@$PGHOST:$PGPORT/tpr_dev"
TEST_DATABASE_URL="postgres://tpr@$PGHOST:$PGPORT/tpr_test"

# A PostgreSQL data directory cannot be shared between platforms, and one
# worktree may be opened natively and in the dev container, so each gets its
# own cluster. The cluster has no Unix-domain socket; see scripts/postgres.
DEV_PLATFORM="$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m)"
PGDATA="${PGDATA:-$PWD/.dev/$DEV_PLATFORM/postgres/data}"
PGUSER="${PGUSER:-postgres}"
PGDATABASE="${PGDATABASE:-tpr_dev}"
TEST_DATABASE="${TEST_DATABASE:-tpr_test}"
TEST_DATABASE_COUNT="${TEST_DATABASE_COUNT:-8}"
TERN_CONFIG="${TERN_CONFIG:-postgresql/tern.conf}"
TERN_MIGRATIONS="${TERN_MIGRATIONS:-postgresql/migrations}"
PC_ADDRESS="${PC_ADDRESS:-127.0.0.1}"
PC_SERVER_LOG_FILE="${PC_SERVER_LOG_FILE:-$PWD/.dev/process-compose.log}"
PC_LOG_FILE="${PC_LOG_FILE:-$PWD/.dev/process-compose-client.log}"
PC_POSTGRES_PID_FILE="${PC_POSTGRES_PID_FILE:-$PWD/.dev/$DEV_PLATFORM/postgres/process-compose.pid}"

# Every process needs an explicit locale. Interactive shells set one, but
# agents, CI, and other non-interactive environments often set none, and then
# macOS's libc consults CoreFoundation, which starts a thread and makes
# PostgreSQL refuse to start ("postmaster became multithreaded during
# startup"). Any valid locale avoids that; en_US.UTF-8 is what macOS terminals
# set and what the dev container generates, so non-interactive runs behave like
# interactive ones. The cluster's collation is fixed at initdb time and does
# not depend on this value.
LANG="${LANG:-en_US.UTF-8}"

# Resolved here rather than at the point of use so that a machine without the
# PostgreSQL server binaries says so immediately, with installation
# instructions, instead of failing later as a process that will not start.
PGBIN="${PGBIN:-$(scripts/pgbin)}"

PATH="$PGBIN:$PATH"
PORT="$BACKEND_PORT"

set +a

mkdir -p .dev/logs

# tern.conf is deliberately not in git: it is a local file a developer may
# point elsewhere.
if [ ! -f postgresql/tern.conf ]; then
	cp postgresql/tern.example.conf postgresql/tern.conf
fi
