# Subscription Billing API

一个用 Go 编写的个人订阅账单管理后端 API，可记录 Netflix、云服务器、健身房等周期性订阅。

## 功能

- 新增订阅
- 查看全部订阅
- 按状态筛选订阅
- 修改续费日期
- 删除订阅
- 返回本月预计支出
- 返回即将扣费列表
- 统一 JSON 响应与错误格式
- 日期、金额、状态等输入校验
- 本地 JSON 文件持久化，支持并发安全读写

## 目录结构

```text
.
├── cmd
│   └── server
│       └── main.go
├── internal
│   └── subscription
│       ├── handler
│       │   └── handler.go
│       ├── model
│       │   └── subscription.go
│       ├── repository
│       │   ├── json_repository.go
│       │   └── repository.go
│       └── service
│           ├── errors.go
│           └── service.go
├── pkg
│   ├── config
│   │   └── config.go
│   ├── middleware
│   │   └── middleware.go
│   └── response
│       └── response.go
├── data
│   └── .gitkeep
├── Dockerfile
└── go.mod
```

## 启动方式

### 使用 Docker

```bash
docker build -t subscription-billing-api .
docker run --rm -p 18081:8080 subscription-billing-api
```

服务启动后监听 `http://localhost:18081`。

如需使用宿主机目录持久化数据：

```bash
docker run --rm -p 18081:8080 \
  -v "$(pwd)/data:/app/data" \
  subscription-billing-api
```

### 本地运行

需要 Go 1.22 或更高版本。

```bash
go run ./cmd/server
```

## 环境变量

| 变量名 | 默认值 | 说明 |
| --- | --- | --- |
| `PORT` | `8080` | HTTP 服务监听端口 |
| `DATA_FILE` | `data/subscriptions.json` | 本地 JSON 数据文件路径 |

## API 路径

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/healthz` | 健康检查 |
| `POST` | `/api/v1/subscriptions` | 新增订阅 |
| `GET` | `/api/v1/subscriptions` | 查看全部订阅，可用 `status` 筛选 |
| `GET` | `/api/v1/subscriptions/{id}` | 查看单个订阅 |
| `PATCH` | `/api/v1/subscriptions/{id}/renewal-date` | 修改续费日期 |
| `DELETE` | `/api/v1/subscriptions/{id}` | 删除订阅 |
| `GET` | `/api/v1/subscriptions/monthly-total` | 返回本月预计支出 |
| `GET` | `/api/v1/subscriptions/upcoming-charges?days=7` | 返回即将扣费列表 |

订阅状态支持 `active`、`paused`、`cancelled`。

## 数据格式

创建订阅请求示例：

```json
{
  "name": "Netflix",
  "description": "4K family plan",
  "amount": 39.9,
  "status": "active",
  "next_renewal_date": "2026-09-01"
}
```

金额会保留两位小数，日期必须使用 `YYYY-MM-DD` 且不能早于当天。

## curl 示例

新增订阅：

```bash
curl -sS -X POST http://localhost:18081/api/v1/subscriptions \
  -H 'Content-Type: application/json' \
  -d '{"name":"Netflix","amount":39.9,"status":"active","next_renewal_date":"2026-09-01"}'
```

查看全部订阅：

```bash
curl -sS http://localhost:18081/api/v1/subscriptions
```

按状态筛选：

```bash
curl -sS 'http://localhost:18081/api/v1/subscriptions?status=active'
```

修改续费日期：

```bash
curl -sS -X PATCH http://localhost:18081/api/v1/subscriptions/{id}/renewal-date \
  -H 'Content-Type: application/json' \
  -d '{"next_renewal_date":"2026-10-15"}'
```

删除订阅：

```bash
curl -sS -X DELETE http://localhost:18081/api/v1/subscriptions/{id}
```

本月预计支出：

```bash
curl -sS http://localhost:18081/api/v1/subscriptions/monthly-total
```

即将扣费列表：

```bash
curl -sS 'http://localhost:18081/api/v1/subscriptions/upcoming-charges?days=7'
```
