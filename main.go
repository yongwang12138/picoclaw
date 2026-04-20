package main

import (
	"fmt"
	"os"

	"picoclaw/internal/runner"
)

// main 程序入口函数
// 启动picoclaw服务，如果启动失败则打印错误信息并退出
func main() {
	if err := runner.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
