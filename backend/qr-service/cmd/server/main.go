package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"qr-service/internal/httpapi"
	"qr-service/internal/middleware"
	"qr-service/internal/store"
)

func main() {
	port := envOr("PORT", "8080")
	allowedOrigins := splitCSV(envOr("CORS_ALLOW_ORIGINS", "http://localhost:5173"))
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	adminKey := envOr("ADMIN_API_KEY", "")

	ctx := context.Background()

	var st store.Store
	var closeStore func()
	if databaseURL != "" {
		pg, err := store.NewPostgresStore(ctx, databaseURL)
		if err != nil {
			log.Fatalf("postgres init failed: %v", err)
		}
		st = pg
		closeStore = func() { _ = pg.Close() }
		log.Printf("qr-service using postgres storage")
	} else {
		st = store.NewMemoryStore()
		closeStore = func() {}
		log.Printf("qr-service using in-memory storage (set DATABASE_URL to persist)")
	}

	router := httpapi.NewRouter(httpapi.Server{Store: st, AdminAPIKey: adminKey})

	// Apply middleware layers (order matters!)
	var handler http.Handler = router

	// 1. CORS (outermost)
	handler = httpapi.NewCorsMiddleware(httpapi.CorsOptions{
		AllowedOrigins:   allowedOrigins,
		AllowCredentials: true,
	})(handler)

	// 2. Rate limiting (200 requests per minute per IP for QR service)
	rateLimiter := middleware.NewRateLimiter(200, time.Minute)
	handler = rateLimiter.Middleware(handler)

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Printf("qr-service listening on http://localhost:%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	closeStore()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

func envOr(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}
