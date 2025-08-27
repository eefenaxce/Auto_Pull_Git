package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ========= 统一入口：保证仓库存在且更新 =========
func (r *Repo) ensureGit() error {
	// 第一次 clone
	if _, err := os.Stat(filepath.Join(r.CloneDir, ".git")); os.IsNotExist(err) {
		if err := os.MkdirAll(r.CloneDir, 0755); err != nil {
			return err
		}
		// 第一次 clone
		if err := r.gitClone(); err != nil {
			return err
		}
		// 第一次克隆后也需要构建
		return r.build()
	}

	// 已存在就 pull
	if err := r.gitPull(); err != nil {
		return err
	}
	return nil
}

// ========= clone =========
func (r *Repo) gitClone() error {
	// 1. SSH 认证：注入 GIT_SSH_COMMAND
	if r.Auth != nil && r.Auth.Type == "ssh" && r.Auth.SSHKey != "" {
		sshCmd := fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=no", r.Auth.SSHKey)
		if r.Auth.SSHPass != "" {
			sshCmd = fmt.Sprintf("sshpass -p '%s' %s", r.Auth.SSHPass, sshCmd)
		}
		os.Setenv("GIT_SSH_COMMAND", sshCmd)
	}

	// 2. HTTPS 私有仓库：自动在 URL 里嵌入用户名+token
	url := r.authURL()

	cmd := exec.Command("git", "clone", "-b", r.Branch, url, ".")
	cmd.Dir = r.CloneDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ========= pull =========
func (r *Repo) gitPull() error {
	// HTTPS 私有仓库：保证 remote 用的是带 token 的 url
	if r.Auth != nil && r.Auth.Type == "https" {
		cmd := exec.Command("git", "remote", "set-url", "origin", r.authURL())
		cmd.Dir = r.CloneDir
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	// SSH 环境变量已在 clone 时注入，pull 复用即可
	cmd := exec.Command("git", "pull", "origin", r.Branch)
	cmd.Dir = r.CloneDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ========= 工具函数 =========

// 根据认证方式组装最终 clone/pull 用的 url
func (r *Repo) authURL() string {
	if r.Auth == nil {
		return r.URL
	}
	switch r.Auth.Type {
	case "https":
		if r.Auth.Username != "" && r.Auth.Token != "" {
			return strings.Replace(r.URL, "https://",
				fmt.Sprintf("https://%s:%s@", r.Auth.Username, r.Auth.Token), 1)
		}
	}
	return r.URL // ssh 或公开仓库
}

// 当前 HEAD commit
func (r *Repo) currentCommit() (string, error) {
	// 先执行 git fetch origin 确保本地远程分支信息是最新的
	fetchCmd := exec.Command("git", "fetch", "origin")
	fetchCmd.Dir = r.CloneDir
	log.Printf("[%s] executing git fetch origin in %s", r.Name, r.CloneDir)
	fetchOut, fetchErr := fetchCmd.CombinedOutput()
	if fetchErr != nil {
		log.Printf("[%s] git fetch origin failed: %v\nOutput: %s", r.Name, fetchErr, string(fetchOut))
		// 不返回错误，继续尝试获取当前 commit，因为 fetch 失败不代表 rev-parse 也会失败
	}
	log.Printf("[%s] git fetch origin output:\n%s", r.Name, string(fetchOut))

	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = r.CloneDir
	log.Printf("[%s] executing git rev-parse HEAD in %s", r.Name, r.CloneDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[%s] git rev-parse HEAD failed: %v\nOutput: %s", r.Name, err, string(out))
		return "", err
	}
	commit := strings.TrimSpace(string(out))
	log.Printf("[%s] current commit: %s", r.Name, commit)
	return commit, nil
}

// 是否有新 commit
func (r *Repo) hasNewCommit() (bool, error) {
	// 获取本地 HEAD commit
	localHead, err := r.currentCommit()
	if err != nil {
		return false, fmt.Errorf("failed to get local HEAD commit: %w", err)
	}

	// 获取远程 HEAD commit
	remoteHeadCmd := exec.Command("git", "rev-parse", "origin/"+r.Branch)
	remoteHeadCmd.Dir = r.CloneDir
	log.Printf("[%s] executing git rev-parse origin/%s in %s", r.Name, r.Branch, r.CloneDir)
	remoteOut, remoteErr := remoteHeadCmd.CombinedOutput()
	if remoteErr != nil {
		log.Printf("[%s] git rev-parse origin/%s failed: %v\nOutput: %s", r.Name, r.Branch, remoteErr, string(remoteOut))
		return false, fmt.Errorf("failed to get remote HEAD commit: %w", remoteErr)
	}
	remoteHead := strings.TrimSpace(string(remoteOut))
	log.Printf("[%s] remote HEAD commit: %s", r.Name, remoteHead)

	// 比较本地 HEAD 和远程 HEAD
	if localHead != remoteHead {
		log.Printf("[%s] local HEAD (%s) is different from remote HEAD (%s), new commit detected.", r.Name, localHead, remoteHead)
		return true, nil
	}

	log.Printf("[%s] local HEAD (%s) is same as remote HEAD (%s), no new commit.", r.Name, localHead, remoteHead)
	return false, nil
}

// 把当前 commit 写回文件，用于下次对比
func (r *Repo) saveCommit() error {
	cur, err := r.currentCommit()
	if err != nil {
		return err
	}
	return os.WriteFile(r.LastCommitFile(), []byte(cur), 0644)
}
