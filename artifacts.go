package conveyor

import "github.com/jmoiron/sqlx"

// Artifact represents an image that was successfully created from a build.
type Artifact struct {
	// The name of the image that was produced.
	Image string
}

type artifact struct {
	BuildID string `db:"build_id"`
	*Artifact
}

// artifactsCreate creates a new artifact linked to the build.
func artifactsCreate(tx *sqlx.Tx, buildID string, a *Artifact) error {
	const createArtifactSql = `INSERT INTO artifacts (build_id, image) VALUES (:build_id, :image)`
	_, err := tx.NamedExec(createArtifactSql, artifact{BuildID: buildID, Artifact: a})
	return err
}
