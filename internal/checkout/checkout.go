package checkout

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Handler struct {
	stripeKey string
	priceID   string
	baseURL   string
}

func New(stripeKey, priceID, baseURL string) *Handler {
	return &Handler{
		stripeKey: stripeKey,
		priceID:   priceID,
		baseURL:   strings.TrimRight(baseURL, "/"),
	}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/checkout", h.handleCheckout)
}

func (h *Handler) handleCheckout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	form := url.Values{}
	form.Set("mode", "payment")
	form.Set("line_items[0][price]", h.priceID)
	form.Set("line_items[0][quantity]", "1")
	form.Set("success_url", h.baseURL+"/success?session_id={CHECKOUT_SESSION_ID}")
	form.Set("cancel_url", h.baseURL+"/")

	req, err := http.NewRequest("POST", "https://api.stripe.com/v1/checkout/sessions", strings.NewReader(form.Encode()))
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "internal error"})
		return
	}
	req.SetBasicAuth(h.stripeKey, "")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		writeJSON(w, 502, map[string]string{"error": "stripe unreachable"})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		writeJSON(w, resp.StatusCode, map[string]string{"error": fmt.Sprintf("stripe error: %s", string(body))})
		return
	}

	var session struct {
		URL string `json:"url"`
	}
	json.Unmarshal(body, &session)

	writeJSON(w, 200, map[string]string{"url": session.URL})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	var buf bytes.Buffer
	json.NewEncoder(&buf).Encode(v)
	w.Write(buf.Bytes())
}
