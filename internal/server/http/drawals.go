package gophmarkthttpserver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	luhn "github.com/EClaesson/go-luhn"

	storage "github.com/zvfkjytytw/gophmarkt/internal/server/storage"
)

func (h *HTTPServer) drawalsPut(w http.ResponseWriter, r *http.Request) {
	login := fmt.Sprintf("%v", r.Context().Value(contextAuthUser))
	contentType, ok := r.Header["Content-Type"]
	if !ok || contentType[0] != "application/json" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("wrong Content-Type. Expect application/json"))
		return
	}

	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("failed read body"))
		return
	}

	var drawal storage.Drawal
	err = json.Unmarshal(body, &drawal)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("failed unmarshal body"))
		return
	}

	ok, err = luhn.IsValid(drawal.Order)
	if err != nil {
		h.logger.Sugar().Errorf("failed upload order %s: %v", drawal.Order, err)
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte("invalid order number format"))
		return
	}

	if !ok {
		h.logger.Sugar().Errorf("failed upload order %s: invalid format", drawal.Order)
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte("invalid order number format"))
		return
	}

	status, err := h.storage.AddDrawal(r.Context(), drawal.Order, login, drawal.Sum)
	if err != nil {
		switch status {
		case storage.DrawalAddBefore:
			h.logger.Sugar().Errorf("failed upload drawal order %s: %v", drawal.Order, err)
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(fmt.Sprintf("drawal order %s is already upload", drawal.Order)))
			return
		case storage.DrawalAddByOther:
			h.logger.Sugar().Errorf("failed upload drawal order %s: %v", drawal.Order, err)
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(fmt.Sprintf("drawal order %s is upload by other", drawal.Order)))
			return
		case storage.DrawalNotEnoughPoints:
			h.logger.Sugar().Errorf("failed upload drawal order %s: %v", drawal.Order, err)
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(fmt.Sprintf("not enough points on balance for login %s", login)))
			return
		case storage.DrawalOperationFailed:
			h.logger.Sugar().Errorf("failed upload drawal order %s: %v", drawal.Order, err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("drawal order %s is not upload", drawal.Order)))
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Drawal order %s is upload", drawal.Order)))
}

func (h *HTTPServer) drawalsGet(w http.ResponseWriter, r *http.Request) {
	login := fmt.Sprintf("%v", r.Context().Value(contextAuthUser))

	drawals, err := h.storage.GetDrawals(r.Context(), login)
	if err != nil {
		h.logger.Sugar().Errorf("failed get drawal orders for %s: %v", login, err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("failed get drawal orders for %s", login)))
		return
	}

	if len(drawals) == 0 {
		w.WriteHeader(http.StatusNoContent)
		w.Write([]byte(fmt.Sprintf("no drawal orders for %s", login)))
		return
	}

	body, err := json.Marshal(drawals)
	if err != nil {
		h.logger.Sugar().Errorf("failed marshaling drawal orders for %s: %v", login, err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("failed get drawal orders for %s", login)))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(body)
}
