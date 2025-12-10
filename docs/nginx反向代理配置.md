# Nginx 反向代理配置指南

## 概述

使用公网服务器 `ng.smartxy.com.cn` 的 nginx 反向代理，将请求转发到内网服务器，解决微信回调问题。

## 架构说明

```
微信服务器 → project.smartxy.com.cn (公网) → nginx (ng.smartxy.com.cn) → 内网服务器
```

- **公网域名**：`project.smartxy.com.cn`（微信回调地址）
- **nginx 服务器**：`ng.smartxy.com.cn`（公网服务器）
- **内网服务器**：运行后端（8080）和前端（3001）的服务器

## 配置步骤

### 1. DNS 配置

确保 `project.smartxy.com.cn` 的 DNS 解析指向 `ng.smartxy.com.cn` 的 IP 地址：

```bash
# 检查 DNS 解析
nslookup project.smartxy.com.cn
# 或
dig project.smartxy.com.cn
```

### 2. Nginx 配置

在 `ng.smartxy.com.cn` 服务器上创建 nginx 配置文件：

```bash
# 创建配置文件
sudo nano /etc/nginx/sites-available/project.smartxy.com.cn
```

复制 `nginx.conf.example` 的内容，并修改以下部分：

1. **替换内网服务器IP**：将 `内网服务器IP` 替换为实际的内网服务器 IP 地址
   - 如果 nginx 服务器和内网服务器在同一网络，使用内网 IP（如 `192.168.1.100`）
   - 如果不在同一网络，需要配置 SSH 隧道或 VPN

2. **配置 SSL（推荐）**：
   ```nginx
   # 使用 Let's Encrypt 免费证书
   listen 443 ssl http2;
   ssl_certificate /etc/letsencrypt/live/project.smartxy.com.cn/fullchain.pem;
   ssl_certificate_key /etc/letsencrypt/live/project.smartxy.com.cn/privkey.pem;
   ```

### 3. 启用配置

```bash
# 创建符号链接
sudo ln -s /etc/nginx/sites-available/project.smartxy.com.cn /etc/nginx/sites-enabled/

# 测试配置
sudo nginx -t

# 重载 nginx
sudo systemctl reload nginx
```

### 4. 配置 SSL 证书（推荐）

使用 Let's Encrypt 免费证书：

```bash
# 安装 certbot
sudo apt-get update
sudo apt-get install certbot python3-certbot-nginx

# 获取证书
sudo certbot --nginx -d project.smartxy.com.cn

# 自动续期（已自动配置）
sudo certbot renew --dry-run
```

### 5. 更新后端配置

在 `backend/config.yaml` 中设置回调域名：

```yaml
wechat:
  callback_domain: "<https://你的域名>/"
```

### 6. 配置微信后台

在微信后台（开放平台或公众号）配置授权回调域名：

- **开放平台**：`project.smartxy.com.cn`（不包含协议和端口）
- **公众号**：`project.smartxy.com.cn`（不包含协议和端口）

## 网络配置

### 情况1：nginx 服务器和内网服务器在同一网络

直接使用内网 IP：

```nginx
proxy_pass http://192.168.1.100:8080;  # 内网服务器IP
```

### 情况2：nginx 服务器和内网服务器不在同一网络

#### 方案A：SSH 隧道

有两种方式建立 SSH 隧道：

**方式1：在公网服务器上建立隧道（-L，本地端口转发）**

在 nginx 服务器（公网服务器）上执行：

```bash
# 建立 SSH 隧道（公网服务器监听 8080，转发到内网服务器的 8080）
ssh -N -L 8080:localhost:8080 aiweb

# 或使用 autossh 保持连接
autossh -M 20000 -N -L 8080:localhost:8080 aiweb
```

**方式2：在内网服务器上建立反向隧道（-R，远程端口转发，推荐）**

在内网服务器上执行（主动连接到公网服务器）：

```bash
# 反向隧道：内网服务器主动连接，在公网服务器上监听 8080
ssh -N -R 8080:localhost:8080 user@ng.smartxy.com.cn

# 或使用 autossh 保持连接
autossh -M 20000 -N -R 8080:localhost:8080 user@ng.smartxy.com.cn
```

**两种方式的区别：**

- **方式1（-L）**：公网服务器主动连接内网服务器，需要公网服务器能访问内网服务器
- **方式2（-R）**：内网服务器主动连接公网服务器，适合内网服务器无法被公网直接访问的情况（推荐）

**nginx 配置（两种方式都相同）：**

```nginx
proxy_pass http://localhost:8080;
```

#### 方案B：VPN

配置 VPN 连接，使 nginx 服务器可以访问内网服务器。

#### 方案C：frp 内网穿透

在内网服务器运行 frpc，在公网服务器运行 frps，然后 nginx 代理到 frps。

## 测试

### 1. 测试 nginx 配置

```bash
# 测试配置语法
sudo nginx -t

# 检查 nginx 状态
sudo systemctl status nginx
```

### 2. 测试反向代理

```bash
# 测试后端 API
curl <https://你的域名>/api/health

# 测试前端
curl <https://你的域名>/

# 测试微信验证文件（如果已上传）
curl <https://你的域名>/MP_verify_xxxxx.txt
```

### 3. 测试微信回调

1. 访问系统初始化页面：`<https://你的域名>/init`
2. 配置微信 AppID 和 AppSecret
3. 获取二维码
4. 用手机微信扫码
5. 确认能正常回调

## 常见问题

### 问题1：502 Bad Gateway

**原因**：nginx 无法连接到内网服务器

**解决方案**：
1. 检查内网服务器是否运行
2. 检查防火墙是否开放端口
3. 检查 nginx 配置中的 IP 和端口是否正确
4. 如果使用 SSH 隧道，检查隧道是否建立

### 问题2：WebSocket 连接失败

**原因**：nginx 未正确配置 WebSocket 代理

**解决方案**：
确保 nginx 配置中包含 WebSocket 相关配置：
```nginx
proxy_http_version 1.1;
proxy_set_header Upgrade $http_upgrade;
proxy_set_header Connection "upgrade";
```

### 问题3：SSL 证书问题

**原因**：证书未正确配置或已过期

**解决方案**：
1. 检查证书路径是否正确
2. 使用 `certbot renew` 更新证书
3. 检查证书权限

## 安全建议

1. **使用 HTTPS**：生产环境必须使用 HTTPS
2. **防火墙配置**：只开放必要的端口（80, 443）
3. **限制访问**：可以配置 IP 白名单（如果需要）
4. **日志监控**：定期检查 nginx 日志

## 相关文档

- [微信登录配置指南](./微信登录配置指南.md)
- [微信域名验证文件配置](./微信域名验证文件配置.md)

