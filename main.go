package main

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	mrand "math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge/http01"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
	_ "github.com/mattn/go-sqlite3"
)

type Config struct {
	HTTPPort     string `json:"http_port"`     // HTTP端口，如 ":8080" 或 ":80"
	HTTPSPort    string `json:"https_port"`    // HTTPS端口，如 ":443"
	CertFile     string `json:"cert_file"`     // SSL证书文件路径
	KeyFile      string `json:"key_file"`      // SSL私钥文件路径
	Domain       string `json:"domain"`        // 外部访问域名，如 "example.com"
	EmailForACME string `json:"email_for_acme"` // ACME证书申请邮箱
}




type SendMessageRequest struct {
	ID           int    `json:"id"`
	SecurityCode string `json:"security_code"`
	MsgType      string `json:"msgtype"`
	Content      string `json:"content"`
}

type WeComTextMessage struct {
	MsgType string `json:"msgtype"`
	Text    struct {
		Content string `json:"content"`
	} `json:"text"`
}

type WeComMarkdownMessage struct {
	MsgType  string `json:"msgtype"`
	Markdown struct {
		Content string `json:"content"`
	} `json:"markdown"`
}

type WeComFileMessage struct {
	MsgType string `json:"msgtype"`
	File    struct {
		MediaID string `json:"media_id"`
	} `json:"file"`
}

type UploadResponse struct {
	ErrCode   int    `json:"errcode"`
	ErrMsg    string `json:"errmsg"`
	Type      string `json:"type"`
	MediaID   string `json:"media_id"`
	CreatedAt string `json:"created_at"`
}

type WeComTemplateCardMessage struct {
	MsgType      string       `json:"msgtype"`
	TemplateCard TemplateCard `json:"template_card"`
}



type TemplateCard struct {
	CardType        string           `json:"card_type"`
	Source          *Source          `json:"source,omitempty"`
	MainTitle       MainTitle        `json:"main_title"`
	EmphasisContent *EmphasisContent `json:"emphasis_content,omitempty"`
	SubTitleText    string           `json:"sub_title_text,omitempty"`
	CardAction      CardAction       `json:"card_action"`
	JumpList        []JumpItem       `json:"jump_list,omitempty"`
}

type Source struct {
	IconURL   string `json:"icon_url,omitempty"`
	Desc      string `json:"desc,omitempty"`
	DescColor int    `json:"desc_color,omitempty"`
}

type MainTitle struct {
	Title string `json:"title"`
	Desc  string `json:"desc,omitempty"`
}

type EmphasisContent struct {
	Title string `json:"title,omitempty"`
	Desc  string `json:"desc,omitempty"`
}

type CardAction struct {
	Type int    `json:"type"`
	URL  string `json:"url,omitempty"`
}

type JumpItem struct {
	Type     int    `json:"type"`
	URL      string `json:"url,omitempty"`
	Title    string `json:"title"`
	AppID    string `json:"appid,omitempty"`
	PagePath string `json:"pagepath,omitempty"`
}




var (
	db    *sql.DB
	tmpls *template.Template
	config Config
)

type SuccessData struct {
	ID                int64
	SecurityCode      string
	SendURL           string
	CurlExample       string
	BotExeExample     string
	BotExeSimple      string
	ConfigFileContent string
}



type Bot struct {
	ID           int    `json:"id"`
	URL          string `json:"url"`
	SecurityCode string `json:"security_code"`
	CreatedAt    string `json:"created_at"`
}

