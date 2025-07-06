package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
)

type SendMessageRequest struct {
	ID           int    `json:"id"`
	SecurityCode string `json:"security_code"`
	MsgType      string `json:"msgtype"`
	Content      string `json:"content"`
}

type ApiResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func main() {
	if len(os.Args) < 5 {
		fmt.Println("Usage: bot.exe <server_url> <port> <bot_id> <security_code> <message>")
		fmt.Println("Example: bot.exe localhost 8080 1 123 \"Hello World\"")
		os.Exit(1)
	}

	serverURL := os.Args[1]
	port := os.Args[2]
	botIDStr := os.Args[3]
	securityCode := os.Args[4]
	message := os.Args[5]

	botID, err := strconv.Atoi(botIDStr)
	if err != nil {
		fmt.Printf("Error: Invalid bot ID '%s'\n", botIDStr)
		os.Exit(1)
	}

	fullURL := fmt.Sprintf("http://%s:%s/send", serverURL, port)

	reqBody := SendMessageRequest{
		ID:           botID,
		SecurityCode: securityCode,
		MsgType:      "markdown",
		Content:      message,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		fmt.Printf("Error creating JSON: %v\n", err)
		os.Exit(1)
	}

	resp, err := http.Post(fullURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error sending request: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		os.Exit(1)
	}

	var apiResp ApiResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		fmt.Printf("Error parsing response: %v\n", err)
		os.Exit(1)
	}

	if resp.StatusCode == 200 {
		fmt.Printf("Success: %s\n", apiResp.Message)
	} else {
		fmt.Printf("Error: %s (Status: %d)\n", apiResp.Message, resp.StatusCode)
		os.Exit(1)
	}
}