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
	"strings"
	"time"
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

type BotConfig struct {
	ServerURL    string `json:"server_url"`
	Port         string `json:"port"`
	BotID        int    `json:"bot_id"`
	SecurityCode string `json:"security_code"`
	Protocol     string `json:"protocol"`
	WebhookURL   string `json:"webhook_url"`
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	// 特殊命令处理
	switch command {
	case "init":
		initConfig()
		return
	case "config":
		showConfig()
		return
	case "help", "-h", "--help":
		printUsage()
		return
	}

	// 加载配置文件
	config, err := loadConfig()
	if err != nil {
		fmt.Printf("错误: 无法加载配置文件: %v\n", err)
		fmt.Println("请先运行 'bot.exe init' 来初始化配置文件")
		os.Exit(1)
	}

	// 检查是否为兼容模式（传统完整参数模式）
	if len(os.Args) >= 6 && isCompatibleMode(os.Args) {
		handleCompatibleMode()
		return
	}

	// 新的简化模式
	switch command {
	case "send":
		if len(os.Args) < 3 {
			fmt.Println("用法: bot.exe send <消息内容>")
			os.Exit(1)
		}
		message := strings.Join(os.Args[2:], " ")
		sendMessage(config.ServerURL, config.Port, config.BotID, config.SecurityCode, message)
	case "sendfile":
		if len(os.Args) < 3 {
			fmt.Println("用法: bot.exe sendfile <文件路径>")
			os.Exit(1)
		}
		filePath := os.Args[2]
		sendFile(config.ServerURL, config.Port, config.BotID, config.SecurityCode, filePath)
	case "upload":
		if len(os.Args) < 3 {
			fmt.Println("用法: bot.exe upload <文件路径>")
			os.Exit(1)
		}
		filePath := os.Args[2]
		uploadFile(config.ServerURL, config.Port, config.BotID, config.SecurityCode, filePath)
	default:
		fmt.Printf("错误: 未知命令 '%s'\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("企业微信机器人命令行工具")
	fmt.Println()
	fmt.Println("配置相关:")
	fmt.Println("  bot.exe init                    初始化配置文件")
	fmt.Println("  bot.exe config                  查看当前配置")
	fmt.Println()
	fmt.Println("简化用法 (需要先初始化配置):")
	fmt.Println("  bot.exe send <消息内容>          发送 Markdown 消息")
	fmt.Println("  bot.exe sendfile <文件路径>      上传并发送文件")
	fmt.Println("  bot.exe upload <文件路径>        仅上传文件，返回 media_id")
	fmt.Println()
	fmt.Println("兼容模式 (完整参数):")
	fmt.Println("  bot.exe <command> <server_url> <port> <bot_id> <security_code> <args...>")
	fmt.Println()
	fmt.Println("简化模式示例:")
	fmt.Println("  bot.exe send \"Hello World\"")
	fmt.Println("  bot.exe sendfile \"report.pdf\"")
	fmt.Println("  bot.exe upload \"file.txt\"")
	fmt.Println()
	fmt.Println("兼容模式示例:")
	fmt.Println("  bot.exe send localhost 8080 1 123 \"Hello World\"")
	fmt.Println("  bot.exe sendfile localhost 8080 1 123 \"C:\\docs\\report.pdf\"")
	fmt.Println("  bot.exe upload localhost 8080 1 123 \"C:\\docs\\file.txt\"")
}

func sendMessage(serverURL, port string, botID int, securityCode, message string) {
	config, _ := loadConfig()
	protocol := "http"
	if config != nil && config.Protocol != "" {
		protocol = config.Protocol
	}
	fullURL := fmt.Sprintf("%s://%s:%s/send", protocol, serverURL, port)

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
		fmt.Printf("错误: 文件 '%s' 不存在\n", filePath)
		os.Exit(1)
	}

	config, _ := loadConfig()
	protocol := "http"
	if config != nil && config.Protocol != "" {
		protocol = config.Protocol
	}
	fullURL := fmt.Sprintf("%s://%s:%s/sendfile", protocol, serverURL, port)
	uploadFileToEndpoint(config, fullURL, botID, securityCode, filePath, "文件发送")
}

func uploadFile(serverURL, port string, botID int, securityCode, filePath string) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Printf("错误: 文件 '%s' 不存在\n", filePath)
		os.Exit(1)
	}

	config, _ := loadConfig()
	protocol := "http"
	if config != nil && config.Protocol != "" {
		protocol = config.Protocol
	}
	fullURL := fmt.Sprintf("%s://%s:%s/upload", protocol, serverURL, port)
	uploadFileToEndpoint(config, fullURL, botID, securityCode, filePath, "文件上传")
}

// 从 webhook URL 中提取 key 参数
func extractWebhookKey(webhookURL string) (string, error) {
	if webhookURL == "" {
		return "", fmt.Errorf("webhook URL 为空")
	}
	
	// 提取 key 参数: https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=XXXXXXX
	parts := strings.Split(webhookURL, "key=")
	if len(parts) < 2 {
		return "", fmt.Errorf("无法从 webhook URL 中提取 key 参数")
	}
	
	return parts[1], nil
}

