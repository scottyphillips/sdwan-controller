package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
)

func main() {
	// Connection string: matches the admin/strongpassword123 we set in docker-compose
	// We use localhost:5432 because the port is forwarded from the container to WSL
	connStr := "postgres://admin:strongpassword123@localhost:5432/sdwan_core"

	// context.Background() is the "standard" way to start a long-running process in Go
	conn, err := pgx.Connect(context.Background(), connStr)
	if err != nil {
		fmt.Printf("Unable to connect to database: %v\n", err)
		fmt.Println("Check if your Docker containers are running (docker ps)")
		os.Exit(1)
	}
	
	// Ensure the connection closes when the program finishes
	defer conn.Close(context.Background())

	// A quick health check query
	var version string
	err = conn.QueryRow(context.Background(), "SELECT version()").Scan(&version)
	if err != nil {
		fmt.Printf("QueryRow failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("--- SD-WAN CONTROLLER STATUS ---")
	fmt.Println("Successfully connected to the PostgreSQL database!")
	fmt.Printf("DB Version: %s\n", version)
	fmt.Println("---------------------------------")
}