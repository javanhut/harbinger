# Harbinger - Git branch monitor

A Git conflict monitoring tool that watches your repository in the background and notifies you when your branch needs attention.

## Features

- **Automatic Monitoring**: Continuously polls your Git repository for remote changes
- **Smart Notifications**: Get notified when:
  - Your branch is out of sync with remote
  - Remote branch has new commits
  - Potential merge conflicts are detected
- **Background Processing**: Run monitors in the background with `--detach` flag
- **Interactive Conflict Resolution**: Terminal-based UI for resolving conflicts
  - Color-coded conflict sections (yours vs. theirs)
  - Accept yours or theirs with one keystroke
  - Edit conflicts in your favorite editor
  - Skip files to resolve later
  - Automatic staging of resolved files
- **Configurable**: Control automatic vs. manual conflict resolution
- **Cross-Platform**: Works on macOS, Linux, and Windows

## Installation

### Quick Install

```bash
curl -sSL https://raw.githubusercontent.com/javanhut/harbinger/main/scripts/install.sh | bash
```

### Using Go

```bash
go install github.com/javanhut/harbinger@latest
```

### Build from Source

```bash
git clone https://github.com/javanhut/harbinger.git
cd harbinger
make install
```

### Using Docker

```bash
docker run -v $(pwd):/workspace javanhut/harbinger monitor
```

## Quick Start

### 1. Start monitoring your repository

```bash
# Monitor current repository in foreground
harbinger monitor

# Or run in background (recommended)
harbinger monitor --detach
```

### 2. When conflicts are detected

Harbinger will automatically launch the conflict resolution UI (if `auto_resolve: true` in config) or notify you to run:

```bash
harbinger resolve
```

### 3. Stop background monitoring

```bash
harbinger stop
```

## Usage

### Basic Commands

| Command | Description |
|---------|-------------|
| `harbinger monitor` | Start monitoring current repository |
| `harbinger monitor -d` | Start monitoring in background. Logs are written to `~/.harbinger.<PID>.log` |
| `harbinger logs [PID]` | Read logs from a specific background monitor process |
| `harbinger stop` | Stop background monitors |
| `harbinger resolve` | Manually resolve conflicts |

### Monitor Options

```bash
# Custom polling interval
harbinger monitor --interval 1m

# Monitor specific repository
harbinger monitor --path /path/to/repo

# Background with custom settings
harbinger monitor --detach --interval 30s --path /path/to/repo
```

## Conflict Resolution

Harbinger provides a powerful interactive terminal UI for resolving merge conflicts efficiently.

### Two Resolution Modes

| Mode | Trigger | Configuration |
|------|---------|---------------|
| **Automatic** | Conflicts detected during monitoring | `auto_resolve: true` (default) |
| **Manual** | Run `harbinger resolve` command | `auto_resolve: false` or anytime |

### Interactive UI Walkthrough

When conflicts are detected, harbinger displays:

```
=== Conflict Resolution (1/3) ===
File: src/main.go

<<<<<<< YOURS
func main() {
    fmt.Println("Hello from your branch")
}
>>>>>>> THEIRS
func main() {
    fmt.Println("Hello from remote branch")
}

--------------------------------------------------
Choose an option:
  [1] Accept yours
  [2] Accept theirs  
  [3] Edit in your editor
  [4] Skip this file

Your choice: 
```

### Resolution Options

| Option | Action | Result |
|--------|--------|--------|
| **[1] Accept yours** | Keep your local changes | Runs `git checkout --ours <file>` |
| **[2] Accept theirs** | Accept incoming changes | Runs `git checkout --theirs <file>` |
| **[3] Edit in editor** | Open file in `$EDITOR` | Manual editing + auto-staging |
| **[4] Skip this file** | Leave unresolved | Continue to next conflict |

### Key Features

- **Color-coded sections**: Green for yours, red for theirs
- **Automatic staging**: Resolved files are staged automatically
- **Fast navigation**: Process multiple conflicts quickly
- **Resumable**: Skip files and come back later
- **Editor integration**: Uses your preferred editor (`$EDITOR`)

