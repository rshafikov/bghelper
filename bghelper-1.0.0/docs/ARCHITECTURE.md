# bghelper Architecture & Command Workflows

## Overview

bghelper is a CLI tool for managing background processes. It stores process metadata in YAML files and logs in separate log files.

## Storage

- **Location**: `~/.bghelper/processes/`
- **Process files**: `<id>.yaml` - stores process metadata (id, name, command, pid, timestamps, logs_path)
- **Log files**: `<id>.log` - stores stdout and stderr output

**Key design decision**: Status is NOT stored in files. It's computed dynamically by checking if the PID is alive.

## Package Structure

```
cmd/                    - CLI commands (cobra)
  root.go              - Root command and shared utilities
  start.go             - Start new process
  stop.go              - Stop running process
  restart.go           - Restart existing process
  delete.go            - Delete process
  list.go              - List all processes
  show.go              - Show process details
  logs.go              - View process logs
  utils.go             - Shared helper functions

internal/
  process/
    types.go           - Process struct and Store interface
    manager.go         - Process lifecycle management
    status.go          - Status computation (dynamic)
  storage/
    store.go           - File-based persistence
  output/
    table.go           - Table formatting for list output
```

---

## Command Workflows

### `bgh start <command> [--name <name>]`

**Purpose**: Create and start a new background process.

**Workflow**:
1. Parse command arguments and optional name flag
2. Generate unique sequential ID by checking existing processes
3. Create Manager instance with FileStore
4. Call `manager.Start(id, command)`:
   - Create new Process struct with ID and command
   - Set StartedAt timestamp
   - Create/open log file at `~/.bghelper/processes/<id>.log`
   - Execute command via `exec.Command("bash", "-c", command)`
   - Redirect stdout/stderr to log file
   - Capture PID from started process
   - Save process metadata to YAML file
   - Start goroutine to monitor process exit
5. If name provided, update process and save again
6. Print success message with ID, PID, and command

**Code Flow**:
```
cmd/start.go: RunE()
  └─> generateUniqueID()
  └─> manager.Start()
        └─> NewProcess()
        └─> os.OpenFile(logsPath)
        └─> exec.Command().Start()
        └─> store.Save()
        └─> go monitorProcess()
```

---

### `bgh stop <id>`

**Purpose**: Stop a running process by sending SIGTERM.

**Workflow**:
1. Resolve process ID or name to actual process
2. Load process from storage (status computed dynamically)
3. Check if process is running (status == "running")
4. Load process into Manager's memory
5. Call `manager.Stop(id)`:
   - Find process by PID using `os.FindProcess()`
   - Send `os.Interrupt` signal (SIGTERM)
   - If signal fails, force kill with `proc.Kill()`
   - Set exit code to 130 (128 + SIGINT)
   - Save updated process state
6. Print confirmation with exit code

**Code Flow**:
```
cmd/stop.go: RunE()
  └─> resolveProcessIdentifier()
  └─> store.Load() -> computes status dynamically
  └─> manager.LoadFromStorage()
  └─> manager.Stop()
        └─> os.FindProcess()
        └─> proc.Signal(os.Interrupt)
        └─> store.Save()
```

---

### `bgh restart <id>`

**Purpose**: Restart an existing process (stop if running, then start again).

**Workflow**:
1. Resolve process ID or name
2. Call `manager.Restart(id)`:
   - Load process from storage if not in memory
   - If process is running:
     - Find by PID
     - Send SIGTERM signal
     - Wait 100ms for graceful exit
   - Clear log file (truncate to 0)
   - Reset process state (StartedAt, ExitCode)
   - Execute command again
   - Capture new PID
   - Save updated process state
   - Start new goroutine to monitor process
3. Print confirmation message

**Code Flow**:
```
cmd/restart.go: RunE()
  └─> resolveProcessIdentifier()
  └─> manager.Restart()
        └─> store.Load() (if not in memory)
        └─> os.FindProcess() + proc.Signal() (if running)
        └─> os.Truncate(logsPath, 0)
        └─> exec.Command().Start()
        └─> store.Save()
        └─> go monitorProcess()
```

