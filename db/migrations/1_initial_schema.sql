-- +migrate Up
CREATE EXTENSION IF NOT EXISTS hstore;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE builds (
  id uuid NOT NULL DEFAULT uuid_generate_v4() primary key,
  repository text,
  branch text,
  sha text,
  state text,
  created_at timestamp without time zone default (now() at time zone 'utc'),
  started_at timestamp without time zone,
  completed_at timestamp without time zone
);

CREATE TABLE artifacts (
  id uuid NOT NULL DEFAULT uuid_generate_v4() primary key,
  build_id uuid NOT NULL references builds(id),
  image text,
  created_at timestamp without time zone default (now() at time zone 'utc')
);

-- We should ensure that we only have 1 pending/building build for any given sha.
CREATE UNIQUE INDEX unique_build ON builds USING btree (sha) WHERE (state = 'building' OR state = 'pending');

-- +migrate Down
DROP TABLE artifacts;
DROP TABLE builds;
