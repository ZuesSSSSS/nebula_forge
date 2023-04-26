package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-ini/ini"
)

type RustScanResult struct {
	Host    string `json:"host"`
	Port    int    `json:"port"`
	Service string `json:"service"`
	Status  string `json:"status"`
}

func main() {
	apiKey, port := readConfig("config.ini")

	router := gin.Default()

	// Serve static files
	router.Static("/static", "./static")

	router.GET("/", serveIndex)
	router.GET("/rustscan", createRustScanHandler(apiKey))

	fmt.Printf("Serving on http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}

func readConfig(configFile string) (string, string) {
	cfg, err := ini.Load(configFile)
	if err != nil {
		log.Fatalf("Failed to load config file: %v", err)
	}

	apiKey := cfg.Section("").Key("API_KEY").String()
	port := cfg.Section("").Key("PORT").String()

	return strings.TrimSpace(apiKey), strings.TrimSpace(port)
}

func serveIndex(c *gin.Context) {
	c.File("index.html")
}

func createRustScanHandler(apiKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !validateAPIKey(c, apiKey) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
			return
		}

		args := strings.TrimSpace(c.Query("args"))

		rustScanOutput, err := runRustScan(args)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to run RustScan: %v\n", err)})
			return
		}

		// Write the RustScan output to the response
		writeJSONResponse(c.Writer, http.StatusOK, rustScanOutput)
	}
}

func validateAPIKey(c *gin.Context, apiKey string) bool {
	return c.Query("api_key") == apiKey
}

func parseRustScanResults(rustScanOutput string) []RustScanResult {
	results := make([]RustScanResult, 0)

	// Split RustScan output into individual lines
	lines := strings.Split(strings.TrimSpace(rustScanOutput), "\n")

	// Loop through each line of RustScan output
	for _, line := range lines {
		// Skip any lines that only contain whitespace
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Split each line into individual fields
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		// Extract host, port, and service from fields
		host := fields[0]
		port, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}
		service := fields[2]

		// Extract status if it exists
		var status string
		if len(fields) > 3 {
			status = fields[3]
		}

		// Create RustScanResult struct and append to results array
		result := RustScanResult{
			Host:    host,
			Port:    port,
			Service: service,
			Status:  status,
		}
		results = append(results, result)
	}

	return results
}

func writeJSONResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")

	// Check if data is an empty array and return an empty JSON object if so
	if reflect.TypeOf(data).Kind() == reflect.Slice && reflect.ValueOf(data).Len() == 0 {
		w.Write([]byte(`{}`))
		return
	}

	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func stripANSI(s string) string {
	re := regexp.MustCompile(`\x1B[[:graph:]]*`)
	return re.ReplaceAllString(s, "")
}

func runRustScan(args string) ([]string, error) {
	// Remove leading and trailing spaces from args
	args = strings.TrimSpace(args)

	// Generate a unique container name using a timestamp
	containerName := "rustscan_" + strconv.FormatInt(time.Now().UnixNano(), 10)

	// Build the RustScan command
	command := []string{
		"docker",
		"run",
		"--rm",
		"--name",
		containerName,
		"rustscan",
	}

	if args != "" {
		command = append(command, strings.Split(args, " ")...)
	}

	cmd := exec.Command(command[0], command[1:]...)

	// Create a pipe to capture stdout
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("error creating stdout pipe: %v", err)
	}

	// Start the RustScan command
	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("error starting RustScan command: %v", err)
	}

	// Read RustScan output line by line and append to result array
	scanner := bufio.NewScanner(stdoutPipe)
	var results []string
	for scanner.Scan() {
		line := stripANSI(scanner.Text())
		results = append(results, line)
		fmt.Println(line) // add this line to print each output line
	}

	// Wait for RustScan to complete and check for errors
	err = cmd.Wait()
	if err != nil {
		return nil, fmt.Errorf("error running RustScan command: %v", err)
	}

	return results, nil
}
