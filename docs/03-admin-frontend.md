# Admin 前端目录

按 Vue 惯例按功能划分：

```text
admin/src/
  api/                 # 请求层（类型 + 封装）
    request.ts         # 全局 HTTP 工具（默认 /niuma/cloud）
    types.ts           # 接口类型
    auth.ts / stats.ts / users.ts / feedback.ts
    releases.ts / admin.ts / ai.ts
    index.ts           # 统一导出
  components/          # 可复用组件（含 ui.ts）
  layouts/             # 布局
  views/               # 路由页面（按业务域分子目录）
    login/
    dashboard/
    users/
    staff/
    feedback/
    releases/
    ai/                # 接入点 + 用量监控
    audit/
  router/              # history base=/niuma/ops/
  styles/
  App.vue
  main.ts
```

业务视图只调用 `@/api`，不直接 `fetch`。

生产访问：`https://www.niuma007.com/niuma/ops/`
