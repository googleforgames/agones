# Development Workflow Guide

This guide covers advanced development patterns, debugging, and workflow optimization for Agones development.

## Performance and Profiling Issues

### I want to use pprof to profile the controller.

Run `make build-images GO_BUILD_TAGS=profile` and this will build images with [pprof](https://golang.org/pkg/net/http/pprof/)
enabled in the controller, which you can then push and install on your cluster.

To get the pprof ui working, run `make controller-portforward PORT=6060` (or `minikube-controller-portforward PORT=6060` if you are on minikube),
which will setup the port forwarding to the pprof http endpoint.

To view CPU profiling, run `make pprof-cpu-web`, which will start the web interface with a CPU usage graph
on [http://localhost:6061](http://localhost:6061).

To view heap metrics, run `make pprof-heap-web`, which will start the web interface with a Heap usage graph.
on [http://localhost:6062](http://localhost:6062).

## Remote Debugging with Minikube

This section covers how to set up remote debugging for Agones services running in a Minikube cluster, allowing you to debug with breakpoints and step-through debugging from your IDE

### Overview

Remote debugging allows you to:
- Set breakpoints in Agones service code (allocator, controller, extensions, processor)
- Step through code execution in real-time
- Inspect variables and application state
- Debug issues that only occur in a Kubernetes environment

The debugging workflow uses [Delve](https://github.com/go-delve/delve) (the Go debugger) to create debug-enabled images that expose a debug port for remote connection

### Prerequisites

Before starting, ensure you have:
- A running Minikube cluster (`make minikube-test-cluster`)
- VS Code with the Go extension installed
- Docker installed and configured
- Basic familiarity with VS Code debugging

**Note**: While this guide uses VS Code as an example, the remote debugging approach works with any IDE or editor that supports connecting to Delve's remote debugging protocol. Contributions documenting setup instructions for other development environments are more than welcome !

### Debug Workflow Overview

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Build Debug   │    │   Push & Deploy  │    │   Port Forward  │
│     Image       │───▶│   to Minikube    │───▶│   & Debug       │
└─────────────────┘    └──────────────────┘    └─────────────────┘
        │                       │                       │
        ▼                       ▼                       ▼
  • Add Delve              • Push debug            • kubectl port-forward
  • Compile with            image to cluster         :2346 (debug port)
    debug flags            • Install with           • Attach VS Code
  • Expose port 2346        1 replica               debugger
```

### Architecture Overview

The remote debugging setup creates a connection between your local development environment and the Agones service running in Minikube:

```
                     ┌──────────────────────────────────────────┐
                     │             Minikube Cluster             │
                     │                                          │
                     │   ┌──────────────────────────────────┐   │
                     │   │            Agones Pod            │   │
                     │   │                                  │   │
                     │   │  ┌─────────────┐   ┌────────────────┐│
                     │   │  │    Delve    │◄──► Agones Binary   ││
                     │   │  │  Debugger   │   │ (debug build)   ││
                     │   │  │   :2346     │   │ e.g. /agones-   ││
                     │   │  └─────────────┘   │ allocator       ││
                     │   │                    └────────────────┘│
                     │   └──────────────────────────────────┘   │
                     │                                          │
                     │       ▲                                   │
                     │       │ kubectl port-forward 2346:2346    │
┌────────────────────┴───────┼───────────────────────────────────┐
│                            │                                    │
│                            ▼                                    │
│        ┌──────────────────────────────────────────────────┐     │
│        │               Local Machine                      │     │
│        │                                                  │     │
│        │   ┌─────────────────────┐      ┌──────────────┐  │     │
│        │   │  VS Code            │◄────►│ localhost:2346│ │     │
│        │   │  Debug Mode         │ Debug│ (Delve Remote │ │     │
│        │   │  Breakpoints        │      │   Endpoint)   │ │     │
│        │   └─────────────────────┘      └──────────────┘  │     │
│        │                                                  │     │
│        │   - Local source code (matches pod build)        │     │
│        │   - launch.json (remote attach)                  │     │
│        └──────────────────────────────────────────────────┘     │
└─────────────────────────────────────────────────────────────────┘
```

### Setting up Debug Environment

1. **Start Minikube cluster (if not already running):**
   ```bash
   make minikube-test-cluster
   ```

2. **Build debug image for the desired service:**
   ```bash
   make build-debug-images WITH_WINDOWS=0 WITH_ARM64=0
   ```

3. **Push debug image to Minikube:**
   ```bash
   make minikube-push WITH_WINDOWS=0 WITH_ARM64=0
   ```

4. **Install Agones with debug configuration:**
   ```bash
   make minikube-install-debug
   ```
   This installs Agones with:
   - All services set to 1 replica for easier debugging
   - Debug image deployed all services

5. **Set up port forwarding:**

   **For debugging Agones system services (deployments)**
   ```bash
   # This will start port forwarding for all Agones services with default ports:
   # - Controller: localhost:2346 -> agones-controller:2346
   # - Extensions: localhost:2347 -> agones-extensions:2346  
   # - Ping: localhost:2348 -> agones-ping:2346
   # - Allocator: localhost:2349 -> agones-allocator:2346
   # - Processor: localhost:2350 -> agones-processor:2346
   make minikube-debug-portforward
   
   # Or customize the ports:
   make minikube-debug-portforward MINIKUBE_DEBUG_CONTROLLER_PORT=3000 MINIKUBE_DEBUG_ALLOCATOR_PORT=3001
   ```
   Use Ctrl+C to stop all port forwards. The command includes proper cleanup to ensure all background processes are terminated.

   **For debugging the Agones SDK sidecar in game server pods**
   ```bash
   # Interactive mode - shows a list of game server pods to choose from
   make minikube-debug-sdk-portforward
   
   # Or specify a specific game server pod directly
   make minikube-debug-sdk-portforward MINIKUBE_DEBUG_POD_NAME=simple-game-server-abc123 MINIKUBE_DEBUG_SDK_PORT=2351
   ```

   **Example output for interactive mode:**
   ```
   Searching for pods with agones-gameserver-sidecar container...
   Found pods with agones-gameserver-sidecar container:
        1  simple-game-server-7d94f
        2  xonotic-gameserver-9x8k2
        3  unity-gameserver-m3n7q
   Select pod number (1-3): 2
   Port forwarding to pod: xonotic-gameserver-9x8k2
   Forwarding from 127.0.0.1:2351 -> 2346
   Forwarding from [::1]:2351 -> 2346
   ```

   This forwards local ports to the debug ports in the respective pods/deployments.

### VS Code Configuration

Create or update `.vscode/launch.json` in your workspace root (the Agones repository root directory):

**Note:** The `${workspaceFolder}` variable must point to the Agones repository root path (e.g., `/path/to/agones`) for proper source file mapping between your local workspace and the remote debugging session

```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Debug Agones Controller (Remote)",
            "type": "go",
            "request": "attach",
            "mode": "remote",
            "host": "127.0.0.1",
            "port": 2346,
            "cwd": "${workspaceFolder}",
            "substitutePath": [
                {
                    "from": "${workspaceFolder}",
                    "to": "/go/src/agones.dev/agones"
                }
            ]
        },
        {
            "name": "Debug Agones Extensions (Remote)",
            "type": "go",
            "request": "attach",
            "mode": "remote", 
            "host": "127.0.0.1",
            "port": 2347,
            "cwd": "${workspaceFolder}",
            "substitutePath": [
                {
                    "from": "${workspaceFolder}",
                    "to": "/go/src/agones.dev/agones"
                }
            ]
        },
        {
            "name": "Debug Agones Ping (Remote)",
            "type": "go",
            "request": "attach",
            "mode": "remote", 
            "host": "127.0.0.1",
            "port": 2348,
            "cwd": "${workspaceFolder}",
            "substitutePath": [
                {
                    "from": "${workspaceFolder}",
                    "to": "/go/src/agones.dev/agones"
                }
            ]
        },
        {
            "name": "Debug Agones Allocator (Remote)",
            "type": "go",
            "request": "attach",
            "mode": "remote", 
            "host": "127.0.0.1",
            "port": 2349,
            "cwd": "${workspaceFolder}",
            "substitutePath": [
                {
                    "from": "${workspaceFolder}",
                    "to": "/go/src/agones.dev/agones"
                }
            ]
        },
        {
            "name": "Debug Agones Processor (Remote)",
            "type": "go",
            "request": "attach",
            "mode": "remote", 
            "host": "127.0.0.1",
            "port": 2350,
            "cwd": "${workspaceFolder}",
            "substitutePath": [
                {
                    "from": "${workspaceFolder}",
                    "to": "/go/src/agones.dev/agones"
                }
            ]
        },
        {
            "name": "Debug Agones SDK Sidecar (Remote)",
            "type": "go",
            "request": "attach",
            "mode": "remote", 
            "host": "127.0.0.1",
            "port": 2351,
            "cwd": "${workspaceFolder}",
            "substitutePath": [
                {
                    "from": "${workspaceFolder}",
                    "to": "/go/src/agones.dev/agones"
                }
            ]
        }
    ]
}
```

**Start multiple debug sessions** in VS Code:
   - Each debug session runs in its own debug console
   - You can set breakpoints in different services simultaneously
   - Switch between debug sessions using the VS Code debug console dropdown
   - The new port forwarding setup allows debugging all services concurrently

### Available Environment Variables

You can customize the debug setup using these environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `MINIKUBE_DEBUG_CONTROLLER_PORT` | 2346 | Local port for controller debugging |
| `MINIKUBE_DEBUG_EXTENSIONS_PORT` | 2347 | Local port for extensions debugging |
| `MINIKUBE_DEBUG_PING_PORT` | 2348 | Local port for ping debugging |
| `MINIKUBE_DEBUG_ALLOCATOR_PORT` | 2349 | Local port for allocator debugging |
| `MINIKUBE_DEBUG_PROCESSOR_PORT` | 2350 | Local port for processor debugging |
| `MINIKUBE_DEBUG_SDK_PORT` | 2351 | Local port for SDK sidecar debugging |
| `MINIKUBE_DEBUG_NAMESPACE` | agones-system | Namespace for Agones services |
| `MINIKUBE_DEBUG_POD_NAME` | (none) | Specific pod name for SDK debugging |

This is particularly useful when debugging interactions between services, such as when the controller communicates with the allocator, or when investigating issues that span multiple Agones components.

## Next Steps

- See [Make Reference](make-reference.md) for all available development and debugging make targets
- See [Troubleshooting Guide](troubleshooting.md) for common development issues and solutions