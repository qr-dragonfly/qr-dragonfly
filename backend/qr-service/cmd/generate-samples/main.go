package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"qr-service/internal/model"
)

func main() {
	// Get database URL from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable required")
	}

	// Get user ID from command line or use default
	userID := "sample-user"
	if len(os.Args) > 1 {
		userID = os.Args[1]
	}

	// Connect to database
	db, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Sample QR codes to generate
	samples := []struct {
		label string
		url   string
	}{
		{"Product Landing Page", "https://example.com/products/wireless-headphones"},
		{"Marketing Campaign", "https://example.com/promo/summer-sale?utm_source=qr&utm_campaign=summer2026"},
		{"Event Registration", "https://example.com/events/tech-conference-2026/register"},
		{"Restaurant Menu", "https://example.com/menu/downtown-bistro"},
		{"Feedback Survey", "https://forms.example.com/customer-feedback/q12345"},
		{"App Download", "https://example.com/app/download?platform=mobile"},
		{"Contact Card", "https://example.com/contact/john-smith"},
		{"WiFi Access", "https://example.com/wifi/guest-network"},
		{"Document Share", "https://docs.example.com/reports/annual-2025.pdf"},
		{"Video Tutorial", "https://example.com/videos/getting-started-guide"},
	}

	ctx := context.Background()
	now := time.Now().UTC()
	created := 0

	for _, sample := range samples {
		qr := model.QrCode{
			Label:     sample.label,
			URL:       sample.url,
			Active:    true,
			CreatedAt: now,
		}

		result := db.WithContext(ctx).Create(&qr)
		if result.Error != nil {
			log.Printf("Failed to create QR code '%s': %v", sample.label, result.Error)
			continue
		}

		created++
		fmt.Printf("Created: %s (ID: %s)\n", sample.label, qr.ID)
	}

	fmt.Printf("\nSuccessfully created %d sample QR codes for user: %s\n", created, userID)
}
