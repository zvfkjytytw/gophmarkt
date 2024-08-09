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

var initQuerys = []string{
	// DROP
	// `DROP TABLE IF EXISTS gophmarkt.withdrawals;`,
	// `DROP TABLE IF EXISTS gophmarkt.orders;`,
	// `DROP INDEX IF EXISTS idx_gophmarkt_balance;`,
	// `DROP TABLE IF EXISTS gophmarkt.balance;`,
	// `DROP INDEX IF EXISTS idx_gophmarkt_users;`,
	// `DROP TABLE IF EXISTS gophmarkt.users;`,
	// `DROP TYPE IF EXISTS gophmarkt.order_status;`,
	// `DROP SCHEMA IF EXISTS gophmarkt;`,
	//CREATE
	`CREATE SCHEMA IF NOT EXISTS gophmarkt;`,
	// Type of the order statuses
	`CREATE TYPE gophmarkt.order_status AS ENUM (
		'NEW',
		'PROCESSING',
		'INVALID',
		'PROCESSED'
	);`,
	// USERS
	// Table of users
	`CREATE TABLE IF NOT EXISTS gophmarkt.users (
		login    text not null, -- username
		password text not null  -- password
	);`,
	// Set the user as the defining one
	`ALTER TABLE gophmarkt.users ADD PRIMARY KEY (login);`,
	// Index to optimize the search for user
	`CREATE UNIQUE INDEX idx_gophmarkt_users ON gophmarkt.users (login);`,
	// BALANCE
	// Table of balance
	`CREATE TABLE IF NOT EXISTS gophmarkt.balance (
		login     text not null,             -- username
		current   double precision not null, -- accumulated points
		withdrawn double precision           -- drawn points
	);`,
	// Set the user as the defining one
	`ALTER TABLE gophmarkt.balance ADD PRIMARY KEY (login);`,
	// Index to optimize the search for balance
	`CREATE UNIQUE INDEX idx_gophmarkt_balance ON gophmarkt.balance (login);`,
	// ORDERS
	// Table of orders
	`CREATE TABLE IF NOT EXISTS gophmarkt.orders (
		order_id    text not null,                   -- order id 
		login       text not null,                   -- username
		status      gophmarkt.order_status not null, -- order status
		accrual     double precision,                -- order points
		date_upload timestamp not null,              -- order upload date
		date_update timestamp not null               -- order last update date
	);`,
	// Set the user as the defining one
	`ALTER TABLE gophmarkt.orders ADD PRIMARY KEY (order_id);`,
	// WITHDRAWALS
	// Table of withdrawals
	`CREATE TABLE IF NOT EXISTS gophmarkt.withdrawals (
		order_id text not null,             -- order id 
		login    text not null,             -- username
		count    double precision not null, -- deducted points
		offdate  timestamp not null         -- date of debiting
	);`,
	`ALTER TABLE gophmarkt.withdrawals ADD PRIMARY KEY (order_id);`,
}

// up migration via db connect
func (s *PGStorage) ApplyMigrations() error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed init DB transaction: %v", err)
	}

	for _, query := range initQuerys {
		_, err := tx.Exec(query)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed execute init query %v: %v", query, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed commit init querys: %v", err)
	}

	return nil
}

// up migration from sql files
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