func main() {
	loadConfig()

	var err error
	db, err = sql.Open("sqlite3", "./bots.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	createTable()

	if len(os.Args) > 1 {
		handleCommandLine()
		return
	}

	manageCertificate()

	tmpls = template.Must(template.ParseGlob("templates/*.html"))

	log.Println("注册根路径处理器: /")
	http.HandleFunc("/", handler)
	log.Println("注册发送消息处理器: /send")
	http.HandleFunc("/send", sendHandler)
	log.Println("注册上传文件处理器: /upload")
	http.HandleFunc("/upload", uploadHandler)
	log.Println("注册发送文件处理器: /sendfile")
	http.HandleFunc("/sendfile", sendfileHandler)

	log.Println("注册静态文件服务: /web/")
	http.Handle("/web/", http.StripPrefix("/web/", http.FileServer(http.Dir("web"))))

	if config.CertFile != "" && config.KeyFile != "" {
		log.Printf("HTTPS 服务器正在 %s 启动...", config.HTTPSPort)
		log.Fatal(http.ListenAndServeTLS(config.HTTPSPort, config.CertFile, config.KeyFile, nil))
	} else {
		log.Printf("HTTP 服务器正在 %s 启动...", config.HTTPPort)
		log.Fatal(http.ListenAndServe(config.HTTPPort, nil))
	}
}

func loadConfig() {
	file, err := os.Open("config.json")
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("配置文件 config.json 未找到，正在创建默认配置...")
			config.HTTPPort = ":8080"
			config.HTTPSPort = ":443"
			config.CertFile = ""
			config.KeyFile = ""
			config.Domain = ""
			config.EmailForACME = ""
			file, err := os.Create("config.json")
			if err != nil {
				log.Fatalf("创建配置文件失败: %v", err)
			}
			defer file.Close()
			encoder := json.NewEncoder(file)
			encoder.SetIndent("", "  ")
			if err := encoder.Encode(config); err != nil {
				log.Fatalf("写入默认配置失败: %v", err)
			}
			log.Println("默认配置文件 config.json 创建成功。")
			return
		}
		log.Fatalf("打开配置文件失败: %v", err)
	}
	defer file.Close()

	if err := json.NewDecoder(file).Decode(&config); err != nil {
		log.Fatalf("解析配置文件失败: %v", err)
	}
}

// MyUser You'll need a user or account type that implements acme.User
type MyUser struct {
	Email        string
	Registration *registration.Resource
	key          crypto.PrivateKey
}

func (u *MyUser) GetEmail() string {
	return u.Email
}
func (u MyUser) GetRegistration() *registration.Resource {
	return u.Registration
}
func (u *MyUser) GetPrivateKey() crypto.PrivateKey {
	return u.key
}

func manageCertificate() {
	if config.Domain == "" || config.EmailForACME == "" {
		return
	}

	// Create a user. New accounts need an email and private key to start.
	user, err := ensureACMEUser()
	if err != nil {
		log.Fatalf("ACME user management failed: %v", err)
	}

	legoConfig := lego.NewConfig(user)

	// This CA URL is configured for a local dev instance of Boulder running in Docker in a VM.
	legoConfig.CADirURL = lego.LEDirectoryProduction
	legoConfig.Certificate.KeyType = certcrypto.RSA2048

	// A client facilitates communication with the CA server.
	client, err := lego.NewClient(legoConfig)
	if err != nil {
		log.Fatal(err)
	}

	err = client.Challenge.SetHTTP01Provider(http01.NewProviderServer(":80", ""))
	if err != nil {
		log.Fatal(err)
	}

	// New users will need to register
	if user.GetRegistration() == nil {
		reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
		if err != nil {
			log.Fatal(err)
		}
		user.Registration = reg
	}

	request := certificate.ObtainRequest{
		Domains: []string{config.Domain},
		Bundle:  true,
	}
	certificates, err := client.Certificate.Obtain(request)
	if err != nil {
		log.Fatal(err)
	}

	// Each certificate comes back with the cert bytes, the bytes of the client's
	// private key, and a certificate URL. SAVE THESE TO DISK.
	certPath := "certs"
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		os.Mkdir(certPath, 0755)
	}

	err = os.WriteFile(filepath.Join(certPath, config.Domain+".crt"), certificates.Certificate, 0600)
	if err != nil {
		log.Fatalf("Failed to write certificate to disk: %v", err)
	}

	err = os.WriteFile(filepath.Join(certPath, config.Domain+".key"), certificates.PrivateKey, 0600)
	if err != nil {
		log.Fatalf("Failed to write private key to disk: %v", err)
	}
	config.CertFile = filepath.Join(certPath, config.Domain+".crt")
	config.KeyFile = filepath.Join(certPath, config.Domain+".key")
}

func ensureACMEUser() (*MyUser, error) {
	keyPath := ".acme_user_key"
	var privateKey *ecdsa.PrivateKey
	keyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		if os.IsNotExist(err) {
			privateKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
			if err != nil {
				return nil, err
			}
			keyBytes, err = x509.MarshalECPrivateKey(privateKey)
			if err != nil {
				return nil, err
			}
			if err := os.WriteFile(keyPath, keyBytes, 0600); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	} else {
		privateKey, err = x509.ParseECPrivateKey(keyBytes)
		if err != nil {
			return nil, err
		}
	}

	return &MyUser{
		Email: config.EmailForACME,
		key:   privateKey,
	}, nil
}



