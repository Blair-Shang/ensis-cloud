# Ensis Cloud 概览

## 仓库边界

```text
ensis-ui       # 设计系统
ensis-site     # 品牌站 + 下载计数
ensis-cloud    # 本仓：Identity + Feedback + Updates + 系统 AI 代理 + Admin
NiuMa          # 桌面客户端
```

## 目录划分（合理，保持）

```text
ensis-cloud/
  admin/                 # Ops 前端（Vue，部署路径 /niuma/ops/）
  server/                # Go API + 嵌入 Admin 静态资源
    cmd/server/          # 进程入口
    internal/
      auth/              # 密码 / JWT / OTP
      llm/               # 系统模型 OpenAI 兼容代理
      cryptutil/         # 上游密钥 AES-GCM 封装
      httpapi/           # HTTP 路由与 handler
      store/             # 数据访问
      db/                # 连接；migrations/{mysql,sqlite}/*.sql
      mail/              # SMTP
      config/ logging/   # 基础设施
      web/               # embed Admin dist
  config/                # app.yaml（gitignore 真实密钥）
  docs/                  # 契约与部署
  script/                # build / start / stop
  run/                   # 二进制产物
```

原则：

| 目录 | 职责 | 评价 |
|------|------|------|
| `admin/` | 仅员工运营 UI | 与品牌站分离，正确 |
| `server/` | 唯一后端进程 | 与 ensis-site 同构，易维护 |
| `config/` | 运行配置 | 密钥不进仓库 |
| 不设 `data/*.db` 为主存储 | 持久化走 MySQL | 符合生产 |

后续若模块膨胀，可在 `server/internal/` 下按域再拆（如 `identity/`、`feedback/`），首期无需。

## 生产域名与路径

统一挂在 **https://www.niuma007.com**，cloud 工程根为 **`/niuma`**：

| 路径 | 职责 |
|------|------|
| `/` | 官网（ensis-site） |
| `/api/*` | 官网下载计数后 302 到 cloud `/updates/download` |
| `/niuma/cloud/api/v1/*` | Cloud API（含最新包入口 `/updates/download`） |
| `/niuma/cloud/healthz` | Cloud 健康检查 |
| `/niuma/ops/` | 运营后台 |

## 技术栈

- Go API（`server/`）
- Vue Admin（`admin/`，`base=/niuma/ops/`）
- **MySQL**（持久化，默认）
- SMTP 发 OTP（dev 可打日志）
