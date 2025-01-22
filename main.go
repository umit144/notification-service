package main

import (
	"context"
	"log"

	"github.com/umit144/notification-service/database"
	"github.com/umit144/notification-service/notification"
)

func main() {
	logger := log.New(log.Writer(), "[NOTIFICATION-SERVICE] ", log.LstdFlags)

	dbClient, err := database.NewClient()
	if err != nil {
		logger.Fatalf("Database initialization error: %v", err)
	}
	defer dbClient.Close()

	service, err := notification.NewService(dbClient, logger)
	if err != nil {
		logger.Fatalf("Service initialization error: %v", err)
	}
	defer service.Redis.Close()

	ctx := context.Background()
	if err := service.Start(ctx); err != nil {
		logger.Fatalf("Service runtime error: %v", err)
	}
}
