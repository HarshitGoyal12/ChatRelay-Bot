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


# Configuration(Locally)

## 📦 Prerequisites

Before setting up the local development environment, ensure you have the following installed:

- **Go 1.20 or later**: Required for building and running the application.
- **Slack Workspace**: Administrative access to create and configure a Slack App.
- **Git**: For cloning and managing the repository.
- **Text Editor / IDE**: For editing configuration and source files.

### 🔍 Optional (for full observability setup):

- **OpenTelemetry Collector**: For receiving and processing telemetry data locally.
- **Docker**: For running observability backends like **Jaeger**, **Prometheus**, or **Grafana**.

## 🛠️ Slack App Configuration

### 🔧 Creating the Slack App

The **ChatRelay Bot** requires a Slack App with specific permissions and **Socket Mode** enabled. Follow these steps to set it up:

1. **Navigate to Slack API**:  
   Visit [api.slack.com/apps](https://api.slack.com/apps) and click **"Create New App"**.

2. **Choose Manifest Option**:  
   Select **"From an app manifest"** for streamlined setup.

3. **Apply Configuration**:  
   Use the following app manifest (replace `YOUR_BOT_NAME` and `YOUR_URL` as needed):
   
   ```yaml
   _metadata:
     major_version: 1
     minor_version: 1
   display_information:
     name: ChatRelay Bot
     description: A Slack bot that relays messages to a backend and streams the response
     background_color: "#2eb67d"
   features:
     bot_user:
       display_name: ChatRelay
       always_online: true
   oauth_config:
     scopes:
       bot:
         - app_mentions:read
         - chat:write
         - chat:write.public
         - channels:history
         - groups:history
         - im:history
         - mpim:history
   settings:
     event_subscriptions:
       request_url: ""
       bot_events:
         - app_mention
     interactivity:
       is_enabled: true
     socket_mode_enabled: true
     org_deploy_enabled: false
     token_rotation_enabled: false

## 🔐 Token Extraction

After creating the Slack App, extract the required tokens to enable API access and Socket Mode functionality:

- **App-Level Token**  
  - Navigate to **Basic Information** → **App-Level Tokens**.
  - Click **"Generate Token and Scopes"**.
  - Select the `connections:write` scope.
  - Copy the token that starts with `xapp-`.

- **Bot OAuth Token**  
  - Navigate to **OAuth & Permissions** in the sidebar.
  - Scroll down to **OAuth Tokens for Your Workspace**.
  - Copy the token that starts with `xoxb-`.

### 📁 Add the Tokens to Your `.env` File

```env
SLACK_BOT_TOKEN=xoxb-your-bot-token
SLACK_APP_TOKEN=xapp-your-app-token
``` 

## ⚙️ Configuration Overview

This document provides a comprehensive guide to configuring the **ChatRelay Bot** system, including:

- Environment variables
- Authentication tokens
- Network settings
- Operational parameters

It covers both required and optional configuration options, their sources, validation rules, and usage throughout the system.

📌 For detailed information about the data structures used to hold configuration, see **[Data Models](#)**.  
💻 For development-specific setup procedures, refer to **[Local Setup](#)**.

The ChatRelay Bot uses a hierarchical configuration system that prioritizes environment variables over .env file settings. Configuration is loaded during application startup and validated before the system begins operation.


![ChatRelay Bot Developemnt Mode](assets/app_starting.png)

## 📥 Configuration Sources

The system loads configuration from two sources, in order of precedence:

1. **Environment Variables** (highest priority)
2. **`.env` File** (fallback for development)

The `LoadConfig()` function performs the following steps:

- First attempts to load a `.env` file using `godotenv.Load()`.
- Then reads environment variables using `os.LookupEnv()`.

> ✅ **Note**: Environment variables always **override** values from the `.env` file.


![ChatRelay Bot Developemnt Mode](assets/env_variable.png)


# Required Configuration Parameters
These parameters must be set or the application will fail to start:

![ChatRelay Bot Developemnt Mode](assets/required_token.png)

The getRequiredEnv() helper function validates that these parameters are both present and non-empty. Missing required parameters cause LoadConfig() to return an error.

![ChatRelay Bot Developemnt Mode](assets/optional_config.png)

# Configuration Loading Process
The configuration loading process follows these steps:

![ChatRelay Bot Developemnt Mode](assets/optional_para.png)


The LoadConfig() function creates helper functions getEnv() and getRequiredEnv() to standardize parameter loading with proper error handling and default value application.


## ✅ Configuration Validation and Error Handling

The system performs validation during the configuration loading process to ensure stability and correctness. Key validations include:

- **Type Conversion and Validation**:
  - **Duration Parameters**: Parsed using `time.ParseDuration()`, with fallback to default values on parse errors.
  - **Integer Parameters**: Converted using `strconv.Atoi()` with proper error handling to prevent panics.
  - **String Parameters**: Used directly after checking for presence and non-empty values.


![ChatRelay Bot Developemnt Mode](assets/example_config.png)



## 🚀 Running the System

Once the prerequisites are met and the environment is configured, you can start the ChatRelay Bot locally:

