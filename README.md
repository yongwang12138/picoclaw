# Picoclaw 项目

这是一个用于快速启动和配置 Picoclaw 服务的 Go 项目。

## 项目结构

```
picoclaw/
├── main.go                 # 程序入口
├── README.md              # 项目说明文档
├── go.mod                 # Go 模块定义
├── go.sum                 # Go 模块依赖校验
├── builder/               # 构建相关模块
│   └── build.go           # 源代码构建功能
├── config/                # 配置管理模块
│   └── config.go          # 配置相关功能
├── core/                  # 核心运行模块
│   └── core.go            # 服务启动和运行控制
├── ports/                 # 端口管理模块
│   └── ports.go           # 端口检查和清理功能
└── tool/                  # 工具函数模块
    └── tool.go            # 通用工具功能
```

## 功能说明

### config 模块
- 管理 Picoclaw 的配置信息
- 提供默认配置值
- 负责配置文件的读写

### builder 模块
- 检测和准备 Picoclaw 源代码
- 构建二进制文件
- 管理前端资源构建
- 精简网关配置

### core 模块
- 协调各个模块完成服务启动
- 初始化运行时目录结构
- 启动 Picoclaw 服务

### ports 模块
- 检查端口占用情况
- 清理占用端口的进程
- 确保服务端口可用

### tool 模块
- 提供通用工具函数
- 辅助其他模块完成特定任务

## 配置说明

默认配置位于 `internal/config/config.go`，包括：

- **APIBase**: API 基础 URL (默认: https://dashscope.aliyuncs.com/compatible-mode/v1)
- **ModelName**: 模型名称 (默认: deepseek-v3)
- **APIKey**: API 密钥
- **WebPort**: Web 服务端口 (默认: 18800)
- **Host**: 服务主机地址 (默认: 127.0.0.1)
- **GatewayPort**: 网关服务端口 (默认: 18790)
- **MCPURL**: MCP 服务器 URL (默认: http://127.0.0.1:18081/mcp)
- **MCPName**: MCP 服务器名称 (默认: jyddms-mcp)

## 使用方法

### 运行项目

```bash
go run main.go
```

### 构建项目

```bash
go build -o picoclaw.exe
```

### 运行构建后的程序

```bash
./picoclaw.exe
```

## 注意事项

1. 首次运行时会自动下载依赖并构建二进制文件，可能需要较长时间
2. 程序会自动检查并清理占用的端口
3. 配置文件会生成在 `.picoclaw-runtime/config.json`
4. 工作目录位于 `.picoclaw-runtime/home/workspace`

## 依赖项

- github.com/sipeed/picoclaw: Picoclaw 核心库

## 更新 Picoclaw 依赖

```bash
# 查看当前版本
go list -m github.com/sipeed/picoclaw
# 检查最新版本
go list -m -versions github.com/sipeed/picoclaw
# 升级到最新版本
go get -u github.com/sipeed/picoclaw@latest
```
