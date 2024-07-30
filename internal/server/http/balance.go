package gophmarkthttpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func (h *HTTPServer) balanceGet(w http.ResponseWriter, r *http.Request) {
	login := fmt.Sprintf("%v", r.Context().Value(contextAuthUser))

	balance, err := h.storage.GetBalance(login)
	if err != nil {
		h.logger.Sugar().Errorf("failed get balance for %s: %v", login, err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("failed get balance for %s", login)))
		return
	}

	body, err := json.Marshal(balance)
	if err != nil {
		h.logger.Sugar().Errorf("failed marshaling balance for %s: %v", login, err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("failed get balance for %s", login)))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(body)
}
