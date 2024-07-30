package gophmarktstorage

import (
	"database/sql"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
)

type UserOperationResult int

const (
	usersTable = "gophmarkt.users"

	UserAddSuccess UserOperationResult = iota
	UserExist
	UserNotFound
	UserPasswordWrong
	UserOperationFailed
)

func (s *PGStorage) AddUser(login, password string) (UserOperationResult, error) {
	query, args, err := sq.Select("login").From(usersTable).Where(sq.Eq{"login": login}).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return UserOperationFailed, fmt.Errorf("failed generate select login query for login %s: %v", login, err)
	}

	row := s.db.QueryRow(query, args...)
	if row.Err() != nil {
		return UserOperationFailed, fmt.Errorf("failed execute select login query for login %s: %v", login, err)
	}

	var user string
	if row.Scan(&user) != sql.ErrNoRows {
		return UserExist, errors.New("user already exist")
	}

	if !validatePassword(password) {
		return UserPasswordWrong, errors.New("unsuitable password")
	}

	query, args, err = sq.Insert(usersTable).Columns("login", "password").Values(login, password).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return UserOperationFailed, fmt.Errorf("failed generate select login, password query for login %s: %v", login, err)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return UserOperationFailed, fmt.Errorf("failed init DB transaction: %v", err)
	}

	result, err := tx.Exec(query, args...)
	if err != nil {
		tx.Rollback()
		return UserOperationFailed, fmt.Errorf("failed execute insert login, password query: %v", err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		tx.Rollback()
		return UserOperationFailed, fmt.Errorf("failed get count of the affected rows: %v", err)
	}

	if n != 1 {
		tx.Rollback()
		return UserOperationFailed, fmt.Errorf("affected %d rows instead 1", n)
	}

	query, args, err = sq.Insert(balanceTable).Columns("login", "current", "withdrawn").Values(login, startBalance, 0).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		tx.Rollback()
		return UserOperationFailed, fmt.Errorf("failed generate init balance query for login %s: %v", login, err)
	}

	_, err = s.db.Exec(query, args...)
	if err != nil {
		tx.Rollback()
		return UserOperationFailed, fmt.Errorf("failed execute init balance query for login %s: %v", login, err)
	}

	if err = tx.Commit(); err != nil {
		return UserOperationFailed, fmt.Errorf("failed commit query result: %v", err)
	}

	return UserAddSuccess, nil
}

func (s *PGStorage) CheckUser(login, password string) (UserOperationResult, error) {
	query, args, err := sq.Select("login").From(usersTable).Where(sq.Eq{"login": login}).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return UserOperationFailed, fmt.Errorf("failed generate select query for login %s: %v", login, err)
	}

	row := s.db.QueryRow(query, args...)
	if row.Err() != nil {
		return UserOperationFailed, fmt.Errorf("failed execute select query for login %s: %v", login, err)
	}

	var user string
	if row.Scan(&user) == sql.ErrNoRows {
		return UserNotFound, errors.New("user not found")
	}

	query, args, err = sq.Select("password").From(usersTable).Where(sq.Eq{"login": login}).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return UserOperationFailed, fmt.Errorf("failed generate select query for login %s: %v", login, err)
	}

	row = s.db.QueryRow(query, args...)
	if row.Err() != nil {
		return UserOperationFailed, fmt.Errorf("failed execute select query for login %s: %v", login, err)
	}

	var pass string
	row.Scan(&pass)
	if password != pass {
		return UserPasswordWrong, errors.New("wrong password")
	}

	return UserExist, nil
}

func validatePassword(password string) bool {
	if len(password) < 4 {
		return false
	}

	return true
}
