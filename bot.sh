#!/bin/bash

# 用于与 qywxbot 服务器交互的简化版命令行工具

# 当任何命令失败时立即退出
set -e

# 默认服务器 URL 和机器人 ID
SERVER_URL="http://localhost:8080"
DEFAULT_BOT_ID="1" # 在这里设置您的默认机器人 ID

# --- 使用说明函数 ---
usage() {
    echo "用法: $0 <command> <argument>"
    echo ""
    echo "命令:"
    echo "  send     <message>      将文本作为 Markdown 消息发送。"
    echo "  sendfile <file_path>     发送一个文件。"
    echo ""
    echo "示例:"
    echo "  $0 send \"### 新的警告\n请立即检查系统状态。\""
    echo "  $0 sendfile ./API.md"
    echo ""
    echo "默认机器人 ID 设置为: ${DEFAULT_BOT_ID}"
    exit 1
}

# --- 主脚本 ---
if [ "$#" -ne 2 ]; then
    echo "错误: 需要两个参数：一个命令和一个参数。"
    usage
fi

COMMAND=$1
ARGUMENT=$2

case $COMMAND in
    send)
        # --- 发送 Markdown 消息 ---
        echo "正在向机器人 ID: ${DEFAULT_BOT_ID} 发送 Markdown 消息..."
        curl -s -X POST "${SERVER_URL}/send" \
             -H "Content-Type: application/json" \
             -d "{\"id\": ${DEFAULT_BOT_ID}, \"msgtype\": \"markdown\", \"content\": \"${ARGUMENT}\"}"
        echo
        echo "消息发送完成。"
        ;;

    sendfile)
        # --- 发送文件 ---
        if [ ! -f "$ARGUMENT" ]; then
            echo "错误: 文件 '$ARGUMENT' 未找到"
            exit 1
        fi
        echo "正在向机器人 ID: ${DEFAULT_BOT_ID} 发送文件 '$ARGUMENT'..."
        curl -s -X POST "${SERVER_URL}/sendfile" \
             -F "id=${DEFAULT_BOT_ID}" \
             -F "media=@${ARGUMENT}"
        echo
        echo "文件发送完成。"
        ;;

    *)
        echo "错误: 未知命令 '$COMMAND'"
        usage
        ;;
esac
