# Bug 复现说明

## Bug 是什么

status 留空时被默认成 paused，本月预计支出把 paused 订阅也计入总额。

## 如何触发

创建订阅时不传 status，再查询本月预计支出。

```bash
CGO_ENABLED=0 go test ./...
```

## 错误信息

- 新创建订阅状态实际为 paused，而不是 active。
- 本月预计支出把 paused 订阅金额也加进去，金额和条数都偏大。
