# Bug 复现说明

## Bug 是什么

创建订阅时，当请求未携带 metadata 或 status 留空，服务会向 nil map 写入字段，导致 panic。

## 如何触发

发送一个不含 metadata 字段、status 为空的创建订阅请求。

```bash
CGO_ENABLED=0 go test ./... -count=20
```

## 错误信息

日志出现：

```text
assignment to entry in nil map
```

HTTP 请求返回 500。
