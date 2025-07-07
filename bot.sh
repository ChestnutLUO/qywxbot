#!/bin/bash

# 机器人配置
BOT_ID="{BOT_ID_Template}"
SECURITY_CODE="{SECURITY_CODE_Template}"
SERVER_URL="{SERVER_URL_Template}"

# 发送消息函数
send_message() {
    local msgtype="$1"
    local content="$2"

    curl -X POST "${SERVER_URL}/send" \
        -H "Content-Type: application/json" \
        -d "{
            \"id\": ${BOT_ID},
            \"security_code\": \"${SECURITY_CODE}\",
            \"msgtype\": \"${msgtype}\",
            \"content\": \"${content}\"
        }"
}

# 发送文本消息
send_text() {
    send_message "text" "$1"
}

# 发送Markdown消息
send_markdown() {
    send_message "markdown" "$1"
}

# 上传并发送文件
send_file() {
    local filepath="$1"

    if [ ! -f "$filepath" ]; then
        echo "文件不存在: $filepath"
        return 1
    fi

    curl -X POST "${SERVER_URL}/sendfile" \
        -F "id=${BOT_ID}" \
        -F "security_code=${SECURITY_CODE}" \
        -F "media=@${filepath}"
}

# 使用示例
if [ $# -eq 0 ]; then
    echo "使用方法:"
    echo "  发送文本消息: $0 text \"消息内容\""
    echo "  发送Markdown: $0 markdown \"**粗体文本**\""
    echo "  发送文件: $0 file \"/path/to/file.txt\""
    exit 1
fi

case "$1" in
    "send")
        send_text "$2"
        ;;
    "markdown")
        send_markdown "$2"
        ;;
    "send_file")
        send_file "$2"
        ;;
    *)
        echo "不支持的消息类型: $1"
        echo "支持的类型: text, markdown, file"
        exit 1
        ;;
esac
