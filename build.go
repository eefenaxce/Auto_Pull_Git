package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
)

func (r *Repo) build() error {
	if err := os.MkdirAll(r.OutputDir, 0755); err != nil {
		return err
	}

	for _, cmdStr := range r.BuildCmd {
		parts := splitCmd(cmdStr)
		if len(parts) == 0 {
			continue
		}
		cmd := exec.Command(parts[0], parts[1:]...)
		cmd.Dir = r.SourceDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("build failed: %w", err)
		}
	}

	if r.RestartCmd != "" {
		log.Printf("[%s] restarting service: %s", r.Name, r.RestartCmd)
		cmd := exec.Command("sh", "-c", r.RestartCmd)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("restart failed: %w", err)
		}
	}

	return nil
}

// 简单切分命令为可执行文件和参数
func splitCmd(s string) []string {
	if os.PathSeparator == '\\' { // Windows
		return []string{"cmd", "/C", s}
	}
	return []string{"sh", "-c", s}
}
