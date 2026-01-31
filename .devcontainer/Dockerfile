FROM mcr.microsoft.com/devcontainers/base:ubuntu24.04

# Get the PostgreSQL 18 client. The Rails PostgreSQL client feature doesn't get the later versions.
RUN apt-get update && apt-get install -y postgresql-common \
    && yes | /usr/share/postgresql-common/pgdg/apt.postgresql.org.sh \
    && apt-get install -y postgresql-client-18 \
    && rm -rf /var/lib/apt/lists/*