// 直接上传文件到企业微信并发送
func uploadFileToWebhook(config *BotConfig, filePath string) error {
	fmt.Println("正在尝试 fallback 到企业微信直接文件上传...")
	
	// 提取 webhook key
	webhookKey, err := extractWebhookKey(config.WebhookURL)
	if err != nil {
		return fmt.Errorf("提取 webhook key 失败: %v", err)
	}
	
	fmt.Printf("提取的 webhook key: %s...\n", webhookKey[:min(10, len(webhookKey))])
	
	// 企业微信文件上传 API
	uploadURL := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/webhook/upload_media?key=%s&type=file", webhookKey)
	
	fmt.Println("正在上传文件到企业微信...")
	
	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("打开文件失败: %v", err)
	}
	defer file.Close()
	
	// 创建 multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	
	part, err := writer.CreateFormFile("media", filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("创建表单文件失败: %v", err)
	}
	
	_, err = io.Copy(part, file)
	if err != nil {
		return fmt.Errorf("复制文件内容失败: %v", err)
	}
	
	err = writer.Close()
	if err != nil {
		return fmt.Errorf("关闭 writer 失败: %v", err)
	}
	
	// 上传文件
	req, err := http.NewRequest("POST", uploadURL, body)
	if err != nil {
		return fmt.Errorf("创建上传请求失败: %v", err)
	}
	
	req.Header.Set("Content-Type", writer.FormDataContentType())
	
	client := &http.Client{
		Timeout: 60 * time.Second,
	}
	
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("上传文件失败: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("文件上传失败，状态码: %d", resp.StatusCode)
	}
	
	// 解析上传响应
	var uploadResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
		return fmt.Errorf("解析上传响应失败: %v", err)
	}
	
	// 检查错误码
	if errCode, exists := uploadResp["errcode"]; exists {
		if errCode.(float64) != 0 {
			return fmt.Errorf("文件上传错误: %v", uploadResp["errmsg"])
		}
	}
	
	// 获取 media_id
	mediaID, exists := uploadResp["media_id"]
	if !exists {
		return fmt.Errorf("上传响应中没有 media_id")
	}
	
	fmt.Printf("获取到 media_id: %s...\n", mediaID.(string)[:min(20, len(mediaID.(string)))])
	
	// 发送文件消息
	fmt.Println("正在发送文件消息...")
	
	fileMessage := map[string]interface{}{
		"msgtype": "file",
		"file": map[string]interface{}{
			"media_id": mediaID,
		},
	}
	
	messageData, err := json.Marshal(fileMessage)
	if err != nil {
		return fmt.Errorf("序列化文件消息失败: %v", err)
	}
	
	req, err = http.NewRequest("POST", config.WebhookURL, bytes.NewBuffer(messageData))
	if err != nil {
		return fmt.Errorf("创建发送请求失败: %v", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	client = &http.Client{
		Timeout: 30 * time.Second,
	}
	
	resp, err = client.Do(req)
	if err != nil {
		return fmt.Errorf("发送文件消息失败: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("文件消息发送失败，状态码: %d", resp.StatusCode)
	}
	
	fmt.Println("✓ 文件发送成功 (通过企业微信直接上传)")
	return nil
}

// 辅助函数：获取两个数的最小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func uploadFileToEndpoint(config *BotConfig, fullURL string, botID int, securityCode, filePath, operation string) {
	// 尝试正常上传到 qywxbot 服务器
	fmt.Printf("正在尝试%s到 qywxbot 服务器...\n", operation)
	
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

	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("qywxbot 服务器连接失败: %v\n", err)
		
		// 尝试 fallback 到企业微信直接上传（仅支持 sendfile 操作）
		if operation == "文件发送" && config.WebhookURL != "" {
			fmt.Println("尝试 fallback 到企业微信直接文件上传...")
			if fallbackErr := uploadFileToWebhook(config, filePath); fallbackErr != nil {
				fmt.Printf("Fallback 上传也失败: %v\n", fallbackErr)
				os.Exit(1)
			}
			return
		} else {
			fmt.Printf("无法连接到 qywxbot 服务器，且不支持 fallback\n")
			os.Exit(1)
		}
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("qywxbot 服务器响应错误 (状态码: %d)\n", resp.StatusCode)
		
		// 尝试 fallback 到企业微信直接上传（仅支持 sendfile 操作）
		if operation == "文件发送" && config.WebhookURL != "" {
			fmt.Println("尝试 fallback 到企业微信直接文件上传...")
			if fallbackErr := uploadFileToWebhook(config, filePath); fallbackErr != nil {
				fmt.Printf("Fallback 上传也失败: %v\n", fallbackErr)
				os.Exit(1)
			}
			return
		} else {
			fmt.Printf("服务器响应错误，且不支持 fallback\n")
			os.Exit(1)
		}
	}

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
		fmt.Printf("成功: %s - %s\n", operation, apiResp.Message)
		if apiResp.MediaID != "" {
			fmt.Printf("Media ID: %s\n", apiResp.MediaID)
		}
	} else {
		fmt.Printf("错误: %s失败 - %s (状态码: %d)\n", operation, apiResp.Message, resp.StatusCode)
		os.Exit(1)
	}
}

