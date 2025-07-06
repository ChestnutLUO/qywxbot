# 企业微信机器人 API 文档

本文档介绍了如何使用机器人管理和消息推送 API。

## 1. 注册机器人

通过网页界面注册新的机器人。

- **URL:** `/`
- **方法:** `GET`, `POST`

### 使用方法

1.  使用浏览器访问服务器的根 URL（例如 `http://localhost:8080/`）。
2.  您将看到一个表单，用于输入机器人的 Webhook URL。
3.  在 “机器人 Webhook URL” 字段中，粘贴您的企业微信机器人的 Webhook 地址。
4.  点击 “注册” 按钮。

### 响应

- **注册成功:**
  - **网页:** 页面上会显示 “机器人注册成功，ID 为: [ID], 安全码为: [安全码]”。
  - **企业微信:** 对应的机器人会收到一张 “机器人注册成功” 的模板卡片消息，其中包含新注册的机器人 ID 和安全码。
- **已注册:**
  - **网页:** 如果该 Webhook URL 已被注册，页面上会显示 “该 URL 的机器人已注册，ID 为: [ID], 安全码为: [安全码]”。

---

## 2. 推送消息

通过 API 向已注册的机器人推送消息。

- **URL:** `/send`
- **方法:** `POST`
- **Content-Type:** `application/json`

### 请求体

| 参数           | 类型    | 是否必填 | 描述                                       |
| :------------- | :------ | :------- | :----------------------------------------- |
| `id`           | Integer | 是       | 要接收消息的机器人的 ID。                  |
| `security_code`| String  | 是       | 机器人的三位数安全码。                     |
| `msgtype`      | String  | 是       | 消息类型，目前支持 `text`、`markdown` 或 `file`。 |
| `content`      | String  | 是       | 消息的具体内容。对于 `file` 类型，此字段应为 `media_id`。                           |

### 示例请求

#### 推送文本消息

```bash
curl -X POST http://localhost:8080/send \
-H "Content-Type: application/json" \
-d '{
    "id": 1,
    "security_code": "123",
    "msgtype": "text",
    "content": "这是一条文本测试消息。"
}'
```

#### 推送 Markdown 消息

```bash
curl -X POST http://localhost:8080/send \
-H "Content-Type: application/json" \
-d '{
    "id": 1,
    "security_code": "123",
    "msgtype": "markdown",
    "content": "### 这是一条 Markdown 消息\n> 引用内容\n- 列表项 1\n- 列表项 2\n\n请<font color=\"info\">注意</font>查收。"
}'
```

### 响应

#### 成功响应

- **状态码:** `200 OK`
- **内容:**

```json
{
    "status": "success",
    "message": "消息发送成功"
}
```

#### 失败响应

- **状态码:** `400 Bad Request`
  - **原因:** 请求体无效或消息类型不被支持。
- **状态码:** `404 Not Found`
  - **原因:** 未找到机器人或安全码不正确。
- **状态码:** `405 Method Not Allowed`
  - **原因:** 使用了非 `POST` 的 HTTP 方法。
- **状态码:** `500 Internal Server Error`
  - **原因:** 数据库错误或向企业微信 Webhook 发送消息时失败。
  - **内容示例:**
    ```json
    {
        "status": "error",
        "message": "发送消息失败，状态码: 400"
    }
    ```

---

## 3. 上传文件

上传文件以获取 `media_id`，该 ID 可用于通过 `/send` 端点发送文件消息。

- **URL:** `/upload`
- **方法:** `POST`
- **Content-Type:** `multipart/form-data`

### 表单数据

| 参数          | 类型   | 是否必填 | 描述                  |
| :------------ | :----- | :------- | :-------------------- |
| `id`          | String | 是       | 要用于上传的机器人 ID。   |
| `security_code` | String | 是       | 机器人的三位数安全码。 |
| `media`       | File   | 是       | 要上传的文件。        |

### 示例请求

```bash
curl -X POST http://localhost:8080/upload \
-F "id=1" \
-F "security_code=123" \
-F "media=@/path/to/your/file.txt"
```

### 成功响应

- **状态码:** `200 OK`
- **内容:**

```json
{
    "errcode": 0,
    "errmsg": "ok",
    "type": "file",
    "media_id": "3a8asd892asd8asd",
    "created_at": "1380000000"
}
```

### 失败响应

- **状态码:** `400 Bad Request`
- **状态码:** `404 Not Found`
  - **原因:** 未找到机器人或安全码不正确。
- **状态码:** `500 Internal Server Error`

---

## 4. 直接发送文件（一步完成）

此端点将文件的上传和发送合并为一个操作。

- **URL:** `/sendfile`
- **方法:** `POST`
- **Content-Type:** `multipart/form-data`

### 表单数据

| 参数          | 类型   | 是否必填 | 描述                  |
| :------------ | :----- | :------- | :-------------------- |
| `id`          | String | 是       | 要用于发送的机器人 ID。   |
| `security_code` | String | 是       | 机器人的三位数安全码。 |
| `media`       | File   | 是       | 要发送的文件。        |

### 示例请求

```bash
curl -X POST http://localhost:8080/sendfile \
-F "id=1" \
-F "security_code=123" \
-F "media=@/path/to/your/file.txt"
```

### 成功响应

- **状态码:** `200 OK`
- **内容:**

```json
{
    "status": "success",
    "message": "文件��送成功",
    "media_id": "3CEVBKZnCEm4Khu6b-C5fT3e-EfCZ277VmOcbsKNxeqsPQjpCiljPmiREkpcG-TaD"
}
```

### 失败响应

- **状态码:** `400 Bad Request`
- **状态码:** `404 Not Found`
  - **原因:** 未找到机器人或安全码不正确。
- **状态码:** `500 Internal Server Error`
