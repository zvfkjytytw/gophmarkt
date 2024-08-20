package gophmarktstorage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
)

type (
	DrawalOperationResult int
	Drawal                struct {
		Order       string    `json:"order"`
		Sum         float64   `json:"sum"`
		ProcessedAt time.Time `json:"processed_at,omitempty"`
	}
)

const (
	drawalTable = "gophmarkt.withdrawals"

	DrawalAddSuccess DrawalOperationResult = iota
	DrawalAddBefore
	DrawalAddByOther
	DrawalNotEnoughPoints
	DrawalAddError
	DrawalOperationFailed
)

func (s *PGStorage) AddDrawal(ctx context.Context, oid, login string, count float64) (DrawalOperationResult, error) {
	query, args, err := sq.Select("login").From(drawalTable).Where(sq.Eq{"order_id": oid}).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return DrawalOperationFailed, fmt.Errorf("failed generate select drawal login query for order %s: %v", oid, err)
	}

	row := s.db.QueryRowContext(ctx, query, args...)
	if row.Err() != nil {
		return DrawalOperationFailed, fmt.Errorf("failed execute select drawal login query for order %s: %v", oid, err)
	}

	var own string
	err = row.Scan(&own)
	if err != sql.ErrNoRows {
		if own == login {
			return DrawalAddBefore, fmt.Errorf("Drawal by order %s already upload", oid)
		}

		return DrawalAddByOther, fmt.Errorf("Drawal by order %s upload by other", oid)
	}

	if err != nil {
		return DrawalOperationFailed, fmt.Errorf("failed check drawal order %s: %v", oid, err)
	}

	now := time.Now().Format(time.DateTime)
	// now := time.Now().Format(time.RFC3339)
	query, args, err = sq.Insert(drawalTable).Columns("order_id", "login", "count", "offdate").Values(oid, login, count, now).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return DrawalOperationFailed, fmt.Errorf("failed generate insert order query for order %s: %v", oid, err)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return DrawalOperationFailed, fmt.Errorf("failed init DB transaction: %v", err)
	}
	defer tx.Rollback()

	result, err := tx.Exec(query, args...)
	if err != nil {
		return DrawalOperationFailed, fmt.Errorf("failed execute insert query: %v", err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		return DrawalOperationFailed, fmt.Errorf("failed get count of the affected rows: %v", err)
	}

	if n != 1 {
		return DrawalOperationFailed, fmt.Errorf("affected %d rows instead 1", n)
	}

	query, args, err = sq.Select("current", "withdrawn").
		From(balanceTable).Where(sq.Eq{"login": login}).Suffix("FOR UPDATE").
		PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return DrawalOperationFailed, fmt.Errorf("failed generate select balance with drawn query for login %s: %v", login, err)
	}

	row = s.db.QueryRowContext(ctx, query, args...)
	if row.Err() != nil {
		return DrawalOperationFailed, fmt.Errorf("failed execute select balance with drawn query for login %s: %v", login, err)
	}

	var current, withdrawn float64
	err = row.Scan(&current, &withdrawn)
	if err == sql.ErrNoRows {
		return DrawalOperationFailed, fmt.Errorf("no balance for login %s: %v", login, err)
	}

	if err != nil {
		return DrawalOperationFailed, fmt.Errorf("failed get balance for login %s: %v", login, err)
	}

	if count > current {
		return DrawalNotEnoughPoints, fmt.Errorf("not enough points on balance for login %s: %v", login, err)
	}

	query, args, err = sq.Update(balanceTable).Set("current", current-count).Set("withdrawn", withdrawn+count).Where(sq.Eq{"login": login}).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return DrawalOperationFailed, fmt.Errorf("failed generate update balance with drawn query for login %s: %v", login, err)
	}

	_, err = s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return DrawalOperationFailed, fmt.Errorf("failed execute update balance with drawn query for login %s: %v", login, err)
	}

	if err = tx.Commit(); err != nil {
		return DrawalOperationFailed, fmt.Errorf("failed commit query result: %v", err)
	}

	return DrawalAddSuccess, nil
}

func (s *PGStorage) GetDrawals(ctx context.Context, login string) ([]*Drawal, error) {
	drawals := make([]*Drawal, 0)
	query, args, err := sq.Select("order_id", "count", "offdate").From(drawalTable).Where(sq.Eq{"login": login}).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed generate select drawals query for login %s: %v", login, err)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed execute select drawals query for login %s: %v", login, err)
	}
	for rows.Next() {
		var oid string
		var sum float64
		var processedAt time.Time

		err = rows.Scan(&oid, &sum, &processedAt)
		if err == nil {
			drawal := &Drawal{
				Order:       oid,
				Sum:         sum,
				ProcessedAt: processedAt,
			}

			drawals = append(drawals, drawal)
		}
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("error scan drawals rows for login %s: %v", login, err)
	}

	return drawals, nil
}