func createTable() {
	createTableSQL := `CREATE TABLE IF NOT EXISTS bots (
		"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		"url" TEXT,
		"security_code" TEXT,
		"created_at" DATETIME DEFAULT CURRENT_TIMESTAMP
	  );`

	statement, err := db.Prepare(createTableSQL)
	if err != nil {
		log.Fatal(err)
	}
	statement.Exec()
}

func handler(w http.ResponseWriter, r *http.Request) {
	log.Printf("接收到来自 %s 的请求: %s %s", r.RemoteAddr, r.Method, r.URL.Path)
	if r.Method == http.MethodPost {
		url := r.FormValue("url")
		if url == "" {
			log.Printf("处理请求失败: URL 为空")
			http.Error(w, "URL 为必填项", http.StatusBadRequest)
			return
		}
		log.Printf("收到新的机器人注册请求，URL: %s", url)

		// Check if the bot already exists
		var existingID int
		var existingCode string
		err := db.QueryRow("SELECT id, security_code FROM bots WHERE url = ?", url).Scan(&existingID, &existingCode)
		if err != nil && err != sql.ErrNoRows {
			log.Printf("数据库查询失败: %v", err)
			http.Error(w, "数据库错误", http.StatusInternalServerError)
			return
		}

		if existingID != 0 {
			log.Printf("机器人已存在，ID: %d。直接显示信息。", existingID)
			// Bot already exists, just show the info page without sending notifications.
			serverURL := getServerURL()
			sendURL := fmt.Sprintf("%s/send", serverURL)
			curlExample := fmt.Sprintf(`curl -X POST -H "Content-Type: application/json" -d '{"id": %d, "security_code": "%s", "msgtype": "text", "content": "Hello from your bot!"}' %s`, existingID, existingCode, sendURL)

			domain := config.Domain
			if domain == "" {
				domain = "localhost"
			}
			port := strings.TrimPrefix(config.HTTPPort, ":")
			if config.CertFile != "" && config.KeyFile != "" {
				port = strings.TrimPrefix(config.HTTPSPort, ":")
			}

			botExeExample := fmt.Sprintf(`bot.exe send %s %s %d %s "你的消息"`, domain, port, existingID, existingCode)

			data := SuccessData{
				ID:            int64(existingID),
				SecurityCode:  existingCode,
				SendURL:       sendURL,
				CurlExample:   curlExample,
				BotExeExample: botExeExample,
			}

			err = tmpls.ExecuteTemplate(w, "success.html", data)
			if err != nil {
				http.Error(w, "无法呈现成功���面", http.StatusInternalServerError)
			}
			return
		}
		mrand.Seed(time.Now().UnixNano())
		securityCode := fmt.Sprintf("%03d", mrand.Intn(1000))

		log.Printf("正在为 URL %s 插入新的机器人记录", url)
		id, err := insertBot(url, securityCode)
		if err != nil {
			log.Printf("插入机器人记录失败: %v", err)
			http.Error(w, "机器人注册失败", http.StatusInternalServerError)
			return
		}
		log.Printf("新机器人注册成功，ID: %d", id)

		serverAddr := "localhost"
		if config.Domain != "" {
			serverAddr = config.Domain
		}
		port := config.HTTPPort
		if config.CertFile != "" && config.KeyFile != "" {
			port = config.HTTPSPort
		}
		port = strings.TrimPrefix(port, ":")

		err = sendTemplateCardMessage(url, id, securityCode)
		if err != nil {
			log.Printf("发送确认消息失败 %s: %v", url, err)
		}

		// 发送配置文件
		serverURL := getServerURL()
		err = sendConfigFile(url, id, securityCode, serverURL)
		if err != nil {
			log.Printf("发送配置文件失败 %s: %v", url, err)
		}

		// 发送简化命令使用提示
		simpleCommandMessage := fmt.Sprintf(` Windows 使用方法

**第一步：下载配置文件**
从上面的文件中下载 bot-config.json 文件，放在 bot.exe 同目录下。

**第二步：使用简化命令**
现在你可以使用简化命令发送消息：
` + "```" + `
bot.exe send "你的消息"
bot.exe sendfile "文件路径"
bot.exe upload "文件路径"
` + "```" + `

**兼容模式**
传统完整参数方式仍然支持：
` + "```" + `
bot.exe send %s %s %d %s "来自 bot.exe 的消息"
` + "```", serverAddr, port, id, securityCode)

		err = sendMarkdownMessage(url, simpleCommandMessage)
		if err != nil {
			log.Printf("发送简化命令提示失败 %s: %v", url, err)
		}

		// 发送 Linux 使用方法
		linuxMessage := fmt.Sprintf(`Linux/macOS 使用方法

**前提条件：需要安装 curl**
` + "```bash" + `
# Ubuntu/Debian
sudo apt-get install curl

# CentOS/RHEL/Fedora
sudo yum install curl
# 或者 (较新版本)
sudo dnf install curl

# macOS
brew install curl
` + "```" + `

**使用 Shell 脚本 (推荐)**
从上面的文件中下载 bot.sh 脚本，然后：
` + "```bash" + `
chmod +x bot.sh
./bot.sh send "你的消息"
./bot.sh sendfile "/path/to/file"
` , id, securityCode, serverURL)

		err = sendMarkdownMessage(url, linuxMessage)
		if err != nil {
			log.Printf("发送 Linux 使用提示失败 %s: %v", url, err)
		}

		sendURL := fmt.Sprintf("%s/send", serverURL)
		curlExample := fmt.Sprintf(`curl -X POST -H "Content-Type: application/json" -d '{"id": %d, "security_code": "%s", "msgtype": "text", "content": "Hello from your bot!"}' %s`, id, securityCode, sendURL)

		botExeExample := fmt.Sprintf(`bot.exe send %s %s %d %s "来自 bot.exe 的消息"`, serverAddr, port, id, securityCode)
		botExeSimple := `bot.exe send "你的消息"`

		data := SuccessData{
			ID:                id,
			SecurityCode:      securityCode,
			SendURL:           sendURL,
			CurlExample:       curlExample,
			BotExeExample:     botExeExample,
			BotExeSimple:      botExeSimple,
			ConfigFileContent: generateConfigFileContent(id, securityCode, serverURL),
		}

		err = sendBotScripts(url, id, securityCode, serverURL)
		if err != nil {
			log.Printf("发送脚本文件失败 %s: %v", url, err)
		}

		err = tmpls.ExecuteTemplate(w, "success.html", data)
		if err != nil {
			http.Error(w, "无法呈现成功页面", http.StatusInternalServerError)
		}
		return
	}

	tmpls.ExecuteTemplate(w, "index.html", nil)
}

func sendHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("接收到来自 %s 的请求: %s %s", r.RemoteAddr, r.Method, r.URL.Path)
	if r.Method != http.MethodPost {
		log.Printf("方法不允许: %s", r.Method)
		http.Error(w, "仅支持 POST 方法", http.StatusMethodNotAllowed)
		return
	}

	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("解析请求体失败: %v", err)
		http.Error(w, "无效的请求体", http.StatusBadRequest)
		return
	}
	log.Printf("收到发送消息请求: Bot ID=%d, MsgType=%s", req.ID, req.MsgType)

	var botURL string
	err := db.QueryRow("SELECT url FROM bots WHERE id = ? AND security_code = ?", req.ID, req.SecurityCode).Scan(&botURL)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("未找到机器人或安全码不正确: Bot ID=%d", req.ID)
			http.Error(w, "未找到机器人或安全码不正确", http.StatusNotFound)
		} else {
			log.Printf("数据库错误: %v", err)
			http.Error(w, "数据库错误", http.StatusInternalServerError)
		}
		return
	}
	log.Printf("找到机器人 URL: %s (ID: %d)", botURL, req.ID)

	var sendErr error
	switch req.MsgType {
	case "text":
		sendErr = sendTextMessage(botURL, req.Content)
	case "markdown":
		sendErr = sendMarkdownMessage(botURL, req.Content)
	case "file":
		sendErr = sendFileMessage(botURL, req.Content)
	default:
		http.Error(w, "不支持的消息类型", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if sendErr != nil {
		log.Printf("发送消息失败 (Bot ID: %d): %v", req.ID, sendErr)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": sendErr.Error()})
	} else {
		log.Printf("消息发送成功 (Bot ID: %d)", req.ID)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "消息发送成功"})
	}
}

func sendTextMessage(url, content string) error {
	msg := WeComTextMessage{
		MsgType: "text",
	}
	msg.Text.Content = content
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return postMessage(url, payload)
}

