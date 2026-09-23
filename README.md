# vps-billing-Gemini

**vps-billing-Gemini** 是一个高性能、高可靠的开源 VPS / 云主机商业销售、财务计费、复式记账与自动化资源编排交付平台（V1 Release Candidate）。

采用清晰的**模块化单体（Modular Monolith）**架构，前后端分离，内置多底层虚拟化适配器、分布式任务状态机与异步工作流引擎。

---

## 核心设计与架构原则

1. **商业真相源唯一性**：业务 PostgreSQL 是全系统唯一真相源，底层虚拟化与 Agent 仅作为执行与观测端，严禁倒置。
2. **领域职责正交隔离**：`Order` $\neq$ `Payment` $\neq$ `Invoice` $\neq$ `Subscription` $\neq$ `Instance`，全独立实体与状态机。
3. **不可篡改复式记账**：所有资金流水均记录借贷相等的 `ledger_transaction`，采用 `BIGINT` 最小货币单位（分/美分），历史记录不可 UPDATE/DELETE。钱包余额仅为账本投影。
4. **长任务全异步解耦**：HTTP Handler 绝不阻塞调用底层宿主机。创建、重装、启停、密码重置均派发 `Operation`，返回 `HTTP 202 Accepted`，通过 SSE 实时流式推送进度。
5. **底层 Provider 契约化**：支持 Mock、CLICD Direct API 及 Runman 分布式 gRPC Agent 网关，替换或扩展底层无须修改任何商业核心代码。
6. **企业级安全与韧性**：管理员 RFC 6238 TOTP 2FA、敏感端点滑动窗口限流、并发排队 409 互斥锁、自动故障自愈协调器（Reconciler）。
7. **界面多语言 100% 对齐**：`zh-CN` 与 `en-US` 双语对齐，无任何硬编码文案。

---

## 核心功能矩阵

### 1. 用户前台自服务系统 (User Web)
- **免登录公开产品大厅**：访客无需注册即可浏览 VPS 规格配置、CPU/内存/磁盘/带宽参数及价格，点击购买直接拉起结账向导并引导注册。
- **5 秒全局业务看板**：实时聚合显示活跃实例、可用钱包余额、待支付订单、进行中工单，关键信息一览无余。
- **全功能实例管理中心**：
  - 一键复制标准 SSH 登录命令（`ssh root@<ip>`）。
  - 安全查看/隐藏 Root 初始密码，支持一键剪贴板复制。
  - 当月网络流量消耗动态进度条，直观掌握流量用量与额度告警。
  - 异步全生命周期电源控制：开机、关机、强制重启，状态实时刷新。
  - 在线重装操作系统向导：支持选择目标发行版镜像，自动派发异步 Operation。
  - 实例快捷续费向导：支持按月自动展期。
- **多币种钱包与复式记账**：支持预设面额与自定义充值，底层完全基于复式借贷凭证（Double-entry Ledger），出入金流水不可篡改。
- **双向交互工单支持系统**：完整的客户支持工单提交、工单消息历史列表与多轮追加沟通交流。

### 2. 运营管理后台控制台 (Admin Web)
- **底层资源动态接入**：提供图形化弹窗快速录入与配置 Node 节点服务器及 Provider 底层虚拟化驱动。
- **商业产品与套餐管理**：可视化表单创建硬件产品规格与周期计费套餐（Plan）。
- **异步运维诊断抽屉 (Operation Inspector)**：实时下钻查看任意异步长任务的执行阶段、Trace ID、进度百分比及底层原始诊断 JSON 数据。
- **客服工单调度台**：集中处理全平台客户工单，支持管理员多轮回复、关闭或重新开启工单。
- **企业级安全 2FA**：内置标准 RFC 6238 TOTP 双因素认证，支持扫码配置与登录验证。

---

## 技术选型

- **后端**：Go 1.23 + Chi + pgx/v5 + sqlc + golang-migrate
- **数据库与缓存**：PostgreSQL 17 + Redis 8 Alpine
- **前端**：React 19 + TypeScript 5.7 + Vite + Tailwind CSS + TanStack Query + i18next
- **容器与部署**：Docker Multi-Stage Build + Docker Compose + 多架构镜像（原生支持 `linux/amd64` 与 `linux/arm64`，适配 x86 服务器、Apple Silicon 及 ARM 云主机）
- **监控与审计**：Prometheus 指标端点 (`/metrics`) + 不可篡改 Audit Log

---

## 目录结构

```text
.
├── .github/workflows/      # GitHub Actions CI/CD (Go Lint, Test, Frontend Build, Docker Build)
├── backend/                # Go 后端工程
│   ├── cmd/
│   │   ├── server/         # REST API、SSE 流与业务事务协调主服务
│   │   └── worker/         # 异步任务消费、工作流执行与 Reconciler 进程
│   └── internal/           # 领域层、业务服务层、Provider 驱动层、数据仓储与中间件
├── frontend/               # 前端 Monorepo (npm workspaces)
│   ├── shared/             # 共享 UI 组件库、类型定义与 i18n 语言包
│   ├── user-web/           # 用户前台自服务应用 (端口 3000)
│   └── admin-web/          # 管理后台控制台应用 (端口 3001)
├── db/
│   ├── migrations/         # 增量数据库迁移脚本 (.up.sql / .down.sql)
│   └── schema.sql          # 数据库基础模式
├── deploy/                 # 部署参考配置
├── docs/
│   ├── openapi/            # OpenAPI 3.1 接口契约定义 (openapi.yaml)
│   └── adr/                # 架构决策记录 (ADR)
├── scripts/                # 自动化生产数据库备份与单事务原子恢复脚本
└── docker-compose.yml      # 全量微服务与基础中间件编排文件
```

