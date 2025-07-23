# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Sistema de aprobación de compras Amazon usando Temporal.io. Es un workflow de larga duración que orquesta validación de productos, flujo de aprobación y compra automática.

**Tech Stack**: Go + Temporal.io + Docker + HTML/JS simple

## Common Development Commands

### Environment Setup
```bash
make temporal-up    # Start Temporal server (Docker)
make worker        # Start Temporal worker
make web          # Start web server (port 8081)
make temporal-down # Stop Temporal server
```

### Development Workflow
```bash
make deps         # Download Go dependencies  
make dev          # Start full development environment
make test         # Run tests
make clean        # Clean Docker containers
```

### Temporal CLI Commands
```bash
make workflow-list                    # List all workflows
make workflow-show ID=workflow-id     # Show specific workflow
temporal workflow list --address localhost:7233
```

## Architecture

### Core Components
- **PurchaseApprovalWorkflow**: Main workflow orchestrating the entire process
- **Amazon Activities**: Product validation and purchase execution
- **Approval Activities**: Notification system and approval logic
- **Web Interface**: Employee and responsible interfaces

### Key Files
- `internal/workflows/purchase_approval.go`: Main workflow definition
- `internal/activities/amazon.go`: Amazon integration activities
- `internal/activities/approval.go`: Approval flow activities  
- `internal/models/purchase.go`: Data structures
- `cmd/worker/main.go`: Temporal worker
- `cmd/web/main.go`: Web server

### Workflow Patterns Used
- **Long-running workflows**: Processes can take days (approval timeouts)
- **Signals**: For approval responses from responsibles
- **Queries**: For real-time status checking
- **Activities with retry**: For external API calls
- **Timeouts**: 7-day approval deadline

### Data Flow
1. Employee submits request via web form
2. Workflow validates Amazon products
3. Sends approval requests to responsibles  
4. Waits for approval signals or timeout
5. Executes Amazon purchase if approved
6. Notifies all parties of completion

### Task Queue
All components use: `"purchase-approval-task-queue"`

### Testing URLs
Use these Amazon product URLs for testing:
- `https://amazon.es/dp/B08N5WRWNW` (Echo Dot - valid)
- `https://amazon.es/dp/B07XJ8C8F5` (Fire TV Stick - valid)
- `https://amazon.es/dp/PROHIBITED1` (Prohibited product)

## Development Guidelines

### Temporal Workflow Rules
- Workflows must be deterministic
- Use `workflow.Sleep()` not `time.Sleep()`
- Activities handle all external interactions
- Always set timeouts for activities

### Code Organization
- Activities are grouped by domain (Amazon, Approval)
- Models define shared data structures
- Each activity has retry policies configured
- Workflows use signals for external input

### Testing Strategy
- Use Temporal's test environment for workflow testing
- Activities can be unit tested independently
- Integration testing via web interface
- Monitor via Temporal UI (localhost:8080)

### Logging & Debugging
- Worker logs show activity execution
- Temporal UI provides full workflow history
- Web interface auto-refreshes status
- Use workflow queries for real-time state

### Configuration
- Approval thresholds in `activities/approval.go`
- Prohibited products list in `activities/amazon.go`
- Timeout values in workflow configuration

## Multi-Language Architecture Principles

### 🌍 Language Agnostic Design
This project follows **best-tool-for-the-job** philosophy with clear service boundaries:

**Current Stack**:
- **Workflows & Core Logic**: Go + Temporal.io (performance, concurrency)
- **Web Interface**: Go templates (rapid prototyping)
- **Infrastructure**: Docker + Make (simplicity)

**Evolution Path**:
```
Phase 1: Go + Temporal (✅ current)
Phase 2: Modern Frontend + Go Backend
Phase 3: ML/Analytics Services  
Phase 4: Enterprise Integrations
```

### 📋 Architecture Rules

**1. API-First Design**
- All services communicate via REST/HTTP
- OpenAPI specs for all endpoints
- No direct database sharing between services
- Clear service boundaries with contracts