func sendMarkdownMessage(url, content string) error {
	msg := WeComMarkdownMessage{
		MsgType: "markdown",
	}
	msg.Markdown.Content = content
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return postMessage(url, payload)
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("接收到来自 %s 的请求: %s %s", r.RemoteAddr, r.Method, r.URL.Path)
	if r.Method != http.MethodPost {
		log.Printf("方法不允���: %s", r.Method)
		http.Error(w, "仅支持 POST 方法", http.StatusMethodNotAllowed)
		return
	}

	botID := r.FormValue("id")
	securityCode := r.FormValue("security_code")
	if botID == "" || securityCode == "" {
		http.Error(w, "机器人 ID 和安全码是必填项", http.StatusBadRequest)
		return
	}

	var botURL string
	err := db.QueryRow("SELECT url FROM bots WHERE id = ? AND security_code = ?", botID, securityCode).Scan(&botURL)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "未找到机器人或安全码不正确", http.StatusNotFound)
		} else {
			http.Error(w, "数据库错误", http.StatusInternalServerError)
		}
		return
	}

	file, handler, err := r.FormFile("media")
	if err != nil {
		http.Error(w, "无法读取文件", http.StatusBadRequest)
		return
	}
	defer file.Close()

	key := strings.Split(botURL, "?key=")[1]
	uploadURL := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/webhook/upload_media?key=%s&type=file", key)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("media", filepath.Base(handler.Filename))
	if err != nil {
		http.Error(w, "无法创建表单文件", http.StatusInternalServerError)
		return
	}
	_, err = io.Copy(part, file)
	if err != nil {
		http.Error(w, "无法将文件内容写入请求", http.StatusInternalServerError)
		return
	}
	writer.Close()

	req, err := http.NewRequest("POST", uploadURL, body)
	if err != nil {
		http.Error(w, "无法创建上传请求", http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "文件上传失败", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var uploadResp UploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
		http.Error(w, "无法解析上传响应", http.StatusInternalServerError)
		return
	}

	if uploadResp.ErrCode != 0 {
		http.Error(w, fmt.Sprintf("文件发送错误: %s", uploadResp.ErrMsg), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(uploadResp)
}

func sendfileHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("接收到来自 %s 的请求: %s %s", r.RemoteAddr, r.Method, r.URL.Path)
	if r.Method != http.MethodPost {
		log.Printf("方法不允许: %s", r.Method)
		http.Error(w, "仅支持 POST 方法", http.StatusMethodNotAllowed)
		return
	}

	botID := r.FormValue("id")
	securityCode := r.FormValue("security_code")
	if botID == "" || securityCode == "" {
		http.Error(w, "机器人 ID 和安全码是必填项", http.StatusBadRequest)
		return
	}

	var botURL string
	err := db.QueryRow("SELECT url FROM bots WHERE id = ? AND security_code = ?", botID, securityCode).Scan(&botURL)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "未找到机器人或安全码不正确", http.StatusNotFound)
		} else {
			http.Error(w, "数据库错误", http.StatusInternalServerError)
		}
		return
	}

	file, handler, err := r.FormFile("media")
	if err != nil {
		http.Error(w, "无法读取文件", http.StatusBadRequest)
		return
	}
	defer file.Close()

	key := strings.Split(botURL, "?key=")[1]
	uploadURL := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/webhook/upload_media?key=%s&type=file", key)

	// Create a new buffer to hold the multipart request body
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("media", filepath.Base(handler.Filename))
	if err != nil {
		http.Error(w, "无法创建表单文件", http.StatusInternalServerError)
		return
	}
	// Copy file content to the multipart writer
	_, err = io.Copy(part, file)
	if err != nil {
		http.Error(w, "无法将文件内容写入请求", http.StatusInternalServerError)
		return
	}
	writer.Close()

	// Create and send the upload request
	req, err := http.NewRequest("POST", uploadURL, body)
	if err != nil {
		http.Error(w, "无法创建上传请求", http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "文件上传失败", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Decode the upload response
	var uploadResp UploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
		http.Error(w, "无法解析上传响应", http.StatusInternalServerError)
		return
	}

	// Check for upload errors
	if uploadResp.ErrCode != 0 {
		http.Error(w, fmt.Sprintf("文件上传错误: %s", uploadResp.ErrMsg), http.StatusInternalServerError)
		return
	}

	// If upload is successful, send the file message
	sendErr := sendFileMessage(botURL, uploadResp.MediaID)

	w.Header().Set("Content-Type", "application/json")
	if sendErr != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": sendErr.Error()})
	} else {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status":   "success",
			"message":  "文件发送成功",
			"media_id": uploadResp.MediaID,
		})
	}
}

func sendFileMessage(url, mediaID string) error {
	msg := WeComFileMessage{
		MsgType: "file",
	}
	msg.File.MediaID = mediaID
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return postMessage(url, payload)
}

