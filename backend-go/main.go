package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

func main() {
	// 1. Load the .env file
	// Inside main.go
	err := godotenv.Load("../.env") // This tells Go to step out of backend-go and find the file
	if err != nil {
		fmt.Println("Error loading .env file from root")
	}

	// 2. Pull the secret from the environment instead of hardcoding it
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		fmt.Println("DATABASE_URL not set in environment")
		os.Exit(1)
	}

	conn, err := pgx.Connect(context.Background(), connStr)
	if err != nil {
		fmt.Printf("Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())

	fmt.Println("Successfully connected using Environment Variables!")
}
