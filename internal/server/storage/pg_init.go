package gophmarktstorage

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	migrateFile "github.com/golang-migrate/migrate/v4/source/file"
)

// up migration
func ApplyMigrations(dsn, src string) error {
	absSrcPath, err := filepath.Abs(src)
	if err != nil {
		return fmt.Errorf("failed get abs sql path: %v", err)

	}

	f := &migrateFile.File{}
	srcDriver, err := f.Open(fmt.Sprintf("file://%s", absSrcPath))
	if err != nil {
		return fmt.Errorf("failed to read source directory %s: %v", absSrcPath, err)
	}

	migrator, err := migrate.NewWithSourceInstance(src, srcDriver, dsn)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %v", err)
	}
	defer migrator.Close()

	if err = migrator.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		migrator.Down()
		return fmt.Errorf("failed to apply migrations: %v", err)
	}

	return nil
}
