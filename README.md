# Vantage

**Centralized GitHub Space Mission Control & Pipeline Cockpit**

Vantage provides unified, high-ground visibility and control over all repositories, commit vitality, release tags, and CI/CD workflow runs across your entire GitHub space (user account or organization).

---

## Key Capabilities

- **Fleet Overview**: Discovers and tracks all repositories in your space with real-time status on default branches, commit vitality, stars, forks, and tags.
- **Unified Pipeline Feed**: Aggregates GitHub Actions workflow runs across all repos in a single real-time stream.
- **Workflow Dispatcher**: Trigger any `workflow_dispatch` pipeline on demand with branch selection and custom inputs.
- **Run Management**: Rerun failed pipelines or cancel in-progress runs directly from the terminal or web dashboard.
- **Zero Configuration**: Automatically resolves credentials via `GITHUB_TOKEN`, `GH_TOKEN`, or local `gh auth token`.
- **Dual Formats**:
  - **Embedded Web Dashboard**: Modern dark-mode interface with live metrics and one-click dispatch at `http://localhost:8080`.
  - **Terminal Cockpit**: Fast CLI commands (`vantage status`, `vantage runs`, `vantage trigger`) for scriptable operations.

---

## Installation

```bash
go install github.com/AndreaPallotta/vantage@latest
```

Or download pre-compiled binaries for Windows, Linux, and macOS from the [Releases](https://github.com/AndreaPallotta/vantage/releases) page.

---

## Quick Start

### Launch Web Cockpit
```bash
vantage
```
Opens the interactive dark-mode dashboard at `http://localhost:8080`.

### CLI Fleet Summary
```bash
# Print fleet status table
vantage status

# Monitor a specific space/organization
vantage status --space AndreaPallotta
```

### Stream Pipeline Runs
```bash
# View recent workflow runs across all repos in space
vantage runs

# Filter runs for a specific repository
vantage runs zephyr
```

### Trigger a Workflow
```bash
# Trigger a release or CI pipeline on main
vantage trigger civet release.yml --ref main
```

---

## Configuration

Vantage reads configuration from `~/.vantage/config.json` or environment variables:

| Option | Env Var | Flag | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `space` | `VANTAGE_SPACE` | `--space`, `-s` | Authenticated user | GitHub user or organization |
| `token` | `GITHUB_TOKEN` | `--token`, `-t` | `gh auth token` | GitHub Personal Access Token |
| `port` | `VANTAGE_PORT` | `--port`, `-p` | `8080` | Port for the web dashboard |
| `auto_open` | `VANTAGE_AUTO_OPEN` | `--no-open` | `true` | Open browser on startup |

---

## License

MIT License. Built by [Andrea Pallotta](https://github.com/AndreaPallotta).
