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
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge/http01"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
	_ "github.com/mattn/go-sqlite3"
)

type Config struct {
	ListenAddress string `json:"listen_address"`
	CertFile      string `json:"cert_file"`
	KeyFile       string `json:"key_file"`
	Domain        string `json:"domain"`
	EmailForACME  string `json:"email_for_acme"`
}




type SendMessageRequest struct {
	ID      int    `json:"id"`
	MsgType string `json:"msgtype"`
	Content string `json:"content"`
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





var db *sql.DB
var tmpl *template.Template
var config Config


type Bot struct {
	ID  int
	URL string
}

func main() {
	loadConfig()
	manageCertificate()

	var err error
	db, err = sql.Open("sqlite3", "./bots.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	createTable()

	tmpl = template.Must(template.ParseFiles("templates/index.html"))

	http.HandleFunc("/", handler)
	http.HandleFunc("/send", sendHandler)
	http.HandleFunc("/upload", uploadHandler)
	http.HandleFunc("/sendfile", sendfileHandler)

	if config.CertFile != "" && config.KeyFile != "" {
		log.Printf("HTTPS 服务器正在 %s 启动...", config.ListenAddress)
		log.Fatal(http.ListenAndServeTLS(config.ListenAddress, config.CertFile, config.KeyFile, nil))
	} else {
		log.Printf("HTTP 服务器正在 %s 启动...", config.ListenAddress)
		log.Fatal(http.ListenAndServe(config.ListenAddress, nil))
	}
}

func loadConfig() {
	file, err := os.Open("config.json")
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("配置文件 config.json 未找到，正在创建默认配置...")
			config.ListenAddress = ":8080"
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

	err = client.Challenge.SetHTTP01Provider(http01.NewProviderServer(config.ListenAddress, ""))
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
		"url" TEXT
	  );`

	statement, err := db.Prepare(createTableSQL)
	if err != nil {
		log.Fatal(err)
	}
	statement.Exec()
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		url := r.FormValue("url")
		if url == "" {
			http.Error(w, "URL 为必填项", http.StatusBadRequest)
			return
		}

		// Check if the bot already exists
		var existingID int
		err := db.QueryRow("SELECT id FROM bots WHERE url = ?", url).Scan(&existingID)
		if err != nil && err != sql.ErrNoRows {
			http.Error(w, "数据库错误", http.StatusInternalServerError)
			return
		}

		if existingID != 0 {
			fmt.Fprintf(w, "该 URL 的机器人已注册，ID 为: %d", existingID)
			return
		}

		id, err := insertBot(url)
		if err != nil {
			http.Error(w, "机器人注册失败", http.StatusInternalServerError)
			return
		}

		err = sendTemplateCardMessage(url, id)
		if err != nil {
			log.Printf("发送确认消息失败 %s: %v", url, err)
		}

		fmt.Fprintf(w, "机器人注册成功，ID 为: %d", id)
		return
	}

	tmpl.Execute(w, nil)
}

func sendHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "仅支持 POST 方法", http.StatusMethodNotAllowed)
		return
	}

	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "无效的请求体", http.StatusBadRequest)
		return
	}

	var botURL string
	err := db.QueryRow("SELECT url FROM bots WHERE id = ?", req.ID).Scan(&botURL)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "未找到机器人", http.StatusNotFound)
		} else {
			http.Error(w, "数据库错误", http.StatusInternalServerError)
		}
		return
	}

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
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": sendErr.Error()})
	} else {
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
	if r.Method != http.MethodPost {
		http.Error(w, "仅支持 POST 方法", http.StatusMethodNotAllowed)
		return
	}

	botID := r.FormValue("id")
	if botID == "" {
		http.Error(w, "机器人 ID 是必填项", http.StatusBadRequest)
		return
	}

	var botURL string
	err := db.QueryRow("SELECT url FROM bots WHERE id = ?", botID).Scan(&botURL)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "未找到机器人", http.StatusNotFound)
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
		http.Error(w, fmt.Sprintf("文件��传错误: %s", uploadResp.ErrMsg), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(uploadResp)
}

func sendfileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "仅支持 POST 方法", http.StatusMethodNotAllowed)
		return
	}

	botID := r.FormValue("id")
	if botID == "" {
		http.Error(w, "机器人 ID 是必填项", http.StatusBadRequest)
		return
	}

	var botURL string
	err := db.QueryRow("SELECT url FROM bots WHERE id = ?", botID).Scan(&botURL)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "未找到机器人", http.StatusNotFound)
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
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("发送消息失败，状态码: %d", resp.StatusCode)
	}
	return nil
}

func sendTemplateCardMessage(webhookURL string, botID int64) error {
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
				Desc:  "机器人 ID",
			},
			SubTitleText: "感谢您的使用！",
			CardAction: CardAction{
				Type: 1,
				URL:  "https://work.weixin.qq.com",
			},
		},
	}

	payload, err := json.Marshal(card)
	if err != nil {
		return err
	}
	return postMessage(webhookURL, payload)
}



func insertBot(url string) (int64, error) {
	insertSQL := "INSERT INTO bots(url) VALUES (?)"
	statement, err := db.Prepare(insertSQL)
	if err != nil {
		return 0, err
	}
	result, err := statement.Exec(url)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}
