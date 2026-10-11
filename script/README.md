# 运维脚本

仓库根目录执行。Windows 用 `.ps1`，Linux 用 `.sh`。

| 脚本 | 作用 |
|------|------|
| `build` | `pnpm` 构建 Admin 到 `server/internal/web/dist`，再 `go build` 打出单二进制（**内嵌静态资源**） |
| `pack-release` | 校验 CHANGELOG 版本段，打出 Linux/Windows amd64 发版归档到 `output/`。自动发版会先把 `[Unreleased]` 收成版本段再调用本脚本 |
| `start` | 后台启动 `run/ensis-cloud[.exe]`，写 pid / 日志；缺失时从样例生成 `config/app.yaml` |
| `stop` | 按 pid 停止（先 SIGTERM，超时再强杀） |
| `restart` | stop + start |
| `status` | 看 pid 是否存活，并探测 `/niuma/cloud/healthz` |

`config/app.yaml` 含数据库口令、JWT、SMTP、pipeline token，**不要提交**；仓库只保留 `config/app.yaml.example`。

## Windows

```powershell
powershell -ExecutionPolicy Bypass -File script/build.ps1
powershell -ExecutionPolicy Bypass -File script/start.ps1
powershell -ExecutionPolicy Bypass -File script/status.ps1
powershell -ExecutionPolicy Bypass -File script/stop.ps1
powershell -ExecutionPolicy Bypass -File script/restart.ps1
```

## Linux

```bash
chmod +x script/*.sh
bash script/build.sh
bash script/start.sh
bash script/status.sh
bash script/stop.sh
bash script/restart.sh
```

## 产物

| 路径 | 说明 |
|------|------|
| `run/ensis-cloud.exe` / `run/ensis-cloud` | 可执行文件（内嵌 Admin） |
| `run/ensis-cloud.pid` | 进程号 |
| `config/` | 启动时自动创建；`app.yaml` 首次从 `app.yaml.example` 生成 |
| `logs/` | 启动时自动创建；业务日志 `ensis-cloud.log`（zap + 滚动） |
| `data/` | 启动时自动创建；安装包目录默认 `{data_dir}/releases` |

默认监听 `http://127.0.0.1:8090/`（`http_addr`，见 `config/app.yaml.example`）：

- 健康检查：`/niuma/cloud/healthz`
- 运营后台：`/niuma/ops/`

开发联调仍可用 `pnpm dev`（Vite 热更新，不走内嵌）。
