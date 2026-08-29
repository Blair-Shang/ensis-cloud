# 部署（www.niuma007.com 路径分流）

## 反代示意

```text
https://www.niuma007.com/           → niuma-site :8080
https://www.niuma007.com/api/*      → niuma-site :8080
https://www.niuma007.com/healthz    → niuma-site :8080
https://www.niuma007.com/niuma/*    → niuma-cloud :8090
```

官网仓已写入 `niuma-site/config/conf.d/niuma-site.conf`。本仓副本见 `config/nginx-locations.conf`。

Nginx 示例：

```nginx
upstream niuma_cloud {
  server 127.0.0.1:8090;
  keepalive 16;
}

location = /niuma {
  return 302 /niuma/ops/;
}

location /niuma/ {
  client_max_body_size 2g;
  proxy_pass http://127.0.0.1:8090;
  proxy_set_header Host $host;
  proxy_set_header X-Real-IP $remote_addr;
  proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
  proxy_set_header X-Forwarded-Proto $scheme;
  proxy_request_buffering off;
  proxy_buffering off;
  proxy_read_timeout 1800s;
  proxy_send_timeout 1800s;
}
```

客户端配置：

- 桌面 / 官网：`VITE_CLOUD_API_BASE=https://www.niuma007.com/niuma/cloud`
- Ops 前端构建 `base=/niuma/ops/`，API 默认相对 `/niuma/cloud`

GitHub 流水线（NiuMa `Pack and Release`）：

- Cloud：`pipeline.token` 或环境变量 `NIUMA_PIPELINE_TOKEN`（至少 16 字符）
- 仓库 Secrets：`NIUMA_CLOUD_PIPELINE_TOKEN`（与上相同）
- 可选 Variables：`NIUMA_CLOUD_API_BASE`（默认 `https://www.niuma007.com/niuma/cloud`）
- 流水线先 `POST /api/v1/pipeline/releases/assets` 把安装包落到 `{data_dir}/releases/`，再登记 published
- 对外下载：`https://www.niuma007.com/niuma/cloud/api/v1/updates/files/{product}/{channel}/{platform}/{arch}/{version}/{name}`
- `updates.public_base` 默认即上述域名；`http_timeout_min` 建议 ≥ 30（上传/下载大包）
- Nginx `/niuma/` 须 `client_max_body_size 2g` 且读写超时约 1800s；改完后 `nginx -s reload`
- 磁盘：按平台保留历史版本，注意 `data/releases` 容量
- 官网首页下载：把 niuma-site `download.windows_url` 填成对应 files 地址（与桌面更新同一条链）

## MySQL

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
