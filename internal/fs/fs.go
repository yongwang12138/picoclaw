package fs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// MustMkdir 创建目录，如果失败则返回错误
// 参数:
//   - path: 要创建的目录路径
// 返回:
//   - error: 创建失败时返回错误
func MustMkdir(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", path, err)
	}
	return nil
}

// BinaryName 根据操作系统返回正确的二进制文件名
// Windows系统会添加.exe后缀
// 参数:
//   - name: 基础文件名
// 返回:
//   - string: 完整的二进制文件名
func BinaryName(name string) string {
	if os.PathSeparator == '\\' {
		return name + ".exe"
	}
	return name
}

// CopyDir 递归复制整个目录
// 参数:
//   - src: 源目录路径
//   - dst: 目标目录路径
// 返回:
//   - error: 复制失败时返回错误
func CopyDir(src, dst string) error {
	// 验证源目录
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat %s: %w", src, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("source is not a directory: %s", src)
	}

	// 创建目标目录
	if err = os.MkdirAll(dst, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dst, err)
	}

	// 读取源目录内容
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("readdir %s: %w", src, err)
	}

	// 递归复制所有文件和子目录
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			// 递归复制子目录
			if err = CopyDir(srcPath, dstPath); err != nil {
				return err
			}
			continue
		}
		// 复制文件
		if err = copyFile(srcPath, dstPath); err != nil {
			return err
		}
	}
	return nil
}

// copyFile 复制单个文件
// 参数:
//   - src: 源文件路径
//   - dst: 目标文件路径
// 返回:
//   - error: 复制失败时返回错误
func copyFile(src, dst string) error {
	// 打开源文件
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open %s: %w", src, err)
	}
	defer in.Close()

	// 创建目标文件
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("create %s: %w", dst, err)
	}
	defer out.Close()

	// 执行文件复制
	if _, err = io.Copy(out, in); err != nil {
		return fmt.Errorf("copy %s -> %s: %w", src, dst, err)
	}
	return nil
}
