package gophmarktaccrual

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	storage "github.com/zvfkjytytw/gophmarkt/internal/server/storage"
)

var tooManyRequests error

type OrderStatus string

type AccrualOrder struct {
	Order   string      `json:"order"`
	Status  OrderStatus `json:"status"`
	Accrual float64     `json:"accrual"`
}

const (
	accrualHandler      = `/api/orders/%s`
	accrualInterval     = 5000
	waitAfterToMany     = 1000
	waitBetweenRequests = 50

	orderRegistered OrderStatus = "REGISTERED"
	orderInvalid    OrderStatus = "INVALID"
	orderProcessing OrderStatus = "PROCESSING"
	orderProcessed  OrderStatus = "PROCESSED"
)

var accrualToStorage = map[OrderStatus]storage.OrderStatus{
	orderInvalid:    storage.OrderStatusInvalid,
	orderProcessing: storage.OrderStatusProcessing,
	orderProcessed:  storage.OrderStatusProcessed,
}

type Accrual struct {
	address string
	client  http.Client
	storage *storage.PGStorage
	logger  *zap.Logger
	stop    chan struct{}
}

func NewAccrual(address string, storage *storage.PGStorage, logger *zap.Logger) (*Accrual, error) {
	tr := &http.Transport{
		MaxIdleConns:    1,
		IdleConnTimeout: 60 * time.Second,
	}
	client := http.Client{Transport: tr}

	stop := make(chan struct{})

	return &Accrual{
		address: address,
		client:  client,
		storage: storage,
		logger:  logger,
		stop:    stop,
	}, nil
}

func (a *Accrual) Start(ctx context.Context) error {
	accrualTicker := time.NewTicker(accrualInterval * time.Millisecond)
	defer accrualTicker.Stop()

	for {
		select {
		case <-a.stop:
			return nil
		case <-accrualTicker.C:
			a.checkOrders(ctx)
		}
	}
}

func (a *Accrual) Stop(ctx context.Context) error {
	close(a.stop)
	a.client.CloseIdleConnections()

	return nil
}

func (a *Accrual) checkOrders(ctx context.Context) {
	orders, err := a.storage.GetUnprocessedOrders(ctx)
	if err != nil {
		a.logger.Sugar().Errorf("failed get unprocessed orders: %v", err)
		return
	}

	for _, order := range orders {
		if ctx.Err() != nil {
			a.logger.Error("accrual contex error")
			return
		}

		err := a.checkOrder(ctx, order)
		if err == tooManyRequests {
			a.logger.Error("too many requests to accrual server")
			time.Sleep(waitAfterToMany * time.Second)
			return
		}
		if err != nil {
			a.logger.Sugar().Errorf("failed check order %s: %v", order.Number, err)
		}

		time.Sleep(waitBetweenRequests * time.Millisecond)
	}
}

func (a *Accrual) checkOrder(ctx context.Context, order *storage.Order) error {
	var body string
	req, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("%s%s", a.address, fmt.Sprintf(accrualHandler, order.Number)),
		strings.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("failed init request for order %s: %v", order.Number, err)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed request accrual data from %s for order %s: %v", a.address, order.Number, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return tooManyRequests
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed read accrual data for order %s: %v", order.Number, err)
	}

	var accrualOrder AccrualOrder
	err = json.Unmarshal(bodyBytes, &accrualOrder)
	if err != nil {
		return fmt.Errorf("failed unmarshal accrual data for order %s: %v", order.Number, err)
	}

	if accrualOrder.Order != order.Number {
		return fmt.Errorf("orders numbers is not equal send %s receive %s ", order.Number, accrualOrder.Order)
	}

	if accrualOrder.Status == orderRegistered {
		return nil
	}

	if accrualToStorage[accrualOrder.Status] != order.Status {
		newOrder := &storage.Order{
			Number:     accrualOrder.Order,
			Status:     accrualToStorage[accrualOrder.Status],
			Accrual:    accrualOrder.Accrual,
			UploadedAt: time.Now(),
		}

		err := a.storage.UpdateOrder(ctx, newOrder)
		if err != nil {
			return err
		}
	}

	return nil
}
