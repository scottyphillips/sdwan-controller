package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// DeviceAudit matches our Postgres table schema
type DeviceAudit struct {
	IP         string
	Tier       string
	Hostname   string
	Confidence int
	Logic      string
}

// OllamaRequest is the payload sent to the Local AI
type OllamaRequest struct {
	Model   string                 `json:"model"`
	Prompt  string                 `json:"prompt"`
	Stream  bool                   `json:"stream"`
	Options map[string]interface{} `json:"options"`
}

// DeviceAnalysis matches the JSON structure we want from the AI
type DeviceAnalysis struct {
	Tier              string `json:"tier"`
	IsPrivate         bool   `json:"is_private"`
	ConfidenceScore   int    `json:"confidence_score"`
	SuggestedHostname string `json:"suggested_hostname"`
	SuggestedLogic    string `json:"suggested_logic"`
}

// loadEnv loads .env from project root (works when run from repo root or backend-go/)
func loadEnv() {
	_ = godotenv.Load(".env")
	if _, set := os.LookupEnv("POSTGRES_USER"); !set {
		_ = godotenv.Load("../.env")
	}
}

// cleanJSON strips the <think> tags and Markdown formatting from DeepSeek
func cleanJSON(s string) string {
	// Remove the <think>...</think> block entirely
	reThink := regexp.MustCompile(`(?s)<think>.*?</think>`)
	s = reThink.ReplaceAllString(s, "")

	// Remove Markdown code blocks
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")

	return strings.TrimSpace(s)
}

func getDB() (*sql.DB, error) {
	connStr := fmt.Sprintf("host=localhost port=5432 user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_DB"),
	)
	return sql.Open("postgres", connStr)
}

func saveToDB(audit DeviceAudit) error {
	db, err := getDB()
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec(
		`INSERT INTO core.device_audits (ip_address, tier, hostname, confidence, logic)
		 VALUES ($1, $2, $3, $4, $5)`,
		audit.IP, audit.Tier, audit.Hostname, audit.Confidence, audit.Logic,
	)
	if err != nil {
		return fmt.Errorf("insert: %w", err)
	}
	return nil
}

// redactPasswords replaces encrypted-password values in JSON with [REDACTED] before sending to LLM.
func redactPasswords(b []byte) []byte {
	return regexp.MustCompile(`"\$[0-9]\$[^"]*"`).ReplaceAll(b, []byte(`"[REDACTED]"`))
}

func runAnalyzeConfig() {
	db, err := getDB()
	if err != nil {
		log.Fatalf("❌ DB: %v", err)
	}
	defer db.Close()

	var hostname string
	var factsJSON []byte
	err = db.QueryRow(`
		SELECT d.hostname, df.facts
		FROM core.device_facts df
		JOIN core.devices d ON d.id = df.device_id
		ORDER BY df.gathered_at DESC
		LIMIT 1
	`).Scan(&hostname, &factsJSON)
	if err != nil {
		log.Fatalf("❌ No device_facts found or DB error: %v", err)
	}

	var facts map[string]interface{}
	if err := json.Unmarshal(factsJSON, &facts); err != nil {
		log.Fatalf("❌ Failed to parse facts JSON: %v", err)
	}

	runningConfig, _ := facts["running_config"].(map[string]interface{})
	if runningConfig == nil {
		log.Fatalf("❌ No running_config in latest device_facts (device: %s)", hostname)
	}

	configBytes, err := json.MarshalIndent(runningConfig, "", "  ")
	if err != nil {
		log.Fatalf("❌ Marshal config: %v", err)
	}
	configStr := string(redactPasswords(configBytes))

	prompt := fmt.Sprintf(`You are an expert at Junos and SD-WAN design. Analyze this Junos running configuration (device: %s) and respond in plain language with:

1. **Topology**: What is this device (edge, hub, vSRX role)? What interfaces and tunnels do you see?
2. **VRFs / segmentation**: List any VRFs or routing instances and what they seem to be used for.
3. **Security**: Summarize security zones and policies (trust/untrust, etc.).
4. **Fit for SD-WAN**: How would you map this into a Realm / Organization / Department (Context) model? Any suggestions or risks?

Configuration (JSON, passwords redacted):
%s`, hostname, configStr)

	url := "http://localhost:11434/api/generate"
	payload := OllamaRequest{
		Model:  "deepseek-r1:14b",
		Prompt: prompt,
		Stream: false,
		Options: map[string]interface{}{
			"temperature": 0.3,
			"num_predict": 2048,
		},
	}
	jsonData, _ := json.Marshal(payload)
	fmt.Printf("📥 Loaded config for %s (%d bytes, redacted).\n", hostname, len(configStr))
	fmt.Println("🧠 Sending to LLM for analysis...")
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatalf("❌ LLM request: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var olResp struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(body, &olResp); err != nil {
		log.Fatalf("❌ LLM response parse: %v", err)
	}
	fmt.Println("\n--- LLM analysis ---\n")
	fmt.Println(strings.TrimSpace(olResp.Response))
	fmt.Println("\n--- end ---")
}

