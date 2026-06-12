# btcwave-web

Website, checkout, and license key API for Bitcoin Wave. Single Go binary with embedded HTML.

## Components

- **Landing page** — product info, pricing, checkout entry
- **Success page** — post-checkout key display with copy-pasteable agent instruction
- **Key API** — issuance, single-use redemption, validation
- **Stripe webhook** — checkout.session.completed → automatic key issuance
- **Health endpoint** — service status + key count

## API

### Redeem a key (CLI calls this during setup)

```
POST /api/keys/redeem
{"key": "WAVE-FULL-7K3M-ABCD"}

→ 200 {"status": "redeemed", "tier": "FULL", "config": {...}}
→ 403 {"error": "key already redeemed"}
```

### Validate a key (check without redeeming)

```
GET /api/keys/validate?key=WAVE-FULL-7K3M-ABCD

→ 200 {"valid": true, "tier": "FULL", "redeemed": false, "revoked": false}
```

### Health

```
GET /api/health

→ 200 {"status": "ok", "keys": 42}
```

## Usage

```sh
./btcwave-web --listen :8080 --data /var/lib/btcwave-web --stripe-webhook-secret whsec_...
```

## Building

```sh
go build -o btcwave-web ./cmd/web/

# Cross-compile for deployment server
GOOS=linux GOARCH=amd64 go build -o btcwave-web-linux ./cmd/web/
```

## Key Format

Keys follow the pattern `WAVE-{TIER}-{4CHAR}-{4CHAR}`, e.g. `WAVE-FULL-7K3M-ABCD`. Character set excludes ambiguous characters (I/O/0/1). Keys are single-use — after redemption the node is fully independent and never contacts the API again.

## License

MIT
