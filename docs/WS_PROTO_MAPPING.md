# WebSocket ↔ Protobuf 映射

- **单一真相**：`proto/bridge/v1/bridge.proto` 定义的 `Message` / `TransportEnvelope` 即 WebSocket payload。所有客户端按
  proto JSON Mapping 规则序列化。
- **版本协商**：`TransportEnvelope.envelope_version` 与 `Message.version` 均默认为 `2025-01`，Sidecar & Chat Worker 可在
  RegisterFrame.supported_versions/bridge_version 中声明能力。
- **Trace 透传**：`trace_id` 字段 + `attributes["trace_id"]` 需与 `pkg/tracing` 保持一致，便于 Jaeger/Grafana 追踪。
- **广播字段**：
    - 单播：`target_connection_ids`/`target_user_ids` 为空，Deliver 指定 connection。
    - 群播：填充 `target_user_ids` 或 `target_connection_ids`。
    - 全局广播：字段为空，由 Sidecar fan-out。
- **JSON Schema**：`schema/transport-envelope.json` 由 proto 派生，可供前端校验。
- **Go Helper**：`pkg/envelope` 直接别名 proto 生成代码，并提供 `NormalizeMessage/ValidateIngress/NormalizeEnvelope` 等函数。
