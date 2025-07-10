#!/bin/bash

# QyWxBot 本地构建脚本
# 用于测试 GitHub Actions 构建流程

set -e

echo "🚀 开始构建 QyWxBot..."

# 清理之前的构建
echo "🧹 清理之前的构建文件..."
rm -rf release/
rm -f qywxbot_server qywxbot_server.exe
rm -f cmd/bot-cli/cmd/bot cmd/bot-cli/cmd/bot.exe

# 设置版本
VERSION=${1:-"dev-build"}
echo "📋 构建版本: $VERSION"

# 构建函数
build_platform() {
    local goos=$1
    local goarch=$2
    local platform=$3
    local extension=$4
    
    echo "🔨 构建 $platform..."
    
    # 设置环境变量
    export GOOS=$goos
    export GOARCH=$goarch
    export CGO_ENABLED=1
    
    # 构建主服务器
    echo "  📦 构建 qywxbot_server..."
    go build -ldflags="-s -w" -o qywxbot_server${extension} main.go
    
    # 构建 bot-cli
    echo "  🔧 构建 bot-cli..."
    cd cmd/bot-cli/cmd
    go build -ldflags="-s -w" -o bot${extension} bot-cli.go
    cd ../../../
    
    # 创建发布包
    PACKAGE_NAME="qywxbot-${platform}"
    echo "  📁 创建发布包: $PACKAGE_NAME"
    mkdir -p release/$PACKAGE_NAME
    
    # 复制二进制文件
    cp qywxbot_server${extension} release/$PACKAGE_NAME/
    cp cmd/bot-cli/cmd/bot${extension} release/$PACKAGE_NAME/
    
    # 复制脚本文件
    if [ "$goos" != "windows" ]; then
        cp bot.sh release/$PACKAGE_NAME/
        chmod +x release/$PACKAGE_NAME/bot.sh
    else
        cp bot.bat release/$PACKAGE_NAME/
    fi
    
    # 复制资源文件
    cp -r web release/$PACKAGE_NAME/
    cp -r templates release/$PACKAGE_NAME/
    
    # 复制文档
    cp README.md release/$PACKAGE_NAME/ 2>/dev/null || echo "  ⚠️  README.md 不存在"
    cp API.md release/$PACKAGE_NAME/ 2>/dev/null || echo "  ⚠️  API.md 不存在"
    cp CLAUDE.md release/$PACKAGE_NAME/ 2>/dev/null || echo "  ⚠️  CLAUDE.md 不存在"
    
    # 创建示例配置
    cat > release/$PACKAGE_NAME/config.sample.json << 'EOF'
{
  "http_port": ":8080",
  "https_port": ":443",
  "cert_file": "",
  "key_file": "",
  "domain": "",
  "email_for_acme": ""
}
EOF
    
    # 创建安装指南
    cat > release/$PACKAGE_NAME/INSTALL.md << 'EOF'
# QyWxBot 安装指南

## 快速开始

1. 解压文件包
2. 复制 `config.sample.json` 为 `config.json` 并根据需要修改配置
3. 运行服务器:
   - Linux/macOS: `./qywxbot_server`
   - Windows: `qywxbot_server.exe`
4. 打开浏览器访问 `http://localhost:8080` 注册机器人

## 文件说明

- `qywxbot_server` / `qywxbot_server.exe`: 主服务器程序
- `bot` / `bot.exe`: 命令行客户端工具
- `bot.sh` / `bot.bat`: Shell 脚本工具
- `web/`: Web 界面文件
- `templates/`: HTML 模板文件
- `config.sample.json`: 示例配置文件

## 使用文档

详细使用说明请参考:
- `README.md`: 项目介绍
- `API.md`: API 文档
- `CLAUDE.md`: 开发文档
- `web/index.html`: 在线帮助文档
EOF
    
    # 创建压缩包
    cd release
    if [ "$goos" = "windows" ]; then
        echo "  📦 创建 ZIP 压缩包..."
        zip -r ${PACKAGE_NAME}.zip $PACKAGE_NAME > /dev/null
    else
        echo "  📦 创建 tar.gz 压缩包..."
        tar -czf ${PACKAGE_NAME}.tar.gz $PACKAGE_NAME
    fi
    cd ..
    
    echo "  ✅ $platform 构建完成"
}

# 检查依赖
echo "🔍 检查构建环境..."
if ! command -v go &> /dev/null; then
    echo "❌ Go 未安装，请先安装 Go"
    exit 1
fi

echo "  Go 版本: $(go version)"

# 下载依赖
echo "📥 下载 Go 模块..."
go mod download

# 构建当前平台
CURRENT_OS=$(go env GOOS)
CURRENT_ARCH=$(go env GOARCH)

echo "🎯 构建当前平台: $CURRENT_OS-$CURRENT_ARCH"

if [ "$CURRENT_OS" = "windows" ]; then
    build_platform "windows" "amd64" "windows-amd64" ".exe"
else
    build_platform "$CURRENT_OS" "$CURRENT_ARCH" "$CURRENT_OS-$CURRENT_ARCH" ""
fi

# 构建所有平台（可选）
if [ "$2" = "--all" ]; then
    echo "🌍 构建所有平台..."
    
    # Linux AMD64
    build_platform "linux" "amd64" "linux-amd64" ""
    
    # Linux ARM64
    build_platform "linux" "arm64" "linux-arm64" ""
    
    # Windows AMD64
    build_platform "windows" "amd64" "windows-amd64" ".exe"
    
    # macOS AMD64
    build_platform "darwin" "amd64" "darwin-amd64" ""
    
    # macOS ARM64
    build_platform "darwin" "arm64" "darwin-arm64" ""
fi

echo "🎉 构建完成！"
echo "📂 构建结果在 release/ 目录中:"
ls -la release/

echo ""
echo "💡 使用提示:"
echo "  - 测试当前平台: ./build.sh"
echo "  - 构建所有平台: ./build.sh dev-build --all"
echo "  - 指定版本: ./build.sh v1.0.0"