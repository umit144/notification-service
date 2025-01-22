# Notification Service

A notification service that listens to Redis pub/sub events and forwards them to registered callback URLs. This service works in conjunction with a Laravel application that manages the database.

## Architecture

- Laravel Application (subscription-management-api)
  - Manages the database
  - Stores callback URLs
  - Handles business logic

- Notification Service (This App)
  - Subscribes to Redis events
  - Fetches callback URLs from Laravel's database
  - Handles notification delivery with retry mechanism

## Features

- Redis pub/sub subscription
- Reads callback URLs from subscription-management-api's MySQL database
- Concurrent notification processing
- Retry mechanism (3 attempts) for failed requests
- HTTP request timeout handling

## Installation

```bash
git clone github.com/umit144/notification-service
cd notification-service
go mod download
```

## Configuration

Make sure the following settings match your Laravel environment:

```go
RedisAddress  = "localhost:6379"
RedisPassword = ""
RedisDB      = 0
DatabaseDSN   = "sail:password@tcp(localhost:3306)/laravel"
```

## Running

```bash
go run main.go
```