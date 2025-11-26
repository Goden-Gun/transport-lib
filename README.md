# GGA Transport Library

GGA 项目的通用通信协议库，包含所有服务间通信的 Protocol Buffers 定义。

## 目录结构

```
transport-lib/
├── proto/                    # Protocol Buffers 定义文件
│   └── bridge/
│       └── v1/
│           └── bridge.proto  # Sidecar 桥接协议
├── gen/                      # 生成的代码
│   └── go/                   # Go 语言生成代码
│       └── bridge/
│           └── v1/
│               ├── bridge.pb.go       # Protobuf 消息定义
│               └── bridge_grpc.pb.go  # gRPC 服务定义
├── scripts/
│   └── generate.sh           # 代码生成脚本
├── go.mod                    # Go 模块定义
└── README.md                 # 本文档
```

## 快速开始

### 1. 生成代码

运行代码生成脚本：

```bash
./scripts/generate.sh
```

该脚本会：
- 检查并安装必要的工具（protoc, protoc-gen-go, protoc-gen-go-grpc）
- 清理旧的生成文件
- 生成新的 Go 代码

### 2. 在项目中使用

在你的 Go 项目中引入该库：

```bash
go get github.com/your-org/gga-transport-lib
```

在代码中导入：

```go
import (
    bridgepb "gga-transport-lib/gen/go/bridge/v1"
)
```

## 协议说明

### Bridge Protocol (v1)

Sidecar 与 Biz 服务之间的 gRPC 双向流通信协议。

#### 主要消息类型

- **TransportEnvelope**: 传输信封，包含连接信息和业务消息
- **Message**: 业务消息，支持文本和音频载荷
- **ErrorPayload**: 统一错误格式

#### 错误码系统

所有错误使用统一的错误码格式：

```protobuf
message ErrorPayload {
  int32 code = 1;          // 数值错误码（40101-50302）
  string error_code = 2;   // 字符串错误码（如 TOKEN_REVOKED）
  string message = 3;      // 用户可读的错误消息
  string details = 4;      // 可选的详细信息
}
```

错误码分类：
- `40101-40199`: 认证错误
- `40201-40299`: 权限错误
- `40301-40399`: 授权错误
- `41001-41099`: 请求格式错误
- `42001-42099`: 业务逻辑错误
- `42901-42999`: 限流错误
- `50001-50099`: 服务器内部错误
- `50201-50299`: 依赖服务错误
- `50301-50399`: 业务处理错误

详细错误码列表请参考各服务的文档。

#### 流帧类型

**客户端 → 服务端：**
- `RegisterFrame`: 注册节点
- `IngressFrame`: 客户端消息入站
- `AckFrame`: 确认消息
- `HeartbeatFrame`: 心跳

**服务端 → 客户端：**
- `DeliverFrame`: 点对点消息投递
- `BroadcastFrame`: 广播/组播消息
- `HeartbeatFrame`: 心跳响应

## 开发指南

### 修改协议

1. 编辑 `proto/bridge/v1/bridge.proto` 文件
2. 运行生成脚本：`./scripts/generate.sh`
3. 提交更改（proto 文件 + 生成的代码）

### 版本管理

- 使用语义化版本：`vX.Y.Z`
- Breaking changes 升级主版本号
- 新增功能升级次版本号
- Bug 修复升级修订号

### 向后兼容性

修改协议时请遵循以下原则：

✅ **允许的修改：**
- 添加新的消息类型
- 添加新的字段（使用新的字段编号）
- 添加新的枚举值
- 添加新的服务方法

❌ **禁止的修改：**
- 删除或重命名字段
- 修改字段编号
- 修改字段类型
- 删除或重命名消息类型

## 使用示例

### Go 语言

```go
package main

import (
    "context"
    "log"

    bridgepb "gga-transport-lib/gen/go/bridge/v1"
    "google.golang.org/grpc"
)

func main() {
    // 连接到 Bridge 服务
    conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
    if err != nil {
        log.Fatalf("连接失败: %v", err)
    }
    defer conn.Close()

    // 创建客户端
    client := bridgepb.NewSidecarBridgeClient(conn)

    // 创建双向流
    stream, err := client.Stream(context.Background())
    if err != nil {
        log.Fatalf("创建流失败: %v", err)
    }

    // 发送注册消息
    err = stream.Send(&bridgepb.StreamRequest{
        Payload: &bridgepb.StreamRequest_Register{
            Register: &bridgepb.RegisterFrame{
                NodeId:    "sidecar-001",
                Namespace: "production",
            },
        },
    })
    if err != nil {
        log.Fatalf("发送注册消息失败: %v", err)
    }

    // 接收消息
    for {
        resp, err := stream.Recv()
        if err != nil {
            log.Fatalf("接收消息失败: %v", err)
        }

        switch payload := resp.Payload.(type) {
        case *bridgepb.StreamResponse_Deliver:
            log.Printf("收到投递消息: %+v", payload.Deliver)
        case *bridgepb.StreamResponse_Broadcast:
            log.Printf("收到广播消息: %+v", payload.Broadcast)
        case *bridgepb.StreamResponse_Heartbeat:
            log.Printf("收到心跳: %s", payload.Heartbeat.Nonce)
        }
    }
}
```

## 工具要求

- Go 1.25.3+
- protoc 3.0+
- protoc-gen-go
- protoc-gen-go-grpc

## 相关项目

- [gga-sidecar](https://github.com/your-org/gga-sidecar) - WebSocket Sidecar 服务
- [GGA](https://github.com/your-org/GGA) - 主业务服务

## 许可证

[您的许可证]

## 维护者

- GGA Team

## 更新日志

### v1.0.0 (2025-01-27)

- 初始版本
- 添加 Bridge Protocol v1
- 统一错误码系统（numeric + string）
- 支持文本和音频载荷
- 支持双向流通信
