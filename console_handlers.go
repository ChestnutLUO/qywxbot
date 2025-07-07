package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func consoleHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "templates/console.html")
}

func botAPIHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	if r.Method == http.MethodGet {
		// 验证机器人ID和安全码
		botIDStr := r.URL.Query().Get("id")
		securityCode := r.URL.Query().Get("security_code")
		
		if botIDStr == "" || securityCode == "" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "缺少机器人ID或安全码",
			})
			return
		}
		
		botID, err := strconv.Atoi(botIDStr)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "无效的机器人ID",
			})
			return
		}
		
		// 查询机器人信息
		var bot Bot
		err = db.QueryRow("SELECT id, url, security_code, created_at FROM bots WHERE id = ? AND security_code = ?", 
			botID, securityCode).Scan(&bot.ID, &bot.URL, &bot.SecurityCode, &bot.CreatedAt)
		
		if err != nil {
			if err == sql.ErrNoRows {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"success": false,
					"message": "机器人不存在或安全码错误",
				})
			} else {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"success": false,
					"message": "数据库查询失败",
				})
			}
			return
		}
		
		// 获取服务器URL用于生成端点信息
		serverURL := getServerURL()
		
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"bot":        bot,
				"server_url": serverURL,
				"endpoints": map[string]string{
					"send":     serverURL + "/send",
					"upload":   serverURL + "/upload", 
					"sendfile": serverURL + "/sendfile",
				},
			},
		})
		
	} else if r.Method == http.MethodPost {
		// 处理重置安全码
		var req struct {
			ID           int    `json:"id"`
			SecurityCode string `json:"security_code"`
			Action       string `json:"action"`
		}
		
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "无效的请求数据",
			})
			return
		}
		
		if req.Action == "reset_security_code" {
			resetSecurityCode(w, req.ID, req.SecurityCode)
		} else {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "不支持的操作",
			})
		}
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func resetSecurityCode(w http.ResponseWriter, botID int, oldSecurityCode string) {
	// 验证当前安全码
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM bots WHERE id = ? AND security_code = ?)", 
		botID, oldSecurityCode).Scan(&exists)
	
	if err != nil || !exists {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "机器人不存在或安全码错误",
		})
		return
	}
	
	// 生成新的安全码
	newSecurityCode := generateSecurityCode()
	
	// 更新数据库
	_, err = db.Exec("UPDATE bots SET security_code = ? WHERE id = ?", newSecurityCode, botID)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "更新安全码失败",
		})
		return
	}
	
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "安全码重置成功",
		"data": map[string]interface{}{
			"new_security_code": newSecurityCode,
		},
	})
}

func generateSecurityCode() string {
	// 生成3位随机数字安全码
	bytes := make([]byte, 2)
	rand.Read(bytes)
	code := int(bytes[0])<<8 | int(bytes[1])
	return fmt.Sprintf("%03d", code%1000)
}