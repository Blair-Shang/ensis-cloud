# 变更日志

本文件记录 niuma-cloud 的重要变更。

格式参考 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)，版本遵循 [Semantic Versioning](https://semver.org/lang/zh-CN/)。

发版时把 `[Unreleased]` 下的条目移入新的 `## [x.y.z] - YYYY-MM-DD` 段；GitHub Release 正文由该段生成。`package.json` 的 `version`、git tag（`v*`）与本文件版本号必须一致。公开基线为 **1.1.0**。

## [Unreleased]

## [1.1.9] - 2026-08-31

### 新增

- Ops 概览「快捷入口」增加官方网站跳转（默认 `https://www.niuma007.com`，可用 `VITE_SITE_URL` 覆盖）。

### 变更

- Ops 编辑发布的标题 / 更新说明时，同一产品+版本的全部未撤回内核（平台/架构/渠道）一并改掉，不再只改当前这一条。下载地址与 SHA 仍按单条内核维护。
- 公开 updates（`latest` / `check` / `releases`）增加 `notesExcerpt` 与 `notesTruncated`。桌面帮助只展示最近三个版本全文；官网左侧列出版本与时间，右侧展示选中版本说明。Ops 继续保存完整说明（上限 2 万字）。
- 公开 `GET /updates/releases` 列表不再返回全文 / 下载地址；`limit` 走服务端分页上限（默认 20、最大 50）。带 `version` 时才返回单条完整说明。
- Ops「AI / 接入点」可手动刷新；从「用量」切回时重新拉列表。编辑中会一并刷新模型目录。

## [1.1.8] - 2026-08-31

### 新增
- Ops 接入点可「拉取模型」：`POST /api/v1/ops/ai/endpoints/pull-models` 读上游 OpenAI 兼容 `/models`，在表单里选默认模型或写入目录。
- 系统模型：登录用户 `GET /api/v1/ai/catalog`、`POST /api/v1/ai/v1/chat/completions`（OpenAI 兼容代理）。上游密钥只存云端密文，客户端只用账号 token。
- Ops「AI」：接入点 / 模型目录入库；厂商预设填充 Base URL 与推荐模型；对话框编辑；页脚「测试连接」（`POST /api/v1/ops/ai/endpoints/test`）。
- Ops 用量监控：`GET /api/v1/ops/ai/usage` 按 UTC 日列出用户次数，附 `summary` / 近 7 日 `trend` / 排行 `top`。Admin「用量」用与对话围栏同一套内联 SVG（折线 + 柱状），概览增加当日调用卡片。
- 按用户按日计量表 `ai_usage_daily`；日请求限额内原子累加，超限 `429 ai_quota_exceeded`。

### 安全

- 上游 Base URL 必须是云端能访问的公网 HTTPS，禁止本机 / 内网 / 元数据地址；出站不跟随重定向。
- 接入点密钥密文入库，列表与编辑不回显明文；`GET .../secret` 只返回 `{found}`。密封优先 `crypto.secret` / `NIUMA_CRYPTO_SECRET`。
- 同一用户最多 2 路并发对话（`429 ai_busy`）；Ops 探测上游员工 3 秒冷却。客户端请求体去掉 `api_key` / `authorization`，并限制 `n` 与 `max_tokens`。
- 用户 JWT 与员工 JWT 分 `kind`，终端用户不能调 `/ops/*`，员工 token 不能走用户态 `/ai/*`。

### 变更

- `GET /api/v1/ops/stats` 增加 `aiUsersToday` / `aiRequestsToday`。
- 未开放接入点或缺少密钥时：`catalog.enabled=false`，对话 `503 ai_disabled`。

## [1.1.7] - 2026-08-30

### 变更

- 邮箱验证码登录对未注册邮箱也会发码；校验通过后自动建号并登录。密码登录仍要求账号已存在。验证码建号没有可用登录密码，之后可用「忘记密码」设置。
- 公开 `GET /api/v1/updates/hits`：官网下载与桌面自动更新共用 `update_hits` 全局计数。

### 修复

- 流水线登记发布不再按 IP 冷却 2 秒。一次发版连登 beta 与 stable 时，第二渠道不再 429。
- 公开 updates 探针不再在进程内按 IP 冷却。清单 / 检查接口限流交给 Nginx 固定 10m zone，下载页并行查询不再被 200ms 冷却打成 429。

## [1.1.6] - 2026-08-30

### 修复

- Admin 对齐 niuma-ui **1.2.5**：打包走包主入口，不再依赖 `niumaUiHost` 改写导入。Ops 生产构建不再因 Vue SFC / 桶导出失败。

## [1.1.5] - 2026-08-30

### 变更

- Admin 对齐 niuma-ui **1.2.4**：设计系统 `styles.css` 自己收口 `box-sizing`，Ops 生产宽高不再只赌本仓 Tailwind Preflight。
- 随 niuma-ui **1.2.2**，`niumaUiHost` 能解析 npm `dist` 桶导出名。CI 装 `@latest` 后 `vite build` 不再把 Rs* 收成 type-only 而报 `MISSING_EXPORT`。

## [1.1.4] - 2026-08-30

### 修复

- Admin CSS 重新 `@import 'tailwindcss'`，让扫描根在 `admin/src`。依赖 `niuma-ui` 1.2.3 起把 `RsTable` 等脚本旁路 CSS 打进 npm，生产表格不再掉皮。

## [1.1.3] - 2026-08-30

### 变更

- Admin 业务 CSS 不再自己 `@import 'tailwindcss'`，改由 `niuma-ui/styles.css` 透传。
- Admin 启用 `niumaUiHost`，不再把 `@niuma/ui` 别名到 `src/index.ts`。

## [1.1.2] - 2026-08-30

### 变更

- Admin 从 `@niuma/ui` 包入口具名导入，不再走源码路径别名；与 niuma-ui ≥ 1.2.0 编译包对齐后不再需要 `shamefully-hoist`。
- CI / 打包从 npm 安装 `niuma-ui@latest`（可用 `NIUMA_UI_VERSION` 钉死）。本机仍用 `link:../niuma-ui` 联调。

## [1.1.1] - 2026-08-30

### 修复

- CI / 打包从 npm 安装 `niuma-ui` 后，`vue-tsc` 能解析其源码依赖（CodeMirror / Reka / marked 等）；不再因缺模块在 Admin 构建阶段失败、产物里没有 `SHA256SUMS.txt`。

## [1.1.0] - 2026-08-30

### 新增

- 公开 `GET /api/v1/updates/download`：按产品维度解析当前 latest published，记 `update_hits`，再 302 到版本化安装包。供官网与投放使用稳定「最新包」地址，避免发版后改配置。
- 该响应用 `Cache-Control: no-store`，防止浏览器把「最新」缓存成旧版本；落地仍走可缓存的 `/updates/files/...`。
- Ops 列表（用户 / 员工 / 反馈 / 发布 / 审计）：按字段等值筛选 + 分页（默认 20，上限 50），去掉模糊 `LIKE`，避免扫全表。
- 启动拒绝文档里的样例 JWT；`NIUMA_JWT_SECRET` / `NIUMA_DB_DSN` / `NIUMA_SMTP_PASSWORD` / `NIUMA_PIPELINE_TOKEN` 可覆盖 yaml。`log_otp` 默认 false，默认监听 `127.0.0.1:8090`。

### 修复

- SQLite 读取发布记录时兼容 `time.Time.String()` 格式的时间戳，避免本地库扫描 `published_at` 失败。

### 变更

- CI / 打包改为 npm 安装 `niuma-ui@latest`，不再 checkout 兄弟 Git 仓库。

### 说明

- 桌面端继续 `/updates/check` + 版本化 `files` + `POST /updates/hit`。`/updates/download` 已计次，官网跳转不必再打 hit。
- Linux / macOS 尚未稳定：官网与流水线使用 `channel=beta`；Windows 为 `stable`。

## [0.1.0] - 2026-08-29

### 新增

- Identity（邮箱密码 / 验证码）、Feedback、Updates、Admin Ops。
- 生产路径挂在 `https://www.niuma007.com/niuma`：`/niuma/cloud` API、`/niuma/ops` 运营后台。
- Windows / Linux 启停脚本：`build` / `start` / `stop` / `restart` / `status`；缺失时从 `app.yaml.example` 生成运行配置。
- GitHub Actions：CI（类型检查 + Admin 构建 + Go 测试）、推送 `v*` 标签打包 Linux/Windows amd64 并发布 Release。

### 安全

- `config/app.yaml` 不入库，仓库只提交 `config/app.yaml.example`。
