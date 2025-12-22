#!/bin/bash
# 安装 Docker Compose V2 脚本
# 适用于未配置 Docker 官方仓库的系统

set -e

echo "正在下载 Docker Compose V2..."

# 获取最新版本号
COMPOSE_VERSION=$(curl -s https://api.github.com/repos/docker/compose/releases/latest | grep -oP '"tag_name": "\K[^"]+' | head -1)

if [ -z "$COMPOSE_VERSION" ]; then
    echo "无法获取最新版本，使用默认版本 v2.24.0"
    COMPOSE_VERSION="v2.24.0"
fi

echo "版本: $COMPOSE_VERSION"

# 检测架构
ARCH=$(uname -m)
case $ARCH in
    x86_64)
        ARCH="x86_64"
        ;;
    aarch64|arm64)
        ARCH="aarch64"
        ;;
    *)
        echo "不支持的架构: $ARCH"
        exit 1
        ;;
esac

# 下载 Docker Compose
DOCKER_COMPOSE_URL="https://github.com/docker/compose/releases/download/${COMPOSE_VERSION}/docker-compose-linux-${ARCH}"

echo "下载地址: $DOCKER_COMPOSE_URL"

# 创建目录
mkdir -p ~/.docker/cli-plugins

# 下载并安装
curl -L "$DOCKER_COMPOSE_URL" -o ~/.docker/cli-plugins/docker-compose

# 设置执行权限
chmod +x ~/.docker/cli-plugins/docker-compose

# 验证安装
if ~/.docker/cli-plugins/docker-compose version > /dev/null 2>&1; then
    echo "✓ Docker Compose V2 安装成功！"
    echo ""
    echo "使用方法:"
    echo "  docker compose up -d    # 注意是空格，不是连字符"
    echo ""
    echo "验证安装:"
    ~/.docker/cli-plugins/docker-compose version
else
    echo "✗ 安装失败，请检查网络连接"
    exit 1
fi

