# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-08-17

### Added
- **Core Engine & Architecture**:
  - Initial implementation of Vantage centralized multi-platform CI/CD dashboard.
  - Multi-platform provider interface (`internal/provider/provider.go`) supporting GitHub and GitLab (public and self-hosted).
  - Multi-namespace manager (`internal/manager/manager.go`) for coordinating and aggregating across single or multiple spaces concurrently.
  - Configuration manager (`internal/config/config.go`) persisting spaces and authentication tokens to `~/.vantage/config.json`.
- **User Interface & Web Cockpit**:
  - Embedded web interface with cyber-telemetry HUD styling, dark palette, and monospace typography.
  - Interactive **"+ Add Space"** modal enabling real-time connection of GitHub and GitLab spaces without restart.
  - Step-by-step pipeline job inspector modal displaying nested execution trees, step progress (`[OK]`, `[ERR]`, `[RUN]`, `[QUE]`), and execution durations.
  - Interactive metric filter cards to dynamically isolate active and failing pipelines.
  - Bespoke vector SVG favicon for browser tabs.
- **CLI Commands**:
  - `vantage` root command running embedded dashboard web server.
  - `vantage status` tabular summary of fleet assets, languages, commits, and pipeline health.
  - `vantage spaces` listing all configured spaces and credential statuses.
  - `vantage add-space` CLI command for configuring new spaces.
  - `vantage runs` real-time stream of recent workflow runs across all spaces.
  - `vantage trigger` workflow and pipeline dispatcher.
- **CI/CD Automation**:
  - GitHub Actions CI workflow (`ci.yml`) with automated tests, race detector, and coverage reporting.
  - Multi-platform automated release workflow (`release.yml`) cross-compiling binary archives for Windows (amd64/arm64), Linux (amd64/arm64), and macOS (amd64/arm64) on tag push.

### Changed
- **Pipeline Vitality Metrics**:
  - Filtered health indicators to evaluate only the latest run per repository rather than historical runs.
- **Design & Polish**:
  - Removed star counts and non-essential telemetry metrics.
  - Removed all emojis across the web UI, CLI output, and codebase in favor of clean monospace tags.
  - Normalized all em-dashes to standard hyphens.
  - Simplified browser tab title to `Vantage`.
