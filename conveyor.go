package conveyor

import (
	"io"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/remind101/conveyor/builder"
	"github.com/remind101/conveyor/logs"
	"golang.org/x/net/context"

	"code.google.com/p/go-uuid/uuid"
)

// newID returns a new unique identifier.
var newID = uuid.New

// Conveyor provides the primary api for triggering builds.
type Conveyor struct {
	// BuildQueue is used to enqueue a build.
	BuildQueue

	// Logger is the log storage backend to read and write logs for builds.
	Logger logs.Logger

	GitHub GitHubAPI

	db *sqlx.DB
}

// New returns a new Conveyor instance.
func New(db *sqlx.DB) *Conveyor {
	return &Conveyor{db: db}
}

// BuildRequest is provided when triggering a new build.
type BuildRequest struct {
	// Repository is the repo to build. This is always required.
	Repository string
	// Sha is the git commit to build. If this is not provided, and a Branch
	// is provided, the sha will be auto-resolved.
	Sha string
	// Branch is the name of the branch that this build relates to.
	Branch string
	// Set to true to disable the layer cache. The zero value is to enable
	// caching.
	NoCache bool
}

// Build enqueues a build to run.
func (c *Conveyor) Build(ctx context.Context, req BuildRequest) (*Build, error) {
	// A branch is provied with no sha. Use the GitHub API to resolve the
	// branch to the sha of HEAD of the branch.
	if req.Sha == "" && req.Branch != "" {
		owner, repo := splitRepo(req.Repository)
		sha, err := c.GitHub.ResolveBranch(owner, repo, req.Branch)
		if err != nil {
			return nil, err
		}
		req.Sha = sha
	}

	tx, err := c.db.Beginx()
	if err != nil {
		return nil, err
	}

	b := &Build{
		Repository: req.Repository,
		Sha:        req.Sha,
		Branch:     req.Branch,
	}

	if err := buildsCreate(tx, b); err != nil {
		tx.Rollback()
		return b, err
	}

	// Commit before we push the build into the queue. We need to do this
	// because it's possible that two inflight transactions will get
	// commited and one will raise an error.
	if err := tx.Commit(); err != nil {
		return b, err
	}

	return b, c.BuildQueue.Push(ctx, builder.BuildOptions{
		ID:         b.ID,
		Repository: req.Repository,
		Sha:        req.Sha,
		Branch:     req.Branch,
		NoCache:    req.NoCache,
	})

}

// FindBuild finds a build by its identity.
func (c *Conveyor) FindBuild(ctx context.Context, buildIdentity string) (*Build, error) {
	tx, err := c.db.Beginx()
	if err != nil {
		return nil, err
	}

	var find func(*sqlx.Tx, string) (*Build, error)
	switch strings.Contains(buildIdentity, "@") {
	case true:
		find = buildsFindByRepoSha
	default:
		find = buildsFindByID
	}

	b, err := find(tx, buildIdentity)
	if err != nil {
		tx.Rollback()
		return b, err
	}

	return b, tx.Commit()
}

// FindArtifact finds an artifact by its identity.
func (c *Conveyor) FindArtifact(ctx context.Context, artifactIdentity string) (*Artifact, error) {
	tx, err := c.db.Beginx()
	if err != nil {
		return nil, err
	}

	var find func(*sqlx.Tx, string) (*Artifact, error)
	switch strings.Contains(artifactIdentity, "@") {
	case true:
		find = artifactsFindByRepoSha
	default:
		find = artifactsFindByID
	}

	a, err := find(tx, artifactIdentity)
	if err != nil {
		tx.Rollback()
		return a, err
	}

	return a, tx.Commit()
}

// Writer returns an io.Writer to write logs for the build.
func (c *Conveyor) Writer(ctx context.Context, buildID string) (io.Writer, error) {
	return c.Logger.Create(buildID)
}

// Logs returns an io.Reader to read logs for the build.
func (c *Conveyor) Logs(ctx context.Context, buildID string) (io.Reader, error) {
	return c.Logger.Open(buildID)
}

// BuildStarted marks the build as started.
func (c *Conveyor) BuildStarted(ctx context.Context, buildID string) error {
	tx, err := c.db.Beginx()
	if err != nil {
		return err
	}

	if err := buildsUpdateState(tx, buildID, StateBuilding); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// BuildComplete marks a build as successful and adds the image as an artifact.
func (c *Conveyor) BuildComplete(ctx context.Context, buildID, image string) error {
	tx, err := c.db.Beginx()
	if err != nil {
		return err
	}

	if err := buildsUpdateState(tx, buildID, StateSucceeded); err != nil {
		tx.Rollback()
		return err
	}

	if err := artifactsCreate(tx, &Artifact{
		BuildID: buildID,
		Image:   image,
	}); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// BuildFailed marks the build as failed.
func (c *Conveyor) BuildFailed(ctx context.Context, buildID string, err error) error {
	tx, err := c.db.Beginx()
	if err != nil {
		return err
	}

	if err := buildsUpdateState(tx, buildID, StateFailed); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func insert(tx *sqlx.Tx, sql string, v interface{}, returns ...interface{}) error {
	rows, err := tx.NamedQuery(sql, v)
	if err != nil {
		return err
	}
	defer rows.Close()
	if rows.Next() {
		for _, r := range returns {
			rows.Scan(r)
		}
	} else {
		panic("expected id to be returned")
	}
	return nil
}
