-- +migrate Up
ALTER TABLE artifacts ADD COLUMN sha text;
ALTER TABLE artifacts ADD COLUMN repository text;
UPDATE artifacts AS a SET sha = b.sha, repository = b.repository FROM builds AS b WHERE a.build_id = b.id;

-- +migrate Down
ALTER TABLE builds DROP COLUMN sha;
ALTER TABLE artifacts DROP COLUMN repository;
