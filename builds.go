package conveyor

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// ErrDuplicateBuild can be returned when we try to start a build for a sha that
// is already in a "pending" or "building" state. We want to ensure that we only
// have 1 concurrent build for a given sha.
//
// This is also enforced at the db level with the `index_builds_on_sha_and_status`
// constraint.
var ErrDuplicateBuild = errors.New("a build for this sha is already pending or building")

// The database constraint that counts as an ErrDuplicateBuild.
const uniqueBuildConstraint = "unique_build"

// Build represents a build of a commit.
type Build struct {
	// A unique identifier for this build.
	ID string `db:"id"`
	// The repository that this build relates to.
	Repository string `db:"repository"`
	// The branch that this build relates to.
	Branch string `db:"branch"`
	// The sha that this build relates to.
	Sha string `db:"sha"`
	// The current status of the build.
	Status BuildStatus `db:"status"`
	// The time that this build was created.
	CreatedAt time.Time `db:"created_at"`
	// The time that the build was started.
	StartedAt *time.Time `db:"started_at"`
	// The time that the build was completed.
	CompletedAt *time.Time `db:"completed_at"`
}

type BuildStatus int

const (
	StatusPending BuildStatus = iota
	StatusBuilding
	StatusFailed
	StatusSucceeded
)

func (s BuildStatus) String() string {
	switch s {
	case StatusPending:
		return "pending"
	case StatusBuilding:
		return "building"
	case StatusFailed:
		return "failed"
	case StatusSucceeded:
		return "succeeded"
	default:
		panic(fmt.Sprintf("unknown build status: %v", s))
	}
}

// Scan implements the sql.Scanner interface.
func (s *BuildStatus) Scan(src interface{}) error {
	if v, ok := src.([]byte); ok {
		switch string(v) {
		case "pending":
			*s = StatusPending
		case "building":
			*s = StatusBuilding
		case "failed":
			*s = StatusFailed
		case "succeeded":
			*s = StatusSucceeded
		default:
			return fmt.Errorf("unknown build status: %v", string(v))
		}
	}

	return nil
}

// Value implements the driver.Value interface.
func (s BuildStatus) Value() (driver.Value, error) {
	return driver.Value(s.String()), nil
}

// buildsCreate inserts a new build into the database.
func buildsCreate(tx *sqlx.Tx, b *Build) error {
	const createBuildSql = `INSERT INTO builds (repository, branch, sha, status) VALUES (:repository, :branch, :sha, :status) RETURNING id`
	err := insert(tx, createBuildSql, b, &b.ID)
	if err, ok := err.(*pq.Error); ok {
		if err.Constraint == uniqueBuildConstraint {
			return ErrDuplicateBuild
		}
	}
	return err
}

// buildsFind finds a build by ID.
func buildsFind(tx *sqlx.Tx, buildID string) (*Build, error) {
	const findBuildSql = `SELECT * FROM builds where id = ?`
	var b Build
	err := tx.Get(&b, tx.Rebind(findBuildSql), buildID)
	if err != nil {
		return nil, err
	}
	return &b, err
}

// buildsUpdateStatus changes the status of a build.
func buildsUpdateStatus(tx *sqlx.Tx, buildID string, status BuildStatus) error {
	var sql string
	switch status {
	case StatusBuilding:
		sql = `UPDATE builds SET status = ?, started_at = ? WHERE id = ?`
	case StatusSucceeded, StatusFailed:
		sql = `UPDATE builds SET status = ?, completed_at = ? WHERE id = ?`
	default:
		panic(fmt.Sprintf("UpdateStatus for %s not implemented", status))
	}

	_, err := tx.Exec(tx.Rebind(sql), status, time.Now(), buildID)
	return err
}
