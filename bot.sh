#!/bin/bash

# 机器人配置
BOT_ID="{BOT_ID_Template}"
SECURITY_CODE="{SECURITY_CODE_Template}"
SERVER_URL="{SERVER_URL_Template}"
WEBHOOK_URL="{WEBHOOK_URL_Template}"

# 从 webhook URL 中提取 key 参数
extract_webhook_key() {
    if [ -z "$WEBHOOK_URL" ] || [ "$WEBHOOK_URL" = "{WEBHOOK_URL_Template}" ]; then
        return 1
    fi
    
    # 提取 key 参数: https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=XXXXXXX
    echo "$WEBHOOK_URL" | sed 's/.*key=//'
}

# 直接上传文件到企业微信并发送
upload_file_to_webhook() {
    local filepath="$1"
    
    echo "正在尝试直接上传文件到企业微信..."
    
    # 提取 webhook key
    local webhook_key
    webhook_key=$(extract_webhook_key)
    if [ $? -ne 0 ] || [ -z "$webhook_key" ]; then
        echo "错误: 无法从 webhook URL 中提取 key 参数"
        return 1
    fi
    
    echo "提取的 webhook key: ${webhook_key:0:10}..."
    
    # 企业微信文件上传 API
    local upload_url="https://qyapi.weixin.qq.com/cgi-bin/webhook/upload_media?key=${webhook_key}&type=file"
    
    echo "正在上传文件到企业微信..."
    
    # 上传文件
    local upload_response
    upload_response=$(curl -X POST "$upload_url" \
        -F "media=@${filepath}" \
        --max-time 60 \
        --connect-timeout 15 \
        -w "HTTP_CODE:%{http_code}" \
        2>/dev/null)
    
    local upload_exit_code=$?
    local upload_http_code=$(echo "$upload_response" | grep -o "HTTP_CODE:[0-9]*" | cut -d: -f2)
    local upload_body=$(echo "$upload_response" | sed 's/HTTP_CODE:[0-9]*$//')
    
    if [ $upload_exit_code -ne 0 ] || [ "$upload_http_code" != "200" ]; then
        echo "文件上传失败 (HTTP: $upload_http_code, Exit: $upload_exit_code)"
        echo "响应: $upload_body"
        return 1
    fi
    
    echo "文件上传成功，正在解析响应..."
    
    # 解析 media_id (简单的 JSON 解析)
    local media_id
    media_id=$(echo "$upload_body" | grep -o '"media_id":"[^"]*"' | cut -d'"' -f4)
    
    if [ -z "$media_id" ]; then
        echo "错误: 无法从上传响应中提取 media_id"
        echo "响应: $upload_body"
        return 1
    fi
    
    echo "获取到 media_id: ${media_id:0:20}..."
    
    # 发送文件消息
    echo "正在发送文件消息..."
    local send_response
    send_response=$(curl -X POST "$WEBHOOK_URL" \
        -H "Content-Type: application/json" \
        -d "{
            \"msgtype\": \"file\",
            \"file\": {
                \"media_id\": \"$media_id\"
            }
        }" \
        --max-time 30 \
        --connect-timeout 10 \
        -w "HTTP_CODE:%{http_code}" \
        2>/dev/null)
    
    local send_exit_code=$?
    local send_http_code=$(echo "$send_response" | grep -o "HTTP_CODE:[0-9]*" | cut -d: -f2)
    
    if [ $send_exit_code -eq 0 ] && [ "$send_http_code" = "200" ]; then
        echo "文件发送成功 (直接到企业微信 webhook)"
        return 0
    else
        echo "文件消息发送失败 (HTTP: $send_http_code, Exit: $send_exit_code)"
        echo "响应: $(echo "$send_response" | sed 's/HTTP_CODE:[0-9]*$//')"
        return 1
    fi
}

# 直接发送到企业微信 webhook 的函数
send_to_webhook() {
    local msgtype="$1"
    local content="$2"
    
    echo "正在尝试直接发送到企业微信 webhook..."
    
    case "$msgtype" in
        "text")
            curl -X POST "$WEBHOOK_URL" \
                -H "Content-Type: application/json" \
                -d "{
                    \"msgtype\": \"text\",
                    \"text\": {
                        \"content\": \"$content\"
                    }
                }"
            ;;
        "markdown")
            curl -X POST "$WEBHOOK_URL" \
                -H "Content-Type: application/json" \
                -d "{
                    \"msgtype\": \"markdown\",
                    \"markdown\": {
                        \"content\": \"$content\"
                    }
                }"
            ;;
        *)
            echo "错误: 直接发送到 webhook 时不支持的消息类型: $msgtype"
            return 1
            ;;
    esac
}

