package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

	// 判断是否是 Node.js 项目
	if _, err := os.Stat(filepath.Join(r.SourceDir, "package.json")); err == nil {
		log.Printf("[%s] detected Node.js project, copying /dist to OutputDir", r.Name)
		// 如果是 Node.js 项目，则复制 /dist 目录到 OutputDir
		srcDistDir := filepath.Join(r.SourceDir, "dist")
		if _, err := os.Stat(srcDistDir); os.IsNotExist(err) {
			return fmt.Errorf("node.js project detected but /dist directory not found in %s", r.SourceDir)
		}
		if err := copyDir(srcDistDir, r.OutputDir); err != nil {
			return fmt.Errorf("failed to copy /dist directory: %w", err)
		}
		return nil
	} else if os.IsNotExist(err) {
		// 如果不是 Node.js 项目，则复制编译后的文件到 OutputDir
		// 查找 SourceDir 下的二进制文件并重命名
		actualSrcPath, err := findAndRenameBinary(r.SourceDir, r.ArtifactName)
		if err != nil {
			return fmt.Errorf("failed to find or rename compiled binary in %s: %w", r.SourceDir, err)
		}

		// 确定目标文件路径
		dstPath := filepath.Join(r.OutputDir, filepath.Base(actualSrcPath))

		if err := copyFile(actualSrcPath, dstPath); err != nil {
			return fmt.Errorf("failed to copy compiled binary from %s to %s: %w", actualSrcPath, dstPath, err)
		}
		log.Printf("[%s] copied compiled binary from %s to %s", r.Name, actualSrcPath, dstPath)

		// 设置新二进制文件的执行权限
		if err := os.Chmod(dstPath, 0755); err != nil {
			log.Printf("[%s] warning: failed to set executable permission for %s: %v", r.Name, dstPath, err)
		}
	} else {
		return fmt.Errorf("failed to check for package.json: %w", err)
	}

	if r.RestartCmd != "" {
		log.Printf("[%s] restarting service: %s", r.Name, r.RestartCmd)
		parts := splitCmd(r.RestartCmd)
		cmd := exec.Command(parts[0], parts[1:]...)
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

// hasExtension 检查文件名是否包含扩展名
func hasExtension(filename string) bool {
	return filepath.Ext(filename) != "" && filepath.Ext(filename) != filename
}

// copyFile 复制文件
func copyFile(src, dst string) error {
	// 尝试删除目标文件，以解决 "text file busy" 或文件已存在的问题
	if err := os.Remove(dst); err != nil && !os.IsNotExist(err) {
		log.Printf("warning: failed to remove existing destination file %s: %v", dst, err)
		// 不返回错误，继续尝试复制，因为有时删除失败但复制可能成功
	}

	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", src, err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", dst, err)
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return fmt.Errorf("failed to copy file from %s to %s: %w", src, dst, err)
	}
	return out.Close()
}

// findAndRenameBinary 在指定目录中查找编译后的二进制文件，并根据需要重命名
// 返回实际的源文件路径
func findAndRenameBinary(dir, targetArtifactName string) (string, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("failed to read source directory %s: %w", dir, err)
	}

	var foundBinaries []string
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		filePath := filepath.Join(dir, file.Name())
		// 检查是否是可执行文件
		if runtime.GOOS == "windows" {
			if filepath.Ext(file.Name()) == ".exe" {
				foundBinaries = append(foundBinaries, filePath)
			}
		} else {
			info, err := file.Info()
			if err != nil {
				log.Printf("warning: failed to get file info for %s: %v", filePath, err)
				continue
			}
			// 检查文件权限是否包含可执行位
			if info.Mode().IsRegular() && (info.Mode().Perm()&0111) != 0 {
				foundBinaries = append(foundBinaries, filePath)
			}
		}
	}

	if len(foundBinaries) == 0 {
		return "", fmt.Errorf("no compiled binary found in %s", dir)
	}
	if len(foundBinaries) > 1 {
		// 如果找到多个可执行文件，尝试匹配 targetArtifactName
		for _, binPath := range foundBinaries {
			if filepath.Base(binPath) == targetArtifactName || (runtime.GOOS == "windows" && filepath.Base(binPath) == targetArtifactName+".exe") {
				return binPath, nil
			}
		}
		return "", fmt.Errorf("multiple compiled binaries found in %s, and none match target artifact name %s", dir, targetArtifactName)
	}

	// 只有一个二进制文件
	actualSrcPath := foundBinaries[0]
	expectedDstName := targetArtifactName
	if runtime.GOOS == "windows" && !hasExtension(targetArtifactName) {
		expectedDstName += ".exe"
	}

	if filepath.Base(actualSrcPath) != expectedDstName {
		newPath := filepath.Join(dir, expectedDstName)
		log.Printf("renaming compiled binary from %s to %s", actualSrcPath, newPath)
		if err := os.Rename(actualSrcPath, newPath); err != nil {
			return "", fmt.Errorf("failed to rename compiled binary from %s to %s: %w", actualSrcPath, newPath, err)
		}
		return newPath, nil
	}

	return actualSrcPath, nil
}

// copyDir 递归复制目录
func copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source directory %s: %w", src, err)
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to create destination directory %s: %w", dst, err)
	}

	dir, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source directory %s: %w", src, err)
	}
	defer dir.Close()

	objects, err := dir.Readdir(-1)
	if err != nil {
		return fmt.Errorf("failed to read source directory %s: %w", src, err)
	}

	for _, obj := range objects {
		srcPath := filepath.Join(src, obj.Name())
		dstPath := filepath.Join(dst, obj.Name())

		if obj.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	return nil
}
