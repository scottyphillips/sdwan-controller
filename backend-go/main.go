package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type DeviceAnalysis struct {
	Tier              string `json:"tier"`
	IsPrivate         bool   `json:"is_private"`
	ConfidenceScore   int    `json:"confidence_score"`
	SuggestedHostname string `json:"suggested_hostname"`
}

func cleanJSON(s string) string {
	// Remove Markdown code blocks if they exist
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}

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

	url := "http://localhost:11434/api/generate"

	// Our first "Network Engineer" test prompt
	payload := OllamaRequest{
		Model: "deepseek-r1:14b",
		Prompt: `Act as a Senior SD-WAN Architect. Analyze this messy CLI snippet:
         
					interface GigabitEthernet0/0/1
					description WAN_PRIMARY_MPLS
					ip address 10.255.1.2 255.255.255.252
					!
					router bgp 65001
					neighbor 10.255.1.1 remote-as 65000
					description PEER_TO_REGIONAL_HUB
					
					Return ONLY a JSON object: 
					{
					"tier": "Global|Regional|Site|Device",
					"is_private": true/false,
					"confidence_score": 1-100,
					"suggested_hostname": "string",
					"logic": "1-sentence explanation"
					}`,
		Stream: false,
	}

	jsonData, _ := json.Marshal(payload)

	fmt.Println("🚀 Sending request to DeepSeek-R1...")
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	type OllamaResponse struct {
		Response string `json:"response"`
	}

	var olResp OllamaResponse
	json.Unmarshal(body, &olResp)

	cleanedJSON := cleanJSON(olResp.Response)

	// 3. Parse the actual AI analysis into our struct
	var analysis DeviceAnalysis
	err = json.Unmarshal([]byte(cleanedJSON), &analysis)
	if err != nil {
		fmt.Printf("❌ Failed to parse AI JSON: %v\n", err)
	}

	fmt.Printf("✅ Analysis Complete!\nTier: %s\nHostname: %s\nConfidence: %d%%\n",
		analysis.Tier, analysis.SuggestedHostname, analysis.ConfidenceScore)

	// Ollama returns a JSON object with a "response" field
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	fmt.Println("\n🧠 DeepSeek's Analysis:")
	fmt.Println(result["response"])
}
