package conveyor

import (
	"strings"

	"github.com/jmoiron/sqlx"
)

// Artifact represents an image that was successfully created from a build.
type Artifact struct {
	// Unique identifier for this artifact.
	ID string `db:"id"`
	// The build that this artifact was a result of.
	BuildID string `db:"build_id"`
	// The name of the image that was produced.
	Image string `db:"image"`
}

// artifactsCreate creates a new artifact linked to the build.
func artifactsCreate(tx *sqlx.Tx, a *Artifact) error {
	const createArtifactSql = `INSERT INTO artifacts (build_id, image) VALUES (:build_id, :image) RETURNING id`
	return insert(tx, createArtifactSql, a, &a.ID)
}

// artifactsFindByID finds an artifact by ID.
func artifactsFindByID(tx *sqlx.Tx, artifactID string) (*Artifact, error) {
	var sql = `SELECT * FROM artifacts WHERE id = ?`
	var a Artifact
	err := tx.Get(&a, tx.Rebind(sql), artifactID)
	return &a, err
}

// artifactsFindByRepoSha finds an artifact by image.
func artifactsFindByRepoSha(tx *sqlx.Tx, repoSha string) (*Artifact, error) {
	parts := strings.Split(repoSha, "@")
	var sql = `SELECT * FROM artifacts WHERE build_id = (SELECT id FROM builds WHERE repository = ? AND sha = ?)`
	var a Artifact
	err := tx.Get(&a, tx.Rebind(sql), parts[0], parts[1])
	return &a, err
}
