package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func handleSelfUpdate(su *SelfUpdate) error {
	// 确保输出目录存在
	if err := os.MkdirAll(su.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// 获取更新前的提交哈希
	oldCommitHash, err := getCurrentCommitHash(su.CloneDir)
	if err != nil {
		log.Printf("[self-update] warning: could not get old commit hash: %v", err)
	}

	// 1. 拉取
	if err := gitPullOrClone(su.URL, su.Branch, su.CloneDir); err != nil {
		return err
	}

	// 获取更新后的提交哈希
	newCommitHash, err := getCurrentCommitHash(su.CloneDir)
	if err != nil {
		log.Printf("[self-update] warning: could not get new commit hash: %v", err)
	}

	// 如果提交哈希相同，则表示没有更新，不需要重启
	if oldCommitHash != "" && newCommitHash != "" && oldCommitHash == newCommitHash {
		log.Println("[self-update] no update found, skipping restart")
		return nil
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

	// 3. 复制编译后的文件到 OutputDir
	// 查找 SourceDir 下的二进制文件并重命名
	actualSrcPath, err := findAndRenameBinary(su.SourceDir, su.ArtifactName)
	if err != nil {
		return fmt.Errorf("failed to find or rename compiled binary in %s: %w", su.SourceDir, err)
	}

	// 确定目标文件路径
	dstPath := filepath.Join(su.OutputDir, filepath.Base(actualSrcPath))

	if err := copyFile(actualSrcPath, dstPath); err != nil {
		return fmt.Errorf("failed to copy compiled binary from %s to %s: %w", actualSrcPath, dstPath, err)
	}
	log.Printf("[self-update] copied compiled binary from %s to %s", actualSrcPath, dstPath)

	// 设置新二进制文件的执行权限
	if err := os.Chmod(dstPath, 0755); err != nil {
		log.Printf("[self-update] warning: failed to set executable permission for %s: %v", dstPath, err)
	}

	// 4. 热重启
	newBin := dstPath

	// Ensure newBin is an absolute path
	absoluteNewBin, err := filepath.Abs(newBin)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for new binary: %w", err)
	}
	newBin = absoluteNewBin

	if _, err := os.Stat(newBin); err != nil {
		return err
	}
	log.Printf("[self-update] exec %s", newBin)
	time.Sleep(500 * time.Millisecond) // 日志 flush
	cmd := exec.Command(absoluteNewBin, os.Args[1:]...)
	cmd.Dir = su.OutputDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start new process: %w", err)
	}

	// 确保新进程有足够时间启动
	time.Sleep(1 * time.Second)
	os.Exit(0)
	return nil
}

// clone 或 pull 时都在当前目录（su.CloneDir == "."）
func gitPullOrClone(url, branch, dir string) error {
	if _, err := os.Stat(filepath.Join(dir, ".git")); os.IsNotExist(err) {
		cloneCmd := fmt.Sprintf("git clone -b %s %s %s", branch, url, dir)
		parts := splitCmd(cloneCmd)
		cmd := exec.Command(parts[0], parts[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	pullCmd := fmt.Sprintf("git -C %s pull origin %s", dir, branch)
	parts := splitCmd(pullCmd)
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// getCurrentCommitHash 获取指定目录的当前 Git 提交哈希
func getCurrentCommitHash(dir string) (string, error) {
	cmdStr := fmt.Sprintf("git -C %s rev-parse HEAD", dir)
	parts := splitCmd(cmdStr)
	cmd := exec.Command(parts[0], parts[1:]...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current commit hash: %w", err)
	}
	return string(output), nil
}
