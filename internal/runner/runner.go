package runner

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	_ "modernc.org/sqlite"

	"picoclaw/internal/builder"
	"picoclaw/internal/config"
	"picoclaw/internal/fs"
	"picoclaw/internal/ports"

	picocfg "github.com/sipeed/picoclaw/pkg/config"
)

// launcherDefaultPasswordHash 预生成的 bcrypt hash (密码：picoclaw123)
// 用于跳过首次启动时的密码设置界面
const launcherDefaultPasswordHash = "JDJhJDEyJFZQLjNrVVRnZ01idFNzZlJzSjlwMnVHcTQ3S2pUcHZYME80WXEyRWdUUUdGUWc1dTJnaWZX"

// writeLauncherConfig 预先写入默认密码的 launcher-config.json
func writeLauncherConfig(configPath string) error {
	launcherConfig := map[string]interface{}{
		"port":                      18800,
		"public":                    false,
		"dashboard_password_hash":   launcherDefaultPasswordHash,
	}

	data, err := json.MarshalIndent(launcherConfig, "", "  ")
	if err != nil {
		return err
	}

	launcherPath := filepath.Join(configPath, "launcher-config.json")
	return os.WriteFile(launcherPath, data, 0o644)
}

// initSQLiteAuthDB 初始化 SQLite 数据库并预置默认密码
// picoclaw 使用 SQLite 存储 dashboard 密码，需要在启动前创建并写入 hash
func initSQLiteAuthDB(homeDir string) error {
	dbPath := filepath.Join(homeDir, "launcher-auth.db")

	// 先删除旧数据库文件，确保干净启动
	os.Remove(dbPath)

	// 创建新的 SQLite 数据库并初始化 schema
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// 执行 schema 初始化（从 picoclaw 源代码中复制）
	schema := `
CREATE TABLE IF NOT EXISTS dashboard_credentials (
    id          INTEGER PRIMARY KEY CHECK (id = 1),
    bcrypt_hash TEXT NOT NULL
);
`
	_, err = db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	// 直接插入带默认密码 hash 的记录
	_, err = db.Exec("INSERT INTO dashboard_credentials (id, bcrypt_hash) VALUES (1, ?)", launcherDefaultPasswordHash)
	if err != nil {
		return fmt.Errorf("failed to insert password: %w", err)
	}

	fmt.Println("✓ SQLite auth DB initialized with default password")
	return nil
}

// Run 运行 picoclaw 服务
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

	// 清理 workspace 目录中的缓存和会话数据
	workspaceDir := filepath.Join(homeDir, "workspace")
	cacheDir := filepath.Join(workspaceDir, "cache")
	historyDir := filepath.Join(workspaceDir, "history")

	// 删除缓存目录
	if err = fs.CleanDir(cacheDir); err != nil {
		fmt.Printf("Warning: failed to clean cache directory: %v\n", err)
	}
	// 删除历史目录
	if err = fs.CleanDir(historyDir); err != nil {
		fmt.Printf("Warning: failed to clean history directory: %v\n", err)
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

	// 预先写入 launcher-config.json 带默认密码 (避免首次启动要求设置密码)
	if err = writeLauncherConfig(runtimeDir); err != nil {
		fmt.Printf("Warning: failed to write launcher config: %v\n", err)
	}

	// 初始化 SQLite 数据库并预置默认密码
	if err = initSQLiteAuthDB(homeDir); err != nil {
		fmt.Printf("Warning: failed to init auth DB: %v\n", err)
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
