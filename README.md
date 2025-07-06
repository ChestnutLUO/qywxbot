# 企业微信机器人 Webhook 服务

这是一个使用 Go 语言编写的简单 Web 服务，用于管理企业微信机器人的 Webhook。它提供了一个 Web 界面来注册机器人，并提供了一个 JSON API 来向已注册的机器人发送消息。

## 功能

- 通过 Web 界面快速注册新的企业微信机器人。
- 支持通过 JSON API 发送文本、Markdown 和文件消息。
- 通过 `config.json` 文件进行灵活配置。
- 支持三种服务器运行模式：
  1.  **HTTP** (默认)
  2.  **手动 HTTPS** (使用您自己的证书)
  3.  **自动 HTTPS** (使用 Let's Encrypt 自动获取和续订证书)

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
- `cert_file` (可选): 手动 HTTPS ���式下，您的 SSL 证书文件路径 (例如, `certs/mycert.pem`)。
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
- 服���器必须可以从公网通过 **80 端口** 访问，以便 Let's Encrypt 完成域名验证 (HTTP-01 质询)。
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

## 使用方法

### 1. 编译

```bash
go build -o qywxbot_server
```

### 2. 运行

```bash
./qywxbot_server
```

服务器将根据您的 `config.json` 配置启动。

## API 文档

关于如何调用 `/send`, `/upload` 等接口的详细信息，请参阅 [API.md](API.md)。
