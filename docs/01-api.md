# Cloud API

生产 Base：`https://www.niuma007.com/niuma/cloud`  
本地 Base：`http://127.0.0.1:8090/niuma/cloud`

完整前缀：`{base}/api/v1`

## 用户 Auth

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/auth/register/start` | `{ email }` → 发注册 OTP |
| POST | `/api/v1/auth/register/complete` | `{ email, code, password, displayName? }` |
| POST | `/api/v1/auth/login` | `{ email, password }` |
| POST | `/api/v1/auth/login/otp/start` | `{ email }` |
| POST | `/api/v1/auth/login/otp/complete` | `{ email, code }` |
| POST | `/api/v1/auth/password/forgot` | `{ email }` |
| POST | `/api/v1/auth/password/reset` | `{ email, code, password }` |
| POST | `/api/v1/auth/password/change` | `{ currentPassword, newPassword }`（Bearer） |
| POST | `/api/v1/auth/refresh` | `{ refreshToken }` |
| POST | `/api/v1/auth/logout` | `{ refreshToken }` |
| GET | `/api/v1/me` | Bearer access |
| PATCH | `/api/v1/me` | `{ displayName }` |
| POST | `/api/v1/feedback` | 登录或匿名（匿名须 contact）；可带 `images[]` |
| GET | `/api/v1/feedback/mine` | 登录用户：我的反馈（含 `staffReply`） |
| GET | `/api/v1/feedback/lookup` | 官网匿名查询：`?id=&contact=` |
| GET | `/api/v1/feedback/{id}/attachments/{attId}` | 反馈截图 |

## 系统 AI（用户 Bearer）

桌面端登录后写入本机 `niuma-system` Provider。接入点 / 模型目录在库表管理（Ops「AI」），上游密钥密文入库，客户端只用账号 token。

未开放接入点或缺少密钥时：`catalog.enabled=false`，对话返回 `503 ai_disabled`。

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/ai/catalog` | `{ enabled, providerName, defaultModel, models[], dailyRequestLimit, requestsUsed, … }` |
| POST | `/api/v1/ai/v1/chat/completions` | OpenAI 兼容；`Authorization: Bearer <access>`；超配额 `429 ai_quota_exceeded`；并发超限 `429 ai_busy` |

## 公开 Updates

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/updates/latest` | `?product=&channel=&platform=&arch=` 最新 published 元数据（展示用，勿拿来下包） |
| GET | `/api/v1/updates/check` | 另需 `current`（semver） |
| GET | `/api/v1/updates/releases` | published 列表 |
| GET | `/api/v1/updates/download` | 同上查询参数；点击当下解析最新 published，记 `update_hits`，**302** 到版本化安装包。响应 `Cache-Control: no-store` |
| GET | `/api/v1/updates/files/{product}/{channel}/{platform}/{arch}/{version}/{name}` | 本机安装包（仅 published）；桌面更新 / 302 落地地址，可缓存 |
| POST | `/api/v1/updates/hit` | `{ product?, channel?, platform, arch, version? }` 仅计次不代理文件；`version` 空则记最新 published。桌面端在拉版本化文件前调用；`/updates/download` 已计次，官网不必再打本接口 |

## Pipeline（流水线 Token）

GitHub Actions 在打完安装包后：**先把文件传到本机**，再登记 published。国内用户与桌面更新都从本站下载，不走 GitHub。鉴权（二选一）：

- `Authorization: Bearer <pipeline.token>`
- `X-Niuma-Pipeline-Token: <pipeline.token>`

`pipeline.token` 配在 `config/app.yaml`，或环境变量 `NIUMA_PIPELINE_TOKEN`（至少 16 字符）。未配置时接口返回 `503 pipeline_disabled`。

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/pipeline/releases/assets` | `multipart/form-data`：`file` + `product`/`channel`/`platform`/`arch`/`version`/`sha256`；落盘 `{data_dir}/releases/...` |
| POST | `/api/v1/pipeline/releases` | 按「产品+渠道+平台+架构+版本」upsert；本机已有文件时自动写官方 `downloadUrl` |

上传字段：`file`（必填）、`product`（默认 niuma）、`channel`（默认 stable）、`platform`、`arch`、`version`、`sha256`（可选，有则必须一致）、`name`（可选，覆盖文件名）。

登记请求体（`downloadUrl` 可省略：本机已有安装包时服务端生成）：

