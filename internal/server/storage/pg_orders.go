package gophmarktstorage

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
)

type (
	OrderStatus          string
	OrderOperationResult int
	Order                struct {
		Number     string      `json:"number"`
		Status     OrderStatus `json:"status"`
		Accrual    float64     `json:"accrual,omitempty"`
		UploadedAt time.Time   `json:"uploaded_at"`
	}
)

const (
	ordersTable = "gophmarkt.orders"

	OrderStatusNew        OrderStatus = "NEW"
	OrderStatusProcessing OrderStatus = "PROCESSING"
	OrderStatusInvalid    OrderStatus = "INVALID"
	OrderStatusProcessed  OrderStatus = "PROCESSED"

	OrderAddSuccess OrderOperationResult = iota
	OrderAddBefore
	OrderAddByOther
	OrderAddError
	OrderOperationFailed
)

func (s *PGStorage) AddOrder(oid, login string) (OrderOperationResult, error) {
	query, args, err := sq.Select("login").From(ordersTable).Where(sq.Eq{"order_id": oid}).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return OrderOperationFailed, fmt.Errorf("failed generate select login query for order %s: %v", oid, err)
	}

	row := s.db.QueryRow(query, args...)
	if row.Err() != nil {
		return OrderOperationFailed, fmt.Errorf("failed execute select login query for order %s: %v", oid, err)
	}

	var own string
	if row.Scan(&own) != sql.ErrNoRows {
		if own == login {
			return OrderAddBefore, errors.New("order already upload")
		}

		return OrderAddByOther, errors.New("order upload by other")
	}

	now := time.Now().Format(time.DateTime)
	// now := time.Now().Format(time.RFC3339)
	query, args, err = sq.Insert(ordersTable).Columns("order_id", "login", "status", "date_upload", "date_update").
		Values(oid, login, OrderStatusNew, now, now).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return OrderOperationFailed, fmt.Errorf("failed generate insert order query for order %s: %v", oid, err)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return OrderOperationFailed, fmt.Errorf("failed init DB transaction: %v", err)
	}

	result, err := tx.Exec(query, args...)
	if err != nil {
		tx.Rollback()
		return OrderOperationFailed, fmt.Errorf("failed execute insert query: %v", err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		tx.Rollback()
		return OrderOperationFailed, fmt.Errorf("failed get count of the affected rows: %v", err)
	}

	if n != 1 {
		tx.Rollback()
		return OrderOperationFailed, fmt.Errorf("affected %d rows instead 1", n)
	}

	if err = tx.Commit(); err != nil {
		return OrderOperationFailed, fmt.Errorf("failed commit query result: %v", err)
	}

	return OrderAddSuccess, nil
}

func (s *PGStorage) GetOrders(login string) ([]*Order, error) {
	orders := make([]*Order, 0)
	query, args, err := sq.Select("order_id", "status", "date_upload", "accrual").From(ordersTable).
		Where(sq.Eq{"login": login}).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed generate select orders query for login %s: %v", login, err)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed execute select orders query for login %s: %v", login, err)
	}
	for rows.Next() {
		var oid string
		var status OrderStatus
		var upload time.Time
		var accrual float64

		rows.Scan(&oid, &status, &upload, &accrual)

		order := &Order{
			Number:     oid,
			Status:     status,
			UploadedAt: upload,
		}
		if accrual > 0 {
			order.Accrual = accrual
		}

		orders = append(orders, order)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("error scan orders rows for login %s: %v", login, err)
	}

	return orders, nil
}

func (s *PGStorage) GetUnprocessedOrders() ([]*Order, error) {
	orders := make([]*Order, 0)
	query, args, err := sq.Select("order_id", "status").From(ordersTable).
		Where(sq.Eq{"status": []OrderStatus{OrderStatusNew, OrderStatusProcessing}}).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed generate select unprocessed orders query: %v", err)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed execute select unprocessed orders query: %v", err)
	}
	for rows.Next() {
		var oid string
		var status OrderStatus

		rows.Scan(&oid, &status)

		order := &Order{
			Number: oid,
			Status: status,
		}

		orders = append(orders, order)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("error scan unprocessed orders rows: %v", err)
	}

	return orders, nil
}

func (s *PGStorage) UpdateOrder(order *Order) error {
	query, args, err := sq.Select("status", "login").From(ordersTable).
		Where(sq.Eq{"order_id": order.Number}).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return fmt.Errorf("failed generate select status query for order %s: %v", order.Number, err)
	}

	row := s.db.QueryRow(query, args...)
	if row.Err() != nil {
		return fmt.Errorf("failed execute select status query for order %s: %v", order.Number, err)
	}

	var status OrderStatus
	var login string
	if row.Scan(&status, &login) == sql.ErrNoRows {
		return fmt.Errorf("order %s not found", order.Number)
	}

	if status == order.Status {
		return fmt.Errorf("order %s status not changed", order.Number)
	}

	query, args, err = sq.Update(ordersTable).Set("status", order.Status).Set("date_update", order.UploadedAt).
		Where(sq.Eq{"order_id": order.Number}).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return fmt.Errorf("failed generate update query for order %s: %v", order.Number, err)
	}

	if order.Status == OrderStatusProcessed {
		query, args, err = sq.Update(ordersTable).
			Set("status", order.Status).Set("accrual", order.Accrual).Set("date_update", order.UploadedAt).
			Where(sq.Eq{"order_id": order.Number}).PlaceholderFormat(sq.Dollar).ToSql()
		if err != nil {
			return fmt.Errorf("failed generate update query for order %s: %v", order.Number, err)
		}
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed init DB transaction: %v", err)
	}

	result, err := tx.Exec(query, args...)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed execute insert query: %v", err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed get count of the affected rows: %v", err)
	}

	if n != 1 {
		tx.Rollback()
		return fmt.Errorf("affected %d rows instead 1", n)
	}

	if order.Status == OrderStatusProcessed {
		query, args, err := sq.Select("current").From(balanceTable).
			Where(sq.Eq{"login": login}).PlaceholderFormat(sq.Dollar).ToSql()
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed generate select balance query for login %s: %v", login, err)
		}

		row := s.db.QueryRow(query, args...)
		if row.Err() != nil {
			tx.Rollback()
			return fmt.Errorf("failed execute select balance query for login %s: %v", login, err)
		}

		var current float64
		if row.Scan(&current) == sql.ErrNoRows {
			tx.Rollback()
			return fmt.Errorf("balance for login %s not found ", login)
		}

		query, args, err = sq.Update(balanceTable).Set("current", current+order.Accrual).
			Where(sq.Eq{"login": login}).PlaceholderFormat(sq.Dollar).ToSql()
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed generate update balance query for login %s: %v", login, err)
		}

		_, err = s.db.Exec(query, args...)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed execute update balance query for login %s: %v", login, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed commit query result: %v", err)
	}

	return nil
}
