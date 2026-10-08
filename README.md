# niuma-cloud

NiuMa 云侧工程：**Identity（邮箱密码/验证码）+ Feedback + Updates + Admin Ops**。

与 [niuma-site](../niuma-site) 品牌站分离。生产挂在 **https://www.niuma007.com** 的工程根 **`/niuma`** 下：

| 路径 | 服务 |
|------|------|
| `/niuma/cloud/api/v1/*` | Identity / Feedback / Updates / Ops API |
| `/niuma/cloud/healthz` | 健康检查 |
| `/niuma/ops/` | 运营后台 |

## 快速开始

```bash
pnpm install
Copy-Item config\app.yaml.example config\app.yaml
# 编辑 config/app.yaml：填写 db.dsn（默认驱动 mysql）
# app.yaml 含密钥，已加入 .gitignore，只提交 app.yaml.example

pnpm dev          # 同时启动 :8090 后端 + :5174 Admin
pnpm dev:server   # 仅后端（/niuma/cloud、/niuma/ops）
pnpm dev:admin    # 仅 Admin（base=/niuma/ops/，代理 /niuma/cloud）
```

需要同级 [`../niuma-ui`](https://github.com/Blair-Shang/niuma-ui)。持久化使用 **MySQL**（见 `config/app.yaml`）；SQLite 仅作临时调试。

生产构建与启停：

Windows：

```powershell
powershell -ExecutionPolicy Bypass -File script/build.ps1
powershell -ExecutionPolicy Bypass -File script/start.ps1
# status / stop / restart 同理
```

Linux：

```bash
chmod +x script/*.sh
bash script/build.sh
bash script/start.sh
```

默认员工种子账号见 `config/app.yaml.example`（`staff.email` / `staff.password`）。

## CI

GitHub Actions 仓库：[Blair-Shang/niuma-cloud](https://github.com/Blair-Shang/niuma-cloud)

| 工作流 | 触发 | 作用 |
|--------|------|------|
| [CI](./.github/workflows/ci.yml) | `main` 推送 / PR | 类型检查、Admin 构建、Go 测试 |
| [Release](./.github/workflows/release.yml) | `main` 推送 | `package.json` 版本尚未发布且 changelog 有列表条目时，打包 Linux/Windows amd64 并发布 GitHub Release |
| [Pack and Release](./.github/workflows/pack.yml) | `v*` 标签 / 手动 | 手工打标签时同样打包发版；手动运行只上传 artifact |

自动发版看 `package.json` 的 `x.y.z`。把本次变更写在 `[Unreleased]` 的 `-` / `*` 列表里，再推 `main`。随后那次 changelog 提交的说明里带 `[release notes]`，用来标记版本段已经记下。

CI / 打包从 npm 安装 `niuma-ui@latest`（可用变量 `NIUMA_UI_VERSION` 钉死如 `1.2.0`）。本机仍用 `link:../niuma-ui` 联调。niuma-ui ≥ 1.2.0 为编译包，按包名导入即可，不必 `shamefully-hoist`。

## 文档

| 文档 | 内容 |
|------|------|
| [docs/00-overview.md](./docs/00-overview.md) | 边界与路径 |
| [docs/01-api.md](./docs/01-api.md) | 用户 / Updates / Ops API |
| [docs/02-deploy.md](./docs/02-deploy.md) | www.niuma007.com 反代 |
| [docs/03-admin-frontend.md](./docs/03-admin-frontend.md) | Admin 前端目录 |
