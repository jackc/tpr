#!/bin/bash
set -e

SCRIPTDIR=$(dirname -- "$(readlink -f -- "$0")")
"$SCRIPTDIR/tern/install"
"$SCRIPTDIR/watchexec/install"
