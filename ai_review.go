package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func (this *GitReviewer) PrepareAIReviewDir() string {
	if this.config.AIReviewer == "" {
		return ""
	}
	if len(this.aiReviewable) == 0 {
		return ""
	}
	if this.config.AIReviewer != "claude-code" {
		log.Printf("Unsupported AI reviewer: %q (supported: \"claude-code\")", this.config.AIReviewer)
		return ""
	}
	if _, err := exec.LookPath("claude"); err != nil {
		log.Println("claude not found in PATH, skipping AI reviews.")
		return ""
	}

	dir := filepath.Join(this.aiAuditFolder, "code-review", time.Now().Format("2006-01-02-150405"))
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("Could not create AI review directory %s: %v", dir, err)
		return ""
	}
	return dir
}

func (this *GitReviewer) AIReviewRepo(repoPath, branch, outputDir string) {
	log.Printf("AI reviewing %s (branch: %s)", repoPath, branch)

	now := time.Now()
	review, err := runAIReview(repoPath, branch)

	filename := deriveRepoFilename(repoPath) + ".md"
	filePath := filepath.Join(outputDir, filename)

	file, fileErr := os.Create(filePath)
	if fileErr != nil {
		log.Printf("Could not create AI review file %s: %v", filePath, fileErr)
		return
	}
	defer func() { _ = file.Close() }()

	_, _ = fmt.Fprintf(file, "# AI Code Review: %s\n\n", repoPath)
	_, _ = fmt.Fprintf(file, "**Branch:** %s\n", branch)
	_, _ = fmt.Fprintf(file, "**Date:** %s\n\n", now.Format("2006-01-02 15:04:05"))

	if err != nil {
		log.Printf("AI review error for %s: %v", repoPath, err)
		_, _ = fmt.Fprintf(file, "ERROR: %v\n", err)
	} else {
		_, _ = fmt.Fprintf(file, "%s\n", review)
	}

	log.Printf("AI review written to %s", filePath)
	_ = exec.Command("subl", filePath).Start()
}

func deriveRepoFilename(repoPath string) string {
	parts := strings.Split(filepath.Clean(repoPath), string(filepath.Separator))
	if len(parts) >= 2 {
		return parts[len(parts)-2] + "--" + parts[len(parts)-1]
	}
	return parts[len(parts)-1]
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
