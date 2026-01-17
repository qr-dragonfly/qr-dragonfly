package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"user-service/internal/cognito"
	"user-service/internal/httpapi"
	"user-service/internal/stripe"
)

func main() {
	port := envOr("PORT", "8081")
	allowedOrigins := splitCSV(envOr("CORS_ALLOW_ORIGINS", "http://localhost:5173"))

	region := envOr("AWS_REGION", "us-east-1")
	userPoolID := envOr("COGNITO_USER_POOL_ID", "")
	clientID := envOr("COGNITO_CLIENT_ID", "")
	clientSecret := envOr("COGNITO_CLIENT_SECRET", "")
	adminKey := envOr("ADMIN_API_KEY", "")

	cookieSecure := envBool("COOKIE_SECURE", false)
	sameSite := parseSameSite(envOr("COOKIE_SAMESITE", "Lax"))

	// Stripe config (optional)
	stripeSecretKey := envOr("STRIPE_SECRET_KEY", "")
	stripeWebhookSecret := envOr("STRIPE_WEBHOOK_SECRET", "")
	stripeBasicPriceID := envOr("STRIPE_BASIC_PRICE_ID", "")
	stripeEnterprisePriceID := envOr("STRIPE_ENTERPRISE_PRICE_ID", "")
	stripeSuccessURL := envOr("STRIPE_SUCCESS_URL", "http://localhost:5173/subscription?success=true")
	stripeCancelURL := envOr("STRIPE_CANCEL_URL", "http://localhost:5173/subscription")
	stripePortalReturnURL := envOr("STRIPE_PORTAL_RETURN_URL", "http://localhost:5173/account")

	if userPoolID == "" || clientID == "" {
		log.Fatal("missing required env: COGNITO_USER_POOL_ID and/or COGNITO_CLIENT_ID")
	}

	ctx := context.Background()
	awsClient, err := cognito.NewAWSClient(ctx, cognito.AWSConfig{Region: region})
	if err != nil {
		log.Fatalf("aws config error: %v", err)
	}

	var stripeClient *stripe.Client
	if stripeSecretKey != "" && stripeWebhookSecret != "" {
		stripeClient = stripe.NewClient(stripe.Config{
			SecretKey:         stripeSecretKey,
			WebhookSecret:     stripeWebhookSecret,
			BasicPriceID:      stripeBasicPriceID,
			EnterprisePriceID: stripeEnterprisePriceID,
			SuccessURL:        stripeSuccessURL,
			PortalReturnURL:   stripePortalReturnURL,
			CancelURL:         stripeCancelURL,
		})
		log.Printf("stripe configured with basic price: %s, enterprise price: %s", stripeBasicPriceID, stripeEnterprisePriceID)
	} else {
		log.Printf("stripe not configured (missing STRIPE_SECRET_KEY or STRIPE_WEBHOOK_SECRET)")
	}

	router := httpapi.NewRouter(httpapi.Server{
		Cognito:        awsClient,
		UserPoolID:     userPoolID,
		ClientID:       clientID,
		ClientSecret:   clientSecret,
		AdminAPIKey:    adminKey,
		CookieSecure:   cookieSecure,
		CookieSameSite: sameSite,
		StripeClient:   stripeClient,
	})

	handler := httpapi.NewCorsMiddleware(httpapi.CorsOptions{
		AllowedOrigins:   allowedOrigins,
		AllowCredentials: true,
	})(router)

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Printf("user-service listening on http://localhost:%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
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

func envBool(key string, fallback bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func parseSameSite(raw string) http.SameSite {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "none":
		return http.SameSiteNoneMode
	case "strict":
		return http.SameSiteStrictMode
	case "lax", "":
		return http.SameSiteLaxMode
	default:
		return http.SameSiteLaxMode
	}
}
