package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
)

type chargeRequest struct {
	AccountID int     `json:"account_id"`
	Items     []int64 `json:"items"` // line item amounts in cents
}

type chargeHandler struct {
	db *DB
}

func (h *chargeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req chargeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.AccountID <= 0 || len(req.Items) == 0 {
		http.Error(w, `{"error":"invalid charge"}`, http.StatusBadRequest)
		return
	}

	var total int64
	for _, amount := range req.Items {
		total += amount
	}

	id := newChargeID()
	if err := h.db.recordCharge(r.Context(), id, req.AccountID, total, req.Items); err != nil {
		slog.Error("charge failed", "charge_id", id, "err", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"id": id, "amount": total})
}

func newChargeID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return "ch_" + hex.EncodeToString(b)
}
