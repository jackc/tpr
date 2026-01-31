# The Pithy Reader

The Pithy Reader is a simple RSS/Atom aggregator.

## Keyboard Shortcuts

* j - Select next item
* k - Select previous item
* v - Open item
* shift+a - mark all read

## Development

The preferred development environment is the provided devcontainer.

Tests are run with `rake`.

```
rake
```

There is a rake task that will automatically recompile and restart the backend server whenever any Go code changes.

```
rake rerun
```

In another terminal start the vite development server.

```
npx vite serve frontend
```

## Deployment

Configure `PRODUCTION_HOST` in `.mise.local.toml`. Run `bin/deploy production`.