---

## 快速部署与启动

### 方式一：预构建镜像一键拉取运行（线上生产推荐，无需本地编译）

GitHub Actions 已自动将最新容器镜像发布至 GitHub Container Registry (GHCR)。
在云服务器上只需 `docker-compose.yml` 与 `.env` 两份文件即可一键拉取镜像启动：

```bash
# 1. 新建部署目录并下载预设配置
mkdir -p vps-billing && cd vps-billing
curl -fsSL https://raw.githubusercontent.com/snail46/vps-billing-Gemini/main/docker-compose.prod.yml -o docker-compose.yml
curl -fsSL https://raw.githubusercontent.com/snail46/vps-billing-Gemini/main/deploy/.env.example -o .env

# 2. 修改 .env 密码与秘钥配置（生产环境务必替换默认密钥）
# nano .env

# 3. 一键拉取镜像并启动全量容器集群
docker compose up -d
```

若已克隆本仓库代码，也可直接在项目根目录下执行：
```bash
docker compose -f docker-compose.prod.yml up -d
```

---

### 方式二：本地源码构建启动（适合二次开发与定制）

确保本地已安装 Docker 和 Docker Compose，在项目根目录下执行：

```bash
docker compose up -d --build
```

系统将按依赖关系与健康检查自动拉起以下 6 个容器服务：

| 服务名称 | 监听端口 | 镜像来源 | 说明 |
|---|---|---|---|
| **postgres** | `5432` | `postgres:17` | 数据库（启动自动执行增量迁移） |
| **redis** | `6379` | `redis:8-alpine` | 缓存与异步任务消息队列 |
| **server** | `8080` | `ghcr.io/snail46/vps-billing-gemini/server` | 后端 REST API、SSE 实时事件与 Prometheus 指标 |
| **worker** | - | `ghcr.io/snail46/vps-billing-gemini/worker` | 异步任务处理器与定期对账协调器 |
| **user-web** | `3000` | `ghcr.io/snail46/vps-billing-gemini/user-web` | 用户自服务前台 Web SPA |
| **admin-web** | `3001` | `ghcr.io/snail46/vps-billing-gemini/admin-web` | 管理员运营控制台 Web SPA |

---

### 方式三：本地裸机研发模式运行

#### 1. 启动数据库与 Redis
```bash
docker compose up -d postgres redis
```

#### 2. 运行后端服务
```bash
# 启动 API 服务
cd backend
go run ./cmd/server

# 另起终端启动后台 Worker
go run ./cmd/worker
```

#### 3. 运行前端应用
```bash
cd frontend
npm install

# 启动用户前台 (http://localhost:3000)
cd user-web
npm run dev

# 启动管理后台 (http://localhost:3001)
cd ../admin-web
npm run dev
```

---

## 初始账号与管理员 2FA 配置

服务启动时会通过安全机制自动初始化超级管理员与演示产品规格：

- **管理后台访问**：`http://localhost:3001`
- **默认管理员邮箱**：`admin@vps-billing.local`
- **默认管理员密码**：`Admin123456!`（系统同时双向兼容 `AdminPassword123!`）
- **自定义初始凭证**：支持在 `.env` 中设置 `INITIAL_ADMIN_EMAIL` 与 `INITIAL_ADMIN_PASSWORD` 自定义账号密码。
- **启用双因素认证 (2FA)**：
  1. 登录管理后台，进入右上角 **Settings $\rightarrow$ Security (2FA)**。
  2. 点击启用，使用标准 TOTP 认证器（Google Authenticator、1Password 等）扫描二维码。
  3. 输入 6 位动态验证码确认，后续登录均强制校验。

---

## 生产容灾与备份恢复

项目自带高标准单事务完整性恢复脚本，内置 SHA-256 校验和审计：

- **创建完整快照备份**：
  ```bash
  ./scripts/backup_db.sh
  ```
  在 `backups/` 目录生成带有时间戳的 `.sql.gz` 压缩文件与 `.sha256` 校验和。

- **单事务灾难恢复**：
  ```bash
  ./scripts/restore_db.sh backups/backup_vps_billing_YYYYMMDD_HHMMSS.sql.gz
  ```
  自动验证哈希防篡改，并在单次事务中执行还原与行数统计审计。

---

## 自动化测试与质量保障

全仓库严格满足 Definition of Done，测试 100% 自动化覆盖：

```bash
# 后端单元测试与验收套件（覆盖 100 次并发 Webhook 幂等、并发排队互斥等高风险用例）
cd backend
go test -v -count=1 ./...

# 前端全量 TypeScript 静态类型检查与生产打包
cd ../frontend
npm run typecheck
npm run build
```

---

## 开源协议

本项目采用 [MIT License](LICENSE) 授权。
