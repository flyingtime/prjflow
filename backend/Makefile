.PHONY: build build-linux build-windows build-mac clean run test frontend-build

# 构建前端（用于 embed）
frontend-build:
	@echo "构建前端..."
	@cd ../frontend && yarn build
	@echo "复制前端文件到 embed 目录..."
	@rm -rf cmd/server/frontend-dist
	@cp -r ../frontend/dist cmd/server/frontend-dist
	@echo "✓ 前端构建完成"

# 默认构建（当前平台）
build: frontend-build
	@echo "构建当前平台..."
	@go build -ldflags="-s -w" -o server cmd/server/main.go
	@echo "✓ 构建完成: ./server"

# Linux构建（静态编译，不依赖GLIBC，仅支持MySQL）
build-linux: frontend-build
	@echo "构建Linux版本（静态编译，仅支持MySQL）..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o server cmd/server/main.go
	@echo "✓ 构建完成: ./server"
	@ls -lh server

# Linux构建（支持SQLite，需要CGO，需要GLIBC匹配）
build-linux-sqlite: frontend-build
	@echo "构建Linux版本（支持SQLite，需要CGO）..."
	@echo "⚠️  警告：此版本需要服务器GLIBC版本匹配"
	@CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o server cmd/server/main.go
	@echo "✓ 构建完成: ./server"
	@ls -lh server

# Windows构建
build-windows: frontend-build
	@echo "构建Windows版本..."
	@CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o server.exe cmd/server/main.go
	@echo "✓ 构建完成: ./server.exe"

# macOS构建
build-mac: frontend-build
	@echo "构建macOS版本..."
	@CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o server-mac cmd/server/main.go
	@echo "✓ 构建完成: ./server-mac"

# 清理构建文件
clean:
	@echo "清理构建文件..."
	@rm -f server server.exe server-mac
	@rm -rf cmd/server/frontend-dist
	@echo "✓ 清理完成"

# 运行服务
run:
	@go run cmd/server/main.go

# 运行测试
test:
	@go test ./...

# 安装依赖
deps:
	@go mod download
	@go mod tidy

