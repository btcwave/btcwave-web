package main

import (
	"embed"
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/btcwave/btcwave-web/internal/api"
	"github.com/btcwave/btcwave-web/internal/checkout"
	"github.com/btcwave/btcwave-web/internal/keys"
	"github.com/btcwave/btcwave-web/internal/stripe"
)

//go:embed static
var staticFS embed.FS

func main() {
	listen := flag.String("listen", ":8080", "Listen address")
	dataDir := flag.String("data", "/var/lib/btcwave-web", "Data directory for key storage")
	stripeKey := flag.String("stripe-key", "", "Stripe secret key (sk_test_... or sk_live_...)")
	stripePriceID := flag.String("stripe-price", "", "Stripe price ID for the $49 license")
	stripeWebhookSecret := flag.String("stripe-webhook-secret", "", "Stripe webhook signing secret")
	baseURL := flag.String("base-url", "https://btcwave.app", "Public base URL for redirects")
	flag.Parse()

	if *stripeKey == "" {
		*stripeKey = os.Getenv("STRIPE_KEY")
	}
	if *stripePriceID == "" {
		*stripePriceID = os.Getenv("STRIPE_PRICE")
	}
	if *stripeWebhookSecret == "" {
		*stripeWebhookSecret = os.Getenv("STRIPE_WEBHOOK_SECRET")
	}
	if *baseURL == "https://btcwave.app" {
		if v := os.Getenv("BASE_URL"); v != "" {
			*baseURL = v
		}
	}

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

	webhookHandler := stripe.NewWebhookHandler(store, *stripeWebhookSecret, func(key *keys.Key) {
		log.Printf("New key issued: %s (%s) for %s", key.Code, key.Tier, key.Email)
	})
	webhookHandler.Register(mux)

	if *stripeKey != "" && *stripePriceID != "" {
		checkoutHandler := checkout.New(*stripeKey, *stripePriceID, *baseURL)
		checkoutHandler.Register(mux)
	}

	mux.HandleFunc("/", serveIndex)
	mux.HandleFunc("/success", serveSuccess)
	mux.HandleFunc("/privacy", serveStatic("static/privacy.html"))
	mux.HandleFunc("/terms", serveStatic("static/terms.html"))
	mux.HandleFunc("/docs", serveStatic("static/docs/index.html"))
	mux.HandleFunc("/docs/", serveStatic("static/docs/index.html"))
	mux.HandleFunc("/docs/vps-setup", serveStatic("static/docs/vps-setup.html"))
	mux.HandleFunc("/docs/raspberry-pi", serveStatic("static/docs/raspberry-pi.html"))
	mux.HandleFunc("/docs/hardware", serveStatic("static/docs/hardware.html"))
	mux.HandleFunc("/docs/how-it-works", serveStatic("static/docs/how-it-works.html"))
	mux.HandleFunc("/agent.txt", serveStaticPlain("static/agent.txt"))
	mux.HandleFunc("/llms.txt", serveStaticPlain("static/llms.txt"))
	mux.HandleFunc("/robots.txt", serveStaticPlain("static/robots.txt"))
	mux.HandleFunc("/sitemap.xml", serveStaticXML("static/sitemap.xml"))
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

func serveStatic(path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := staticFS.ReadFile(path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
	}
}

func serveStaticPlain(path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := staticFS.ReadFile(path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write(data)
	}
}

func serveStaticXML(path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := staticFS.ReadFile(path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.Write(data)
	}
}
