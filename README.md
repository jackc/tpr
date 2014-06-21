# The Pithy Reader

The Pithy Reader is a simple RSS/Atom aggregator.

## Keyboard Shortcuts

* j - Select next item
* k - Select previous item
* v - Open item
* shift+a - mark all read

## Development

The Pithy Reader backend is written in [Go](http://golang.org/). The frontend is built with [Middleman](http://middlemanapp.com/) which is a [Ruby](https://www.ruby-lang.org/) project. First ensure [Go](http://golang.org/) and [Ruby](https://www.ruby-lang.org/) are installed.

The Pithy Reader uses [godep](https://github.com/tools/godep) to manage its dependencies. If you don't already have it installed "go get" it.

    go get github.com/tools/godep

Get The Pithy Reader.

    go get github.com/jackc/tpr/backend

Go to repository you just checked out.

    cd $GOPATH/src/github.com/jackc/tpr

Install the Ruby dependencies:

    bundle install

All source code and required libraries for The Pithy Reader should now be installed.

The Pithy Reader requires a PostgreSQL database. Create one for testing and one for development.

    createdb tpr_development
    createdb tpr_test

For security reasons, The Pithy Reader is designed to run with a limited database user. Create this user (remember the password - you will need it later).

    createuser -P tpr

Database migrations are managed with [tern](https://github.com/jackc/tern). Install tern if you don't already have it.

    go get github.com/jackc/tern
    cd $GOPATH/src/github.com/jackc/tern
    godep go install

Go back to The Pithy Reader directory.

    cd $GOPATH/src/github.com/jackc/tpr

Make a copy of the example config files without the "example".

    cp tern.example.conf tern.conf
    cp tern.test.example.conf tern.test.conf
    cp tpr.example.conf tpr.conf
    cp tpr.test.example.conf tpr.test.conf

The "tern" files are used the the database migrator. The "tpr" files are used by The Pithy Reader server. Conf files with "test" in them are used for the test environment. The ones without "test" are used by the development environment.

Edit tern.conf and tern.test.conf and enter the database connection information for the development and test environments respectively. The database user tern runs as should be a superuser.

Migrate the development and test databases.

    tern migrate -m migrate -c tern.conf
    tern migrate -m migrate -c tern.test.conf

Edit tpr.conf and tpr.test.conf. Configure the database connection to use the "tpr" user created above.

The Pithy Reader development environment should be set up now. The Pithy Reader has Go tests for the server and RSpec/Capybara/Selenium integration tests. The default rake task runs the both test suites.

    bundle exec rake

There is a rake task that will automatically recompile and restart the backend server whenever any Go code changes.

    bundle exec rake rerun

In another terminal start the Middleman frontend server.

    cd frontend
    bundle exec middleman
