-- +migrate Up
ALTER TABLE builds ADD COLUMN seq SERIAL;
ALTER TABLE artifacts ADD COLUMN seq SERIAL;

CREATE INDEX index_builds_on_seq ON builds USING btree (seq);
CREATE INDEX index_artifacts_on_seq ON artifacts USING btree (seq);

-- +migrate Down
ALTER TABLE builds DROP COLUMN seq;
ALTER TABLE artifacts DROP COLUMN seq;
