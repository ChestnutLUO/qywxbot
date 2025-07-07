package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
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
	MediaID string `json:"media_id,omitempty"`
}

func main() {
	if len(os.Args) < 6 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	serverURL := os.Args[2]
	port := os.Args[3]
	botIDStr := os.Args[4]
	securityCode := os.Args[5]

	botID, err := strconv.Atoi(botIDStr)
	if err != nil {
		fmt.Printf("Error: Invalid bot ID '%s'\n", botIDStr)
		os.Exit(1)
	}

	switch command {
	case "send":
		if len(os.Args) < 7 {
			fmt.Println("Usage: bot.exe send <server_url> <port> <bot_id> <security_code> <message>")
			os.Exit(1)
		}
		message := os.Args[6]
		sendMessage(serverURL, port, botID, securityCode, message)
	case "sendfile":
		if len(os.Args) < 7 {
			fmt.Println("Usage: bot.exe sendfile <server_url> <port> <bot_id> <security_code> <file_path>")
			os.Exit(1)
		}
		filePath := os.Args[6]
		sendFile(serverURL, port, botID, securityCode, filePath)
	case "upload":
		if len(os.Args) < 7 {
			fmt.Println("Usage: bot.exe upload <server_url> <port> <bot_id> <security_code> <file_path>")
			os.Exit(1)
		}
		filePath := os.Args[6]
		uploadFile(serverURL, port, botID, securityCode, filePath)
	default:
		fmt.Printf("Error: Unknown command '%s'\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("企业微信机器人命令行工具")
	fmt.Println()
	fmt.Println("用法:")
	fmt.Println("  bot.exe <command> <server_url> <port> <bot_id> <security_code> <args...>")
	fmt.Println()
	fmt.Println("命令:")
	fmt.Println("  send     <message>    发送 Markdown 消息")
	fmt.Println("  sendfile <file_path>  上传并发送文件")
	fmt.Println("  upload   <file_path>  仅上传文件，返回 media_id")
	fmt.Println()
	fmt.Println("示例:")
	fmt.Println("  bot.exe send localhost 8080 1 123 \"Hello World\"")
	fmt.Println("  bot.exe sendfile localhost 8080 1 123 \"C:\\docs\\report.pdf\"")
	fmt.Println("  bot.exe upload localhost 8080 1 123 \"C:\\docs\\file.txt\"")
}

func sendMessage(serverURL, port string, botID int, securityCode, message string) {
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

	handleResponse(resp, "消息发送")
}

func sendFile(serverURL, port string, botID int, securityCode, filePath string) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Printf("Error: File '%s' does not exist\n", filePath)
		os.Exit(1)
	}

	fullURL := fmt.Sprintf("http://%s:%s/sendfile", serverURL, port)
	uploadFileToEndpoint(fullURL, botID, securityCode, filePath, "文件发送")
}

func uploadFile(serverURL, port string, botID int, securityCode, filePath string) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Printf("Error: File '%s' does not exist\n", filePath)
		os.Exit(1)
	}

	fullURL := fmt.Sprintf("http://%s:%s/upload", serverURL, port)
	uploadFileToEndpoint(fullURL, botID, securityCode, filePath, "文件上传")
}

func uploadFileToEndpoint(fullURL string, botID int, securityCode, filePath, operation string) {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("media", filepath.Base(filePath))
	if err != nil {
		fmt.Printf("Error creating form file: %v\n", err)
		os.Exit(1)
	}

	_, err = io.Copy(part, file)
	if err != nil {
		fmt.Printf("Error copying file: %v\n", err)
		os.Exit(1)
	}

	err = writer.WriteField("id", strconv.Itoa(botID))
	if err != nil {
		fmt.Printf("Error writing ID field: %v\n", err)
		os.Exit(1)
	}

	err = writer.WriteField("security_code", securityCode)
	if err != nil {
		fmt.Printf("Error writing security code field: %v\n", err)
		os.Exit(1)
	}

	err = writer.Close()
	if err != nil {
		fmt.Printf("Error closing writer: %v\n", err)
		os.Exit(1)
	}

	req, err := http.NewRequest("POST", fullURL, body)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		os.Exit(1)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error sending request: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	handleResponse(resp, operation)
}

func handleResponse(resp *http.Response, operation string) {
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
		fmt.Printf("Success: %s - %s\n", operation, apiResp.Message)
		if apiResp.MediaID != "" {
			fmt.Printf("Media ID: %s\n", apiResp.MediaID)
		}
	} else {
		fmt.Printf("Error: %s失败 - %s (Status: %d)\n", operation, apiResp.Message, resp.StatusCode)
		os.Exit(1)
	}
}