package runner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"picoclaw/internal/builder"
	"picoclaw/internal/config"
	"picoclaw/internal/fs"
	"picoclaw/internal/ports"

	picocfg "github.com/sipeed/picoclaw/pkg/config"
)

// Run 运行picoclaw服务
// 主要步骤：
// 1. 初始化目录结构
// 2. 检查并清理端口占用
// 3. 准备并构建源代码
// 4. 生成配置文件
// 5. 启动服务
// 返回:
//   - error: 运行失败时返回错误
func Run() error {
	// 获取当前工作目录
	rootDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	// 初始化运行时目录结构
	runtimeDir := filepath.Join(rootDir, ".picoclaw-runtime")
	binDir := filepath.Join(runtimeDir, "bin")
	homeDir := filepath.Join(runtimeDir, "home")
	configPath := filepath.Join(runtimeDir, "config.json")
	sourceDir := filepath.Join(runtimeDir, "picoclaw-src")

	// 创建必要的目录
	if err = fs.MustMkdir(binDir); err != nil {
		return err
	}
	if err = fs.MustMkdir(homeDir); err != nil {
		return err
	}

	// 加载配置
	cfg := config.NewConfig()
	webPortInt, err := strconv.Atoi(cfg.WebPort)
	if err != nil {
		return fmt.Errorf("invalid web port %q: %w", cfg.WebPort, err)
	}

	// 检查端口并清理占用
	if err = ports.EnsureReady(cfg.Host, webPortInt, cfg.GatewayPort); err != nil {
		return err
	}

	// 写入配置文件
	if err = config.WriteConfig(configPath, homeDir, cfg); err != nil {
		return err
	}

	// 检测并准备源代码
	moduleSource, err := builder.DetectPicoclawModuleSource()
	if err != nil {
		return err
	}
	if err = builder.PrepareBuildSource(moduleSource, sourceDir); err != nil {
		return err
	}

	// 精简网关配置
	if err = builder.DisableMatrixGatewayChannel(sourceDir); err != nil {
		return err
	}
	if err = builder.SlimGatewayToPicoOnly(sourceDir); err != nil {
		return err
	}

	// 确保前端资源已构建
	if err = builder.EnsureFrontendDist(sourceDir); err != nil {
		return err
	}

	// 构建二进制文件
	gatewayBinary := filepath.Join(binDir, fs.BinaryName("picoclaw"))
	launcherBinary := filepath.Join(binDir, fs.BinaryName("picoclaw-web"))

	if err = builder.BuildBinaryIfNeeded(sourceDir, gatewayBinary, "./cmd/picoclaw"); err != nil {
		return err
	}
	if err = builder.BuildBinaryFromSource(sourceDir, launcherBinary, "./web/backend"); err != nil {
		return err
	}

	// 打印启动信息
	fmt.Printf("Config ready: %s\n", configPath)
	fmt.Printf("Model: %s (%s)\n", cfg.ModelName, cfg.APIBase)
	fmt.Printf("MCP: %s (%s, type=http)\n", cfg.MCPName, cfg.MCPURL)
	fmt.Printf("Service ports: web=http://%s:%s, gateway=http://%s:%d\n", cfg.Host, cfg.WebPort, cfg.Host, cfg.GatewayPort)
	fmt.Printf("Starting web frontend at: http://%s:%s\n", cfg.Host, cfg.WebPort)
	fmt.Println("Press Ctrl+C to stop.")

	// 启动服务
	cmd := exec.Command(launcherBinary, "-console", "-port", cfg.WebPort, configPath)
	cmd.Env = append(os.Environ(),
		picocfg.EnvBinary+"="+gatewayBinary,
		picocfg.EnvHome+"="+homeDir,
		picocfg.EnvConfig+"="+configPath,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err = cmd.Run(); err != nil {
		return fmt.Errorf("failed to start picoclaw-web: %w", err)
	}
	return nil
}
