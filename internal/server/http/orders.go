package gophmarkthttpserver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	luhn "github.com/EClaesson/go-luhn"

	storage "github.com/zvfkjytytw/gophmarkt/internal/server/storage"
)

func (h *HTTPServer) ordersPut(w http.ResponseWriter, r *http.Request) {
	login := fmt.Sprintf("%v", r.Context().Value(contextAuthUser))
	contentType, ok := r.Header["Content-Type"]
	if !ok || contentType[0] != "text/plain" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("wrong Content-Type. Expect text/plain"))
		return
	}

	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("failed read body"))
		return
	}

	orderID := string(body[:])
	ok, err = luhn.IsValid(orderID)
	if err != nil {
		h.logger.Sugar().Errorf("failed upload order %s: %v", orderID, err)
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte("invalid order number format"))
		return
	}

	if !ok {
		h.logger.Sugar().Errorf("failed upload order %s: invalid format", orderID)
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte("invalid order number format"))
		return
	}

	status, err := h.storage.AddOrder(r.Context(), orderID, login)
	if err != nil {
		switch status {
		case storage.OrderAddBefore:
			h.logger.Sugar().Errorf("failed upload order %s: %v", orderID, err)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fmt.Sprintf("order %s is already exists", orderID)))
			return
		case storage.OrderAddByOther:
			h.logger.Sugar().Errorf("failed upload order %s: %v", orderID, err)
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(fmt.Sprintf("order %s is upload by other", orderID)))
			return
		case storage.OrderOperationFailed:
			h.logger.Sugar().Errorf("failed upload order %s: %v", orderID, err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("order %s is not upload", orderID)))
			return
		}
	}

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(fmt.Sprintf("Order %s is upload", orderID)))
}

func (h *HTTPServer) ordersGet(w http.ResponseWriter, r *http.Request) {
	login := fmt.Sprintf("%v", r.Context().Value(contextAuthUser))

	orders, err := h.storage.GetOrders(r.Context(), login)
	if err != nil {
		h.logger.Sugar().Errorf("failed get orders for %s: %v", login, err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("failed get orders for %s", login)))
		return
	}

	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		w.Write([]byte(fmt.Sprintf("no orders for %s", login)))
		return
	}

	body, err := json.Marshal(orders)
	if err != nil {
		h.logger.Sugar().Errorf("failed marshaling orders for %s: %v", login, err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("failed get orders for %s", login)))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(body)
}
