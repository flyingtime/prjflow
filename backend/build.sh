#!/bin/bash

# 后端构建脚本
# 使用静态编译，避免GLIBC版本依赖问题
# 注意：此脚本会自动构建前端并嵌入到后端二进制文件中

set -e

echo "开始构建后端服务..."

# 先构建前端（用于 embed）
echo "步骤 1/2: 构建前端..."
cd ../frontend
if [ ! -f "package.json" ]; then
    echo "错误: 未找到 frontend/package.json，请确保在项目根目录或 backend 目录运行此脚本"
    exit 1
fi

# 检查是否已安装依赖
if [ ! -d "node_modules" ]; then
    echo "安装前端依赖..."
    yarn install
fi

# 构建前端
yarn build

if [ ! -d "dist" ] || [ ! -f "dist/index.html" ]; then
    echo "错误: 前端构建失败，未找到 dist/index.html"
    exit 1
fi

echo "✓ 前端构建完成"
echo ""

# 返回 backend 目录
cd ../backend

# 复制前端文件到 embed 目录
echo "复制前端文件到 embed 目录..."
rm -rf cmd/server/frontend-dist
cp -r ../frontend/dist cmd/server/frontend-dist
echo "✓ 前端文件复制完成"
echo ""

echo "步骤 2/2: 构建后端..."

# 设置编译参数
export CGO_ENABLED=0
export GOOS=linux
export GOARCH=amd64

# 编译参数说明：
# - CGO_ENABLED=0: 禁用CGO，静态编译，不依赖系统库
# - GOOS=linux: 目标操作系统
# - GOARCH=amd64: 目标架构
# - -ldflags="-s -w": 减小二进制文件大小（-s去掉符号表，-w去掉调试信息）

echo "编译参数："
echo "  CGO_ENABLED=$CGO_ENABLED"
echo "  GOOS=$GOOS"
echo "  GOARCH=$GOARCH"
echo ""

# 编译
go build -ldflags="-s -w" -o server cmd/server/main.go

if [ $? -eq 0 ]; then
    echo "✓ 构建成功！"
    echo "  输出文件: ./server"
    echo "  文件大小: $(du -h server | cut -f1)"
    echo ""
    echo "提示："
    echo "  - 这是一个静态编译的二进制文件，不依赖系统GLIBC"
    echo "  - 可以直接在Linux服务器上运行，无需安装Go环境"
    echo "  - 支持SQLite（纯Go实现）和MySQL数据库"
    echo "  - 记得同时上传 config.yaml 和数据库文件（如果使用SQLite）"
else
    echo "✗ 构建失败！"
    exit 1
fi

