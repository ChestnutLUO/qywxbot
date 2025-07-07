# 企业微信机器人 Webhook 服务

这是一个功能完整的企业微信机器人管理服务，使用 Go 语言编写。它提供了 Web 界面注册机器人、REST API 接口、脚本工具和 Windows 二进制程序，满足各种使用场景的需求。

## ✨ 核心功能

- 🌐 **Web 界面注册**：通过简洁的网页界面快速注册企业微信机器人
- 🚀 **REST API**：完整的 JSON API 支持文本、Markdown 和文件消息
- 📁 **文件服务**：支持文件上传、发送和 media_id 管理
- 🔒 **安全验证**：每个机器人独立的安全码验证机制
- 📜 **脚本生成**：自动生成 Shell、批处理和 Windows 二进制工具
- 📚 **在线文档**：完整的 Web 文档系统，包含使用指南和示例
- 🌍 **多模式部署**：支持 HTTP、手动 HTTPS 和自动 HTTPS（Let's Encrypt）

## 🛠️ 支持的工具

### 1. Shell 脚本 (Linux/macOS)
- `bot.sh` - 支持发送消息和文件的 Shell 脚本

### 2. 批处理脚本 (Windows)
- `bot.bat` - Windows 批处理脚本，功能与 Shell 脚本相同

### 3. Windows 二进制程序
- `bot.exe` - 功能完整的 Windows 命令行工具
- 支持 `send`、`sendfile`、`upload` 命令
- 零依赖，单文件部署

## 📖 在线文档

启动服务后，您可以通过 `/web/` 访问完整的在线文档：

- **文档首页** (`/web/`) - 服务概述和快速导航
- **API 文档** (`/web/api.html`) - 接口参考文档
- **API 使用文档** (`/web/api-usage.html`) - 详细的 API 使用指南
- **Bot 脚本文档** (`/web/bot-scripts.html`) - Shell 和批处理脚本说明
- **Windows 程序** (`/web/windows-binary.html`) - bot.exe 完整使用文档
- **API 使用示例** (`/web/examples.html`) - 多语言调用示例

## 配置

在首次运行应用程序之前，请在项目根目录创建一个 `config.json` 文件。如果该文件不存在，系统将在首次启动时自动创建一个默认配置文件。

以下是一个完整的配置示例：

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

### 字段说明

- `http_port` (必需): HTTP 服务器监听的端口 (例如, `:8080` 或 `:80`)。
- `https_port` (可选): HTTPS 服务器监听的端口 (例如, `:443`)。
- `cert_file` (可选): 手动 HTTPS 模式下，您的 SSL 证书文件路径 (例如, `certs/mycert.pem`)。
- `key_file` (可选): 手动 HTTPS 模式下，您的 SSL 私钥文件路径 (例如, `certs/mykey.pem`)。
- `domain` (可选): 外部访问域名，用于自动 HTTPS 模式 (例如, `bot.example.com`)。留空则为内网模式。
- `email_for_acme` (可选): 自动 HTTPS 模式下，用于 Let's Encrypt 注册和通知的电子邮件地址。

## 运行模式

服务可以根据 `config.json` 的配置以三种不同的模式运行。

### 1. HTTP 模式 (默认)

这是最简单的模式，适用于本地测试、内网使用或在反向代理之后运行。

**配置**:
确保 `cert_file`, `key_file`, 和 `domain` 字段均为空。

**内网使用示例**:
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

**访问方式**：
- 本机访问：`http://localhost:8080`
- 内网访问：`http://[服务器IP]:8080`

**优点**：
- 配置简单，无需证书
- 适合内网开发和测试
- 避免ACME证书申请的复杂性

### 2. 手动 HTTPS 模式

如果您已经拥有自己的 SSL 证书，可以使用此模式。

**配置**:
在 `cert_file` 和 `key_file` 字段中提供证书和私钥的路径。

```json
{
  "http_port": ":80",
  "https_port": ":443",
  "cert_file": "path/to/your/cert.pem",
  "key_file": "path/to/your/key.pem",
  "domain": "your.domain.com",
  "email_for_acme": ""
}
```

### 3. 自动 HTTPS 模式 (Let's Encrypt)

此模式将为您的域名自动获取和管理 SSL 证书。

