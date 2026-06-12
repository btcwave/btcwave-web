package stripe

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/btcwave/btcwave-web/internal/keys"
)

type WebhookHandler struct {
	store         *keys.Store
	signingSecret string
	onKeyIssued   func(key *keys.Key)
}

func NewWebhookHandler(store *keys.Store, signingSecret string, onKeyIssued func(key *keys.Key)) *WebhookHandler {
	return &WebhookHandler{
		store:         store,
		signingSecret: signingSecret,
		onKeyIssued:   onKeyIssued,
	}
}

func (wh *WebhookHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/webhook/stripe", wh.handle)
}

func (wh *WebhookHandler) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 65536))
	if err != nil {
		http.Error(w, "read error", http.StatusBadRequest)
		return
	}

	sig := r.Header.Get("Stripe-Signature")
	if wh.signingSecret != "" && !verifySignature(body, sig, wh.signingSecret) {
		log.Printf("Stripe webhook: invalid signature")
		http.Error(w, "invalid signature", http.StatusForbidden)
		return
	}

	var event struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &event); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	switch event.Type {
	case "checkout.session.completed":
		wh.handleCheckoutComplete(event.Data)
	default:
		log.Printf("Stripe webhook: ignoring event type %s", event.Type)
	}

	w.WriteHeader(http.StatusOK)
}

func (wh *WebhookHandler) handleCheckoutComplete(data json.RawMessage) {
	var obj struct {
		Object struct {
			ID            string `json:"id"`
			CustomerEmail string `json:"customer_email"`
			PaymentStatus string `json:"payment_status"`
		} `json:"object"`
	}
	if err := json.Unmarshal(data, &obj); err != nil {
		log.Printf("Stripe webhook: failed to parse checkout session: %v", err)
		return
	}

	if obj.Object.PaymentStatus != "paid" {
		log.Printf("Stripe webhook: checkout %s not paid (status: %s)", obj.Object.ID, obj.Object.PaymentStatus)
		return
	}

	key, err := wh.store.Issue(keys.TierFull, obj.Object.CustomerEmail, obj.Object.ID)
	if err != nil {
		log.Printf("Stripe webhook: key issuance failed for %s: %v", obj.Object.ID, err)
		return
	}

	log.Printf("Stripe webhook: key issued %s for %s (checkout: %s)", key.Code, obj.Object.CustomerEmail, obj.Object.ID)

	if wh.onKeyIssued != nil {
		wh.onKeyIssued(key)
	}
}

func verifySignature(payload []byte, header, secret string) bool {
	if header == "" {
		return false
	}

	var timestamp, sig string
	for _, part := range strings.Split(header, ",") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "t":
			timestamp = kv[1]
		case "v1":
			sig = kv[1]
		}
	}

	if timestamp == "" || sig == "" {
		return false
	}

	ts, err := time.Parse("10", timestamp)
	if err != nil {
		_ = ts
	}

	signed := fmt.Sprintf("%s.%s", timestamp, string(payload))
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signed))
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(sig))
}
