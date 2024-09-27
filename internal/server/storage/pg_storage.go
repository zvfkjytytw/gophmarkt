package gophmarktstorage

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
)

const (
	postgresDriver = "postgres"
	postgresDSN    = "postgres://%s:%s@%s:%d/%s?sslmode=%s&binary_parameters=yes"
)

type Config struct {
	Host     string `yaml:"host"`
	Port     int32  `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"db_name"`
	SSLMode  string `yaml:"sslmode"`
}

type PGStorage struct {
	db *sql.DB
}

func NewPGStorageFromConfig(config *Config) (*PGStorage, error) {
	dsn, err := GetDSNFromConfig(config)
	if err != nil {
		return nil, err
	}

	return NewPGStorage(dsn)
}

func GetDSNFromConfig(config *Config) (string, error) {
	password, ok := os.LookupEnv(config.Password)
	if !ok {
		return "", errors.New("PostgreSQL password is not found")
	}

	return fmt.Sprintf(postgresDSN, config.User, password, config.Host, config.Port, config.DBName, config.SSLMode), nil
}

// 'postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable&binary_parameters=yes'
func NewPGStorage(dsn string) (*PGStorage, error) {
	db, err := sql.Open(postgresDriver, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed create database connection: %v", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("database is not responding: %v", err)
	}

	return &PGStorage{
		db: db,
	}, nil
}

func (s *PGStorage) Close() error {
	return s.db.Close()
}
