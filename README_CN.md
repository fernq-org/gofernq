# gofernq

[![Go Reference](https://pkg.go.dev/badge/github.com/fernq-org/gofernq.svg)](https://pkg.go.dev/github.com/fernq-org/gofernq)
[![Go Report Card](https://goreportcard.com/badge/github.com/fernq-org/gofernq)](https://goreportcard.com/report/github.com/fernq-org/gofernq)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

<div align="right">

**[English](README.md)** | **[中文](README_CN.md)**

</div>

---

gofernq 是 Fernq 协议的 Go 语言实现。它提供了一种简单高效的方式与 Fernq 服务器通信，具有灵活的路由系统和强大的连接管理功能。

## 特性

- **简单的 API**: 易于使用的客户端接口，设置简单
- **灵活的路由**: 内置路由器，支持基于路径的请求处理
- **连接管理**: 自动重连和优雅关闭
- **连接监控**: 内置 Done() 通道用于跟踪连接状态
- **协议支持**: 完整的 Fernq 协议实现，支持 JSON 和表单内容类型
- **并发安全**: 具有适当锁定机制的线程安全操作
- **错误处理**: 全面的错误类型和消息

## 安装

```bash
go get github.com/fernq-org/gofernq
```

## 快速开始

### 基本用法

```go
package main

import (
    "fmt"
    "github.com/fernq-org/gofernq"
)

func main() {
    // 创建新的客户端
    client := gofernq.NewClient("my-client")

    // 连接到 Fernq 服务器
    // URL 格式: fernq://<host>[:<port>]/<uuid>#<room_name>?room_pass=<password>
    // 示例: fernq://localhost:9147/550e8400-e29b-41d4-a716-446655440000#my-room?room_pass=123456
    if err := client.Connect("fernq://localhost:9147/550e8400-e29b-41d4-a716-446655440000#my-room?room_pass=123456"); err != nil {
        panic(err)
    }
    defer client.Close()

    // 发送请求
    resp, err := client.Request("/api/hello")
    if err != nil {
        panic(err)
    }

    fmt.Printf("响应: %+v\n", resp)
}
```

### 使用路由器

```go
package main

import (
    "github.com/fernq-org/gofernq"
)

func main() {
    // 创建路由器
    router := gofernq.NewRouter()

    // 添加路由处理器
    router.AddRoute("/api/hello", func(ctx *gofernq.Context) {
        ctx.JSON(gofernq.StatusOK, map[string]string{
            "message": "Hello, World!",
        })
    })

    // 使用路由器创建客户端
    client := gofernq.NewClient("my-client", router)

    // 连接并使用
    // URL 格式: fernq://<host>[:<port>]/<uuid>#<room_name>?room_pass=<password>
    if err := client.Connect("fernq://localhost:9147/550e8400-e29b-41d4-a716-446655440000#my-room?room_pass=123456"); err != nil {
        panic(err)
    }
    defer client.Close()
}
```

### 带请求体的请求

```go
// 发送带有 JSON 请求体的请求
data := map[string]interface{}{
    "name": "Alice",
    "age":  30,
}

resp, err := client.Request("/api/user", data)
if err != nil {
    panic(err)
}
```

### 监听连接状态

```go
package main

import (
    "fmt"
    "time"
    "github.com/fernq-org/gofernq"
)

func main() {
    client := gofernq.NewClient("my-client")

    // URL 格式: fernq://<host>[:<port>]/<uuid>#<room_name>?room_pass=<password>
    if err := client.Connect("fernq://localhost:9147/550e8400-e29b-41d4-a716-446655440000#my-room?room_pass=123456"); err != nil {
        panic(err)
    }
    defer client.Close()

    // 监听连接状态
    go func() {
        <-client.Done()
        fmt.Println("连接已关闭！")
        // 在这里处理重连逻辑
    }()

    // 你的主业务逻辑...
    time.Sleep(30 * time.Second)
}
```

## API 参考

### 客户端

#### `NewClient(name string, routers ...*Router) *Client`

创建一个新的客户端实例。

**参数:**
- `name`: 客户端名称标识符
- `routers`: 可选的路由器实例，用于处理传入的请求

**返回值:** 新的 Client 实例

#### `Connect(url string) error`

连接到指定 URL 的 Fernq 服务器。

**参数:**
- `url`: Fernq 服务器的协议 URL（例如："fernq://localhost:9147/550e8400-e29b-41d4-a716-446655440000#my-room?room_pass=123456"）
  - 格式: fernq://<host>[:<port>]/<uuid>#<room_name>?room_pass=<password>

**返回值:** 如果连接失败则返回错误

#### `Request(path string, body ...any) (*ResponseMessage, error)`

向指定路径发送请求。

**参数:**
- `path`: 请求路径（例如："/api/users"）
- `body`: 可选的请求体（将被序列化为 JSON）

**返回值:** ResponseMessage 和错误

#### `Done() <-chan struct{}`

返回一个通道，当客户端连接终止时该通道将被关闭。使用此方法来监听连接状态并处理重连逻辑。

**返回值:** 只读通道，连接丢失时会关闭

#### `Close()`

优雅地关闭客户端连接。

### 路由器

#### `NewRouter() *Router`

创建一个新的路由器实例。

**返回值:** 新的 Router 实例

#### `AddRoute(path string, handler func(*Context)) error`

为指定路径添加路由处理器。

**参数:**
- `path`: 路由路径（支持动态片段）
- `handler`: 处理此路径请求的函数

**返回值:** 如果路由无效则返回错误

### 上下文

Context 对象提供了处理请求的方法：

#### `JSON(state int, body any)`

发送带有指定状态码和请求体的 JSON 响应。

**参数:**
- `state`: HTTP 类型的状态码（例如：200, 404, 500）
- `body`: 将被序列化为 JSON 的响应体

### 状态码

常用的状态码：

- `StatusOK` (200)
- `StatusCreated` (201)
- `StatusBadRequest` (400)
- `StatusUnauthorized` (401)
- `StatusForbidden` (403)
- `StatusNotFound` (404)
- `StatusInternalServerError` (500)

## 许可证

MIT License

## 贡献

欢迎贡献！请随时提交 Pull Request。
