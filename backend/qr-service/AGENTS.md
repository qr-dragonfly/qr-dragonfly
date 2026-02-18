# AGENTS.md

This file provides guidance to WARP (warp.dev) when working with code in this repository.

## Build & Run Commands

```bash
# Run the server (in-memory storage)
go run ./cmd/server

# Run with Postgres persistence
DATABASE_URL='postgres://user:pass@host:5432/db?sslmode=disable' go run ./cmd/server

# Run tests
go test ./...

# Run a single test
go test ./internal/store -run TestMemoryStore

# Build binary
go build -o server ./cmd/server
```

## Environment Variables

- `PORT` - Server port (default: 8080)
- `DATABASE_URL` - Postgres connection string; if unset, uses in-memory storage
- `CORS_ALLOW_ORIGINS` - Comma-separated allowed origins (default: http://localhost:5173)
- `ADMIN_API_KEY` - Key for admin endpoints (optional)

## Architecture

### Store Interface Pattern
The service uses a `Store` interface (`internal/store/store.go`) with two implementations:
- `MemoryStore` - In-memory storage for development/testing
- `PostgresStore` - GORM-based Postgres storage for production

Storage backend is selected at startup based on `DATABASE_URL` presence. All handlers interact only with the `Store` interface, making it easy to swap implementations.

### Middleware Chain
Request flow in `cmd/server/main.go`:
```
Request → CORS → RateLimiter → Router → Handler
```

Per-route middleware applied in `httpapi.NewRouter`:
```
Recoverer → RequestID → ExposeResponseHeaders → EnforceJSON → Handler
```

### Quota System
User quotas are tier-based via `X-User-Type` header (free/basic/enterprise/admin). Quotas limit:
- `maxActive` - Maximum active QR codes
- `maxTotal` - Maximum total QR codes

Quota checks occur in create/update handlers before mutations.

### API Routes
All routes defined in `internal/httpapi/router.go`:
- `/healthz` - Health check
- `/api/qr-codes` - Collection (GET list, POST create)
- `/api/qr-codes/{id}` - Item (GET, PATCH, DELETE)
- `/api/settings` - User settings (GET, PUT)
- `/api/admin/generate-sample-data` - Admin sample data generation

### URL Validation
Only HTTPS URLs are accepted (`isValidHTTPURL` in router.go). This is intentional - do not relax to allow HTTP.
