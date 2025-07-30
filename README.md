# fawa

> 让高性能分布式通信与协作，变得优雅而简单。

fawa 是一套以 Go 语言为内核、面向未来的分布式服务集群。它不仅是工程师的工具箱，更是高效协作与数据流动的桥梁。无论是文件极速传输、多人实时白板，还是流式通信的极致体验，fawa 都以现代云原生理念和极致性能为你赋能。

---

## 技术核心与项目亮点

- **全链路 gRPC/Connect 协议**：统一的高性能流式通信，支持一元、服务端流、客户端流、双向流，轻松应对复杂业务场景。
- **多后端存储适配**：文件服务支持 MinIO 对象存储与 Dragonfly/Redis 分布式缓存，兼容云原生与本地部署。
- **WebTransport/HTTP3 实时协作**：白板服务支持 WebTransport 与 WebSocket，低延迟、高并发，适配未来浏览器生态。
- **模块化多服务架构**：每个服务独立可部署，接口清晰，易于扩展和二次开发。
- **Kubernetes 原生部署**：一键上云，支持多环境配置与自动证书管理。

---

## 服务一览

### 1. greetservice —— 流式通信的极致演绎

greetservice 是 fawa 的问候演示服务，全面展示了 gRPC/Connect 的一元、服务端流、客户端流、双向流等通信模式。它不仅是新手的入门范例，更是理解流式协议的最佳实践。

- **SayHello**：一元 RPC，简单高效。
- **GreetStream**：服务端流，持续推送问候。
- **GreetClientStream**：客户端流，聚合多次问候。
- **GreetBidiStream**：双向流，实时互动。
- **接口定义**：`proto/greet/v1/hello.proto`

### 2. fileservice —— 极速与安全的文件流转

fileservice 让大文件传输变得前所未有的高效与安全。支持分片上传、断点续传、临时直链下载，后端可选 MinIO 或 Dragonfly/Redis，轻松应对海量数据与高并发场景。

- **SendFile**：客户端流式上传，边传边存。
- **ReceiveFile**：服务端流式下载，断点续传。
- **GetDownloadURL**：生成临时直链，安全分享。
- **多存储后端**：对象存储与分布式缓存无缝切换。
- **接口定义**：`proto/file/v1/file.proto`

### 3. canvaxservice —— 云端协作的艺术空间

canvaxservice 以 gRPC/Connect 为桥梁，打造多人实时协作白板。每一笔绘制、每一次同步，都是流式通信的极致体现。无论是远程头脑风暴，还是在线教学，皆可流畅无阻。

- **Collaborate**：双向流式协作，事件与历史实时同步。
- **高并发支持**：每个连接独立管理，广播高效。
- **接口定义**：`proto/canva/v1/canva.proto`

### 4. canvaservice —— WebTransport 赋能的未来白板

canvaservice 站在 WebTransport/HTTP3 与 WebSocket 的前沿，带来极致低延迟的多人实时白板体验。创新的会话管理与事件广播机制，让每一次协作都如身临其境。

- **WebTransport/WS 双协议支持**：兼容新旧浏览器，拥抱未来。
- **会话与历史管理**：自动清理、实时同步，保障体验。
- **极致性能**：适合高并发、低延迟场景。

---

## 一键开发与云原生部署

- **just 命令集成**：`just` 一键列出/执行所有开发、测试、构建、部署命令。
- **Kubernetes 支持**：`deploy/` 目录内含全套 K8s 配置，支持多环境、自动证书、服务发现。
- **性能测试**：`k6_test/` 提供 k6 脚本，助力高并发压测。

---

## 快速上手

1. 安装 Go 1.20+、just、MinIO/Redis（如需文件服务）、k6（如需压测）。
2. `just build <服务名>` 构建，`just run <服务名>` 运行。
3. 参考各服务 `proto/` 目录，快速集成自定义客户端。
4. Kubernetes 用户可直接应用 `deploy/base/` 下的 YAML 文件。

---

## 开源与贡献

- 遵循 Apache 2.0 协议，欢迎任何形式的贡献与建议。
- PR 需通过 lint、测试与 license 检查。

---

> fawa，让分布式协作与数据流动，优雅如诗。
