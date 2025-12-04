# transport-lib Roadmap

1. **Scaffolding (当前)**
   - 引入 `pkg/envelope`, `pkg/codes`, `pkg/bridge`, `pkg/tracing` 骨架。
   - README 增加包结构说明。
2. **消息与 Schema**
   - 实现 JSON Schema、Serde helper、错误码导出文档。
3. **Bridge Helper**
   - 完成 gRPC 客户端/服务端封装（心跳、ACK、Backpressure、Drain）。
4. **Tracing / Examples**
   - 提供 Sidecar / Chat Worker 示例，默认 OTel 配置。
5. **Release**
   - 文档、CHANGELOG、tag 发布，通知下游项目。
