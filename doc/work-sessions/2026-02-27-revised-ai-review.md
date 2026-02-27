# Revised AI Review: Interleaved Per-Repo Workflow

## Motivation

The current AI review runs as a batch phase _before_ the interactive GUI review, producing a single concatenated file. This means the user must wait for all AI reviews to finish before any GUI opens, and the combined file is unwieldy when reviewing individual repos.

The revised flow interleaves AI review with the GUI step on a per-repo basis and writes each repo's audit to its own file within a timestamped folder.

## Revised User-Facing Flow

```
1. GitAnalyzeAll()        — concurrent fetches + analysis (unchanged)
2. For each reviewable repo:
   a. Prompt: "Press <ENTER> to review <repo>..."
   b. Open the git GUI (smerge, gitk, etc.) at the repo
   c. If repo is AI-reviewable, run the AI audit in the background
   d. When the AI audit completes, open the resulting text file
3. PrintCodeReviewLogEntry()  — journal output (unchanged)
```

The user sees the GUI immediately, and the AI audit file opens shortly after (typically a few seconds) while they are already inspecting the repo in the GUI.

## Output Structure

Currently all reviews go to a single file:

```
/tmp/code-review/2026-02-27.md    ← single file, appended
```

New structure — one file per repo, grouped by run timestamp:

```
/tmp/code-review/2026-02-27-143052/
  smarty--assertions.md
  smarty--httpstatus.md
  smarty--smartystreets-go-sdk.md
```

The folder name uses `YYYY-MM-DD-HHMMSS` format (the timestamp of when the review session started). Each file is named by deriving a safe filename from the repo path (last two path segments joined by `--`, e.g., `smarty--assertions`). Multiple runs per day each get their own folder.

## Changes to `main.go`

Remove the standalone `AIReviewAll()` phase:

```go
func main() {
    config := ReadConfig(Version)
    reviewer := NewGitReviewer(config)
    reviewer.GitAnalyzeAll()
    // reviewer.AIReviewAll()   ← REMOVED
    reviewer.ReviewAll()
    reviewer.PrintCodeReviewLogEntry()
}
```

## Changes to `ReviewAll()` in `review.go`

Currently `ReviewAll()` does two things: prints a summary, then opens all GUIs in a rapid loop with a 25ms delay. The revised version loops through repos one at a time with an interactive prompt, integrating the AI audit.

Before the loop begins, create the timestamped output directory (if any repos are AI-reviewable).

The per-repo loop becomes:

```
for each repo in reviewable:
    prompt user to press <ENTER>
    open the git GUI
    if repo is in aiReviewable:
        run AI review, write result to individual file
        open that file (e.g., via `subl` or configured opener)
```

The AI review is sequential (not background/goroutine) since the user is already looking at the GUI and will naturally wait. Running it synchronously keeps the implementation simple and avoids race conditions with file opening. The claude CLI call typically returns in a few seconds.

## Changes to `ai_review.go`

### Remove `AIReviewAll()`

The batch orchestrator is no longer needed. Replace it with a smaller method and a helper.

### New: `AIReviewRepo(repoPath, branch, outputDir string)`

Per-repo method on `GitReviewer` that:

1. Calls `runAIReview(repoPath, branch)` (this function is unchanged)
2. Derives a filename from the repo path
3. Writes the review (or error) to `<outputDir>/<repo-name>.md`
4. Opens the file using the configured text editor

### New: `PrepareAIReviewDir() string`

Creates the timestamped output directory and returns its path. Called once before the review loop, only if AI review is enabled and there are AI-reviewable repos. Returns `""` if AI review is disabled or preconditions fail (claude not in PATH, etc.), signaling the review loop to skip AI steps.

### `runAIReview()` — unchanged

The existing function that runs `git diff` and pipes to `claude -p` remains as-is.

## File Format Per Repo

Each individual file is a self-contained review:

