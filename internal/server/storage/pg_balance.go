package gophmarktstorage

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
)

type Balance struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn,omitempty"`
}

const (
	balanceTable = "gophmarkt.balance"
	startBalance = 0
)

func (s *PGStorage) AddBalance(ctx context.Context, login string, count float64) error {
	query, args, err := sq.Select("current").From(balanceTable).Where(sq.Eq{"login": login}).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return fmt.Errorf("failed generate select balance query for login %s: %v", login, err)
	}

	row := s.db.QueryRowContext(ctx, query, args...)
	if row.Err() != nil {
		return fmt.Errorf("failed execute select balance query for login %s: %v", login, err)
	}

	var current float64
	err = row.Scan(&current)
	if err == sql.ErrNoRows {
		query, args, err = sq.Insert(balanceTable).Columns("login", "current", "withdrawn").Values(login, count, 0).PlaceholderFormat(sq.Dollar).ToSql()
		if err != nil {
			return fmt.Errorf("failed generate init balance query for login %s: %v", login, err)
		}

		_, err = s.db.ExecContext(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("failed execute init balance query for login %s: %v", login, err)
		}

		return nil
	}

	if err != nil {
		return fmt.Errorf("failed get balance for login %s: %v", login, err)
	} else {
		query, args, err = sq.Update(balanceTable).Set("current", current+count).Where(sq.Eq{"login": login}).PlaceholderFormat(sq.Dollar).ToSql()
		if err != nil {
			return fmt.Errorf("failed generate update balance query for login %s: %v", login, err)
		}

		_, err = s.db.ExecContext(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("failed execute update balance query for login %s: %v", login, err)
		}
	}

	return nil
}

func (s *PGStorage) DrawnBalance(ctx context.Context, login string, count float64) error {
	query, args, err := sq.Select("current", "withdrawn").From(balanceTable).Where(sq.Eq{"login": login}).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return fmt.Errorf("failed generate select balance with drawn query for login %s: %v", login, err)
	}

	row := s.db.QueryRowContext(ctx, query, args...)
	if row.Err() != nil {
		return fmt.Errorf("failed execute select balance with drawn query for login %s: %v", login, err)
	}

	var current, withdrawn float64
	err = row.Scan(&current, &withdrawn)
	if err == sql.ErrNoRows {
		return fmt.Errorf("no balance for login %s: %v", login, err)
	}

	if err != nil {
		return fmt.Errorf("failed get balance for login %s: %v", login, err)
	}

	query, args, err = sq.Update(balanceTable).Set("current", current-count).Set("withdrawn", withdrawn+count).Where(sq.Eq{"login": login}).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return fmt.Errorf("failed generate update balance with drawn query for login %s: %v", login, err)
	}

	_, err = s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed execute update balance with drawn query for login %s: %v", login, err)
	}

	return nil
}

func (s *PGStorage) GetBalance(ctx context.Context, login string) (*Balance, error) {
	query, args, err := sq.Select("current", "withdrawn").From(balanceTable).Where(sq.Eq{"login": login}).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed generate select balance with drawn query for login %s: %v", login, err)
	}

	row := s.db.QueryRowContext(ctx, query, args...)
	if row.Err() != nil {
		return nil, fmt.Errorf("failed execute select balance with drawn query for login %s: %v", login, err)
	}

	var current, withdrawn float64
	err = row.Scan(&current, &withdrawn)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no balance for login %s: %v", login, err)
	}

	if err != nil {
		return nil, fmt.Errorf("failed get balance for login %s: %v", login, err)
	}

	return &Balance{
		Current:   current,
		Withdrawn: withdrawn,
	}, nil
}

func (s *PGStorage) DropBalance(ctx context.Context, login string) error {
	query, args, err := sq.Delete(balanceTable).Where(sq.Eq{"login": login}).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return fmt.Errorf("failed generate delete balance query for login %s: %v", login, err)
	}

	_, err = s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed execute delete balance query for login %s: %v", login, err)
	}

	return nil
}
