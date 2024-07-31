package gophmarkthttpserver

import (
	"encoding/json"
	"math/rand"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type OrderStatus string

type AccrualOrder struct {
	Order   string      `json:"order"`
	Status  OrderStatus `json:"status"`
	Accrual float64     `json:"accrual"`
}

const (
	maxPoints float64 = 5

	Registered OrderStatus = "REGISTERED"
	Invalid    OrderStatus = "INVALID"
	Processing OrderStatus = "PROCESSING"
	Processed  OrderStatus = "PROCESSED"
)

var orderStatuses = []OrderStatus{Registered, Invalid, Processing, Processed}

func mockAccrual(w http.ResponseWriter, r *http.Request) {
	order := &AccrualOrder{
		Order:   chi.URLParam(r, "number"),
		Status:  orderStatuses[rand.Intn(len(orderStatuses))],
		Accrual: rand.Float64() * maxPoints,
	}

	body, err := json.Marshal(order)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("failed accrual"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(body)
}
