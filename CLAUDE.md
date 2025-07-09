# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

这是一个用 Go 语言编写的企业微信机器人 Webhook 管理服务。它提供了网页界面注册机器人、REST API 接口、文件上传发送功能，并可生成跨平台客户端工具。服务支持多种部署模式，包括 HTTP、手动 HTTPS 和 Let's Encrypt 自动 HTTPS。

## 架构说明

### 核心应用程序 (`main.go`)
- **单文件架构**: 所有应用逻辑都包含在 `main.go` 中（约1100行代码）
- **HTTP 服务器**: 使用标准 `net/http` 库进行路由
- **数据库**: SQLite 配合 `go-sqlite3` 驱动，在 `bots.db` 中存储机器人信息
- **配置系统**: 基于 JSON 的配置系统，支持自动生成默认配置

### 主要 HTTP 端点
- **`/`**: 机器人注册网页界面（`GET`）和注册处理器（`POST`）
- **`/send`**: 发送文本/Markdown/文件消息的 JSON API
- **`/upload`**: 文件上传端点，返回 media_id
- **`/sendfile`**: 组合上传和发送文件的端点
- **`/console`**: 机器人管理控制台界面
- **`/api/bots`**: RESTful 机器人管理 API
- **`/web/`**: 文档的静态文件服务器

### 消息类型和结构体
- **`WeComTextMessage`**: 文本消息结构
- **`WeComMarkdownMessage`**: Markdown 消息结构
- **`WeComFileMessage`**: 文件消息结构
- **`WeComTemplateCardMessage`**: 富文本卡片消息结构
- **`SendMessageRequest`**: API 请求反序列化结构
- **`UploadResponse`**: 文件上传响应结构

### SSL/TLS 支持
- **手动 HTTPS**: 使用提供的证书文件
- **自动 HTTPS**: Let's Encrypt 集成，使用 HTTP-01 验证
- **ACME 客户端**: 使用 go-acme/lego 库的自定义实现

### 客户端工具生成
- **`bot.sh`**: Shell 脚本模板，支持参数替换
- **`bot.bat`**: Windows 批处理脚本模板
- **`bot.exe`**: Windows 二进制工具（独立的 CLI 应用程序）

## 常用命令

### 构建和运行
```bash
# 构建主服务器
go build -o qywxbot_server main.go

# 构建 Windows CLI 工具
cd cmd/bot-cli && GOOS=windows GOARCH=amd64 go build -o ../../bot.exe bot-cli.go

# 运行服务器
./qywxbot_server

# 使用命令行选项运行
./qywxbot_server list              # 列出所有机器人
./qywxbot_server add <webhook_url> # 通过 CLI 添加机器人
./qywxbot_server delete <id>       # 按 ID 删除机器人
./qywxbot_server update <id> <url> # 更新机器人 webhook URL
./qywxbot_server send <id> <msg>   # 通过 CLI 发送消息
```

### 开发命令
```bash
# 管理依赖
go mod tidy

# 测试应用程序
go run main.go

# 检查 Go 模块问题
go mod verify
```

## 配置系统

应用程序使用 `config.json` 进行配置，支持自动生成：

```json
{
  "http_port": ":8080",
  "https_port": ":443", 
  "cert_file": "",
  "key_file": "",
  "domain": "",
  "email_for_acme": ""
}
```

### 部署模式
1. **HTTP 模式**: 默认模式，`cert_file` 和 `key_file` 为空
2. **手动 HTTPS**: 提供 `cert_file` 和 `key_file` 路径
3. **自动 HTTPS**: 设置 `domain` 和 `email_for_acme` 启用 Let's Encrypt

## 数据库架构

SQLite 表结构：
```sql
CREATE TABLE bots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    url TEXT,
    security_code TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

## 文件结构

- **`templates/`**: HTML 模板文件 (index.html, success.html, console.html)
- **`web/`**: 静态文档文件
- **`cmd/bot-cli/`**: Windows CLI 工具源代码
- **`certs/`**: 自动生成的 SSL 证书目录
- **`bots.db`**: SQLite 数据库文件（自动创建）
- **`.acme_user_key`**: ACME 私钥（Let's Encrypt 自动生成）
