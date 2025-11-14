package main

import (
	"bufio"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func collectGitRepositories(root string) (gits []string) {
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}
		if isGitRepository(path, info) {
			gits = append(gits, path)
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	return gits
}

func isGitRepository(path string, item os.FileInfo) bool {
	if !item.IsDir() {
		return false
	}

	_, err := os.Stat(filepath.Join(path, ".git"))
	if os.IsNotExist(err) {
		return false
	}

	return true
}

func execute(dir, command string) (string, error) {
	args := strings.Fields(command)
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func prompt(message string) string {
	log.Println(message)
	s := bufio.NewScanner(os.Stdin)
	s.Scan()
	return s.Text()
}
