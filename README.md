# Distributed Tracing Demo

Two service distributed tracing POC using OpenTelemetry. Service A (Python/LangGraph) starts a trace, calls Service B (TypeScript/Express), and the `traceparent` header propagates automatically via OTEL auto-instrumentation.

## Architecture

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

OTEL Collector (:4318)
  ├── Galileo (otlphttp)
  └── Debug (stdout)
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

## Project Structure

```
distributed-tracing-demo/
├── python-service/          # Service A: Python/LangGraph
│   ├── app.py               # FastAPI + LangGraph graph
│   ├── tracing.py           # OTEL SDK setup
│   ├── requirements.txt
│   └── Dockerfile
├── ts-service/              # Service B: TypeScript/Express
│   ├── src/
│   │   ├── server.ts        # Express HTTP server
│   │   ├── tracing.ts       # OTEL SDK + HTTP/Express instrumentation
│   │   ├── agent.ts         # Agent logic (LLM calls, tools)
│   │   └── mock-llm.ts      # Simulated LLM responses
│   ├── package.json
│   ├── tsconfig.json
│   └── Dockerfile
├── docker-compose.yml
├── otel-collector-config.yaml
└── .env.example
```
