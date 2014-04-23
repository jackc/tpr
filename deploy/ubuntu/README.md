# Ubuntu

## Building a .deb Package

This is may have some rough edges, but it works.

    sudo rake deb

## Installation from .deb Package

Adjust version as appropriate.

    sudo dpkg -i tpr_VERSION.deb

## PostgreSQL Configuration

PostgreSQL 9.3+ is required.

    sudo -u postgres createuser tpr
    sudo -u postgres createdb

Database migrations are performed with (tern)[https://github.com/JackC/tern].
/etc/tpr/tern.conf is preset for a default installation where the database is
owned by postgres.

    sudo -u postgres tern -c /etc/tpr/tern.conf -m /usr/share/tpr/migrate

## Configuration

/etc/tpr/config.yml contains configuration for The Pithy Reader. By default it
will listen on 127.0.0.1:4000 and connect to a local tpr database as user tpr
through Unix domain sockets.

Consider using apache to serve static assets and handle SSL. This also lets us
listen on port 80 or 443 without any privileges. /deploy/apache2/site.conf
contains a sample vhost configuration file.

## Starting the server

    start tpr
