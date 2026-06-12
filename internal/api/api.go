package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/btcwave/btcwave-web/internal/keys"
)

type Handler struct {
	store *keys.Store
}

func New(store *keys.Store) *Handler {
	return &Handler{store: store}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/keys/redeem", h.handleRedeem)
	mux.HandleFunc("/api/keys/validate", h.handleValidate)
	mux.HandleFunc("/api/health", h.handleHealth)
}

func (h *Handler) handleRedeem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}

	var req struct {
		Key string `json:"key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Key == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "key required"})
		return
	}

	key, err := h.store.Redeem(req.Key)
	if err != nil {
		log.Printf("Redeem failed for %s: %v", req.Key, err)
		writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
		return
	}

	log.Printf("Key redeemed: %s (tier: %s)", key.Code, key.Tier)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "redeemed",
		"tier":   key.Tier,
		"config": tierConfig(key.Tier),
	})
}

func (h *Handler) handleValidate(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("key")
	if code == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "key param required"})
		return
	}

	key, err := h.store.Validate(code)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"valid":    true,
		"tier":     key.Tier,
		"redeemed": key.RedeemedAt != nil,
		"revoked":  key.Revoked,
	})
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "ok",
		"keys":   h.store.Count(),
	})
}

func tierConfig(tier keys.Tier) map[string]interface{} {
	switch tier {
	case keys.TierFull:
		return map[string]interface{}{
			"tier":          "full",
			"pruned":        false,
			"txindex":       true,
			"tor":           true,
			"zmq":           true,
			"spam_filtering": true,
			"dashboard":     true,
			"lightning":     true,
			"electrum":      true,
			"btcpay":        true,
		}
	default:
		return map[string]interface{}{"tier": string(tier)}
	}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
