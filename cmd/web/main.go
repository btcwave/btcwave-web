package main

import (
	"embed"
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/btcwave/btcwave-web/internal/api"
	"github.com/btcwave/btcwave-web/internal/keys"
	"github.com/btcwave/btcwave-web/internal/stripe"
)

//go:embed static
var staticFS embed.FS

func main() {
	listen := flag.String("listen", ":8080", "Listen address")
	dataDir := flag.String("data", "/var/lib/btcwave-web", "Data directory for key storage")
	stripeSecret := flag.String("stripe-webhook-secret", "", "Stripe webhook signing secret")
	flag.Parse()

	if err := os.MkdirAll(*dataDir, 0700); err != nil {
		log.Fatalf("Cannot create data dir: %v", err)
	}

	store, err := keys.NewStore(filepath.Join(*dataDir, "keys.json"))
	if err != nil {
		log.Fatalf("Key store init failed: %v", err)
	}

	mux := http.NewServeMux()

	apiHandler := api.New(store)
	apiHandler.Register(mux)

	webhookHandler := stripe.NewWebhookHandler(store, *stripeSecret, func(key *keys.Key) {
		log.Printf("New key issued: %s (%s) for %s", key.Code, key.Tier, key.Email)
	})
	webhookHandler.Register(mux)

	mux.HandleFunc("/", serveIndex)
	mux.HandleFunc("/success", serveSuccess)
	mux.Handle("/static/", http.FileServer(http.FS(staticFS)))

	log.Printf("btcwave-web listening on %s (data: %s, keys: %d)", *listen, *dataDir, store.Count())
	log.Fatal(http.ListenAndServe(*listen, mux))
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	data, _ := staticFS.ReadFile("static/index.html")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

func serveSuccess(w http.ResponseWriter, r *http.Request) {
	data, _ := staticFS.ReadFile("static/success.html")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}
