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
		Accrual    int32       `json:"accrual,omitempty"`
		UploadedAt time.Time   `json:"uploaded_at"`
	}
)

const (
	ordersTable = "gophmarkt.orders"

	OrderStatusNew        OrderStatus = "NEW"
	OrderStatusProcessing OrderStatus = "PROCESSING"
	OrderStatusInvalid    OrderStatus = "INVALID"
	OrderStatusProcessed  OrderStatus = "PROCEESED"

	OrderAddSuccess OrderOperationResult = iota
	OrderAddBefore
	OrderAddByOther
	OrderAddError
	OrderOperationFailed
)

func (s *PGStorage) AddOrder(order_id, login string) (OrderOperationResult, error) {
	query, args, err := sq.Select("login").From(ordersTable).Where(sq.Eq{"order_id": order_id}).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return OrderOperationFailed, fmt.Errorf("failed generate select login query for order %s: %v", order_id, err)
	}

	row := s.db.QueryRow(query, args...)
	if row.Err() != nil {
		return OrderOperationFailed, fmt.Errorf("failed execute select login query for order %s: %v", order_id, err)
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
	query, args, err = sq.Insert(ordersTable).Columns("order_id", "login", "status", "date_upload", "date_update").Values(order_id, login, OrderStatusNew, now, now).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return OrderOperationFailed, fmt.Errorf("failed generate insert order query for order %s: %v", order_id, err)
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
	query, args, err := sq.Select("order_id", "status", "date_upload", "accrual").From(ordersTable).Where(sq.Eq{"login": login}).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed generate select orders query for login %s: %v", login, err)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed execute select orders query for login %s: %v", login, err)
	}
	for rows.Next() {
		var order_id string
		var status OrderStatus
		var upload time.Time
		var accrual int32

		rows.Scan(&order_id, &status, &upload, &accrual)

		order := &Order{
			Number:     order_id,
			Status:     status,
			UploadedAt: upload,
		}
		if accrual > 0 {
			order.Accrual = accrual
		}

		orders = append(orders, order)
	}

	return orders, nil
}