**2. Language Selection Guidelines**
- **Go**: Workflows, concurrent APIs, infrastructure tools
- **Python**: ML/AI, data science, rapid integration scripts  
- **JavaScript/TypeScript**: Frontend, Node.js for rapid prototypes
- **Java**: Enterprise integrations, legacy system connectors
- **SQL**: Data analysis, reporting, complex queries

**3. Frontend Framework Options**
Choose based on team expertise and requirements:
- **Vue 3**: Excellent DX, gradual adoption, smaller bundle
- **Svelte/SvelteKit**: Superior performance, minimal JS
- **React**: Largest ecosystem, enterprise adoption
- **Solid**: Best performance, React-like syntax
- **Alpine.js**: Progressive enhancement, minimal

**4. Communication Standards**
- **REST APIs**: Standard CRUD operations
- **GraphQL**: Complex frontend data needs
- **gRPC**: High-performance service-to-service
- **WebSockets**: Real-time updates
- **Events**: Async workflows (Temporal, Kafka, Redis)

**5. Data & State Management**
- **Temporal**: Workflow state, long-running processes
- **PostgreSQL**: Transactional data, user management
- **Redis**: Caching, sessions, real-time data
- **S3/MinIO**: File storage, backups
- **ClickHouse/TimescaleDB**: Analytics, time-series

### 🛠 Technology Decision Matrix

| Use Case | Primary Choice | Alternative | Why |
|----------|---------------|-------------|-----|
| **Workflows** | Go + Temporal | Java + Temporal | Performance, determinism |
| **Frontend** | Vue 3 | Svelte, React | DX, performance, adoption |
| **API Gateway** | Go | TypeScript | Concurrency, single binary |
| **ML/AI** | Python | R, Julia | Ecosystem, libraries |
| **Mobile** | Flutter | React Native | Cross-platform, performance |
| **DevOps** | Go, Shell | Python | Binary distribution, speed |
| **Reporting** | Python + SQL | R, Jupyter | Data science ecosystem |

### 🔄 Migration Strategy

**Service Extraction Pattern**:
1. Start with monolithic Go service
2. Identify bounded contexts 
3. Extract services as separate deployments
4. Maintain API contracts
5. Independent scaling and technology choices

**Example Evolution**:
```
// Current: Single Go binary
go run cmd/web/main.go

// Phase 2: Frontend + API separation  
npm run dev        # Vue 3 frontend
go run cmd/api     # Go API service

// Phase 3: Microservices
docker compose up  # Multiple services
```

### 📊 Monitoring & Observability

**Language-Agnostic Standards**:
- **Logs**: JSON structured format
- **Metrics**: Prometheus/OpenTelemetry 
- **Tracing**: Jaeger distributed tracing
- **Health**: Standard /health endpoints
- **Documentation**: OpenAPI + service catalogs

## Frontend Framework Comparison

### Vue 3 ✅ Recommended
```bash
npm create vue@latest frontend
# Excellent DX, composition API, TypeScript support
# Smaller learning curve, great documentation
```

### Svelte/SvelteKit ✅ High Performance  
```bash
npm create svelte@latest frontend
# Best performance, minimal runtime
# Compiled framework, excellent DX
```

### React (if team expertise exists)
```bash
npx create-react-app frontend --template typescript
# Largest ecosystem, more developers
# More complex state management
```

### Alpine.js (progressive enhancement)
```html
<!-- Add to existing Go templates -->
<script defer src="https://cdn.jsdelivr.net/npm/alpinejs@3.x.x/dist/cdn.min.js"></script>
```

**Choose based on**:
- **Team expertise** 
- **Performance requirements**
- **Ecosystem needs**
- **Long-term maintenance**

## Services & Ports
- Temporal Server: localhost:7233
- Temporal UI: localhost:8082  
- Web Application: localhost:8081
- PostgreSQL: localhost:5432
- Elasticsearch: localhost:9200