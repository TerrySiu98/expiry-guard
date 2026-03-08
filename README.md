# Expiry Guard - Cloudflare Workers Edition

资产到期提醒系统，部署在 Cloudflare Workers + R2 存储。

## 功能特性

- 📊 资产管理：添加、编辑、删除、搜索
- 🔔 到期提醒：Telegram Bot 通知
- 👥 多用户支持：用户注册、登录、权限管理
- 🌐 多语言：中文、英文
- 🌙 深色模式
- 📤 导入导出：JSON 格式
- 💾 系统备份：完整数据备份

## 部署步骤

### 1. 创建 R2 存储桶

在 Cloudflare 控制台创建 R2 bucket，例如 `expiry-guard-data`

### 2. 配置生命周期规则

为临时数据设置自动清理：
- 规则 1：前缀 `sessions/`，1 天后删除
- 规则 2：前缀 `2fa/`，1 天后删除

### 3. 部署 Worker

1. 复制 `worker.js` 的全部内容
2. 在 Cloudflare Workers 控制台创建新 Worker
3. 粘贴代码并保存
4. 在 Settings → Variables → R2 Bucket Bindings 添加：
   - Variable name: `BUCKET`
   - R2 bucket: 选择你创建的 bucket

### 4. 配置 Telegram Bot（可选）

1. 访问系统设置页面
2. 填写 TG Bot Username 和 TG Bot Token
3. 在个人设置填写 Telegram Chat ID

## 技术栈

- Cloudflare Workers（边缘计算）
- R2 Storage（对象存储）
- 原生 JavaScript（零依赖）

## License

MIT
