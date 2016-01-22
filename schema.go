package conveyor

import (
	"github.com/jmoiron/sqlx"
	"github.com/rubenv/sql-migrate"
)

var Migrations = &migrate.AssetMigrationSource{
	Asset:    Asset,
	AssetDir: AssetDir,
	Dir:      "db/migrations",
}

func Migrate(db *sqlx.DB, dir migrate.MigrationDirection) error {
	_, err := migrate.Exec(db.DB, db.DriverName(), Migrations, dir)
	return err
}

// MigrateUp migrates the database up.
func MigrateUp(db *sqlx.DB) error {
	return Migrate(db, migrate.Up)
}

// MigrateDown migrates the database down.
func MigrateDown(db *sqlx.DB) error {
	return Migrate(db, migrate.Down)
}

func Reset(db *sqlx.DB) error {
	if err := MigrateDown(db); err != nil {
		return err
	}

	return MigrateUp(db)
}