**重要**:
- 服务器必须可以从公网通过 **80 端口** 访问，以便 Let's Encrypt 完成域名验证 (HTTP-01 质询)。
- 首次运行后，系统会在项目根目录创建一个 `.acme_user_key` 文件。请**不要**将此文件上传到代码仓库。建议将其添加到 `.gitignore`。
- 获取的证书将保存在 `certs/` 目录下。

**配置**:
填写 `domain` 和 `email_for_acme` 字段。`https_port` 通常应设置为 `:443`。

```json
{
  "http_port": ":80",
  "https_port": ":443",
  "cert_file": "",
  "key_file": "",
  "domain": "your.domain.com",
  "email_for_acme": "your-email@example.com"
}
```

## 🚀 快速开始

### 1. 编译服务器

```bash
# 编译主服务器
go build -o qywxbot_server main.go

# 编译 Windows 二进制工具 (可选)
cd cmd && GOOS=windows GOARCH=amd64 go build -o ../bot.exe bot-cli.go
```

### 2. 启动服务

```bash
./qywxbot_server
```

服务器将根据您的 `config.json` 配置启动。

### 3. 注册机器人

1. 访问 `http://localhost:8080` (或您的服务器地址)
2. 输入企业微信机器人的 Webhook URL
3. 点击注册，获取机器人 ID 和安全码
4. 系统会自动发送脚本和工具到您的企业微信群

### 4. 使用工具

注册成功后，您可以使用多种方式发送消息：

#### API 调用
```bash
curl -X POST http://localhost:8080/send \
  -H "Content-Type: application/json" \
  -d '{
    "id": 1,
    "security_code": "123",
    "msgtype": "text",
    "content": "Hello World!"
  }'
```

#### Shell 脚本 (Linux/macOS)
```bash
./bot.sh send "Hello from Shell!"
./bot.sh sendfile "/path/to/file.pdf"
```

#### Windows 批处理
```cmd
bot.bat send "Hello from Windows!"
bot.bat sendfile "C:\path\to\file.pdf"
```

#### Windows 二进制程序
```cmd
bot.exe send localhost 8080 1 123 "Hello from bot.exe!"
bot.exe sendfile localhost 8080 1 123 "C:\path\to\file.pdf"
bot.exe upload localhost 8080 1 123 "C:\path\to\file.pdf"
```

## 📚 文档和 API 参考

### 在线文档
- 🌐 **完整文档**: 访问 `/web/` 获取包含所有功能的在线文档
- 📖 **快速指南**: 每个工具都有详细的使用说明和示例

### API 接口
- **POST** `/send` - 发送文本或 Markdown 消息
- **POST** `/upload` - 上传文件并获取 media_id
- **POST** `/sendfile` - 上传并发送文件 (一步完成)
- **GET** `/` - Web 注册界面
- **GET** `/web/*` - 在线文档系统

详细的 API 使用说明请参阅：
- [API.md](API.md) - 原始 API 文档
- `/web/api-usage.html` - 在线 API 使用指南

## 🔧 高级配置

### 环境变量支持
可以通过环境变量覆盖配置文件设置：

```bash
export HTTP_PORT=":9000"
export DOMAIN="bot.example.com"
export EMAIL_FOR_ACME="admin@example.com"
./qywxbot_server
```

### 反向代理配置
如果在 Nginx 等反向代理后运行，建议配置：

```nginx
location / {
    proxy_pass http://localhost:8080;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}
```

## 🛡️ 安全建议

- 🔒 **安全码管理**: 妥善保管机器人安全码，不要在代码中硬编码
- 🌐 **网络安全**: 生产环境建议使用 HTTPS
- 📁 **文件权限**: 确保数据库文件 `bots.db` 的访问权限
- 🔑 **ACME 密钥**: 保护好 `.acme_user_key` 文件，不要提交到版本控制

## 🤝 常见使用场景

- 📊 **监控告警**: 集成到监控系统发送告警消息
- 🔄 **CI/CD 通知**: 构建和部署状态通知
- 📈 **报告推送**: 定时推送业务报告和数据统计
- 🎯 **运维自动化**: 系统状态检查和日志推送

## 🆘 故障排查

### 常见问题
1. **编译失败**: 确保 Go 版本 >= 1.19
2. **端口占用**: 检查配置的端口是否被占用
3. **权限问题**: 确保程序有读写当前目录的权限
4. **网络问题**: 检查防火墙和网络配置

### 调试模式
启用详细日志：
```bash
export DEBUG=true
./qywxbot_server
```
