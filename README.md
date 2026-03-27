# Distributed Tracing Demo with Custom Galileo OTel Collector

Two-service distributed tracing POC using OpenTelemetry. Service A (Python/LangGraph) starts a trace, calls Service B (TypeScript/Express), and the `traceparent` header propagates automatically via OTel auto-instrumentation.

Includes a **custom OTel Collector distribution** with Galileo-specific components that enable vendor-agnostic application code — services only use standard OTel attributes, and the Collector handles all Galileo routing and auth.

## Architecture

### Standard Mode (stock OTel Collector)

```
User → POST /ask (:8000)
         ↓
  Service A (Python/LangGraph)
    Auto-instruments: FastAPI + httpx
         ↓ traceparent header (automatic)
  Service B (TypeScript/Express :3000)
    Auto-instruments: HTTP + Express
         ↓
    Agent logic (invoke_agent → chat → tool → chat)

OTel Collector (:4318)
  ├── Galileo (otlphttp)  ← project/logstream in headers
  └── Debug (stdout)
```

### Custom Collector Mode (vendor-agnostic routing)

```
User → POST /ask (:8000)
         ↓
  Service A (Python)                Service B (TypeScript)
    service.name: "python-..."        service.name: "otel-ts-..."
    (pure OTel — no Galileo imports)  (pure OTel — no Galileo imports)
         ↓                                ↓
         └──────── OTLP ─────────────────┘
                      ↓
  Galileo OTel Collector (:4318)
    ├── galileo_router processor
    │     Maps service.name → galileo.project.name / galileo.logstream.name
    ├── batch processor
    ├── galileo exporter  ← only place Galileo API key lives
    └── debug exporter (stdout)
```

## Quick Start

```bash
# Copy .env.example to .env and fill in your credentials
cp .env.example .env

# starts all services - the collector, the Python service, and the TS/express
docker compose up -d

# Send a request
curl -X POST http://localhost:8000/ask \
  -H 'Content-Type: application/json' \
  -d '{"question":"Find me a good restaurant"}'
```

Check Galileo for the distributed trace — one trace spanning both services.

## Local Development

```bash
# Start just the collector
docker compose up -d otel-collector

# Terminal 1: Service B (TypeScript)
cd ts-service && npm install && npm start

# Terminal 2: Service A (Python)
cd python-service && python3 -m venv .venv && source .venv/bin/activate
pip install -r requirements.txt
source ../.env && uvicorn app:app --port 8000

# Test
curl -X POST http://localhost:8000/ask \
  -H 'Content-Type: application/json' \
  -d '{"question":"Find me a good restaurant"}'
```

## How It Works

1. Service A receives `POST /ask` — FastAPI auto-instrumentation creates a server span
2. LangGraph graph executes — `call_ts_service` node calls Service B via httpx
3. `opentelemetry-instrumentation-httpx` auto-injects `traceparent` header into the outgoing request
4. Service B receives the request — `@opentelemetry/instrumentation-http` auto-extracts the `traceparent`
5. All spans in Service B inherit the same `trace_id`
6. Both services export spans to OTEL Collector → Galileo reconstructs the full trace tree

## Custom Galileo Collector

The `collector/` directory contains two custom OTel Collector components and a build system that produces a standalone Collector binary.

### Components

**`galileo_router` processor** — Maps standard OTel resource attributes to Galileo project/logstream routing. Application code stays vendor-agnostic.

```yaml
processors:
  galileo_router:
    routes:
      - match:
          service.name: "my-agent"
        project: "team-alpha"
        logstream: "production"
    default_route:
      project_from_attribute: "service.name"
      logstream_from_attribute: "deployment.environment"
      logstream_default: "default"
```

**`galileo` exporter** — Sends traces to Galileo's OTLP endpoint. Handles auth, gzip compression, and retry.

```yaml
exporters:
  galileo:
    api_key: "${GALILEO_API_KEY}"
    endpoint: "https://app.galileo.ai"  # default, or your self-hosted URL
```

### Build

```bash
# Prerequisites: Go 1.25+, OTel Collector Builder (ocb)
go install go.opentelemetry.io/collector/cmd/builder@v0.148.0

# Build the custom collector binary
cd collector && make build

# Run tests
make test

# Verify components are registered
./dist/output/galileo-otel-collector components
```

### Run with Docker Compose

```bash
# Uses the custom Galileo Collector instead of stock otel-collector-contrib
docker compose -f docker-compose-custom.yml up -d

# Send a request
curl -X POST http://localhost:8000/ask \
  -H 'Content-Type: application/json' \
  -d '{"question":"Find me a good restaurant"}'
```

### Why This Matters

Without the custom Collector, Galileo-specific concepts leak into application code or Collector headers:

| Concern | Standard OTel Collector | Custom Galileo Collector |
|---------|------------------------|------------------------|
| **Project routing** | `project:` header on exporter | `galileo_router` derives from `service.name` |
| **Logstream routing** | `logstream:` header on exporter | `galileo_router` derives from `deployment.environment` |
| **API key** | `Galileo-API-Key` header in app or exporter config | `galileo` exporter `api_key` field (masked in logs) |
| **Endpoint URL** | Manual `traces_endpoint` construction | `galileo` exporter auto-constructs from `endpoint` |
| **Multi-service routing** | One project/logstream per Collector | Per-span routing from resource attributes |

## Project Structure

```
distributed-tracing-demo/
├── python-service/                  # Service A: Python/LangGraph
│   ├── app.py                       # FastAPI + LangGraph graph
│   ├── tracing.py                   # OTel SDK setup (vendor-agnostic)
│   ├── requirements.txt
│   └── Dockerfile
├── ts-service/                      # Service B: TypeScript/Express
│   ├── src/
│   │   ├── server.ts                # Express HTTP server
│   │   ├── tracing.ts               # OTel SDK + HTTP/Express instrumentation
│   │   ├── agent.ts                 # Agent logic (LLM calls, tools)
│   │   └── mock-llm.ts             # Simulated LLM responses
│   ├── package.json
│   ├── tsconfig.json
│   └── Dockerfile
├── collector/                       # Custom Galileo OTel Collector
│   ├── processor/
│   │   └── galileorouterprocessor/  # Maps OTel attrs → Galileo project/logstream
│   ├── exporter/
│   │   └── galileoexporter/         # Sends traces to Galileo OTLP endpoint
│   ├── dist/
│   │   └── builder-config.yaml      # OCB manifest for custom distribution
│   ├── Makefile
│   └── Dockerfile
├── docker-compose.yml               # Standard: stock otel-collector-contrib
├── docker-compose-custom.yml        # Custom: Galileo Collector with routing
├── otel-collector-config.yaml       # Config for standard Collector
├── otel-collector-config-custom.yaml # Config for custom Galileo Collector
└── .env.example
```
