# Phase 1 Week 1 实施进度

## 已完成

### Day 1-2: 认证架构设计与数据模型 ✅

**完成的文件**:
1. `ent/schema/api_token.go` - API Token 数据模型
2. `ent/schema/user.go` - User 数据模型
3. `ent/schema/role.go` - Role 数据模型 (更新)

**数据模型关系**:
```
User 1--* APIToken
User *--* Role
```

**Token 字段**:
- id, token (hashed), name, description
- user_id, scopes
- created_at, expires_at, last_used_at
- revoked, revoked_at, revoked_reason

### Day 3-4: Token 服务实现 ✅

**完成的文件**:
1. `internal/auth/token_service.go` - Token 业务逻辑
2. `internal/auth/token_repository.go` - Token 数据访问层
3. `internal/auth/middleware.go` - 认证中间件
4. `internal/server/token_handler.go` - Token HTTP API

**核心功能**:
- ✅ Token 生成 (32字节随机 + SHA256哈希)
- ✅ Token 验证 (哈希比对 + 过期检查 + 撤销检查)
- ✅ Token 撤销
- ✅ Token 列表查询
- ✅ Token 删除

**中间件功能**:
- ✅ Bearer Token 认证
- ✅ 可选认证 (OptionalAuthenticate)
- ✅ Scope 权限检查 (RequireScopes)
- ✅ Context 注入 (token, user_id)

**API 端点**:
- POST /api/v1/auth/tokens - 创建 Token
- GET /api/v1/auth/tokens - 列表查询
- POST /api/v1/auth/tokens/{id}/revoke - 撤销 Token
- DELETE /api/v1/auth/tokens/{id} - 删除 Token

---

## 待完成

### Day 5: 集成与测试

**任务清单**:
- [ ] 集成到主服务器
  - [ ] 更新 cmd/server/main.go
  - [ ] 初始化 TokenService
  - [ ] 注册 Token 路由
  - [ ] 应用认证中间件到现有 API

- [ ] 创建初始用户和 Token
  - [ ] 实现用户创建 API
  - [ ] 创建 bootstrap 脚本
  - [ ] 生成初始 admin token

- [ ] 单元测试
  - [ ] token_service_test.go
  - [ ] token_repository_test.go
  - [ ] middleware_test.go

- [ ] 集成测试
  - [ ] 测试 Token 创建流程
  - [ ] 测试认证中间件
  - [ ] 测试权限检查

- [ ] 文档更新
  - [ ] API 文档
  - [ ] 使用指南
  - [ ] 迁移指南

---

## 下一步 (Week 2)

### RBAC 权限系统

**数据模型**:
- Permission (id, name, resource, action)
- Role-Permission 关联
- User-Role 关联

**核心功能**:
- 权限定义和管理
- 角色权限分配
- 用户角色分配
- 权限检查中间件

**预定义角色**:
- admin: 所有权限
- developer: 任务 CRUD + 项目读取
- viewer: 只读权限

---

## 技术债务

1. **密码哈希**: User 模型中的 password_hash 需要使用 bcrypt
2. **Token 刷新**: 考虑实现 refresh token 机制
3. **审计日志**: 记录所有认证和授权操作
4. **速率限制**: 防止暴力破解
5. **Token 轮转**: 定期强制更新 Token

---

## 风险与问题

### 已识别风险:
1. **向后兼容**: 现有 API 需要逐步迁移到认证模式
   - 缓解: 先实现 OptionalAuthenticate，逐步强制认证

2. **初始化复杂度**: 首次启动需要创建用户和 Token
   - 缓解: 提供 bootstrap 命令和环境变量配置

3. **Token 泄露**: Token 一旦泄露无法撤回已发出的请求
   - 缓解: 实现短期 Token + 撤销机制

### 待解决问题:
1. 如何处理现有的无认证 API 调用?
2. 是否需要实现 OAuth2/OIDC?
3. 多租户隔离如何实现?

---

## 性能考虑

### Token 验证性能:
- 每个请求都需要查询数据库验证 Token
- 优化方案:
  1. 添加内存缓存 (5分钟 TTL)
  2. 使用 Redis 缓存
  3. 实现 JWT (无状态验证)

### 数据库索引:
- ✅ token 字段已添加索引
- ✅ user_id 字段已添加索引
- ✅ revoked 字段已添加索引

---

## 测试策略

### 单元测试覆盖:
- Token 生成和哈希
- Token 验证逻辑
- 过期检查
- 撤销检查
- Scope 权限检查

### 集成测试场景:
1. 创建 Token 并使用
2. Token 过期后拒绝访问
3. 撤销 Token 后拒绝访问
4. 无 Token 访问受保护端点
5. 无效 Token 访问受保护端点
6. Scope 不足访问受限端点

### 性能测试:
- Token 验证吞吐量 (目标: >1000 req/s)
- 并发 Token 创建
- 大量 Token 查询性能

---

## 文档需求

### API 文档:
- Token 管理 API 规范
- 认证流程说明
- 错误码定义
- 示例代码

### 用户指南:
- 如何创建 Token
- 如何使用 Token 调用 API
- 如何管理 Token (撤销、删除)
- 安全最佳实践

### 运维文档:
- 如何初始化系统
- 如何创建初始用户
- 如何备份 Token 数据
- 如何处理 Token 泄露

---

**更新时间**: 2026-05-03 02:20:00  
**负责人**: 开发团队  
**状态**: Week 1 Day 4 完成，Day 5 进行中
