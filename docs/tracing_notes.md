# Bridge Tracing Notes

- 在服务初始化阶段配置 OpenTelemetry TracerProvider + Propagator（例如 Jaeger/OTLP）。
- WebSocket 入口应生成/传入 `trace_id`（放入 `message.metadata.trace_id`），Sidecar 调用 `envelope.StampTrace` 后即可在 gRPC 中透传。
- Bridge Client 调用 `PublishIngress` 前可用 `pkg/tracing.InjectMetadata` 将 ctx 注入 gRPC metadata，确保 TraceID 同步。
- Bridge Server Handler (`OnIngress/OnAck/OnHeartbeat` 等) 需用 `pkg/tracing.ExtractMetadata` 恢复 ctx，并创建 span，记录 node_id/namespace/action 等属性。
- Deliver/Broadcast/ACK 也应把 TraceID 写回 Envelope，保证客户端/后续 Kafka 事件仍可串联。
