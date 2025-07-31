# fawa                [English Version](README.md)

> 让高性能分布式通信与协作，变得优雅而简单。

fawa 是一套基于 Go 语言构建的分布式微服务集群，专注于实时通信与协作场景。它采用现代化的云原生架构，提供文件传输、实时白板协作等核心功能，为开发者提供高性能、可扩展的分布式通信解决方案。



---

## 技术架构与核心特性

- **gRPC/Connect 协议栈**：统一的高性能流式通信框架，支持一元、服务端流、客户端流、双向流等多种通信模式。
- **多存储后端适配**：文件服务支持 MinIO 对象存储与 Dragonfly/Redis 元数据存储，实现存储与元数据分离的架构设计。
- **WebTransport/HTTP3 实时协作**：白板服务支持 WebTransport 与 WebSocket 双协议，低延迟、高并发，适配现代浏览器生态。
- **微服务架构**：每个服务独立部署，接口清晰，支持水平扩展和独立运维。
- **Kubernetes 原生支持**：完整的 K8s 部署配置，支持多环境、自动证书管理、服务发现。

---

## 服务架构详解

### 1. greetservice —— gRPC 流式通信演示服务

greetservice 是 fawa 的入门演示服务，完整展示了 gRPC/Connect 协议的各种流式通信模式。它不仅是新手的入门范例，更是理解现代流式协议的最佳实践。

**核心功能：**
- **SayHello**：一元 RPC，简单高效的请求-响应模式
- **GreetStream**：服务端流式 RPC，持续推送问候消息
- **GreetClientStream**：客户端流式 RPC，聚合多次问候请求
- **GreetBidiStream**：双向流式 RPC，实时双向通信

**技术特点：**
- 基于 Connect 协议，兼容 gRPC 和 REST
- 支持 HTTP/1.1、HTTP/2、HTTP/3 传输
- 自动生成客户端代码，支持多种语言

### 2. fileservice —— 分布式文件传输服务

fileservice 提供高性能的文件上传、下载和分享功能，采用存储与元数据分离的架构设计，支持多种存储后端。

**核心功能：**
- **SendFile**：客户端流式上传，支持大文件分片传输
- **ReceiveFile**：服务端流式下载，支持断点续传
- **GetDownloadURL**：生成临时预签名链接，安全分享文件

**存储架构：**
- **MinIO 对象存储**：负责文件内容的持久化存储
- **Dragonfly/Redis 元数据存储**：存储文件元数据（文件名、大小、存储路径等）
- **存储分离设计**：元数据与文件内容分离，提高系统可扩展性

**技术特点：**
- 支持文件元数据 TTL 管理（25分钟自动过期）
- 预签名 URL 机制，支持临时直链下载
- 可配置的公共端点，支持 CDN 集成

### 3. canvaxservice —— gRPC 实时协作白板

canvaxservice 基于 gRPC 双向流实现多人实时协作白板，提供低延迟的绘图事件同步和会话管理。

**核心功能：**
- **Collaborate**：双向流式协作，实时同步绘图事件
- **会话管理**：自动管理客户端连接和绘图历史
- **事件广播**：高效的绘图事件广播机制

**技术特点：**
- 内存中的绘图历史管理（最多1000个事件）
- 自动清理过期连接和会话
- 支持绘图事件类型：ping、clear、draw 等

### 4. canvaservice —— WebTransport 实时协作白板

canvaservice 基于 WebTransport/HTTP3 和 WebSocket 协议，提供面向浏览器的实时协作白板服务。

**核心功能：**
- **WebTransport 支持**：基于 HTTP/3 的低延迟双向通信
- **WebSocket 降级**：兼容不支持 WebTransport 的浏览器
- **会话管理**：基于代码的会话创建和加入机制
- **自动清理**：10分钟无活动自动清理会话

**技术特点：**
- 双协议支持：WebTransport（优先）+ WebSocket（降级）
- 会话级别的客户端管理
- 绘图历史的内存管理
- 支持多种绘图事件类型

---

## 开发与部署

### 开发环境

- **Go 1.24+**：核心开发语言
- **just**：项目构建和任务管理工具
- **buf**：Protocol Buffers 代码生成
- **MinIO**：对象存储服务（可选）
- **Dragonfly/Redis**：元数据存储服务（可选）

### 快速开始

1. **克隆项目**
   ```bash
   git clone <repository-url>
   cd fawa
   ```

2. **构建服务**
   ```bash
   just build fileservice    # 构建文件服务
   just build greetservice   # 构建问候服务
   just build canvaxservice  # 构建gRPC白板服务
   just build canvaservice   # 构建WebTransport白板服务
   ```

3. **运行服务**
   ```bash
   just run fileservice      # 运行文件服务
   just run greetservice     # 运行问候服务
   just run canvaxservice    # 运行gRPC白板服务
   just run canvaservice     # 运行WebTransport白板服务
   ```

### 性能测试

项目包含 k6 性能测试脚本，支持高并发压测：

```bash
k6 run k6_test/script.js
```

测试配置：
- 200 并发用户
- 3 分钟持续压测
- 95% 请求延迟 < 500ms
- 99% 请求延迟 < 1500ms

### Kubernetes 部署

项目提供完整的 K8s 部署配置：

```bash
# 部署到 Kubernetes
kubectl apply -f deploy/base/
```

支持的功能：
- 自动证书管理（cert-manager）
- 服务发现和负载均衡
- 多环境配置管理
- 自动扩缩容

---

## 项目结构

```
fawa/
├── fileservice/          # 文件传输服务
│   ├── storage/         # 存储抽象层
│   ├── handler/         # gRPC 处理器
│   └── proto/           # Protocol Buffers 定义
├── greetservice/        # 问候演示服务
├── canvaxservice/       # gRPC 白板服务
├── canvaservice/        # WebTransport 白板服务
├── deploy/              # Kubernetes 部署配置
├── k6_test/            # 性能测试脚本
└── justfile            # 项目任务管理
```

---

## 开源协议

- 遵循 Apache 2.0 协议
- 欢迎任何形式的贡献与建议
- PR 需通过 lint、测试与 license 检查

---

> fawa，让分布式协作与数据流动，优雅如诗。 