func postMessage(url string, payload []byte) error {
	log.Printf("正在向 %s 发送消息...", url)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		log.Printf("发送消息到 %s 失败: %v", url, err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("发送消息到 %s 返回非 200 状态码: %d, 响应: %s", url, resp.StatusCode, string(bodyBytes))
		return fmt.Errorf("发送消息失败，状态码: %d, 响应: %s", resp.StatusCode, string(bodyBytes))
	}

	log.Printf("成功发送消息到 %s", url)
	return nil
}

func sendTemplateCardMessage(webhookURL string, botID int64, securityCode string) error {
	serverURL := getServerURL()
	card := WeComTemplateCardMessage{
		MsgType: "template_card",
		TemplateCard: TemplateCard{
			CardType: "text_notice",
			Source: &Source{
				IconURL: "https://wework.qpic.cn/wwpic/252813_jOfDHtcISzuodLa_1629280209/0",
				Desc:    "机器人管家",
			},
			MainTitle: MainTitle{
				Title: "机器人注册成功",
				Desc:  "您的机器人已成功在系统中注册",
			},
			EmphasisContent: &EmphasisContent{
				Title: fmt.Sprintf("%d", botID),
				Desc:  "机器人ID",
			},
			SubTitleText: fmt.Sprintf("安全码: %s", securityCode),
			CardAction: CardAction{
				Type: 1,
				URL:  serverURL,
			},
			JumpList: []JumpItem{
				{
					Type:  1,
					URL:   serverURL + "/web/bot-scripts.html",
					Title: "脚本使用文档",
				},
				{
					Type:  1,
					URL:   serverURL + "/web/api-docs.html",
					Title: "API文档",
				},
			},
		},
	}

	payload, err := json.Marshal(card)
	if err != nil {
		return err
	}
	return postMessage(webhookURL, payload)
}

