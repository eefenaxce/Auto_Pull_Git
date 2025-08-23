package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

func handleSelfUpdate(su *SelfUpdate) error {
	// 1. 拉取
	if err := gitPullOrClone(su.URL, su.Branch, su.CloneDir); err != nil {
		return err
	}

	// 2. 编译
	for _, cmdStr := range su.BuildCmd {
		parts := splitCmd(cmdStr)
		cmd := exec.Command(parts[0], parts[1:]...)
		cmd.Dir = su.SourceDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	// 3. 热重启
	newBin := filepath.Join(su.OutputDir, "autobuild-new")
	if _, err := os.Stat(newBin); err != nil {
		return err
	}
	log.Printf("[self-update] exec %s", newBin)
	time.Sleep(500 * time.Millisecond) // 日志 flush
	return syscall.Exec(newBin, os.Args, os.Environ())
}

// clone 或 pull 时都在当前目录（su.CloneDir == "."）
func gitPullOrClone(url, branch, dir string) error {
	if _, err := os.Stat(filepath.Join(dir, ".git")); os.IsNotExist(err) {
		return exec.Command("git", "clone", "-b", branch, url, dir).Run()
	}
	cmd := exec.Command("git", "-C", dir, "pull", "origin", branch)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
