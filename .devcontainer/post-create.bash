#!/bin/bash
set -e

psql -f postgresql/prepare.sql

[ ! -f postgresql/tern.conf ] && cp postgresql/tern.example.conf postgresql/tern.conf
[ ! -f tpr.conf ] && cp tpr.example.conf tpr.conf
[ ! -f tpr.test.conf ] && cp tpr.test.example.conf tpr.test.conf

mise trust
mise install
eval "$(mise env -s bash)"
bundle install
npm install
npx playwright install --with-deps chromium
go install golang.org/x/tools/cmd/goimports@latest

tern migrate
PGDATABASE=tpr_test tern migrate

# Run any additional setup scripts included in the shared/devcontainer directory. This is to allow for per developer or
# per-environment customizations. These scripts are not checked into source control.
if [ -x "../shared/devcontainer/install" ]; then
  ../shared/devcontainer/install
fi

# Create a symlink to the shared .scratch directory for temporary files if it exists.
if [ -x "../shared/.scratch" ]; then
  if [ ! -e .scratch ] && [ ! -L .scratch ]; then
    ln -s ../shared/.scratch
  fi
fi
