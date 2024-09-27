package gophmarktstorage

import (
	"context"
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

func (s *PGStorage) AddUser(ctx context.Context, login, password string) (UserOperationResult, error) {
	query, args, err := sq.Select("login").From(usersTable).Where(sq.Eq{"login": login}).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return UserOperationFailed, fmt.Errorf("failed generate select login query for login %s: %v", login, err)
	}

	row := s.db.QueryRowContext(ctx, query, args...)
	if row.Err() != nil {
		return UserOperationFailed, fmt.Errorf("failed execute select login query for login %s: %v", login, err)
	}

	var user string
	err = row.Scan(&user)
	if err == nil {
		return UserExist, errors.New("user already exist")
	}

	if !validatePassword(password) {
		return UserPasswordWrong, errors.New("unsuitable password")
	}

	if err != sql.ErrNoRows {
		return UserOperationFailed, fmt.Errorf("failed check login %s: %v", login, err)
	}

	query, args, err = sq.Insert(usersTable).Columns("login", "password").Values(login, password).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return UserOperationFailed, fmt.Errorf("failed generate select login, password query for login %s: %v", login, err)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return UserOperationFailed, fmt.Errorf("failed init DB transaction: %v", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return UserOperationFailed, fmt.Errorf("failed execute insert login, password query: %v", err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		return UserOperationFailed, fmt.Errorf("failed get count of the affected rows: %v", err)
	}

	if n != 1 {
		return UserOperationFailed, fmt.Errorf("affected %d rows instead 1", n)
	}

	query, args, err = sq.Insert(balanceTable).Columns("login", "current", "withdrawn").Values(login, startBalance, 0).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return UserOperationFailed, fmt.Errorf("failed generate init balance query for login %s: %v", login, err)
	}

	_, err = s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return UserOperationFailed, fmt.Errorf("failed execute init balance query for login %s: %v", login, err)
	}

	if err = tx.Commit(); err != nil {
		return UserOperationFailed, fmt.Errorf("failed commit query result: %v", err)
	}

	return UserAddSuccess, nil
}

func (s *PGStorage) CheckUser(ctx context.Context, login, password string) (UserOperationResult, error) {
	query, args, err := sq.Select("login", "password").From(usersTable).Where(sq.Eq{"login": login}).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return UserOperationFailed, fmt.Errorf("failed generate select query for login %s: %v", login, err)
	}

	row := s.db.QueryRowContext(ctx, query, args...)
	if row.Err() != nil {
		return UserOperationFailed, fmt.Errorf("failed execute select query for login %s: %v", login, err)
	}

	var user, pass string
	err = row.Scan(&user, &pass)

	if err == sql.ErrNoRows {
		return UserNotFound, errors.New("user not found")
	}

	if err != nil {
		return UserOperationFailed, fmt.Errorf("failed check login %s: %v", login, err)
	}

	if password != pass {
		return UserPasswordWrong, errors.New("wrong password")
	}

	return UserExist, nil
}

func validatePassword(password string) bool {
	if len(password) < 4 {
		return false
	}

	for _, l := range password {
		switch l {
		case ' ':
			return false
		case '(':
			return false
		case ')':
			return false
		case ':':
			return false
		case '!':
			return false
		}
	}

	return true
}
