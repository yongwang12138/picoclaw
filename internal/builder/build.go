package builder

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"picoclaw/internal/fs"
)

// DetectPicoclawModuleSource 检测picoclaw模块的源代码路径
// 使用go list命令获取模块路径
// 返回:
//   - string: 模块源代码路径
//   - error: 检测失败时返回错误
func DetectPicoclawModuleSource() (string, error) {
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/sipeed/picoclaw")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to detect github.com/sipeed/picoclaw module path: %w: %s", err, strings.TrimSpace(string(out)))
	}

	dir := strings.TrimSpace(string(out))
	if dir == "" {
		return "", fmt.Errorf("failed to detect github.com/sipeed/picoclaw module path: empty path")
	}
	if _, err = os.Stat(dir); err != nil {
		return "", fmt.Errorf("picoclaw module path is not accessible: %w", err)
	}
	return dir, nil
}

// PrepareBuildSource 准备构建源代码
// 将模块源代码复制到目标目录，并设置工作区资源
// 参数:
//   - moduleSource: 模块源代码路径
//   - target: 目标目录路径
// 返回:
//   - error: 准备失败时返回错误
func PrepareBuildSource(moduleSource, target string) error {
	// 如果目标目录不存在，则复制源代码
	if _, err := os.Stat(target); os.IsNotExist(err) {
		if err = fs.CopyDir(moduleSource, target); err != nil {
			return fmt.Errorf("failed to copy source tree: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to access local picoclaw source copy: %w", err)
	}

	// 设置工作区资源
	workspaceSource := filepath.Join(target, "workspace")
	workspaceTarget := filepath.Join(target, "cmd", "picoclaw", "internal", "onboard", "workspace")
	if _, err := os.Stat(workspaceSource); err != nil {
		return fmt.Errorf("workspace assets missing in local picoclaw source: %w", err)
	}

	// 删除旧的工作区资源并复制新的
	_ = os.RemoveAll(workspaceTarget)
	if err := fs.CopyDir(workspaceSource, workspaceTarget); err != nil {
		return fmt.Errorf("failed to stage onboard workspace assets: %w", err)
	}
	return nil
}

// BuildBinaryFromSource 从源代码构建二进制文件
// 参数:
//   - sourceRoot: 源代码根目录
//   - output: 输出二进制文件路径
//   - pkg: 要构建的包路径
// 返回:
//   - error: 构建失败时返回错误
func BuildBinaryFromSource(sourceRoot, output, pkg string) error {
	fmt.Printf("Building %s ... (first run may take several minutes)\n", pkg)
	cmd := exec.Command("go", "build", "-buildvcs=false", "-v", "-o", output, pkg)
	cmd.Dir = sourceRoot
	// 设置国内镜像加速
	cmd.Env = append(os.Environ(),
		"GOPROXY=https://mirrors.aliyun.com/goproxy/,https://goproxy.cn,https://proxy.golang.com.cn,direct",
		"GOSUMDB=off",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("build failed: %s: %w", pkg, err)
	}
	return nil
}

// BuildBinaryIfNeeded 如果需要则构建二进制文件
// 如果二进制文件已存在则跳过构建
// 参数:
//   - sourceRoot: 源代码根目录
//   - output: 输出二进制文件路径
//   - pkg: 要构建的包路径
// 返回:
//   - error: 构建失败时返回错误
func BuildBinaryIfNeeded(sourceRoot, output, pkg string) error {
	if info, err := os.Stat(output); err == nil && !info.IsDir() {
		fmt.Printf("Using existing binary: %s\n", output)
		return nil
	}
	return BuildBinaryFromSource(sourceRoot, output, pkg)
}

// EnsureFrontendDist 确保前端资源已构建
// 如果前端资源不存在则执行构建
// 参数:
//   - sourceRoot: 源代码根目录
// 返回:
//   - error: 构建失败时返回错误
func EnsureFrontendDist(sourceRoot string) error {
	frontendDir := filepath.Join(sourceRoot, "web", "frontend")
	backendDistIndex := filepath.Join(sourceRoot, "web", "backend", "dist", "index.html")

	// 检查前端资源是否已存在
	if _, err := os.Stat(backendDistIndex); err == nil {
		fmt.Println("Using existing frontend dist assets")
		return nil
	}

	// 构建前端资源
	fmt.Println("Frontend dist assets missing, building real web UI ...")
	nodeModules := filepath.Join(frontendDir, "node_modules")
	if _, err := os.Stat(nodeModules); os.IsNotExist(err) {
		// 安装依赖
		if err = runCmd(frontendDir, "pnpm", "install", "--frozen-lockfile"); err != nil {
			return err
		}
	} else if err != nil {
		return fmt.Errorf("failed to check frontend node_modules: %w", err)
	}
	// 构建前端
	if err := runCmd(frontendDir, "pnpm", "build:backend"); err != nil {
		return err
	}

	// 验证构建结果
	if _, err := os.Stat(backendDistIndex); err != nil {
		return fmt.Errorf("frontend build did not produce backend/dist/index.html: %w", err)
	}
	return nil
}

// SlimGatewayToPicoOnly 精简网关为仅Pico模式
// 移除不需要的通道导入，减小二进制文件大小
// 参数:
//   - sourceRoot: 源代码根目录
// 返回:
//   - error: 操作失败时返回错误
func SlimGatewayToPicoOnly(sourceRoot string) error {
	gatewayFile := filepath.Join(sourceRoot, "pkg", "gateway", "gateway.go")
	data, err := os.ReadFile(gatewayFile)
	if err != nil {
		return fmt.Errorf("failed to read gateway source for slimming: %w", err)
	}

	// 移除不需要的通道导入
	lines := strings.Split(string(data), "\n")
	out := make([]string, 0, len(lines))
	removed := 0
	for _, line := range lines {
		if strings.Contains(line, `_ "github.com/sipeed/picoclaw/pkg/channels/`) {
			removed++
			continue
		}
		out = append(out, line)
	}

	// 如果没有移除任何内容则直接返回
	if removed == 0 {
		return nil
	}

	// 写入修改后的文件
	_ = os.Chmod(gatewayFile, 0644)
	if err = os.WriteFile(gatewayFile, []byte(strings.Join(out, "\n")), 0644); err != nil {
		return fmt.Errorf("failed to write slimmed gateway source: %w", err)
	}
	fmt.Printf("Slimmed gateway imports to Pico-only mode (removed %d channel imports)\n", removed)
	return nil
}

// DisableMatrixGatewayChannel 禁用Matrix网关通道
// 删除Matrix网关通道相关文件
// 参数:
//   - sourceRoot: 源代码根目录
// 返回:
//   - error: 操作失败时返回错误
func DisableMatrixGatewayChannel(sourceRoot string) error {
	matrixFile := filepath.Join(sourceRoot, "pkg", "gateway", "channel_matrix.go")
	if _, err := os.Stat(matrixFile); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to check matrix gateway file: %w", err)
	}

	// 删除Matrix网关通道文件
	if err := os.Chmod(matrixFile, 0644); err != nil {
		return fmt.Errorf("failed to make matrix gateway file writable: %w", err)
	}
	if err := os.Remove(matrixFile); err != nil {
		return fmt.Errorf("failed to disable matrix gateway channel: %w", err)
	}
	fmt.Println("Disabled matrix gateway channel for Pico-only build")
	return nil
}

// runCmd 在指定目录执行命令
// 参数:
//   - dir: 工作目录
//   - name: 命令名称
//   - args: 命令参数
// 返回:
//   - error: 执行失败时返回错误
func runCmd(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command failed: %s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}
