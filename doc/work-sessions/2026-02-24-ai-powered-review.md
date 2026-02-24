# AI-Powered Code Review Extension

## Context

gitreview currently identifies repos behind origin, opens a GUI for manual review, and writes a journal entry. We want to additionally pipe `git diff HEAD..origin/<branch>` to the `claude` CLI for each behind+smarty repo and aggregate the AI reviews into `/tmp/code-review/<yyyy-mm-dd>.md`.

## New CLI Flag

`-ai` string flag, default `"claude-code"`. Pass `-ai ""` to disable. Only `"claude-code"` supported initially.

```go
// config.go — add to Config struct
AIReviewer string

// config.go — add in ReadConfig()
flags.StringVar(&config.AIReviewer,
    "ai", "claude-code", ""+
        "AI reviewer for behind-origin repos. Set to empty to disable.\n"+
        "Currently supported: \"claude-code\".\n"+
        "-->",
)
```

## Store Branch on GitReport

`GitRevList()` computes the branch but discards it. Persist it for the diff command later.

```go
// git.go — add field to GitReport
Branch string

// git.go — in GitRevList(), after branch := this.GitDefaultBranch()
this.Branch = branch
```

## Track AI-Reviewable Repos

Add `aiReviewable map[string]string` (path → branch) to `GitReviewer`. Populate during `GitAnalyzeAll()`.

```go
// review.go — add to GitReviewer struct and init in NewGitReviewer
aiReviewable map[string]string

// review.go — in GitAnalyzeAll(), after existing map population
if len(report.RevListBehind) > 0 && this.canJournal(report) && report.Branch != "" {
    this.aiReviewable[report.RepoPath] = report.Branch
}
```

Criteria: repo is behind origin AND `canJournal()` (remote contains "smarty", not omitted).

## New File: `ai_review.go`

Contains all AI review logic (~100 lines). Key functions:

### `AIReviewAll()`

- Early-return if `-ai ""`, no eligible repos, unsupported reviewer, or `claude` not in PATH
- `os.MkdirAll("/tmp/code-review", 0755)`
- Open `/tmp/code-review/<yyyy-mm-dd>.md` with `O_WRONLY|O_CREATE|O_APPEND`
- Write run header: `# AI Code Review — <yyyy-mm-dd HH:MM:SS>`
- Iterate eligible repos sequentially, call `runAIReview()` for each
- Write separator + review (or error) per repo
- After closing file, open it in Sublime Text (`subl`)

### `runAIReview(repoPath, branch) (string, error)`

```go
func runAIReview(repoPath, branch string) (string, error) {
    diff, err := execute(repoPath, fmt.Sprintf("git diff HEAD..origin/%s", branch))
    if err != nil {
        return "", fmt.Errorf("git diff failed: %w", err)
    }
    if strings.TrimSpace(diff) == "" {
        return "No differences found.", nil
    }

    prompt := "Review the following git diff. Summarize the changes and flag " +
        "any concerns (bugs, security, style). Be concise."

    cmd := exec.Command("claude", "-p", prompt)
    cmd.Dir = repoPath
    cmd.Stdin = strings.NewReader(diff)
    out, err := cmd.CombinedOutput()
    if err != nil {
        return "", fmt.Errorf("claude failed: %w\n%s", err, string(out))
    }
    return string(out), nil
}
```

### Report format

```markdown
# AI Code Review — 2026-02-24 14:30:00

================================================================================
## /Users/mike/src/github.com/smarty/some-repo
================================================================================

[claude's review]

================================================================================
## /Users/mike/src/github.com/smarty/another-repo
================================================================================

[claude's review]
```

Multiple runs per day append new `# AI Code Review` sections.

### Error handling

| Failure                       | Action                                         |
|-------------------------------|------------------------------------------------|
| `claude` not in PATH          | Log, skip all AI reviews                       |
| `git diff` fails for one repo | Log, write error in report, continue           |
| `claude` fails for one repo   | Log, write error in report, continue           |
| Can't create output dir/file  | Log, skip all AI reviews                       |
| Empty diff                    | Write "No differences found", skip claude call |
| Unsupported `-ai` value       | Log, skip all AI reviews                       |

## Wire Into Main Flow

```go
// main.go
func main() {
    config := ReadConfig(Version)
    reviewer := NewGitReviewer(config)
    reviewer.GitAnalyzeAll()
    reviewer.AIReviewAll()   // NEW — after analysis, before GUI
    reviewer.ReviewAll()
    reviewer.PrintCodeReviewLogEntry()
}
```

AI review runs before GUI review so the report file is ready when the user starts manual inspection.

## Design Decisions

**Sequential execution** — claude CLI calls are API-rate-limited; eligible repo count is typically small (1-5); avoids file-locking complexity.

**Before GUI review** — the AI report is available for reference during manual review.

**Smarty-only scope** — matches the existing journal filter via `canJournal()`.

**No new dependencies** — only standard library (`os`, `os/exec`, `fmt`, `strings`, `time`, `path/filepath`, `sort`, `log`).

## Files Changed

| File           | Change                                                |
|----------------|-------------------------------------------------------|
| `git.go`       | Add `Branch` field, persist in `GitRevList()`         |
| `config.go`    | Add `AIReviewer` field + `-ai` flag                   |
| `review.go`    | Add `aiReviewable` map, populate in `GitAnalyzeAll()` |
| `ai_review.go` | **New** — `AIReviewAll()`, `runAIReview()`, helpers   |
| `main.go`      | Add `reviewer.AIReviewAll()` call                     |

## Verification

1. `make test` — confirm no regressions
2. Run `gitreview -fetch=false -ai ""` — confirm AI review is skipped (existing behavior unchanged)
3. Run `gitreview -fetch=false` in a tree with a repo behind origin — confirm `/tmp/code-review/<date>.md` is created with review content
4. Run again — confirm the file is appended to, not overwritten
