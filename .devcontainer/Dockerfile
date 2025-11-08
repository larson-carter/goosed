FROM mcr.microsoft.com/devcontainers/go:1.25

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        postgresql-client \
        jq \
        unzip \
    && rm -rf /var/lib/apt/lists/*