func main() {
	loadEnv()
	analyze := flag.Bool("analyze", false, "Load latest device config from DB and send to LLM for analysis")
	flag.Parse()

	if *analyze {
		fmt.Println("🚀 SD-WAN Controller — Config → LLM analysis")
		runAnalyzeConfig()
		return
	}

	fmt.Println("🚀 Starting SD-WAN Controller...")

	// 2. Prepare AI Request
	url := "http://localhost:11434/api/generate"
	payload := OllamaRequest{
		Model: "deepseek-r1:14b",
		Prompt: `You are a Deterministic Network Parser. 
                Rule 1: If 'BGP' and 'Regional Hub' are present, it is 'Regional'.
                Rule 2: If '10.x.x.x/24' is present with no BGP, it is 'Site'.

                Analyze this:
                interface GigabitEthernet0/0/1
                description WAN_PRIMARY_MPLS
                ip address 10.255.1.2 255.255.255.252
                !
                router bgp 65001
                neighbor 10.255.1.1 remote-as 65000
                description PEER_TO_REGIONAL_HUB

                Return ONLY JSON:
                {
                    "tier": "string",
                    "is_private": bool,
                    "confidence_score": int,
                    "suggested_hostname": "string",
                    "suggested_logic": "string"
                }`,
		Stream: false,
		Options: map[string]interface{}{
			"temperature": 0.0,
			"seed":        42,
		},
	}

	// 3. Send to Ollama
	jsonData, _ := json.Marshal(payload)
	fmt.Println("🧠 Sending request to DeepSeek-R1 (Deterministic Mode)...")

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatalf("❌ AI Connection Error: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// 4. Extract Response Content
	var olResp struct {
		Response string `json:"response"`
	}
	json.Unmarshal(body, &olResp)

	// 5. Clean and Parse AI Result
	cleaned := cleanJSON(olResp.Response)
	var analysis DeviceAnalysis
	if err := json.Unmarshal([]byte(cleaned), &analysis); err != nil {
		log.Fatalf("❌ Failed to parse AI JSON: %v\nDEBUG - Cleaned output: %s", err, cleaned)
	}

	fmt.Printf("✅ Analysis Complete!\nTier: %s\nHostname: %s\nConfidence: %d%%\n",
		analysis.Tier, analysis.SuggestedHostname, analysis.ConfidenceScore)

	// 6. Push to Database
	audit := DeviceAudit{
		IP:         "10.255.1.2",
		Tier:       analysis.Tier,
		Hostname:   analysis.SuggestedHostname,
		Confidence: analysis.ConfidenceScore,
		Logic:      analysis.SuggestedLogic,
	}
	if err := saveToDB(audit); err != nil {
		log.Fatalf("❌ Database: %v", err)
	}
	fmt.Println("💾 Audit successfully logged to Postgres!")
}
