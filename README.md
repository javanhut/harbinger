# Harbinger - Git branch monitor

A Git conflict monitoring tool that watches your repository in the background and notifies you when your branch needs attention.

## Features

- **Automatic Monitoring**: Polls your Git repository for remote changes
- **Smart Notifications**: Get notified when:
  - Your branch is out of sync with remote
  - Remote branch has new commits
  - Potential merge conflicts are detected
- **Interactive Conflict Resolution**: Terminal-based UI for resolving conflicts
  - Accept yours or theirs
  - Edit conflicts in your favorite editor
  - View only the conflicting sections
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

## Usage

### Start monitoring the current repository

```bash
harbinger monitor
```

### Monitor with custom interval

```bash
harbinger monitor --interval 1m
```

### Monitor a specific repository

```bash
harbinger monitor --path /path/to/repo
```

## Configuration

Create a configuration file at `~/.harbinger.yaml`:

```yaml
poll_interval: 30s
editor: vim
notifications: true
ignore_branches:
  - main
  - master
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
- `poll_interval`: How often to check for changes
- `editor`: External editor for conflict resolution
- `notifications`: Enable/disable system notifications
- `ignore_branches`: List of branches to skip monitoring
- `auto_fetch`: Automatically fetch before monitoring
- `conflict_strategy`: Default conflict resolution strategy

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
