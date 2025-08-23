package main

import (
	"log"
	"time"
)

func main() {
	cfg, err := LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// 启动后先执行一轮
	log.Println("autobuild initial run...")
	for _, repo := range cfg.Repos {
		if err := handleRepo(&repo); err != nil {
			log.Printf("[%s] initial run error: %v", repo.Name, err)
		}
	}

	// 开始定时轮询
	ticker := time.NewTicker(cfg.Interval())
	defer ticker.Stop()
	log.Printf("autobuild started, interval=%v", cfg.Interval())

	// 在首轮循环里
	if cfg.SelfUpdate != nil && cfg.SelfUpdate.Enable {
		if err := handleSelfUpdate(cfg.SelfUpdate); err != nil {
			log.Printf("[self-update] initial run error: %v", err)
		}
	}

	// 在定时循环里
	for range ticker.C {
		for _, repo := range cfg.Repos {
			if err := handleRepo(&repo); err != nil {
				log.Printf("[%s] error: %v", repo.Name, err)
			}
		}
		if cfg.SelfUpdate != nil && cfg.SelfUpdate.Enable {
			if err := handleSelfUpdate(cfg.SelfUpdate); err != nil {
				log.Printf("[self-update] error: %v", err)
			}
		}
	}
}
func handleRepo(r *Repo) error {
	log.Printf("[%s] checking...", r.Name)

	if err := r.ensureGit(); err != nil {
		return err
	}

	newCommit, err := r.hasNewCommit()
	if err != nil {
		return err
	}
	if !newCommit {
		log.Printf("[%s] already up to date", r.Name)
		return nil
	}

	log.Printf("[%s] new commit detected, building...", r.Name)
	if err := r.build(); err != nil {
		return err
	}

	if err := r.saveCommit(); err != nil {
		return err
	}

	log.Printf("[%s] build success, artifact ready in %s", r.Name, r.OutputDir)
	return nil
}
