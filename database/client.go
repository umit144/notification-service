package database

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

const (
	DatabaseDSN = "sail:password@tcp(localhost:3306)/laravel"
)

type Client struct {
	db *sql.DB
}

func NewClient() (*Client, error) {
	db, err := sql.Open("mysql", DatabaseDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %v", err)
	}

	if err = db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("database ping failed: %v", err)
	}

	return &Client{db: db}, nil
}

func (c *Client) Close() error {
	return c.db.Close()
}

func (c *Client) GetCallbackURLs(ctx context.Context) ([]string, error) {
	rows, err := c.db.QueryContext(ctx, "SELECT endpoint_url FROM callbacks")
	if err != nil {
		return nil, fmt.Errorf("database query error: %v", err)
	}
	defer rows.Close()

	var urls []string
	for rows.Next() {
		var url string
		if err := rows.Scan(&url); err != nil {
			return nil, fmt.Errorf("database scan error: %v", err)
		}
		urls = append(urls, url)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("database iteration error: %v", err)
	}

	return urls, nil
}