```json
{
  "product": "niuma",
  "version": "1.0.2",
  "channel": "stable",
  "title": "NiuMa 1.0.2",
  "notesMd": "## 新增\\n- ...",
  "publish": true,
  "items": [
    {
      "platform": "windows",
      "arch": "x64",
      "name": "NiuMa-1.0.2-windows-x64-Setup.exe",
      "sha256": "64 位 hex",
      "fileSize": 123456789
    }
  ]
}
```

同一维度重复推送会覆盖下载地址 / 校验和 / 说明（已 yank 的记录返回 `409 release_yanked`）。未上传安装包且未给 `downloadUrl` 时返回 `400 missing_installer`。

员工角色：`admin`（超管，可管员工/用户账号）与 `operator`（日常运营：统计、工单、发布、只读账户）。种子账号为 admin。

## Ops（Staff Bearer）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/ops/auth/login` | 员工登录（access 在 JSON；refresh 仅 HttpOnly Cookie `niuma_ops_rt`） |
| POST | `/api/v1/ops/auth/refresh` | 轮转 refresh（Cookie 或 body.refreshToken）并签发新 access |
| POST | `/api/v1/ops/auth/logout` | 吊销当前 refresh 并清 Cookie |
| POST | `/api/v1/ops/auth/password` | 修改自己的密码（吊销旧会话并重签） |
| GET | `/api/v1/ops/stats` | `users` / `feedback` / `feedbackLast7d` / `downloadHits` / `aiUsersToday` / `aiRequestsToday` |
| GET | `/api/v1/ops/audit-logs` | 审计；`action` / `actorKind` / `actorId` / `resourceKind` / `resourceId` 等值；`limit`（默认 20，上限 50）/ `offset` |
| GET / POST | `/api/v1/ops/staff` | 员工列表 / 创建（创建需 admin）；列表：`id` / `email` / `displayName` / `status` / `role` 等值；`limit` / `offset` |
| PATCH | `/api/v1/ops/staff/{id}` | 启停、显示名、角色（需 admin） |
| POST | `/api/v1/ops/staff/{id}/password` | 重置员工密码（需 admin） |
| GET / POST | `/api/v1/ops/users` | 终端用户列表 / 创建（创建需 admin）；列表：`id` / `email` / `displayName` / `status` 等值；`limit` / `offset` |
| GET / PATCH | `/api/v1/ops/users/{id}` | 详情 / 启停（启停需 admin） |
| POST | `/api/v1/ops/users/{id}/password` | 重置用户密码（需 admin） |
| GET | `/api/v1/ops/feedback` | 工单列表；`id` / `title` / `contact` / `category` / `status` / `product` / `userId` / `clientVersion` 等值；`limit` / `offset` |
| PATCH | `/api/v1/ops/feedback/{id}` | `{ status?, adminNote?, staffReply? }` |
| GET / POST | `/api/v1/ops/releases` | 发布列表 / 新建草稿；列表：`product` / `channel` / `platform` / `arch` / `status` / `version` 等值；`limit` / `offset` |
| GET / PATCH | `/api/v1/ops/releases/{id}` | 详情 / 编辑草稿 |
| POST | `/api/v1/ops/releases/{id}/publish` | 发布 |
| POST | `/api/v1/ops/releases/{id}/yank` | 撤回 |
| GET | `/api/v1/ops/ai/usage` | 按 UTC 日用量：`?day=YYYY-MM-DD`（默认今天）`email`（精确）`userId` `limit` `offset`；返回 `summary` / `trend`（近 7 日） / `top`（排行前 8） / `items` / `dailyRequestLimit` |
| GET / POST | `/api/v1/ops/ai/endpoints` | 系统模型接入点列表 / 新建（密钥密文入库，列表仅 `hasApiSecret`） |
| POST | `/api/v1/ops/ai/endpoints/test` | 探测上游：`{ baseUrl, apiSecret, model, endpointId? }`；仅公网 HTTPS；`ok/latencyMs/message`（不记用户配额，员工 3s 冷却） |
| POST | `/api/v1/ops/ai/endpoints/pull-models` | 拉取上游 `GET /models`：同上鉴权与冷却；`ok/items[{id,label}]`，最多 300 条，不写入目录 |
| GET / PATCH / DELETE | `/api/v1/ops/ai/endpoints/{id}` | 接入点详情 / 编辑 / 删除（级联目录） |
| GET | `/api/v1/ops/ai/endpoints/{id}/secret` | 是否已配置密钥（`found`，不回显明文） |
| GET / POST | `/api/v1/ops/ai/endpoints/{id}/models` | 模型目录列表 / 新增 |
| PATCH / DELETE | `/api/v1/ops/ai/models/{modelId}` | 模型目录编辑 / 删除 |

## 健康检查

`GET /niuma/cloud/healthz`