## Configuration

Create a configuration file at `~/.harbinger.yaml`:

```yaml
poll_interval: 30s
editor: vim
notifications: true
auto_resolve: true
ignore_branches:
  - main
  - master
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `poll_interval` | duration | `30s` | How often to check for remote changes |
| `editor` | string | `$EDITOR` | External editor for conflict resolution |
| `notifications` | boolean | `true` | Enable/disable system notifications |
| `auto_resolve` | boolean | `true` | Auto-launch conflict resolution UI |
| `ignore_branches` | array | `[]` | List of branches to skip monitoring |

### Example Configurations

**Minimal monitoring (manual resolution only):**
```yaml
auto_resolve: false
notifications: false
```

**High-frequency monitoring:**
```yaml
poll_interval: 10s
auto_resolve: true
editor: code
```

**Production-safe monitoring:**
```yaml
poll_interval: 5m
auto_resolve: false
ignore_branches:
  - main
  - master
  - production
```

## Common Workflows

### Developer Workflow

```bash
# Start monitoring in background when you begin work
harbinger monitor -d

# Work on your feature branch
git checkout feature/new-feature
# ... make changes ...

# Harbinger notifies you of conflicts when they're detected
# Resolve automatically or manually
harbinger resolve

# Stop monitoring when done
harbinger stop
```

### CI/CD Integration

```bash
# Check for conflicts before merging
harbinger resolve --dry-run  # (future feature)

# Or monitor specific branches
harbinger monitor --path /path/to/repo --interval 1m
```

### Team Collaboration

```bash
# Each team member monitors their branch
harbinger monitor -d

# Conflicts are resolved immediately when detected
# No more "merge conflict" surprises during PRs
```

## Troubleshooting

### Common Issues

**Issue: "not a git repository"**
```bash
# Ensure you're in a git repository
git status

# Or specify the repository path
harbinger monitor --path /path/to/your/repo
```

**Issue: "No background harbinger monitor found"**
```bash
# Check if monitor is running
ps aux | grep harbinger

# Start a new monitor
harbinger monitor -d
```

**Issue: "Conflicts not detected"**
```bash
# Ensure you have remote tracking set up
git remote -v
git branch -vv

# Manually fetch to test
git fetch --all
```

**Issue: "Editor not opening"**
```bash
# Check your EDITOR environment variable
echo $EDITOR

# Or set it in config
echo "editor: code" >> ~/.harbinger.yaml
```

### Debug Mode

```bash
# Check logs for a specific detached process
harbinger logs $(PID)

