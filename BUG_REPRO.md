# Bug 复现说明

## Bug 是什么

订阅账单相关逻辑出现组合错误：金额四舍五入错误，按状态筛选会把排除条件写反，未来 7 天扣费窗口多包含 1 天，HTTP 列表接口没有传递 status 查询参数。

## 如何触发

1. 创建金额为 39.955 的订阅。
2. 请求 `GET /api/v1/subscriptions?status=active`。
3. 请求 `GET /api/v1/subscriptions/upcoming-charges?days=7`。
4. 运行完整测试：

```bash
CGO_ENABLED=0 go test ./... -count=20
```

## 错误信息

- 金额应四舍五入为 39.96，实际为 39.95。
- active 筛选返回错误集合。
- 7 天窗口把第 8 天订阅也返回，报告 count 为 3。
- invalid status 本应返回 400，实际返回 200。