```markdown
# AI Code Review: /Users/mike/src/github.com/smarty/assertions

**Branch:** main
**Date:** 2026-02-27 14:30:52

[claude's review output]
```

Or, on error:

```markdown
# AI Code Review: /Users/mike/src/github.com/smarty/assertions

**Branch:** main
**Date:** 2026-02-27 14:30:52

ERROR: git diff failed: exit status 128
```

## Text Editor for AI Output

Currently hardcoded to `subl`. Consider using `$EDITOR` with a fallback to `open` (macOS) for opening individual review files. This can be deferred — `subl` is fine for now and matches the existing behavior.

## Prompt Changes

Current prompt (before opening all GUIs at once):

```
Press <ENTER> to initiate the review process (will open 5 review windows), or 'q' to quit...
```

New prompt (per-repo, shown before each GUI opens):

```
[1/5] Press <ENTER> to review smarty/assertions...
```

This gives the user a sense of progress through the review session.

## Summary of Changes

| File           | Change                                                              |
|----------------|---------------------------------------------------------------------|
| `main.go`      | Remove `reviewer.AIReviewAll()` call                                |
| `review.go`    | Restructure `ReviewAll()` to per-repo prompt + interleaved AI audit |
| `ai_review.go` | Replace `AIReviewAll()` with `PrepareAIReviewDir()` + `AIReviewRepo()` |

No new files. No new dependencies. `runAIReview()` and all git analysis code remain unchanged.

## Design Decisions

**Sequential per-repo (not background goroutine)** — The user is interacting with the GUI while the audit runs, but opening the file synchronously after the audit completes keeps the UX predictable: GUI opens, then a moment later the audit file appears. No goroutine cleanup or synchronization needed.

**Individual files vs. single file** — Individual files are easier to reference later, can be opened independently, and avoid the growing-monolith problem of the current appended file.

**Timestamped folder** — Groups a review session's output together. Multiple runs per day don't collide. Easy to clean up old sessions.

**Filename derivation** — Using the last two path segments (org + repo) produces readable, unique names without deeply nested paths. Edge cases (repos not under an org directory) fall back to just the repo directory name.

## Implementation Checklist

### `ai_review.go` — replace batch orchestration with per-repo functions

- [x] Remove `AIReviewAll()` method
- [x] Add `PrepareAIReviewDir() string` method that:
  - Returns `""` early if `config.AIReviewer` is empty, `aiReviewable` is empty, reviewer is unsupported, or `claude` not in PATH
  - Creates `/tmp/code-review/<YYYY-MM-DD-HHMMSS>/` directory
  - Returns the directory path
- [x] Add `AIReviewRepo(repoPath, branch, outputDir string)` method that:
  - Calls `runAIReview(repoPath, branch)`
  - Derives filename from last two path segments of `repoPath` (joined by `--`)
  - Writes file header (`# AI Code Review: <full repoPath>`, `**Branch:**`, `**Date:**`) and review content (or error) to `<outputDir>/<filename>.md`
  - Opens the file via `subl`
- [x] Keep `runAIReview()` unchanged

### `review.go` — restructure `ReviewAll()` for per-repo prompts with interleaved AI audit

- [x] Remove the bulk "Press ENTER to open N windows" prompt
- [x] Remove the rapid-fire GUI-open loop with 25ms sleep
- [x] Call `PrepareAIReviewDir()` once before the loop to get the output directory (empty string means skip AI)
- [x] Replace with per-repo loop:
  - Print `[i/N] Press <ENTER> to review <short repo name>...` and wait
  - Open the git GUI at the repo (preserve existing `gitk` vs. other GUI logic)
  - If `outputDir != ""` and repo is in `aiReviewable`, call `AIReviewRepo()`

### `main.go` — remove standalone AI phase

- [x] Remove `reviewer.AIReviewAll()` call

### Verify

- [x] `make test`

## Verification

`make test` — confirm no regressions