# Or tail the log file directly
tail -f ~/.harbinger.<PID>.log
```

## How It Works

### Overview

Harbinger is a background service that monitors your Git repository for changes and potential conflicts. It operates in a non-intrusive manner, fetching remote changes periodically and analyzing them against your local state.

### Architecture

The tool follows a modular architecture with clear separation of concerns:

1. **Monitor Loop**: The core monitoring loop (`internal/monitor/monitor.go`) runs as a goroutine that:
   - Executes at configurable intervals (default: 30 seconds)
   - Performs git fetch operations to retrieve remote changes
   - Compares local and remote branch states
   - Triggers notifications based on detected changes

2. **Git Operations Layer** (`internal/git/repository.go`):
   - Wraps Git commands using the command-line interface
   - Provides high-level abstractions for:
     - Fetching remote changes
     - Comparing commits between branches
     - Detecting merge conflicts
     - Managing branch states
   - Implements error handling and recovery mechanisms

3. **Conflict Detection Engine** (`internal/conflict/resolver.go`):
   - Performs a simulated merge to detect potential conflicts
   - Parses Git conflict markers in files
   - Extracts conflict sections (ours vs. theirs)
   - Provides strategies for automatic resolution

4. **Notification System** (`internal/notify/notifier.go`):
   - Abstracts platform-specific notification APIs
   - Uses native system notifications:
     - macOS: `osascript` for notification center
     - Linux: `notify-send` (requires libnotify)
     - Windows: Windows toast notifications
   - Falls back to terminal output if system notifications fail

5. **Terminal UI** (`internal/ui/terminal.go`):
   - Built using terminal control sequences
   - Provides interactive conflict resolution interface
   - Features:
     - Syntax highlighting for conflict markers
     - Side-by-side diff view
     - Keyboard navigation
     - Editor integration

### Detailed Workflow

1. **Initialization**:
   - Validates Git repository presence
   - Checks for remote configuration
   - Loads user configuration from `~/.harbinger.yaml`
   - Initializes notification system

2. **Monitoring Cycle**:
   ```
   ┌─────────────┐
   │   Start     │
   └──────┬──────┘
          │
          ▼
   ┌─────────────┐
   │ Fetch Remote│
   └──────┬──────┘
          │
          ▼
   ┌─────────────┐     No changes
   │Check Changes├─────────────────┐
   └──────┬──────┘                 │
          │ Changes detected        │
          ▼                         │
   ┌─────────────┐                 │
   │ Check Merge │                 │
   │  Conflicts  │                 │
   └──────┬──────┘                 │
          │                         │
   ┌──────┴──────┐                 │
   │             │                 │
   ▼             ▼                 │
   Conflicts   No conflicts        │
   detected                        │
   │             │                 │
   ▼             ▼                 │
   Launch UI   Send notification   │
   │             │                 │
   └─────────────┴─────────────────┘
                 │
                 ▼
           Wait interval
                 │
                 └─────────────┐
                               │
                               ▼
                         (Repeat cycle)
   ```

3. **Conflict Detection Process**:
   - Creates a temporary merge branch
   - Attempts to merge remote changes
   - If merge fails, parses conflict markers
   - Categorizes conflicts by file and type
   - Cleans up temporary branches

4. **Notification Logic**:
   - **New commits on remote**: "Remote branch has N new commits"
   - **Local ahead of remote**: "Your branch is N commits ahead"
   - **Conflicts detected**: "Merge conflicts detected in N files"
   - **Both ahead and behind**: "Branches have diverged"

5. **Interactive Resolution**:
   - Terminal UI launches automatically for conflicts
   - Displays each conflict with context
   - Options per conflict:
     - Accept current branch version (ours)
     - Accept incoming branch version (theirs)
     - Open in configured editor
     - Skip to next conflict
   - Stages resolved files automatically
   - Provides summary of resolution actions

### Configuration System

The configuration is loaded in the following priority order:
1. Command-line flags (highest priority)
2. Configuration file (`~/.harbinger.yaml`)
3. Environment variables
4. Default values (lowest priority)

Configuration options:
- `poll_interval`: How often to check for changes (default: 30s)
- `editor`: External editor for conflict resolution (default: $EDITOR)
- `notifications`: Enable/disable system notifications (default: true)
- `auto_resolve`: Automatically launch conflict resolution UI when conflicts are detected (default: true)
- `ignore_branches`: List of branches to skip monitoring (default: empty)

### Error Handling

The tool implements comprehensive error handling:
- Network failures: Exponential backoff with retry
- Git command failures: Detailed error messages with recovery suggestions
- Permission issues: Clear guidance on fixing repository permissions
- Missing dependencies: Checks for Git and notification tools at startup

### Performance Considerations

- Minimal CPU usage through event-driven design
- Efficient Git operations using plumbing commands where possible
- Caches remote state to reduce network calls
- Configurable polling interval to balance responsiveness and resource usage

## Development

### Project Structure

```
├── cmd/                    # CLI commands
│   ├── main.go
│   └── monitor.go
├── internal/              # Internal packages
│   ├── git/              # Git operations
│   ├── monitor/          # Monitoring logic
│   ├── conflict/         # Conflict resolution
│   ├── ui/              # Terminal UI
│   └── notify/          # Notification system
└── pkg/                  # Public packages
    └── config/          # Configuration
```

### Building

```bash
go build -o harbinger ./cmd
```

### Testing

```bash
go test ./...
```

## License

MIT
