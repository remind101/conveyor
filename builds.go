package conveyor

import (
	"time"

	"code.google.com/p/go-uuid/uuid"
	"github.com/jinzhu/gorm"
	"github.com/remind101/conveyor/builder"
)

type BuildStatus int

const (
	StatusPending BuildStatus = iota
	StatusFailed
	StatusSucceeded
)

func (s BuildStatus) String() string {
	switch s {
	case StatusFailed:
		return "failed"
	case StatusSucceeded:
		return "succeeded"
	default:
		return "pending"
	}
}

type Build struct {
	// Unique identifier for this build.
	ID string

	// The status that this build is in.
	Status BuildStatus

	// Image is the string identifier of the build image.
	Image string

	// The options provided to start the build.
	builder.BuildOptions

	// The time that this build was created.
	CreatedAt time.Time

	// The time that this build started building.
	StartedAt *time.Time

	// The time that the build completed.
	CompletedAt *time.Time
}

func (b *Build) BeforeCreate() error {
	b.ID = uuid.New()
	return nil
}

func (b *Build) String() string {
	return b.ID
}

// BuildsService is a gorm.DB backed persistence layer for Builds.
type BuildsService struct {
	db *gorm.DB
}

// Create persists the build.
func (s *BuildsService) Create(b *Build) error {
	return s.db.Create(b).Error
}

func (s *BuildsService) Update(b *Build) error {
	return s.db.Update(b).Error
}
