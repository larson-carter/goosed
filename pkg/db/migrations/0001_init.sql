-- +goose Up
CREATE TABLE IF NOT EXISTS blueprints (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    os TEXT NOT NULL,
    version TEXT,
    data JSONB NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now() NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT now() NOT NULL
);

CREATE TABLE IF NOT EXISTS workflows (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    steps JSONB NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now() NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT now() NOT NULL
);

CREATE TABLE IF NOT EXISTS machines (
    id UUID PRIMARY KEY,
    mac TEXT UNIQUE NOT NULL,
    serial TEXT,
    profile JSONB NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now() NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT now() NOT NULL
);

CREATE TABLE IF NOT EXISTS runs (
    id UUID PRIMARY KEY,
    machine_id UUID REFERENCES machines(id),
    blueprint_id UUID REFERENCES blueprints(id),
    status TEXT NOT NULL,
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    logs TEXT
);

CREATE TABLE IF NOT EXISTS artifacts (
    id UUID PRIMARY KEY,
    kind TEXT NOT NULL,
    sha256 TEXT NOT NULL,
    url TEXT NOT NULL,
    meta JSONB,
    created_at TIMESTAMPTZ DEFAULT now() NOT NULL
);

CREATE TABLE IF NOT EXISTS facts (
    id UUID PRIMARY KEY,
    machine_id UUID REFERENCES machines(id),
    snapshot JSONB NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now() NOT NULL
);

CREATE TABLE IF NOT EXISTS audit (
    id BIGSERIAL PRIMARY KEY,
    actor TEXT NOT NULL,
    action TEXT NOT NULL,
    obj TEXT NOT NULL,
    details JSONB,
    at TIMESTAMPTZ DEFAULT now() NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS audit;
DROP TABLE IF EXISTS facts;
DROP TABLE IF EXISTS artifacts;
DROP TABLE IF EXISTS runs;
DROP TABLE IF EXISTS machines;
DROP TABLE IF EXISTS workflows;
DROP TABLE IF EXISTS blueprints;
