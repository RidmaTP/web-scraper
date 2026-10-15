# Web Analyzer

Web Analyzer is a Go backend that renders and analyzes public web pages with
Chromium. It extracts page metadata, classifies links, checks link status, and
streams results through Server-Sent Events (SSE).

This repository contains the backend and observability infrastructure. A
frontend is not included; browser clients can consume the SSE API directly.

## Technology

### Backend

- Go 1.26
- Gin: <https://github.com/gin-gonic/gin>
- Rod and Chromium: <https://github.com/go-rod/rod>
- In-memory result cache: <https://github.com/patrickmn/go-cache>
- Structured JSON logging with Go `log/slog`
- IP rate limiting with `golang.org/x/time/rate`

### DevOps and observability

- Docker Compose
- OpenTelemetry Collector
- Prometheus metrics
- Tempo traces
- Loki and Promtail centralized logs
- Grafana dashboards and Explore

<img width="1573" height="867" alt="image" src="https://github.com/user-attachments/assets/b2267ac4-a586-410e-934b-5c21a39dc691" />

<img width="1572" height="942" alt="image" src="https://github.com/user-attachments/assets/6e70d4bd-58f3-4aa7-b866-3a517c6d4465" />

<img width="1562" height="847" alt="image" src="https://github.com/user-attachments/assets/a4b388c9-6635-4d02-bfcc-2e17faef5b1b" />


## Prerequisites

For Docker Compose:

- Docker Engine or Docker Desktop with Compose v2
- At least 4 GB allocated to Docker Desktop is recommended
- Docker Desktop must allow the project directory, Docker socket, and container
  log directory to be shared with Linux containers

For direct execution:

- Go 1.26
- Chromium or Google Chrome
- Network access for public pages and Go modules

## Run the project

### Docker Compose

From the repository root:

```bash
docker compose up --build
```

To start the application and its telemetry dependencies:

```bash
docker compose up --build web-scraper-app
```

The Compose service is `web-scraper-app`; its container is `web-scraper`.

Stop the stack:

```bash
docker compose down
```

Remove persisted Loki data and Promtail positions:

```bash
docker compose down -v
```

If Tempo or Loki exits with status 137, increase Docker Desktop memory.

### Run directly with Go

```bash
go mod download
go run ./cmd/server
```

Set `CHROME_PATH` to the local Chromium executable first. Direct execution
starts only the backend, not the Docker observability services.

## Service URLs

- API: <http://localhost:8000>
- Grafana: <http://localhost:3000>

Prometheus and OTLP ports are assigned dynamically by the current Compose
configuration. Use `docker compose port prometheus 9090` to find the host port.
Grafana provisions Prometheus, Tempo, and Loki automatically.

## API

### Health

```http
GET /api/v1/health
```

Example response:

```json
{
  "message": "service up and running",
  "status": "success",
  "version": "v1.1"
}
```

### Analyze a page

```http
GET /api/v1/result?url=<public-http-or-https-url>
Accept: text/event-stream
X-Request-ID: <client-request-id>
```

Example:

```bash
curl --no-buffer \
  -H "Accept: text/event-stream" \
  -H "X-Request-ID: demo-request-001" \
  "http://localhost:8000/api/v1/result?url=https%3A%2F%2Fexample.com"
```

The endpoint streams `data:` events containing analysis results such as:

- HTML version and page title
- Header count
- Internal and external links
- Active and inactive link counts
- Login-page indicators

Invalid URLs return JSON with HTTP 400, for example:

```json
{
  "error": "url destination is not allowed",
  "status_code": 400
}
```

## Application behavior

- **Validation:** only HTTP/HTTPS URLs with valid hosts are accepted.
- **SSRF protection:** loopback, private, link-local, multicast, reserved,
  documentation, and cloud-metadata destinations are blocked before browsing.
- **Streaming:** results are sent incrementally over SSE.
- **Caching:** completed results are cached in memory for two hours.
- **Rate limiting:** each client IP receives a token bucket with rate 10 and
  burst 20.
- **Logging:** `X-Request-ID` is stored in context and included in structured
  request logs.
- **CORS:** `X-Request-ID` is allowed and exposed.
- **Tracing:** OpenTelemetry exports traces through the Collector to Tempo.
- **Metrics:** OpenTelemetry exports metrics for Prometheus scraping.
- **Centralized logs:** Promtail forwards Docker logs to Loki for Grafana
  Explore.
- **Authentication:** authentication and authorization are not currently
  implemented. Use an authenticated gateway for public deployments.

## Testing

Run all tests:

```bash
go test ./...
```

Run coverage for the main tested packages:

```bash
go test ./internal/analysis ./internal/application ./internal/browser ./internal/configs ./internal/utils -cover
```

## Challenges and solutions

- **Large pages:** SSE prevents users from waiting for the complete crawl.
- **Many links:** worker-based processing handles link checks efficiently.
- **JavaScript pages:** Rod and Chromium render pages before extraction.
- **SSRF risk:** the initial destination is resolved and checked against
  blocked address ranges.
- **Troubleshooting:** request IDs, structured logs, traces, metrics, Loki,
  Tempo, Prometheus, and Grafana provide cross-service observability.

## Possible improvements

1. Add authentication.
2. Use a shared cache for horizontally scaled deployments.
3. Apply more SSRF checks for initial URL validation and add SSRF checks to the discovered links.
4. Add readiness/liveness endpoints.
5. Improving grafana dashbored.
