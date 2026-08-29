# 变更日志

本文件记录 niuma-cloud 的重要变更。

格式参考 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)，版本遵循 [Semantic Versioning](https://semver.org/lang/zh-CN/)。

发版时把 `[Unreleased]` 下的条目移入新的 `## [x.y.z] - YYYY-MM-DD` 段；GitHub Release 正文由该段生成。`package.json` 的 `version`、git tag（`v*`）与本文件版本号必须一致。

## [Unreleased]

## [0.1.0] - 2026-08-29

### 新增

- Identity（邮箱密码 / 验证码）、Feedback、Updates、Admin Ops。
- 生产路径挂在 `https://www.niuma007.com/niuma`：`/niuma/cloud` API、`/niuma/ops` 运营后台。
- Windows / Linux 启停脚本：`build` / `start` / `stop` / `restart` / `status`；缺失时从 `app.yaml.example` 生成运行配置。
- GitHub Actions：CI（类型检查 + Admin 构建 + Go 测试）、推送 `v*` 标签打包 Linux/Windows amd64 并发布 Release。

### 安全

- `config/app.yaml` 不入库，仓库只提交 `config/app.yaml.example`。
