# Git 推送指南

## 问题诊断

当前 Git 推送失败，错误信息：
```
remote: Permission to leoh081910-ship-it/Multi-Terminal-AI-Orchestration-Platform.git denied to vasm-696.
fatal: unable to access 'https://github.com/leoh081910-ship-it/Multi-Terminal-AI-Orchestration-Platform.git/': The requested URL returned error: 403
```

**原因**: GitHub 认证凭据不匹配或已过期

---

## 解决方案

### 方案 1: 使用 GitHub CLI (推荐)

```bash
# 1. 检查 gh 是否已登录
gh auth status

# 2. 如果未登录，执行登录
gh auth login

# 3. 推送代码
cd "E:\04-Claude\Projects\多终端 AI 编排平台"
git push origin main
```

### 方案 2: 使用 SSH

```bash
# 1. 检查 SSH 密钥
ls ~/.ssh/id_*.pub

# 2. 如果没有密钥，生成新密钥
ssh-keygen -t ed25519 -C "your_email@example.com"

# 3. 添加 SSH 密钥到 GitHub
# 复制公钥内容
cat ~/.ssh/id_ed25519.pub
# 访问 https://github.com/settings/keys 添加

# 4. 修改远程仓库 URL
cd "E:\04-Claude\Projects\多终端 AI 编排平台"
git remote set-url origin git@github.com:leoh081910-ship-it/Multi-Terminal-AI-Orchestration-Platform.git

# 5. 推送代码
git push origin main
```

### 方案 3: 使用 Personal Access Token (PAT)

```bash
# 1. 生成 PAT
# 访问 https://github.com/settings/tokens
# 创建新 token，勾选 repo 权限

# 2. 使用 PAT 推送
cd "E:\04-Claude\Projects\多终端 AI 编排平台"
git push https://YOUR_TOKEN@github.com/leoh081910-ship-it/Multi-Terminal-AI-Orchestration-Platform.git main

# 3. 或者更新远程 URL（不推荐，token 会暴露在配置中）
git remote set-url origin https://YOUR_TOKEN@github.com/leoh081910-ship-it/Multi-Terminal-AI-Orchestration-Platform.git
git push origin main
```

### 方案 4: 使用 Git Credential Manager

```bash
# 1. 检查是否已安装
git credential-manager --version

# 2. 如果未安装，通过 Scoop 安装
scoop install git-credential-manager

# 3. 配置 Git 使用 credential manager
git config --global credential.helper manager

# 4. 推送时会弹出认证窗口
cd "E:\04-Claude\Projects\多终端 AI 编排平台"
git push origin main
```

---

## 待推送的提交

共 9 个提交待推送：

```
b0557c0 docs: add automated remediation summary
45f85d9 docs: add comprehensive test report and update project state
5f5d010 fix: use relative paths in config.yaml for runtime scripts
42d8390 feat: add organization routing workspace features
b9badf8 fix(quick-260502-olx): stabilize local Vite build
cb20964 docs(quick-260502-olx-01): record final summary commit hash
24cc096 docs(quick-260502-olx-01): complete validation quick task
1c77c12 fix(quick-260502-olx-01): update package-relative runtime paths
7ce4970 fix(quick-260502-olx-01): resolve frontend hook lint errors
```

---

## 推送后验证

推送成功后，访问以下链接验证：

1. **仓库主页**:  
   https://github.com/leoh081910-ship-it/Multi-Terminal-AI-Orchestration-Platform

2. **提交历史**:  
   https://github.com/leoh081910-ship-it/Multi-Terminal-AI-Orchestration-Platform/commits/main

3. **测试报告**:  
   https://github.com/leoh081910-ship-it/Multi-Terminal-AI-Orchestration-Platform/blob/main/TEST_REPORT.md

---

## 手动推送命令

如果你已经配置好认证，直接执行：

```bash
cd "E:\04-Claude\Projects\多终端 AI 编排平台"
git push origin main
```

---

**生成时间**: 2026-05-03 01:48:00 +08:00
