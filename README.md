# fawa

Fawa is a high-performance file transfer service built with Go. It leverages gRPC and Connect for efficient, bidirectional streaming of files.

**Core Features:**

*   **File Upload:** Stream files from a client to the server.
*   **File Download:** Stream files from the server to a client.
*   **Built with Go:** Ensures high performance and concurrency.
*   **gRPC & Connect:** Provides a modern, robust, and efficient communication layer.


## Development Commands

This project uses [`just`](https://github.com/casey/just) to manage development commands. Please install it first by following the [official instructions](https://github.com/casey/just#installation).

  * Run `just` to list all available commands.
  * Run `just <command>` to execute a specific task.

### Available Commands:

  - `build` - Builds the binary
  - `run` - Runs the application
  - `test` - Runs unit tests
  - `lint` - Lints the code
  - `tidy` - Tidies Go modules
  - `fmt` - Formats the code
  - `clean` - Cleans up build artifacts
  -  `generate` - Generate gen files
