package config

import (
	"fmt"
	"os"
	"path/filepath"

	picocfg "github.com/sipeed/picoclaw/pkg/config"
)

// 默认配置常量
const (
	// DefaultAPIBase 默认 API 基础 URL
	DefaultAPIBase = "http://188.18.18.55:8802/v1"
	// DefaultModel 默认模型名称
	DefaultModel = "qwen3.5"
	// DefaultAPIKey 默认 API 密钥
	DefaultAPIKey = "sk-test"
	// DefaultPort 默认 Web 服务端口
	DefaultPort = "18800"
	// DefaultHost 默认服务主机地址
	DefaultHost = "127.0.0.1"
	// GatewayPort 网关服务端口
	GatewayPort = 18790
	// DefaultMCPURL 默认 MCP 服务器 URL
	DefaultMCPURL = "http://127.0.0.1:18888/mcp"
	// DefaultMCPName 默认 MCP 服务器名称
	DefaultMCPName = "jyddms-mcp"
	// DefaultPassword 默认控制台密码
	DefaultPassword = "picoclaw123"
)

// Config 配置结构体，用于管理 Picoclaw 的配置信息
type Config struct {
	APIBase     string // API 基础 URL
	ModelName   string // 模型名称
	APIKey      string // API 密钥
	WebPort     string // Web 服务端口
	Host        string // 主机地址
	GatewayPort int    // 网关端口
	MCPURL      string // MCP 服务器 URL
	MCPName     string // MCP 服务器名称
	Password    string // 控制台密码
}

// NewConfig 创建新的配置实例，使用默认值
func NewConfig() *Config {
	return &Config{
		APIBase:     DefaultAPIBase,
		ModelName:   DefaultModel,
		APIKey:      DefaultAPIKey,
		WebPort:     DefaultPort,
		Host:        DefaultHost,
		GatewayPort: GatewayPort,
		MCPURL:      DefaultMCPURL,
		MCPName:     DefaultMCPName,
		Password:    DefaultPassword,
	}
}

// WriteConfig 将配置写入文件
// 参数:
//   - configPath: 配置文件路径
//   - homeDir: 工作目录路径
//   - cfg: 配置对象
//
// 返回:
//   - error: 写入失败时返回错误
func WriteConfig(configPath, homeDir string, cfg *Config) error {
	// 创建默认配置
	picoCfg := picocfg.DefaultConfig()

	// 设置工作目录
	picoCfg.Agents.Defaults.Workspace = filepath.Join(homeDir, "workspace")
	// 设置模型名称
	picoCfg.Agents.Defaults.ModelName = cfg.ModelName
	// 设置网关主机和端口
	picoCfg.Gateway.Host = cfg.Host
	picoCfg.Gateway.Port = cfg.GatewayPort

	// 配置模型列表
	picoCfg.ModelList = []*picocfg.ModelConfig{
		{
			ModelName: cfg.ModelName,
			Model:     "openai/" + cfg.ModelName,
			APIBase:   cfg.APIBase,
			APIKeys:   picocfg.SimpleSecureStrings(cfg.APIKey),
		},
	}

	// 启用 MCP 工具
	picoCfg.Tools.MCP.Enabled = true
	picoCfg.Tools.MCP.Servers = map[string]picocfg.MCPServerConfig{}
	// 配置 MCP 服务器
	picoCfg.Tools.MCP.Servers[cfg.MCPName] = picocfg.MCPServerConfig{
		Enabled: true,
		Type:    "http",
		URL:     cfg.MCPURL,
	}

	// 确保配置文件目录存在
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// 保存配置文件
	if err := picocfg.SaveConfig(configPath, picoCfg); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	return nil
}
