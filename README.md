# Braid Protocol Mock Server

A simple file-based mock server for the [Braid Protocol](https://braid.org/) that allows you to simulate API responses and real-time updates by editing files.

## Features

- File-based mocking system with live updates
- Implements core Braid protocol features (versioning, subscriptions)
- Watches for file changes and pushes updates to subscribed clients
- Simple to set up and use

## Installation

```bash
# Clone the repository
git clone https://github.com/gihan9a/braid-mock-server.git
cd braid-mock-server

# Build the server
go build -o server
```

## How It Works

1. Create `.braid` files in a directory structure that mirrors your API endpoints
2. Start the mock server pointing to this directory
3. Connect clients to the server using the Braid protocol
4. Edit the `.braid` files to simulate updates - changes are automatically pushed to connected clients

## File Structure

The mock server uses `.braid` files to simulate API responses:

- File name represents the API endpoint
- File content is JSON that will be returned as the response body

For example:

```
mock-data/
├── user/
│   ├── me-test.braid         # Endpoint: /user/me-test
│   └── settings.braid        # Endpoint: /user/settings
├── products/
│   ├── 123.braid             # Endpoint: /products/123
│   └── featured.braid        # Endpoint: /products/featured
```

## Usage

### Starting the Server

```bash
# Start server using the current directory
./server

# Start server with a specific directory
./server -d ./mocks

# Start server on a different port
./server -p 8080
```

### Simulating Updates

To simulate updates, simply edit and save the `.braid` file. The server will detect the change and send the updated content to all subscribed clients.

## Braid Protocol Implementation

This mock server implements these Braid protocol features:

1. **Versioning** - Each resource has a unique version ID that changes when the resource is updated
2. **Subscriptions** - Clients can subscribe to resource changes with the `Subscribe: true` header
3. **Real-time Updates** - Changes to `.braid` files are immediately pushed to subscribed clients

## Example Workflow

1. Create file `user/me-test.braid` with some initial JSON data
2. Start the mock server: `./braid-mock-server`
4. Edit the `user/me-test.braid` file to change some values
5. Save the file - you should see the update appear in the client's output

## Command Line Options

### Server

- `-d <directory>`: Directory containing `.braid` mock files (default: current directory)
- `-p <port>`: Port to listen on (default: 3000)

## License

MIT