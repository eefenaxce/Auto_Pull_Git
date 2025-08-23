package main

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	LogLevel        string      `yaml:"log_level"`
	IntervalMinutes int         `yaml:"interval_minutes"`
	Repos           []Repo      `yaml:"repos"`
	SelfUpdate      *SelfUpdate `yaml:"self_update"`
}

type SelfUpdate struct {
	Enable    bool     `yaml:"enable"`
	URL       string   `yaml:"url"`
	Branch    string   `yaml:"branch"`
	CloneDir  string   `yaml:"clone_dir"`
	SourceDir string   `yaml:"source_dir"`
	OutputDir string   `yaml:"output_dir"`
	BuildCmd  []string `yaml:"build_cmd"`
}

type Repo struct {
	Name         string   `yaml:"name"`
	URL          string   `yaml:"url"`
	Branch       string   `yaml:"branch"`
	Auth         *Auth    `yaml:"auth,omitempty"`
	CloneDir     string   `yaml:"clone_dir"`
	SourceDir    string   `yaml:"source_dir"`
	OutputDir    string   `yaml:"output_dir"`
	BuildCmd     []string `yaml:"build_cmd"`
	RestartCmd   string   `yaml:"restart_cmd,omitempty"`
	ArtifactName string   `yaml:"artifact_name"`
}

type Auth struct {
	Type     string `yaml:"type"`     // https | ssh
	Username string `yaml:"username"` // https 时必填
	Token    string `yaml:"token"`    // https 时必填
	SSHKey   string `yaml:"ssh_key"`  // ssh 时必填
	SSHPass  string `yaml:"ssh_passphrase,omitempty"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.IntervalMinutes <= 0 {
		cfg.IntervalMinutes = 5
	}
	return &cfg, nil
}

func (r *Repo) String() string {
	return fmt.Sprintf("%s (%s)", r.Name, r.URL)
}

func (r *Repo) LockFile() string {
	return r.CloneDir + "/.autobuild.lock"
}

func (r *Repo) LastCommitFile() string {
	return r.CloneDir + "/.last_commit"
}

func (c *Config) Interval() time.Duration {
	return time.Duration(c.IntervalMinutes) * time.Minute
}
