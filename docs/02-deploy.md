# 部署（www.niuma007.com 路径分流）

## 上线前加固

1. **密钥**：`jwt.secret` / 流水线 token / SMTP 授权码不要用文档里的样例。优先环境变量（见 `config/app.env.example`）：`NIUMA_JWT_SECRET`、`NIUMA_PIPELINE_TOKEN`、`NIUMA_SMTP_PASSWORD`、`NIUMA_DB_DSN`。含 `change-me` / `dev-niuma` / `example` 的 JWT 会拒绝启动。
2. **`smtp.log_otp`**：生产必须 `false`（未写时默认 false）。
3. **监听**：`http_addr: "127.0.0.1:8090"`，只让 Nginx 443 对外。`trust_proxy: true`。
4. **MySQL**：安全组只放行应用机；不要用公网 3306 + `root`。DSN 可用 `NIUMA_DB_DSN`。
5. **超时**：`http_timeout_min: 30`。`updates.public_base` 与 `download_url_hosts` 必须含官方域名。
6. **员工密码**：`staff.password` 只在库里还没有员工时种子生效；已有账号在 Ops 改密。
7. **Nginx**：`http { }` 里声明 `niuma_dl_latest` / `niuma_dl_conn`；对 `/updates/download` 限请求、对 `/updates/files/` 限连接与带宽。改完 `nginx -t && nginx -s reload`。

## 反代示意

```text
https://www.niuma007.com/           → niuma-site :8080
https://www.niuma007.com/api/*      → niuma-site :8080
https://www.niuma007.com/healthz    → niuma-site :8080
https://www.niuma007.com/niuma/*    → niuma-cloud :8090
```

官网仓已写入 `niuma-site/config/conf.d/niuma-site.conf`。本仓副本见 `config/nginx-locations.conf`。

Nginx 示例：

完整片段见 `config/nginx-locations.conf` 与官网仓 `config/conf.d/niuma-site.conf`。`http { }` 须先声明：

```nginx
limit_req_zone  $binary_remote_addr zone=niuma_dl_latest:10m rate=8r/s;
limit_conn_zone $binary_remote_addr zone=niuma_dl_conn:10m;
```

客户端配置：

- 桌面 / 官网：`VITE_CLOUD_API_BASE=https://www.niuma007.com/niuma/cloud`
- Ops 前端构建 `base=/niuma/ops/`，API 默认相对 `/niuma/cloud`

GitHub 流水线（NiuMa `Pack and Release`）：

- Cloud：`pipeline.token` 或环境变量 `NIUMA_PIPELINE_TOKEN`（至少 16 字符）
- 仓库 Secrets：`NIUMA_CLOUD_PIPELINE_TOKEN`（与上相同）
- 可选 Variables：`NIUMA_CLOUD_API_BASE`（默认 `https://www.niuma007.com/niuma/cloud`）
- 流水线先 `POST /api/v1/pipeline/releases/assets` 把安装包落到 `{data_dir}/releases/`，再登记 published
- 对外「永远最新」下载（官网 / 投放）：`https://www.niuma007.com/niuma/cloud/api/v1/updates/download?product=niuma&channel=stable&platform=windows&arch=x64`
- Linux / macOS 尚未稳定：官网与流水线使用 `channel=beta`（如 `platform=linux&arch=x64`、`platform=macos&arch=arm64`）
- 版本化落地（桌面更新、CDN 缓存）：`https://www.niuma007.com/niuma/cloud/api/v1/updates/files/{product}/{channel}/{platform}/{arch}/{version}/{name}`
- `updates.public_base` 默认即上述域名；`http_timeout_min` 建议 ≥ 30（上传/下载大包）
- Nginx `/niuma/` 须 `client_max_body_size 2g` 且读写超时约 1800s；改完后 `nginx -s reload`
- 磁盘：按平台保留历史版本，注意 `data/releases` 容量
- 官网 hit 默认 302 到 `/updates/download`，不必再改 `download.windows_url`；该字段仅作 cloud 关闭时的应急直链

## MySQL

安全组 / 防火墙只允许应用机访问 3306，不要对 `0.0.0.0/0` 开放。应用账号不要用 `root`。

`config/app.yaml`（已 gitignore，勿提交；仓库只保留 `config/app.yaml.example`）：

```yaml
db:
  driver: mysql
  dsn: "user:pass@tcp(host:3306)/niuma_cloud?parseTime=true&charset=utf8mb4&loc=UTC"
```

库名 `niuma_cloud` 不存在时，启动会自动 `CREATE DATABASE`。

Schema 由 `server/internal/db/migrations/{mysql,sqlite}/*.sql` 版本化维护。  
新增迁移（如 `004_update_hits.sql`）会在启动时自动执行。若本地/开发库仍是未发布阶段改过的旧 `001_init.sql`，需 **DROP 后重建**（`CREATE IF NOT EXISTS` 不会改已有列）：

```sql
DROP DATABASE IF EXISTS niuma_cloud;
```

重启服务后会自动建库并执行迁移。
