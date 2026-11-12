FROM mcr.microsoft.com/devcontainers/base:ubuntu24.04

# Each worktree runs PostgreSQL under process-compose; install binaries only.
RUN mkdir -p /etc/postgresql-common \
    && echo "create_main_cluster = false" > /etc/postgresql-common/createcluster.conf \
    && apt-get update && apt-get install -y postgresql-common \
    && yes | /usr/share/postgresql-common/pgdg/apt.postgresql.org.sh \
    && apt-get install -y postgresql-18 \
    && rm -rf /var/lib/apt/lists/*
