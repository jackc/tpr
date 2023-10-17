# The Pithy Reader

The Pithy Reader is a simple RSS/Atom aggregator.

## Keyboard Shortcuts

* j - Select next item
* k - Select previous item
* v - Open item
* shift+a - mark all read

## Development

The Pithy Reader backend is written in [Go](http://golang.org/). The frontend is built with [Vite](https://vitejs.dev/). First ensure [Go](http://golang.org/), [Ruby](https://www.ruby-lang.org/), and [NodeJS](https://nodejs.org/en/), and are installed.

First, get The Pithy Reader (i.e. close this repo).

Install the Ruby dependencies:

```
bundle install
```

Install the Node packages:

```
npm install
```

All source code and required libraries for The Pithy Reader should now be installed.

The Pithy Reader requires a PostgreSQL database. Create one for testing and one for development.

```
createdb tpr_dev
createdb tpr_test
```

For security reasons, The Pithy Reader is designed to run with a limited database user. Create this user (remember the password - you will need it later).

```
createuser -P tpr
```

Database migrations are managed with [tern](https://github.com/jackc/tern). Install tern if you don't already have it.

```
go install github.com/jackc/tern@latest
```

Automatic server rebuild and restart are managed with [watchexec](https://github.com/watchexec/watchexec). Install watchexec if you don't already have it.

Make a copy of the example config files without the "example".

```
cp postgresql/tern.example.conf postgresql/tern.conf
cp .envrc.example .envrc
cp tpr.example.conf tpr.conf
cp tpr.test.example.conf tpr.test.conf
```

The "tern" files are used the the database migrator. The "tpr" files are used by The Pithy Reader server. Conf files with "test" in them are used for the test environment. The ones without "test" are used by the development environment.

Edit these config files as needed.

Migrate the development and test databases.

```
tern migrate
PGDATABASE=tpr_test tern migrate
```

Edit tpr.conf and tpr.test.conf. Configure the database connection to use the "tpr" user created above.

The Pithy Reader development environment should be set up now. The default rake task runs the tests. You can also run tests directly with `go test ./...`, but the rake task ensures all dependencies are rebuilt if necessary.

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

Configure host in `.envrc`. Run `bin/deploy production`.
