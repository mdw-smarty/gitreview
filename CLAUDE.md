# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Commands

- `make test` — format, vet, and run tests with race detector
- `make fmt` — run `go mod tidy` and `go fmt ./...`
- `make install` — install binary with version from `git describe`
- `make docs` — regenerate README.md from built-in help text
- `go test -run TestName ./...` — run a single test

## Overview

gitreview is a Go CLI tool (zero external dependencies) that scans a directory tree for git repositories, concurrently analyzes their state (dirty, ahead, behind, fetch results), presents repositories needing attention for interactive review via an external git GUI (default: smerge), and appends a review log entry.

## Architecture

**Startup flow:** `main.go` → `ReadConfig()` parses flags/args → `NewGitReviewer(config)` → `GitAnalyzeAll()` → `ReviewAll()` → `PrintCodeReviewLogEntry()`

**Concurrency model:** `analyzer.go` manages a pool of 16 worker goroutines (`worker.go`). Workers receive repo paths from an input channel, run all git commands (`git.go`), and send `GitReport` results to an output channel. `merge()` collects results via fan-in.

**Key types:**
- `Config` (config.go) — CLI flags, root directory, output file
- `GitReport` (git.go) — per-repo analysis results; methods run git commands and build a 7-char progress indicator `[!MABOFS]`
- `GitReviewer` (review.go) — orchestrates analysis, categorizes repos into maps (messy/ahead/behind/fetched/errored/omitted/skipped), drives interactive review
- `Analyzer` / `Worker` (analyzer.go, worker.go) — concurrent worker pool

**I/O helpers** (io.go): `collectGitRepositories()` walks the directory tree, `execute()` shells out to git, `prompt()` reads stdin for interactive review.

## Per-repo Git Config

- `review.skip true` — skip repo entirely
- `review.omit true` — review but omit from final report
- `review.branch <name>` — override default branch detection (main/master)
