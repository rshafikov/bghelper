# bghelper

A CLI tool for managing background processes. Start, stop, restart, and monitor long-running commands like SSH tunnels, dev servers, and other background tasks.

## Features

- 🚀 Start background processes with automatic ID assignment
- 📋 List all processes with real-time status (computed dynamically)
- 🔄 Restart processes without losing configuration
- 📜 View process logs (stdout/stderr)
- 🛑 Stop and delete processes
- 🎨 Color-coded status output
- 💾 Persistent process definitions across restarts

## Installation

### From Source

```bash
git clone https://github.com/ra.shafikov/bghelper.git
cd bghelper
make install
```

### From Release

Download the appropriate binary for your platform from the [releases page](https://github.com/ra.shafikov/bghelper/releases).

## Usage

### Start a process

```bash
# Start a process
bgh start "ssh -L 8080:localhost:8080 user@server"

# Start with a name
bgh start --name "my-tunnel" "ssh -L 8080:localhost:8080 user@server"
```

### List processes

```bash
# List all processes
bgh list

# Output formats
bgh list --format json
bgh list --format yaml
```

### Show process details

```bash
bgh show 1
bgh show my-tunnel
```

### View logs

```bash
# View all logs
bgh logs 1

# Stream logs in real-time
bgh logs 1 --follow

# Show last 50 lines
bgh logs 1 --tail 50
```

### Restart a process

```bash
bgh restart 1
bgh restart my-tunnel
```

### Stop a process

```bash
bgh stop 1
```

### Delete a process

```bash
# Delete stopped process
bgh delete 1

# Force delete running process (stops first)
bgh delete --force 1
```

## How It Works

### Storage

Process metadata is stored in `~/.bghelper/processes/`:
- `<id>.yaml` - Process metadata (id, name, command, pid, timestamps)
- `<id>.log` - Process stdout/stderr output

### Status Computation

Status is **not** stored in files. It's computed dynamically by checking if the process PID is alive. This ensures accurate status even after system restarts.

### Process Monitoring

When a process starts, a goroutine monitors it and:
- Captures exit code when process terminates
- Updates process state in storage
- Closes log file handle

## Development

### Build

```bash
make build
```

### Test

```bash
make test
```

### Run

```bash
make run
```

### Release

```bash
# Create a new tag
git tag v0.1.0
git push origin v0.1.0

# GitHub Actions will automatically build and release
```

## License

MIT License - see [LICENSE](LICENSE) for details.