func insertBot(url, securityCode string) (int64, error) {
	insertSQL := "INSERT INTO bots(url, security_code) VALUES (?, ?)"
	statement, err := db.Prepare(insertSQL)
	if err != nil {
		return 0, err
	}
	result, err := statement.Exec(url, securityCode)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func getServerURL() string {
	if config.Domain != "" {
		if config.CertFile != "" && config.KeyFile != "" {
			// HTTPS: 如果是443端口则不显示端口号
			if config.HTTPSPort == ":443" {
				return fmt.Sprintf("https://%s", config.Domain)
			}
			return fmt.Sprintf("https://%s%s", config.Domain, config.HTTPSPort)
		}
		// HTTP: 如果是80端口则不显示端口号
		if config.HTTPPort == ":80" {
			return fmt.Sprintf("http://%s", config.Domain)
		}
		return fmt.Sprintf("http://%s%s", config.Domain, config.HTTPPort)
	}
	return fmt.Sprintf("http://localhost%s", config.HTTPPort)
}

func sendBotScripts(webhookURL string, botID int64, securityCode, serverURL string) error {
	// 发送 bot.sh 脚本
	err := sendBotScript(webhookURL, botID, securityCode, serverURL, "bot.sh")
	if err != nil {
		return fmt.Errorf("发送 bot.sh 失败: %v", err)
	}

	// 发送 Windows 二进制程序
	err = sendBotBinary(webhookURL, botID, securityCode, "bot.exe")
	if err != nil {
		return fmt.Errorf("发送 bot.exe 失败: %v", err)
	}

	return nil
}

// 生成配置文件内容
func generateConfigFileContent(botID int64, securityCode, serverURL string) string {
	// 解析服务器URL获取协议、主机和端口
	protocol := "http"
	host := "localhost"
	port := "8080"

	if strings.HasPrefix(serverURL, "https://") {
		protocol = "https"
		serverURL = strings.TrimPrefix(serverURL, "https://")
	} else if strings.HasPrefix(serverURL, "http://") {
		protocol = "http"
		serverURL = strings.TrimPrefix(serverURL, "http://")
	}

	// 分离主机和端口
	if strings.Contains(serverURL, ":") {
		parts := strings.Split(serverURL, ":")
		host = parts[0]
		port = parts[1]
	} else {
		host = serverURL
		if protocol == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}

	configContent := fmt.Sprintf(`{
  "server_url": "%s",
  "port": "%s",
  "bot_id": %d,
  "security_code": "%s",
  "protocol": "%s"
}`, host, port, botID, securityCode, protocol)

	return configContent
}

// 发送配置文件到企业微信
func sendConfigFile(webhookURL string, botID int64, securityCode, serverURL string) error {
	// 生成配置文件内容
	configContent := generateConfigFileContent(botID, securityCode, serverURL)

	// 创建临时配置文件
	tempFile, err := os.CreateTemp("", fmt.Sprintf("bot_%d_config.json", botID))
	if err != nil {
		return fmt.Errorf("创建临时配置文件失败: %v", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// 写入配置内容
	_, err = tempFile.WriteString(configContent)
	if err != nil {
		return fmt.Errorf("写入配置文件失败: %v", err)
	}
	tempFile.Close()

	// 上传配置文件到企业微信
	key := strings.Split(webhookURL, "?key=")[1]
	uploadURL := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/webhook/upload_media?key=%s&type=file", key)

	file, err := os.Open(tempFile.Name())
	if err != nil {
		return fmt.Errorf("打开配置文件失败: %v", err)
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("media", "bot-config.json")
	if err != nil {
		return fmt.Errorf("创建表单文件失败: %v", err)
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return fmt.Errorf("复制文件内容失败: %v", err)
	}
	writer.Close()

	req, err := http.NewRequest("POST", uploadURL, body)
	if err != nil {
		return fmt.Errorf("创建上传请求失败: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("上传配置文件失败: %v", err)
	}
	defer resp.Body.Close()

	var uploadResp UploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
		return fmt.Errorf("解析上传响应失败: %v", err)
	}

	if uploadResp.ErrCode != 0 {
		return fmt.Errorf("配置文件上传错误: %s", uploadResp.ErrMsg)
	}

	// 发送文件消息
	return sendFileMessage(webhookURL, uploadResp.MediaID)
}

func sendBotScript(webhookURL string, botID int64, securityCode, serverURL, scriptName string) error {
	// 读取脚本模板
	templateContent, err := os.ReadFile(scriptName)
	if err != nil {
		return fmt.Errorf("读取脚本模板失败: %v", err)
	}

	// 替换占位符
	scriptContent := string(templateContent)
	scriptContent = strings.ReplaceAll(scriptContent, "{BOT_ID_Template}", fmt.Sprintf("%d", botID))
	scriptContent = strings.ReplaceAll(scriptContent, "{SECURITY_CODE_Template}", securityCode)
	scriptContent = strings.ReplaceAll(scriptContent, "{SERVER_URL_Template}", serverURL)

	// 创建临时文件
	tempFile, err := os.CreateTemp("", fmt.Sprintf("bot_%d.%s", botID, scriptName[4:]))
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %v", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// 写入内容
	_, err = tempFile.WriteString(scriptContent)
	if err != nil {
		return fmt.Errorf("写入临时文件失败: %v", err)
	}
	tempFile.Close()

	// 上传文件到企业微信
	key := strings.Split(webhookURL, "?key=")[1]
	uploadURL := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/webhook/upload_media?key=%s&type=file", key)

	file, err := os.Open(tempFile.Name())
	if err != nil {
		return fmt.Errorf("打开临时文件失败: %v", err)
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("media", fmt.Sprintf("bot_%d%s", botID, filepath.Ext(scriptName)))
	if err != nil {
		return fmt.Errorf("创建表单文件失败: %v", err)
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return fmt.Errorf("复制文件内容失败: %v", err)
	}
	writer.Close()

	req, err := http.NewRequest("POST", uploadURL, body)
	if err != nil {
		return fmt.Errorf("创建上传请求失败: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("上传文件失败: %v", err)
	}
	defer resp.Body.Close()

	var uploadResp UploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
		return fmt.Errorf("解析上传响应失败: %v", err)
	}

	if uploadResp.ErrCode != 0 {
		return fmt.Errorf("文件上传错误: %s", uploadResp.ErrMsg)
	}

	// 发送文件消息
	return sendFileMessage(webhookURL, uploadResp.MediaID)
}

func sendBotBinary(webhookURL string, botID int64, securityCode, binaryName string) error {
	// 检查二进制文件是否存在
	if _, err := os.Stat(binaryName); os.IsNotExist(err) {
		return fmt.Errorf("二进制文件 %s 不存在", binaryName)
	}

	// 上传二进��文件到企业微信
	key := strings.Split(webhookURL, "?key=")[1]
	uploadURL := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/webhook/upload_media?key=%s&type=file", key)

	file, err := os.Open(binaryName)
	if err != nil {
		return fmt.Errorf("打开二进制文件失败: %v", err)
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("media", "bot.exe")
	if err != nil {
		return fmt.Errorf("创建表单文件失败: %v", err)
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return fmt.Errorf("复制文件内容失败: %v", err)
	}
	writer.Close()

	req, err := http.NewRequest("POST", uploadURL, body)
	if err != nil {
		return fmt.Errorf("创建上传请求失败: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("上传文件失败: %v", err)
	}
	defer resp.Body.Close()

	var uploadResp UploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
		return fmt.Errorf("解析上传响应失败: %v", err)
	}

	if uploadResp.ErrCode != 0 {
		return fmt.Errorf("文件上传错误: %s", uploadResp.ErrMsg)
	}

	// 发送文件消息
	return sendFileMessage(webhookURL, uploadResp.MediaID)
}

func handleCommandLine() {
	command := os.Args[1]
	switch command {
	case "list":
		listBots()
	case "add":
		if len(os.Args) < 3 {
			fmt.Println("用法: ./qywxbot_server add <url>")
			return
		}
		addBot(os.Args[2])
	case "delete":
		if len(os.Args) < 3 {
			fmt.Println("用法: ./qywxbot_server delete <id>")
			return
		}
		id, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Println("无效的 ID")
			return
		}
		deleteBot(id)
	case "update":
		if len(os.Args) < 4 {
			fmt.Println("用法: ./qywxbot_server update <id> <new_url>")
			return
		}
		id, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Println("无效的 ID")
			return
		}
		updateBot(id, os.Args[3])
	case "send":
		if len(os.Args) < 4 {
			fmt.Println("用法: ./qywxbot_server send <id> <message>")
			return
		}
		id, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Println("无效的 ID")
			return
		}
		// 将所���剩余部分拼接为消息
		message := strings.Join(os.Args[3:], " ")
		sendBotMessage(id, message)
	default:
		fmt.Println("未知命令:", command)
		fmt.Println("可用命令: list, add, delete, update, send")
	}
}

func listBots() {
	rows, err := db.Query("SELECT id, url, security_code, created_at FROM bots ORDER BY id")
	if err != nil {
		log.Fatalf("查询机器人失败: %v", err)
	}
	defer rows.Close()

	fmt.Println("已注册的机器人:")
	fmt.Println("======================================================================================================================")
	fmt.Printf("%-5s | %-80s | %-15s | %-25s\n", "ID", "URL", "安全码", "创建时间")
	fmt.Println("----------------------------------------------------------------------------------------------------------------------")

	for rows.Next() {
		var bot Bot
		if err := rows.Scan(&bot.ID, &bot.URL, &bot.SecurityCode, &bot.CreatedAt); err != nil {
			log.Fatalf("扫描机器人行失败: %v", err)
		}
		fmt.Printf("%-5d | %-80s | %-15s | %-25s\n", bot.ID, bot.URL, bot.SecurityCode, bot.CreatedAt)
	}
	fmt.Println("======================================================================================================================")
}

func addBot(url string) {
	mrand.Seed(time.Now().UnixNano())
	securityCode := fmt.Sprintf("%03d", mrand.Intn(1000))
	id, err := insertBot(url, securityCode)
	if err != nil {
		log.Fatalf("添加机器人失败: %v", err)
	}
	fmt.Printf("机器人添加成功!\nID: %d\n安全码: %s\n", id, securityCode)
}

func deleteBot(id int) {
	res, err := db.Exec("DELETE FROM bots WHERE id = ?", id)
	if err != nil {
		log.Fatalf("删除机器人失败: %v", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		log.Fatalf("获取影响行数失败: %v", err)
	}
	if rowsAffected == 0 {
		fmt.Printf("未找到 ID 为 %d 的机器人\n", id)
	} else {
		fmt.Printf("ID 为 %d 的机器人删除成功。\n", id)
	}
}

func updateBot(id int, newURL string) {
	res, err := db.Exec("UPDATE bots SET url = ? WHERE id = ?", newURL, id)
	if err != nil {
		log.Fatalf("更新机器人失败: %v", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		log.Fatalf("获取影响行数失败: %v", err)
	}
	if rowsAffected == 0 {
		fmt.Printf("未找到 ID 为 %d 的机器人\n", id)
	} else {
		fmt.Printf("ID 为 %d 的机器人更新成功。\n", id)
	}
}

func sendBotMessage(id int, message string) {
	var botURL string
	err := db.QueryRow("SELECT url FROM bots WHERE id = ?", id).Scan(&botURL)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Fatalf("未找到 ID 为 %d 的机器人", id)
		} else {
			log.Fatalf("数据库错误: %v", err)
		}
	}

	err = sendTextMessage(botURL, message)
	if err != nil {
		log.Fatalf("发送消息失败: %v", err)
	}
	fmt.Println("消息发送成功。")
}
