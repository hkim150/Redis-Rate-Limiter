# Redis Rate Limiter

A high-performance distributed rate limiter using Redis that supports multiple rate limiting algorithms.

## Features

- **Multiple Rate Limiting Algorithms**:
  - Fixed Window: Simple counter-based rate limiting
  - Token Bucket: Smooth rate limiting with burst capacity
- **Distributed**: Uses Redis for coordination across multiple instances
- **Per-Client Rate Limiting**: Supports client identification via headers or IP
- **Health Checks**: Built-in health endpoint with Redis connectivity check
- **Graceful Shutdown**: Proper signal handling and graceful server shutdown
- **Request Logging**: Comprehensive HTTP request logging
- **Configurable**: Command-line configuration options
- **Docker Support**: Ready-to-use Docker configuration

## Quick Start

### Using Docker Compose

```bash
# Start the services
docker-compose up -d

# Test the rate limiter
curl http://localhost:8080/health
curl http://localhost:8080/fixed-window
curl http://localhost:8080/token-bucket
```

### Building from Source

```bash
# Build the application
go build -o redis-rate-limiter

# Run with default settings
./redis-rate-limiter

# Run with custom configuration
./redis-rate-limiter --port 8081 --redis-addr localhost:6379
```

## API Endpoints

### Health Check
```
GET /health
```
Returns server status and Redis connectivity.

**Response:**
- `200 OK`: Server is healthy and Redis is connected
- `503 Service Unavailable`: Redis connection failed

### Fixed Window Rate Limiting
```
GET /fixed-window
```
Implements fixed window rate limiting algorithm.

**Headers:**
- `X-Client-ID` (optional): Client identifier for rate limiting

**Response:**
- `200 OK`: Request allowed
- `429 Too Many Requests`: Rate limit exceeded

### Token Bucket Rate Limiting
```
GET /token-bucket
```
Implements token bucket rate limiting algorithm.

**Headers:**
- `X-Client-ID` (optional): Client identifier for rate limiting

**Response:**
- `200 OK`: Request allowed
- `429 Too Many Requests`: Rate limit exceeded

## Configuration

### Command Line Options

```bash
./redis-rate-limiter [flags]

Flags:
  -p, --port string            Port to run the server on (default "8080")
  -r, --redis-addr string      Redis server address (default "redis:6379")
      --redis-password string  Redis password (if required)
  -d, --redis-db int           Redis database number (default 0)
  -h, --help                   Help for redis-rate-limiter
```

### Default Rate Limiting Configuration

- **Fixed Window**:
  - Max Requests: 3 per window
  - Window Duration: 10 seconds

- **Token Bucket**:
  - Max Tokens: 5
  - Refill Rate: 1 token per second

## Architecture

### Fixed Window Algorithm
- Uses Redis INCR for atomic counter increments
- Sets TTL on first request to define the window
- Simple and efficient for basic rate limiting needs

### Token Bucket Algorithm
- Implemented using Lua scripts for atomic operations
- Supports burst traffic while maintaining average rate
- More sophisticated algorithm for smooth rate limiting

### Client Identification
The rate limiter identifies clients using:
1. `X-Client-ID` header (if provided)
2. Remote IP address (fallback)

This allows for flexible client identification strategies.

## Development

### Prerequisites
- Go 1.24+
- Redis server
- Docker (optional)

### Running Tests
```bash
go test ./...
```

### Building Docker Image
```bash
docker build -t redis-rate-limiter .
```

## Deployment

### Docker Compose
The included `compose.yaml` sets up:
- Redis server
- Two rate limiter instances (ports 8080, 8081)

### Kubernetes
For Kubernetes deployment, consider:
- Using Redis Cluster for high availability
- Configuring resource limits
- Setting up health check probes
- Using ConfigMaps for configuration

## Monitoring

The application provides structured logging for:
- HTTP requests (method, path, status, duration, client)
- Redis connection status
- Server lifecycle events
- Error conditions

Consider integrating with log aggregation systems like ELK stack or Prometheus for monitoring.

## Performance Considerations

- **Redis Connection Pooling**: The go-redis client handles connection pooling automatically
- **Lua Scripts**: Token bucket uses Lua scripts for atomic operations
- **Graceful Shutdown**: Ensures in-flight requests complete before shutdown
- **Timeouts**: Configured read/write/idle timeouts prevent resource leaks

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is open source and available under the MIT License.