// 获取配置文件路径
func getConfigPath() string {
	exe, err := os.Executable()
	if err != nil {
		return "bot-config.json"
	}
	exeDir := filepath.Dir(exe)
	return filepath.Join(exeDir, "bot-config.json")
}

// 加载配置文件
func loadConfig() (*BotConfig, error) {
	configPath := getConfigPath()
	
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("配置文件不存在: %s", configPath)
	}
	
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}
	
	var config BotConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}
	
	return &config, nil
}

// 保存配置文件
func saveConfig(config *BotConfig) error {
	configPath := getConfigPath()
	
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %v", err)
	}
	
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("保存配置文件失败: %v", err)
	}
	
	return nil
}

// 初始化配置文件
func initConfig() {
	configPath := getConfigPath()
	
	// 检查配置文件是否已存在
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("配置文件已存在: %s\n", configPath)
		fmt.Print("是否要重新配置? (y/n): ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			return
		}
	}
	
	fmt.Println("初始化企业微信机器人配置...")
	fmt.Println()
	
	config := &BotConfig{
		Protocol: "http",
	}
	
	fmt.Print("服务器地址 (例如: localhost): ")
	fmt.Scanln(&config.ServerURL)
	
	fmt.Print("端口号 (例如: 8080): ")
	fmt.Scanln(&config.Port)
	
	fmt.Print("机器人ID: ")
	fmt.Scanln(&config.BotID)
	
	fmt.Print("安全码: ")
	fmt.Scanln(&config.SecurityCode)
	
	fmt.Print("协议 (http/https, 默认: http): ")
	var protocol string
	fmt.Scanln(&protocol)
	if protocol != "" {
		config.Protocol = protocol
	}
	
	if err := saveConfig(config); err != nil {
		fmt.Printf("保存配置失败: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("配置已保存到: %s\n", configPath)
	fmt.Println()
	fmt.Println("现在你可以使用简化命令:")
	fmt.Println("  bot.exe send \"你的消息\"")
	fmt.Println("  bot.exe sendfile \"文件路径\"")
	fmt.Println("  bot.exe upload \"文件路径\"")
}

// 显示当前配置
func showConfig() {
	config, err := loadConfig()
	if err != nil {
		fmt.Printf("无法加载配置: %v\n", err)
		fmt.Println("请先运行 'bot.exe init' 来初始化配置")
		return
	}
	
	fmt.Println("当前配置:")
	fmt.Printf("  配置文件: %s\n", getConfigPath())
	fmt.Printf("  服务器地址: %s\n", config.ServerURL)
	fmt.Printf("  端口: %s\n", config.Port)
	fmt.Printf("  机器人ID: %d\n", config.BotID)
	fmt.Printf("  安全码: %s\n", config.SecurityCode)
	fmt.Printf("  协议: %s\n", config.Protocol)
	fmt.Printf("  服务器URL: %s://%s:%s\n", config.Protocol, config.ServerURL, config.Port)
}

// 检查是否为兼容模式（传统完整参数模式）
func isCompatibleMode(args []string) bool {
	// 检查第二个参数是否像服务器地址（包含字母或IP格式）
	if len(args) >= 6 {
		port := args[3]
		botID := args[4]
		
		// 简单检查：如果第4个参数是数字，第3个参数是端口格式
		if _, err := strconv.Atoi(botID); err == nil {
			if _, err := strconv.Atoi(port); err == nil {
				return true
			}
		}
	}
	return false
}

// 处理兼容模式
func handleCompatibleMode() {
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
		fmt.Printf("错误: 无效的机器人ID '%s'\n", botIDStr)
		os.Exit(1)
	}
	
	switch command {
	case "send":
		if len(os.Args) < 7 {
			fmt.Println("用法: bot.exe send <server_url> <port> <bot_id> <security_code> <message>")
			os.Exit(1)
		}
		message := strings.Join(os.Args[6:], " ")
		sendMessage(serverURL, port, botID, securityCode, message)
	case "sendfile":
		if len(os.Args) < 7 {
			fmt.Println("用法: bot.exe sendfile <server_url> <port> <bot_id> <security_code> <file_path>")
			os.Exit(1)
		}
		filePath := os.Args[6]
		sendFile(serverURL, port, botID, securityCode, filePath)
	case "upload":
		if len(os.Args) < 7 {
			fmt.Println("用法: bot.exe upload <server_url> <port> <bot_id> <security_code> <file_path>")
			os.Exit(1)
		}
		filePath := os.Args[6]
		uploadFile(serverURL, port, botID, securityCode, filePath)
	default:
		fmt.Printf("错误: 未知命令 '%s'\n", command)
		printUsage()
		os.Exit(1)
	}
}