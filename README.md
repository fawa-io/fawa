# fawa
[中文版 / Chinese Version](README.zh-CN.md)
> Making high-performance distributed communication and collaboration elegant and simple.

fawa is a distributed microservices cluster built with Go, focused on real-time communication and collaboration scenarios. It adopts modern cloud-native architecture to provide core functionalities such as file transfer and real-time whiteboard collaboration, offering developers a high-performance, scalable distributed communication solution.



---

## Technical Architecture & Core Features

- **gRPC/Connect Protocol Stack**: Unified high-performance streaming communication framework supporting unary, server-streaming, client-streaming, and bidirectional streaming communication modes.
- **Multi-Storage Backend Adaptation**: File service supports MinIO object storage and Dragonfly/Redis metadata storage, implementing a storage and metadata separation architecture design.
- **WebTransport/HTTP3 Real-time Collaboration**: Whiteboard service supports WebTransport and WebSocket dual protocols, low-latency, high-concurrency, adapting to modern browser ecosystems.
- **Microservices Architecture**: Each service is independently deployable with clear interfaces, supporting horizontal scaling and independent operations.
- **Kubernetes Native Support**: Complete K8s deployment configurations supporting multi-environment, automatic certificate management, and service discovery.

---

## Service Architecture Details

### 1. greetservice —— gRPC Streaming Communication Demo Service

greetservice is fawa's entry-level demo service that comprehensively demonstrates various streaming communication modes of the gRPC/Connect protocol. It's not only an entry example for newcomers but also the best practice for understanding modern streaming protocols.

**Core Features:**
- **SayHello**: Unary RPC, simple and efficient request-response pattern
- **GreetStream**: Server-streaming RPC, continuously pushing greeting messages
- **GreetClientStream**: Client-streaming RPC, aggregating multiple greeting requests
- **GreetBidiStream**: Bidirectional streaming RPC, real-time bidirectional communication

**Technical Characteristics:**
- Based on Connect protocol, compatible with gRPC and REST
- Supports HTTP/1.1, HTTP/2, HTTP/3 transport
- Auto-generates client code, supporting multiple languages

### 2. fileservice —— Distributed File Transfer Service

fileservice provides high-performance file upload, download, and sharing functionality, adopting a storage and metadata separation architecture design that supports multiple storage backends.

**Core Features:**
- **SendFile**: Client-streaming upload, supporting large file chunked transfer
- **ReceiveFile**: Server-streaming download, supporting resumable transfer
- **GetDownloadURL**: Generates temporary pre-signed links for secure file sharing

**Storage Architecture:**
- **MinIO Object Storage**: Responsible for persistent storage of file content
- **Dragonfly/Redis Metadata Storage**: Stores file metadata (filename, size, storage path, etc.)
- **Storage Separation Design**: Metadata and file content separation, improving system scalability

**Technical Characteristics:**
- Supports file metadata TTL management (25-minute automatic expiration)
- Pre-signed URL mechanism, supporting temporary direct link downloads
- Configurable public endpoints, supporting CDN integration

### 3. canvaxservice —— gRPC Real-time Collaboration Whiteboard

canvaxservice implements a multi-user real-time collaboration whiteboard based on gRPC bidirectional streaming, providing low-latency drawing event synchronization and session management.

**Core Features:**
- **Collaborate**: Bidirectional streaming collaboration, real-time drawing event synchronization
- **Session Management**: Automatic client connection and drawing history management
- **Event Broadcasting**: Efficient drawing event broadcast mechanism

**Technical Characteristics:**
- In-memory drawing history management (up to 1000 events)
- Automatic cleanup of expired connections and sessions
- Supports drawing event types: ping, clear, draw, etc.

### 4. canvaservice —— WebTransport Real-time Collaboration Whiteboard

canvaservice provides browser-oriented real-time collaboration whiteboard services based on WebTransport/HTTP3 and WebSocket protocols.

**Core Features:**
- **WebTransport Support**: Low-latency bidirectional communication based on HTTP/3
- **WebSocket Fallback**: Compatible with browsers that don't support WebTransport
- **Session Management**: Code-based session creation and joining mechanism
- **Auto Cleanup**: 10-minute inactivity automatic session cleanup

**Technical Characteristics:**
- Dual protocol support: WebTransport (priority) + WebSocket (fallback)
- Session-level client management
- In-memory drawing history management
- Supports various drawing event types

---

## Development & Deployment

### Development Environment

- **Go 1.24+**: Core development language
- **just**: Project build and task management tool
- **buf**: Protocol Buffers code generation
- **MinIO**: Object storage service (optional)
- **Dragonfly/Redis**: Metadata storage service (optional)

### Quick Start

1. **Clone the project**
   ```bash
   git clone <repository-url>
   cd fawa
   ```

2. **Build services**
   ```bash
   just build fileservice    # Build file service
   just build greetservice   # Build greeting service
   just build canvaxservice  # Build gRPC whiteboard service
   just build canvaservice   # Build WebTransport whiteboard service
   ```

3. **Run services**
   ```bash
   just run fileservice      # Run file service
   just run greetservice     # Run greeting service
   just run canvaxservice    # Run gRPC whiteboard service
   just run canvaservice     # Run WebTransport whiteboard service
   ```

### Performance Testing

The project includes k6 performance testing scripts supporting high-concurrency load testing:

```bash
k6 run k6_test/script.js
```

Test configuration:
- 200 concurrent users
- 3-minute continuous load testing
- 95% request latency < 500ms
- 99% request latency < 1500ms

### Kubernetes Deployment

The project provides complete K8s deployment configurations:

```bash
# Deploy to Kubernetes
kubectl apply -f deploy/base/
```

Supported features:
- Automatic certificate management (cert-manager)
- Service discovery and load balancing
- Multi-environment configuration management
- Auto-scaling

---

## Project Structure

```
fawa/
├── fileservice/          # File transfer service
│   ├── storage/         # Storage abstraction layer
│   ├── handler/         # gRPC handlers
│   └── proto/           # Protocol Buffers definitions
├── greetservice/        # Greeting demo service
├── canvaxservice/       # gRPC whiteboard service
├── canvaservice/        # WebTransport whiteboard service
├── deploy/              # Kubernetes deployment configs
├── k6_test/            # Performance testing scripts
└── justfile            # Project task management
```

---

## Open Source License

- Licensed under Apache 2.0 License
- Welcome any form of contributions and suggestions
- PRs must pass lint, test, and license checks

---

> fawa, making distributed collaboration and data flow elegant as poetry.
