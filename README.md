# 🛡️ ExpiryGuard

> **Your Ultimate Asset Expiration Manager.**
>
> 一个轻量、安全、现代化的资产过期管理系统。专为个人和中小团队设计，帮助你追踪域名、服务器、SSL 证书等资产的有效期，拒绝意外过期。

[![Build Status](https://img.shields.io/github/actions/workflow/status/TerrySiu98/expiry-guard/docker.yml?style=flat-square)](https://github.com/TerrySiu98/expiry-guard/actions)
[![Docker Pulls](https://img.shields.io/docker/pulls/terrysiu/expiry-guard?style=flat-square)](https://hub.docker.com/r/terrysiu/expiry-guard)
[![License](https://img.shields.io/badge/license-MIT-blue?style=flat-square)](LICENSE)

## ✨ Features (核心功能)

* **🔐 金融级安全**：强制密码重置策略、2FA 双重验证 (Telegram/Email)、登录 IP 审计。
* **🌍 完整国际化**：原生支持 🇨🇳 中文、🇺🇸 English、🇯🇵 日本語、🇰🇷 한국어，自动时区转换。
* **📱 移动端优先**：精心打磨的响应式 UI，支持手势滑动、汉堡菜单，手机管理极其流畅。
* **🔔 多渠道通知**：资产到期前 7/3/1/0 天自动发送 Telegram 和 邮件提醒。
* **⚡ 高效管理**：
    * **智能联想**：输入分类时自动推荐历史记录。
    * **批量导入**：支持 CSV 文件一键导入海量资产。
    * **一键备份**：管理员可随时下载数据库备份。
* **🐳 Docker 部署**：支持 AMD64 和 ARM64 (树莓派/Oracle Cloud) 架构，开箱即用。

## 🚀 Quick Start (快速开始)

### 使用 Docker (推荐)

只需一行命令即可启动：

```sh
docker run -d \
  --name expiry-guard \
  -p 8080:8080 \
  -v $(pwd)/data:/app/data \
  --restart always \
  terrysiu/expiry-guard:latest
```

然后在浏览器中访问：
```sh
http://localhost:8080
```

* **默认管理员**：注册的第一个用户自动成为管理员。
* **初始设置**：登录后请立即进入“个人设置”绑定 Telegram 或 邮箱以开启 2FA。

## 🛠️ Configuration (配置)

进入 **系统管理 (Admin)** -> **全局配置**，填入你的通知参数：

* **Telegram Bot**: \`@BotFather\` 申请的 Token 和 Bot Username。
* **SMTP Email**: 用于发送验证码和提醒邮件的 SMTP 服务配置。

## 📜 License

This project is licensed under the [MIT License](LICENSE).
