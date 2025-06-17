# 📝 Purpose and Scope

This document provides a comprehensive overview of the **ChatRelay Bot** system, a high-performance Slack bot implementation written in Go. The system demonstrates robust architectural patterns including comprehensive observability, concurrency handling, and enterprise-grade reliability features.

ChatRelay Bot serves as a relay service that receives user queries via Slack mentions, forwards them to configurable chat backend services, and streams responses back to users in real-time. The system includes a complete mock backend for development and testing scenarios.

For detailed information about specific components, see **Core Components**. For configuration details, see **Configuration**. For development setup and the mock backend, see **Development**.

---

# 🏗️ System Architecture

The ChatRelay Bot system follows a **modular architecture** with clear separation of concerns across multiple components:

![ChatRelay Bot High Level Design](assets/hld.png)


# Message Processing Flow

The system processes Slack mentions through a well-defined pipeline that demonstrates the integration between all major components:

![ChatRelay Bot High Level Design](assets/hld2.png)

# Technology Stack

ChatRelay Bot leverages a modern Go technology stack optimized for performance, observability, and reliability:


![ChatRelay Bot Technology Stack](assets/stack.png)

## 🧩 Key Dependencies

The system relies on several critical dependencies for its core functionality:

- **Slack SDK**: [`slack-go/slack` v0.17.1](https://github.com/slack-go/slack) — Provides Socket Mode WebSocket connectivity and Slack API integration.
- **OpenTelemetry**: Comprehensive observability stack including gRPC/HTTP exporters, host metrics, and runtime instrumentation.
- **Configuration Management**: [`joho/godotenv` v1.5.1](https://github.com/joho/godotenv) — Enables flexible, environment-based configuration management.


# Core Features
## Real-Time Event Processing

The system uses Slack's Socket Mode to establish persistent WebSocket connections, enabling real-time processing of app_mention events without requiring public HTTP endpoints. This approach simplifies deployment and enhances security by eliminating inbound connection requirements.

## Simulated Response Streaming
ChatRelay Bot implements an intelligent response streaming mechanism that enhances user experience by providing progressive updates. The system splits backend responses into logical chunks and updates Slack messages incrementally with visual indicators (...) to simulate real-time streaming.


## 📊 Enterprise-Grade Observability

Built-in **OpenTelemetry** integration provides comprehensive observability including:

- **Distributed Tracing**: End-to-end request tracking from Slack events to backend responses.
- **Structured Logging**: Correlated logs with trace context using Go's `slog` package.
- **Runtime Metrics**: Go-specific metrics including garbage collection and goroutine monitoring.
- **Host Metrics**: System-level tracking of CPU, memory, and disk utilization.


## 🛡️ Resilience and Error Handling

The system implements multiple layers of resilience to ensure high availability and fault tolerance:

- **Retry Mechanisms**: Configurable retry logic for both Slack API and backend communications.
- **Context Cancellation**: Proper request timeout and cancellation handling using Go's context propagation.
- **Graceful Error Recovery**: Intelligent error handling that maintains system stability and avoids cascading failures.


## Development Support
A complete mock backend service enables local development and testing without external dependencies. The mock service simulates realistic chat backend behavior including response delays and various response formats.


![ChatRelay Bot Developemnt Mode](assets/development.png)