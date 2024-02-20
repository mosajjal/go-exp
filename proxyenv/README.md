# HTTP Proxy Configured by Environment Variables

This is a simple HTTP proxy server written in Go. It is fully configured by environment variables, making it easy to deploy in different environments without changing the code.

## Dependencies

This project uses the following libraries:

- `github.com/armon/go-radix`: A radix tree implementation.
- `github.com/elazarl/goproxy`: A versatile HTTP proxy.
- `github.com/elazarl/goproxy/ext/auth`: An extension for HTTP Basic authentication.
- `github.com/sethvargo/go-envconfig`: A library for managing configuration data from environment variables.

## Configuration

The server's configuration is defined in the Config struct:

- `LISTEN`: The address and port the server should listen on.
- `AUTH`: The credentials for HTTP Basic authentication.
- `ALLOWED`: A semicolon-separated list of domain suffixes that the server will allow.

## Functionality

The server uses a radix tree to store the allowed domain suffixes. The domain names are reversed before being inserted into the tree to make them suffixes.

## Running the Server

To run the server, simply set the appropriate environment variables and start the program:

```bash
export LISTEN=":8080"
export AUTH="user:password"
export ALLOWED="example.com;example.org"
go run main.go
```

This will start the server listening on port 8080, with HTTP Basic authentication requiring the username "user" and password "password", and allowing requests to any subdomain of example.com or example.org.