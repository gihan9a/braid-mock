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
- **Configuration file** - Simplified startup with YAML configuration

## Installation

### Prerequisites

- Go 1.18 or higher

### Building from source

```bash
# Clone the repository
git clone https://github.com/gihan9a/braid-mock.git
cd braid-mock

# Build the server
go build -o braid-mock ./cmd/server

# Generate a default configuration file
./braid-mock -generate-config

# Run the server (will use config.yml by default)
./braid-mock
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

## Configuration

### Configuration File

The server uses a YAML configuration file (`config.yml` by default) with the following structure:

```yaml
server:
  port: 3000                 # Server port
  root_dir: "./mock-data"    # Directory containing .braid files

proxy:
  url: "http://api.example.com"  # URL to proxy requests to when mocks don't exist
  insecure_verify: false     # Skip SSL certificate verification for proxy

tls:
  enabled: false             # Enable/disable TLS (HTTPS)
  cert_file: "cert/cert.pem" # Path to TLS certificate
  key_file: "cert/key.pem"   # Path to TLS key
  generate_cert: false       # Auto-generate self-signed certificate

cors:
  enabled: true              # Enable/disable CORS support
  allow_origins: "*"         # Allowed origins
  allow_methods: "GET, POST, PUT, DELETE, OPTIONS, PATCH"  # Allowed HTTP methods
  allow_headers: "Content-Type, Authorization, Subscribe, Version, Parents"  # Allowed headers
  allow_credentials: false   # Allow credentials
  max_age: 86400            # Max age for preflight requests
```

### Generating a Default Configuration

```bash
# Generate default config in the default location (config.yml)
./braid-mock -generate-config

# Generate config in a custom location
./braid-mock -generate-config -config-path my-custom-config.yml
```

## Usage

### Starting the Server

```bash
# Using the default config.yml file
./braid-mock

# Using a custom configuration file
./braid-mock -config my-custom-config.yml

# Override config file settings
./braid-mock -p 8080 -d ./other-mock-dir
```

### Command Line Options

| Flag | Description | Default |
|------|-------------|---------|
| `-config <file>` | Path to configuration file | `config.yml` |
| `-generate-config` | Generate a default configuration file | `false` |
| `-config-path <path>` | Path where config file should be generated | `config.yml` |
| `-d <dir>` | Directory containing .braid mock files (overrides config) | (from config) |
| `-p <port>` | Port to listen on (overrides config) | (from config) |

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
braid-mock/
├── cmd/
│   └── server/           # Entry point
├── internal/
│   ├── config/           # Configuration handling
│   ├── server/           # Core server implementation
│   ├── tls/              # TLS certificate handling
│   └── utils/            # Utility functions
├── pkg/
│   └── braidproto/       # Braid protocol types
├── mock-data/            # Default directory for .braid files
├── config.yml            # Configuration file
```

## License

MIT