---

### `bgh delete <id> [--force]`

**Purpose**: Delete a stopped process from storage.

**Workflow**:
1. Resolve process ID or name
2. Load process (status computed dynamically)
3. If process is running:
   - Without --force: return error
   - With --force: stop process first
4. Delete process file and log file from storage
5. Print confirmation

**Code Flow**:
```
cmd/delete.go: RunE()
  └─> resolveProcessIdentifier()
  └─> store.Load() -> computes status
  └─> if running && !force: error
  └─> if running && force: manager.Stop()
  └─> store.Delete()
        └─> os.Remove(<id>.yaml)
        └─> os.Remove(<id>.log)
```

---

### `bgh list [--format table|json|yaml]`

**Purpose**: List all processes with their status.

**Workflow**:
1. Get storage directory
2. Call `store.LoadAll()`:
   - List all YAML files in storage directory
   - For each file, load and unmarshal YAML
   - Compute status dynamically via `RefreshStatus()`
3. Sort processes by CreatedAt (newest first)
4. Format output based on --format flag:
   - table: Use TableFormatter with smart column widths
   - json: Marshal to JSON
   - yaml: Marshal to YAML

**Status Computation** (`RefreshStatus`):
```
if PID == 0:
  status = "stopped"
else if IsProcessAlive(PID):
  status = "running"
else:
  status = "stopped" or "crashed" (based on exit code)
```

**Smart Column Widths**:
- ID: Based on longest ID in data
- NAME: Based on longest name (max 20)
- STATUS: Based on longest status string
- CREATED: Fixed 10 chars (MM-DD HH:MM)
- COMMAND: Remaining terminal width

**Code Flow**:
```
cmd/list.go: Run()
  └─> store.LoadAll()
        └─> store.List() -> read directory
        └─> store.Load() for each
              └─> yaml.Unmarshal()
              └─> RefreshStatus()
                    └─> IsProcessAlive(PID)
  └─> sort by CreatedAt
  └─> TableFormatter.FormatProcessList()
```

---

### `bgh show <id> [--format table|json|yaml]`

**Purpose**: Display detailed information about a single process.

**Workflow**:
1. Resolve process ID or name
2. Load process from storage (status computed)
3. Format output based on --format flag
4. For table format: display ID, Name, Command, Status, PID, timestamps, exit code, logs path

**Code Flow**:
```
cmd/show.go: Run()
  └─> resolveProcessIdentifier()
  └─> store.Load() -> computes status
  └─> displayProcessDetails() or JSON/YAML output
```

---

### `bgh logs <id> [--follow] [--tail N]`

**Purpose**: View process output logs.

**Workflow**:
1. Resolve process ID or name
2. Get log file path from store
3. Check if log file exists
4. Open log file
5. Based on flags:
   - Default: Read and print entire file
   - --follow: Stream file continuously (like tail -f)
   - --tail N: Print only last N lines

**Code Flow**:
```
cmd/logs.go: RunE()
  └─> resolveProcessIdentifier()
  └─> store.GetLogsPath()
  └─> os.Open(logsPath)
  └─> io.Copy() or streamLogs() or tailLogs()
```

---

## Key Design Decisions

### 1. Dynamic Status Computation
Status is not stored in YAML files. Instead, it's computed every time a process is loaded:
- Check if PID > 0 and process is alive → "running"
- Otherwise → "stopped" or "crashed"

This ensures status is always accurate, even if bghelper wasn't running when the process exited.

### 2. Log File Management
Each process has a dedicated log file. Logs are:
- Created when process starts
- Truncated on restart
- Deleted when process is deleted

### 3. Process Monitoring
When a process starts, a goroutine monitors it:
- Waits for process to exit via `cmd.Wait()`
- Captures exit code
- Updates process state
- Closes log file handle

### 4. Storage Format
YAML files store:
- id, name, command
- pid (operating system process ID)
- created_at, started_at timestamps
- exit_code (if process has exited)
- logs_path

### 5. ID Generation
IDs are sequential integers (1, 2, 3...). The system finds the maximum existing ID and increments it.
