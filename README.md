# Braid Protocol Mock Server

A file-based mock server for the [Braid Protocol](https://braid.org/) that allows you to simulate API responses and real-time updates by editing files.

## Features

- **File-based mocking system** - Create `.braid` files that correspond to API endpoints
- **Live updates** - Edit files to trigger real-time updates to connected clients
- **Braid protocol support** - Implements versioning, subscriptions, and other core Braid features
- **JSON patch optimization** - Sends only the differences between states for bandwidth efficiency
- **Proxy mode** - Forwards requests to a real backend when mock files aren't found
- **TLS support** - Secure your mock server with HTTPS and auto-generated self-signed certificates
- **CORS support** - Allow cross-origin requests from web applications

## Installation

### Prerequisites

- Go 1.18 or higher

### Building from source

```bash
# Clone the repository
git clone https://github.com/yourusername/braid-mock-server.git
cd braid-mock-server

# Build the server
go build -o braid-mock-server ./cmd/server

# Run the server
./braid-mock-server
```

## How It Works

1. Create `.braid` files in a directory structure that mirrors your API endpoints
2. Start the mock server pointing to this directory
3. Connect clients to the server using the Braid protocol
4. Edit the `.braid` files to simulate updates - changes are automatically pushed to connected clients

## File Structure

The mock server uses `.braid` files to simulate API responses:

```
mock-data/
├── user/
│   ├── me.braid             # Endpoint: /user/me
│   └── settings.braid       # Endpoint: /user/settings
├── products/
│   ├── 123.braid            # Endpoint: /products/123
│   └── featured.braid       # Endpoint: /products/featured
```

## Usage

### Starting the Server

```bash
# Basic usage with default settings
./braid-mock-server

# Specify mock data directory
./braid-mock-server -d ./mock-data

# Run on a different port
./braid-mock-server -p 8080

# Enable TLS with auto-generated certificate
./braid-mock-server -tls -gen-cert

# Proxy mode - forward requests to a real server when mocks don't exist
./braid-mock-server -proxy http://api.example.com:8080

# Proxy with insecure SSL (ignore certificate errors)
./braid-mock-server -proxy https://api.example.com -insecure-proxy

# Enable CORS for web applications
./braid-mock-server -cors

# Complete example with all features
./braid-mock-server -d ./mock-data -p 8443 -tls -gen-cert -proxy https://api.example.com -insecure-proxy -cors
```

### Command Line Options

| Flag | Description | Default |
|------|-------------|---------|
| `-d <dir>` | Directory containing .braid mock files | `.` (current directory) |
| `-p <port>` | Port to listen on | `3000` |
| `-proxy <url>` | URL to proxy requests to when mock files aren't found | (none) |
| `-insecure-proxy` | Skip SSL certificate verification when proxying requests | `false` |
| **TLS Options:** | | |
| `-tls` | Enable TLS (HTTPS) | `false` |
| `-cert <file>` | Path to TLS certificate file | `cert/cert.pem` |
| `-key <file>` | Path to TLS private key file | `cert/key.pem` |
| `-gen-cert` | Generate a self-signed certificate if none exists | `false` |
| **CORS Options:** | | |
| `-cors` | Enable CORS support | `false` |
| `-cors-origins` | Comma-separated list of allowed origins | `*` |
| `-cors-methods` | Comma-separated list of allowed HTTP methods | `GET, POST, PUT, DELETE, OPTIONS, PATCH` |
| `-cors-headers` | Comma-separated list of allowed HTTP headers | `Content-Type, Authorization, Subscribe, Version, Parents` |
| `-cors-credentials` | Allow credentials (cookies, auth headers) | `false` |
| `-cors-max-age` | Max age for CORS preflight requests (seconds) | `86400` (24 hours) |

## Connecting with curl

Test the server with curl:

```bash
# Regular GET request
curl http://localhost:3000/user/me

# Subscribe to updates
curl -H "Subscribe: true" -H "Accept: application/json" http://localhost:3000/user/me

# With TLS (using -k to accept self-signed certificate)
curl -k -H "Subscribe: true" https://localhost:3000/user/me
```

## Braid Protocol Support

This mock server implements these Braid protocol features:

1. **Versioning** - Resources are versioned with CRC32 hashes
2. **Subscriptions** - Subscribe to resource changes with the `Subscribe: true` header
3. **JSON Patches** - Changes are sent as efficient JSON patches when possible
4. **Headers** - Correct Braid protocol headers for versioning and content types

## Project Structure

```
braid-mock-server/
├── cmd/
│   └── server/           # Entry point
├── internal/
│   ├── config/           # Configuration handling
│   ├── server/           # Core server implementation
│   ├── tls/              # TLS certificate handling
│   └── utils/            # Utility functions
├── pkg/
│   └── braidproto/       # Braid protocol types
```

## License

MIT