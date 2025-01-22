package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/umit144/notification-service/database"
)

const (
	MaxRetryAttempts    = 3
	RetryDelay          = 2 * time.Second
	RequestTimeout      = 10 * time.Second
	RedisAddress        = "localhost:6379"
	RedisPassword       = ""
	RedisDB             = 0
	NotificationChannel = "notifications.subscription.updated"
)

type SubscriptionEvent struct {
	AppID    int64  `json:"appId"`
	DeviceID int64  `json:"deviceId"`
	Event    string `json:"event"`
}

type Service struct {
	Database *database.Client
	Redis    *redis.Client
	Logger   *log.Logger
	client   *http.Client
}

func NewService(dbClient *database.Client, logger *log.Logger) (*Service, error) {
	return &Service{
		Database: dbClient,
		Redis:    connectToRedis(),
		Logger:   logger,
		client:   &http.Client{Timeout: RequestTimeout},
	}, nil
}

func connectToRedis() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     RedisAddress,
		Password: RedisPassword,
		DB:       RedisDB,
	})
}

func (s *Service) Start(ctx context.Context) error {
	subscription := s.Redis.Subscribe(ctx, NotificationChannel)
	defer subscription.Close()

	s.Logger.Println("Service started. Listening to Redis channel...")

	for msg := range subscription.Channel() {
		if err := s.processMessage(ctx, msg); err != nil {
			s.Logger.Printf("Message processing error: %v", err)
		}
	}

	return nil
}

func (s *Service) processMessage(ctx context.Context, msg *redis.Message) error {
	var event SubscriptionEvent
	if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
		return fmt.Errorf("message parse error: %v", err)
	}

	s.Logger.Printf("New event received: AppID: %d, DeviceID: %d, Event: %s",
		event.AppID, event.DeviceID, event.Event)

	endpoints, err := s.Database.GetCallbackURLs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get callback URLs: %v", err)
	}

	var wg sync.WaitGroup
	for _, endpoint := range endpoints {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			s.sendNotificationWithRetry(ctx, url, event)
		}(endpoint)
	}
	wg.Wait()

	return nil
}

func (s *Service) sendNotificationWithRetry(ctx context.Context, endpoint string, event SubscriptionEvent) {
	payload, err := json.Marshal(event)
	if err != nil {
		s.Logger.Printf("JSON marshaling error: %v", err)
		return
	}

	for attempt := 1; attempt <= MaxRetryAttempts; attempt++ {
		err := s.sendRequest(ctx, endpoint, payload)
		if err == nil {
			s.Logger.Printf("Successfully sent notification to %s on attempt %d", endpoint, attempt)
			return
		}

		s.Logger.Printf("Attempt %d failed for %s: %v", attempt, endpoint, err)

		if attempt < MaxRetryAttempts {
			time.Sleep(RetryDelay)
			continue
		}

		s.Logger.Printf("Max retry attempts reached for endpoint %s after %d attempts", endpoint, MaxRetryAttempts)
		return
	}
}

func (s *Service) sendRequest(ctx context.Context, endpoint string, payload []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "NotificationService/1.0")

	s.Logger.Printf("[POST] Sending to %s - %s", endpoint, string(payload))

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}
