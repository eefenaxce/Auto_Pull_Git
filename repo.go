package main

import (
	"fmt"
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
		return r.gitClone()
	}
	// 已存在就 pull
	return r.gitPull()
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
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = r.CloneDir
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

// 是否有新 commit
func (r *Repo) hasNewCommit() (bool, error) {
	last, err := os.ReadFile(r.LastCommitFile())
	if err != nil {
		return true, nil // 第一次
	}
	cur, err := r.currentCommit()
	if err != nil {
		return false, err
	}
	return string(last) != cur, nil
}

// 把当前 commit 写回文件，用于下次对比
func (r *Repo) saveCommit() error {
	cur, err := r.currentCommit()
	if err != nil {
		return err
	}
	return os.WriteFile(r.LastCommitFile(), []byte(cur), 0644)
}
