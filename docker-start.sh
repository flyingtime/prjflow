#!/bin/bash
# 使用 Docker 命令直接启动服务（不依赖 docker-compose）
# 使用方法: ./docker-start.sh

set -e

# 检查 Docker 是否运行
if ! docker ps > /dev/null 2>&1; then
    echo "错误: Docker 未运行或无法连接"
    exit 1
fi

# 创建数据目录
mkdir -p data

# 构建镜像
echo "构建镜像..."
docker build -t prjflow:latest .

# 停止并删除旧容器（如果存在）
if docker ps -a | grep -q prjflow; then
    echo "停止并删除旧容器..."
    docker stop prjflow 2>/dev/null || true
    docker rm prjflow 2>/dev/null || true
fi

# 启动容器
echo "启动容器..."
docker run -d \
  --name prjflow \
  -p 8080:8080 \
  -v "$(pwd)/data:/app/data" \
  -e JWT_SECRET="${JWT_SECRET:-your-secret-key-change-in-production}" \
  -e SERVER_PORT=8080 \
  -e DATABASE_TYPE=sqlite \
  -e DATABASE_DSN=/app/data/data.db \
  -e UPLOAD_STORAGE_PATH=/app/data/uploads \
  -e TZ=Asia/Shanghai \
  --restart unless-stopped \
  prjflow:latest

echo "容器已启动！"
echo "查看日志: docker logs -f prjflow"
echo "访问: http://localhost:8080"

