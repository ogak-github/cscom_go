# Control System Commander (CSCom)
A lightweight system monitoring tools. CPU, RAM, Disk, Network, Docker, and Processes. Have to mode TUI only accessed via local and RestAPI and Websocket for realtime data

## Features

- **CPU** — usage per core, load average, model info
- **RAM** — used/free/total with usage bar
- **Swap** — usage tracking
- **Disk** — usage per mount point
- **Network** — RX/TX bytes, packets per interface
- **Processes** — top processes sorted by CPU usage
- **Docker** — container CPU%, memory, network I/O (auto-detect, hidden if no containers)

## Development

### Prerequisites

- Go 1.21+

### Run locally (without installing)

**TUI:**

```bash
go run ./cmd/cscom/
```

**Web server:**

```bash
go run ./cmd/cscom-serve/
```

**Build binaries:**

```bash
go build -o bin/cscom      ./cmd/cscom/
go build -o bin/cscom-serve ./cmd/cscom-serve/
```

---

## Deploy

Both local and remote deploy use the same script. It will prompt for an API key before building.

### Local machine

```bash
./install.sh local
```

### Remote server

Setup `~/.ssh/config` first:

```
Host my-vps
    HostName 192.168.1.100
    User root
```

Then:

```bash
./install.sh my-vps
```

Both commands will:
1. Prompt for `CSCOM_KEY` (leave empty to disable auth) 
2. Build standalone binaries
3. Install to `/opt/cscom` + create systemd service
5. Start/restart the service

### Uninstall

```bash
./uninstall.sh local
./uninstall.sh my-vps
``` 

## Usage

### Terminal (TUI)

```bash
cscom
```

### Web Dashboard

The dashboard is served by the `cscom-serve` binary (managed by systemd):

- Dashboard: `http://<host>:4040`
- REST API: `http://<host>:4040/api/metrics`
- WebSocket: `ws://<host>:4040/ws`

If `CSCOM_KEY` is set, the dashboard will show a login dialog on first visit.

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `4040` | Web server port |
| `CSCOM_KEY` | (empty) | API key. If empty, auth is disabled |

## Struktur Project

```
cscom/
├── cmd/
│   ├── cscom/
│   │   └── main.go              # Entry point — TUI
│   └── cscom-serve/
│       └── main.go              # Entry point — Web server
├── internal/
│   ├── metrics/
│   │   ├── collector.go         # Aggregates all metrics into a Snapshot
│   │   ├── cpu.go
│   │   ├── ram.go
│   │   ├── disk.go
│   │   ├── network.go
│   │   ├── docker.go
│   │   └── processes.go
│   └── transport/
│       └── server.go            # HTTP / REST API / WebSocket
├── web/
│   └── index.html               # Dashboard frontend
├── install.sh
├── uninstall.sh
├── go.mod
├── go.sum
└── .env.example
```