# 发送消息函数（带 fallback）
send_message() {
    local msgtype="$1"
    local content="$2"
    
    echo "正在尝试发送到 qywxbot 服务器..."
    
    # 首先尝试发送到 qywxbot 服务器
    local response
    response=$(curl -X POST "${SERVER_URL}/send" \
        -H "Content-Type: application/json" \
        -d "{
            \"id\": ${BOT_ID},
            \"security_code\": \"${SECURITY_CODE}\",
            \"msgtype\": \"${msgtype}\",
            \"content\": \"${content}\"
        }" \
        --max-time 10 \
        --connect-timeout 5 \
        -w "HTTP_CODE:%{http_code}" \
        2>/dev/null)
    
    local exit_code=$?
    local http_code=$(echo "$response" | grep -o "HTTP_CODE:[0-9]*" | cut -d: -f2)
    
    # 检查是否成功
    if [ $exit_code -eq 0 ] && [ "$http_code" = "200" ]; then
        echo "成功发送到 qywxbot 服务器"
        echo "$response" | sed 's/HTTP_CODE:[0-9]*$//'
        return 0
    else
        echo "qywxbot 服务器不可达或响应错误 (HTTP: $http_code, Exit: $exit_code)"
        
        # 如果有 webhook URL，尝试 fallback
        if [ -n "$WEBHOOK_URL" ] && [ "$WEBHOOK_URL" != "{WEBHOOK_URL_Template}" ]; then
            echo "尝试 fallback 到企业微信 webhook..."
            send_to_webhook "$msgtype" "$content"
        else
            echo "错误: 无法连接到 qywxbot 服务器，且未配置 webhook fallback"
            return 1
        fi
    fi
}

# 发送文件函数（带 fallback）
send_file() {
    local filepath="$1"

    if [ ! -f "$filepath" ]; then
        echo "文件不存在: $filepath"
        return 1
    fi

    echo "正在尝试发送文件到 qywxbot 服务器..."
    
    # 首先尝试发送到 qywxbot 服务器
    local response
    response=$(curl -X POST "${SERVER_URL}/sendfile" \
        -F "id=${BOT_ID}" \
        -F "security_code=${SECURITY_CODE}" \
        -F "media=@${filepath}" \
        --max-time 30 \
        --connect-timeout 10 \
        -w "HTTP_CODE:%{http_code}" \
        2>/dev/null)
    
    local exit_code=$?
    local http_code=$(echo "$response" | grep -o "HTTP_CODE:[0-9]*" | cut -d: -f2)
    
    # 检查是否成功
    if [ $exit_code -eq 0 ] && [ "$http_code" = "200" ]; then
        echo "成功发送文件到 qywxbot 服务器"
        echo "$response" | sed 's/HTTP_CODE:[0-9]*$//'
        return 0
    else
        echo "qywxbot 服务器不可达或响应错误 (HTTP: $http_code, Exit: $exit_code)"
        
        # 如果有 webhook URL，尝试 fallback 直接上传
        if [ -n "$WEBHOOK_URL" ] && [ "$WEBHOOK_URL" != "{WEBHOOK_URL_Template}" ]; then
            echo "尝试 fallback 到企业微信直接文件上传..."
            upload_file_to_webhook "$filepath"
        else
            echo "错误: 无法连接到 qywxbot 服务器，且未配置 webhook fallback"
            return 1
        fi
    fi
}

# 发送文本消息
send_text() {
    send_message "text" "$1"
}

# 发送Markdown消息
send_markdown() {
    send_message "markdown" "$1"
}

# 使用示例
if [ $# -eq 0 ]; then
    echo "企业微信机器人 Shell 脚本"
    echo ""
    echo "配置信息:"
    echo "  机器人ID: $BOT_ID"
    echo "  服务器: $SERVER_URL"
    echo "  Webhook: ${WEBHOOK_URL:0:50}..."
    echo ""
    echo "使用方法:"
    echo "  发送文本消息: $0 send \"消息内容\""
    echo "  发送Markdown: $0 markdown \"**粗体文本**\""
    echo "  发送文件: $0 sendfile \"/path/to/file.txt\""
    echo ""
    echo "Fallback 机制:"
    echo "  优先尝试发送到 qywxbot 服务器"
    echo "  如果失败，自动 fallback 到企业微信 webhook"
    echo "  🔥 新功能: 文件发送现在也支持 fallback!"
    echo ""
    echo "Fallback 详细功能:"
    echo "  📝 文本/Markdown 消息: qywxbot服务器 → webhook直发"
    echo "  📁 文件发送: qywxbot服务器 → 企业微信上传API → webhook发送"
    echo "  ⚡ 自动切换: 检测服务器不可达时自动使用 fallback"
    exit 1
fi

case "$1" in
    "send")
        send_text "$2"
        ;;
    "markdown")
        send_markdown "$2"
        ;;
    "sendfile")
        send_file "$2"
        ;;
    *)
        echo "不支持的消息类型: $1"
        echo "支持的类型: send, markdown, sendfile"
        exit 1
        ;;
esac
