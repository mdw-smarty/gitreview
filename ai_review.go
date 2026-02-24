package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func (this *GitReviewer) AIReviewAll() {
	if this.config.AIReviewer == "" {
		return
	}
	if len(this.aiReviewable) == 0 {
		return
	}
	if this.config.AIReviewer != "claude-code" {
		log.Printf("Unsupported AI reviewer: %q (supported: \"claude-code\")", this.config.AIReviewer)
		return
	}
	if _, err := exec.LookPath("claude"); err != nil {
		log.Println("claude not found in PATH, skipping AI reviews.")
		return
	}

	dir := "/tmp/code-review"
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("Could not create AI review directory %s: %v", dir, err)
		return
	}

	now := time.Now()
	filePath := filepath.Join(dir, now.Format("2006-01-02")+".md")
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("Could not open AI review file %s: %v", filePath, err)
		return
	}
	defer func() { _ = file.Close() }()

	_, _ = fmt.Fprintf(file, "# AI Code Review — %s\n", now.Format("2006-01-02 15:04:05"))

	paths := make([]string, 0, len(this.aiReviewable))
	for p := range this.aiReviewable {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	for _, repoPath := range paths {
		branch := this.aiReviewable[repoPath]
		_, _ = fmt.Fprintf(file, "\n================================================================================\n")
		_, _ = fmt.Fprintf(file, "## %s\n", repoPath)
		_, _ = fmt.Fprintf(file, "================================================================================\n\n")

		review, err := runAIReview(repoPath, branch)
		if err != nil {
			log.Printf("AI review error for %s: %v", repoPath, err)
			_, _ = fmt.Fprintf(file, "ERROR: %v\n", err)
		} else {
			_, _ = fmt.Fprintf(file, "%s\n", review)
		}
	}

	log.Printf("AI review written to %s", filePath)
	_ = exec.Command("subl", filePath).Start()
}

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
