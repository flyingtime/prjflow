# 微信 scope 参数错误排查指南

## 错误信息

如果遇到以下错误：
- "scope参数错误或没有scope权限"
- "errcode: 40125"

## 问题原因

这个错误通常由以下原因导致：

1. **授权回调域名配置不正确**（最常见）
2. **使用了错误的 AppID 类型**
3. **接口权限未申请**
4. **配置文件中的 account_type 设置错误**

## 排查步骤

### 1. 检查 AppID 类型

**重要**：根据您的 AppID 来源，需要配置正确的 `account_type`：

#### 情况A：使用微信开放平台网站应用（推荐，支持扫码登录）

- **AppID 来源**：[微信开放平台](https://open.weixin.qq.com/) > 管理中心 > 网站应用
- **配置文件设置**：
  ```yaml
  wechat:
    account_type: "open_platform"  # 必须设置为 open_platform
    scope: "snsapi_userinfo"       # 此配置对开放平台无效（固定使用 snsapi_login）
  ```
- **授权URL**：`https://open.weixin.qq.com/connect/qrconnect`
- **scope 参数**：固定为 `snsapi_login`（代码自动设置）

#### 情况B：使用微信公众平台公众号

- **AppID 来源**：[微信公众平台](https://mp.weixin.qq.com/) > 开发 > 基本配置
- **配置文件设置**：
  ```yaml
  wechat:
    account_type: "official_account"  # 必须设置为 official_account
    scope: "snsapi_userinfo"         # 或 "snsapi_base"
  ```
- **授权URL**：`https://open.weixin.qq.com/connect/oauth2/authorize`
- **scope 参数**：使用配置的 `scope` 值（`snsapi_userinfo` 或 `snsapi_base`）
- **注意**：公众号网页授权主要用于在微信内打开的网页，不适合扫码登录

### 2. 检查授权回调域名配置

**这是最常见的错误原因！**

#### 对于微信开放平台网站应用：

1. 登录 [微信开放平台](https://open.weixin.qq.com/)
2. 进入"管理中心" > "网站应用" > 您的应用
3. 找到"授权回调域名"配置
4. **只填写域名**，不包含：
   - 协议（http:// 或 https://）
   - 端口号（:8080）
   - 路径（/api/init/callback）

**示例**：
- ✅ 正确：`project.smartxy.com.cn`
- ❌ 错误：`<https://你的域名>`
- ❌ 错误：`project.smartxy.com.cn:8080`
- ❌ 错误：`project.smartxy.com.cn/api/init/callback`

#### 对于微信公众平台公众号：

1. 登录 [微信公众平台](https://mp.weixin.qq.com/)
2. 进入"开发" > "接口权限" > "网页授权"
3. 配置"授权回调页面域名"
4. **只填写域名**，规则同上

### 3. 检查配置文件中的 callback_domain

确保 `backend/config.yaml` 中的 `callback_domain` 配置正确：

```yaml
wechat:
  callback_domain: "<https://你的域名>/"  # 注意：包含协议和末尾斜杠
```

**重要**：
- `callback_domain` 必须与微信后台配置的授权回调域名**匹配**
- 例如：如果微信后台配置的是 `project.smartxy.com.cn`，这里应该填写 `<https://你的域名>/`
- 系统会自动在域名后添加回调路径（如 `/api/init/callback`）

### 4. 检查接口权限

#### 对于微信开放平台网站应用：

1. 登录 [微信开放平台](https://open.weixin.qq.com/)
2. 进入"管理中心" > "网站应用" > 您的应用
3. 找到"接口权限"
4. 确保已申请并获得了"微信登录"接口权限
5. 如果未申请，点击"申请"按钮进行申请

#### 对于微信公众平台公众号：

1. 登录 [微信公众平台](https://mp.weixin.qq.com/)
2. 进入"开发" > "接口权限"
3. 确保"网页授权"接口已开通

### 5. 验证配置

#### 检查当前配置：

1. 查看 `backend/config.yaml` 文件：
   ```yaml
   wechat:
     app_id: "您的AppID"
     app_secret: "您的AppSecret"
     account_type: "open_platform"  # 或 "official_account"
     scope: "snsapi_userinfo"      # 仅公众号有效
     callback_domain: "<https://你的域名>/"
   ```

2. 验证授权回调域名：
   - 打开浏览器开发者工具（F12）
   - 访问初始化页面，获取二维码
   - 查看二维码对应的授权URL
   - 检查 URL 中的 `redirect_uri` 参数
   - 确认域名部分与微信后台配置的授权回调域名一致

#### 测试步骤：

1. **清除浏览器缓存**
2. **重启后端服务**（确保配置生效）
3. **重新访问初始化页面**
4. **获取二维码并扫码**
5. **查看错误信息**（现在会显示更详细的错误提示）

## 常见错误场景

### 场景1：配置了公众号 AppID，但想扫码登录

**问题**：公众号网页授权主要用于微信内打开，不适合扫码登录。

**解决方案**：
1. 使用微信开放平台的网站应用 AppID
2. 修改配置文件：
   ```yaml
   wechat:
     account_type: "open_platform"  # 改为 open_platform
   ```

### 场景2：授权回调域名配置错误

**问题**：在微信后台配置了 `<https://你的域名>`，但应该只填写域名。

**解决方案**：
1. 在微信后台修改为：`project.smartxy.com.cn`（不包含协议）
2. 确保配置文件中的 `callback_domain` 为：`<https://你的域名>/`

### 场景3：使用了错误的 AppID

**问题**：使用了公众号的 AppID，但配置了 `account_type: "open_platform"`。

**解决方案**：
1. 确认 AppID 来源：
   - 开放平台网站应用 → 使用 `account_type: "open_platform"`
   - 公众号 → 使用 `account_type: "official_account"`
2. 修改配置文件中的 `account_type`

### 场景4：本地开发环境

**问题**：本地开发时使用 `localhost`，但微信不支持 `localhost`。

**解决方案**：
1. 使用内网穿透工具（如 ngrok）：
   ```bash
   ngrok http 8080
   # 获取公网地址，如：https://abc123.ngrok.io
   ```
2. 在微信后台配置授权回调域名为：`abc123.ngrok.io`
3. 在配置文件中设置：
   ```yaml
   wechat:
     callback_domain: "https://abc123.ngrok.io/"
   ```

## 修复后的改进

系统已更新，现在会提供更详细的错误信息：

- **code已过期**：提示重新扫码
- **code已被使用**：提示重新扫码
- **scope参数错误**：提供详细的排查建议，包括：
  1. 检查授权回调域名配置
  2. 检查 AppID 类型
  3. 检查接口权限

## 联系支持

如果按照以上步骤仍无法解决问题，请提供以下信息：

1. 错误信息（完整的错误提示）
2. 配置文件内容（隐藏敏感信息）
3. 微信后台配置截图（授权回调域名、接口权限）
4. 授权URL（从浏览器开发者工具中获取）

