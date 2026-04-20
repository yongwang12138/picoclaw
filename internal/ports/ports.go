package ports

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// EnsureReady 确保端口准备就绪
// 检查并清理指定端口上的占用进程
// 参数:
//   - host: 主机地址
//   - webPort: Web服务端口
//   - gatewayPort: 网关服务端口
// 返回:
//   - error: 检查或清理失败时返回错误
func EnsureReady(host string, webPort, gatewayPort int) error {
	fmt.Printf("Checking ports before startup: web=%s:%d gateway=%s:%d\n", host, webPort, host, gatewayPort)
	if err := killPortOccupiers(webPort); err != nil {
		return err
	}
	if err := killPortOccupiers(gatewayPort); err != nil {
		return err
	}
	return nil
}

// killPortOccupiers 终止占用指定端口的进程
// 参数:
//   - port: 端口号
// 返回:
//   - error: 终止失败时返回错误
func killPortOccupiers(port int) error {
	// 查找占用端口的进程
	pids, err := findListeningPIDs(port)
	if err != nil {
		return fmt.Errorf("failed to inspect port %d: %w", port, err)
	}
	if len(pids) == 0 {
		return nil
	}

	// 终止所有占用端口的进程
	for _, pid := range pids {
		fmt.Printf("Port %d is occupied by PID %d, terminating it...\n", port, pid)
		if err = killProcess(pid); err != nil {
			return err
		}
	}

	// 等待进程终止并验证
	time.Sleep(300 * time.Millisecond)
	rest, err := findListeningPIDs(port)
	if err != nil {
		return fmt.Errorf("failed to re-check port %d: %w", port, err)
	}
	if len(rest) > 0 {
		return fmt.Errorf("port %d is still occupied after cleanup: pids=%v", port, rest)
	}
	return nil
}

// findListeningPIDs 查找监听指定端口的进程ID
// 参数:
//   - port: 端口号
// 返回:
//   - []int: 进程ID列表
//   - error: 查找失败时返回错误
func findListeningPIDs(port int) ([]int, error) {
	cmd := exec.Command("netstat", "-ano", "-p", "tcp")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("netstat failed: %w: %s", err, strings.TrimSpace(string(out)))
	}

	targetSuffix := ":" + strconv.Itoa(port)
	seen := map[int]struct{}{}
	var pids []int

	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// 跳过空行和标题行
		if line == "" || strings.HasPrefix(line, "Proto") {
			continue
		}

		fields := strings.Fields(line)
		// 验证TCP监听状态
		if len(fields) < 5 || strings.ToUpper(fields[0]) != "TCP" {
			continue
		}
		local := fields[1]
		state := strings.ToUpper(fields[3])
		if state != "LISTENING" {
			continue
		}
		// 检查是否匹配目标端口
		if !strings.HasSuffix(local, targetSuffix) && !strings.Contains(local, "]"+targetSuffix) {
			continue
		}

		// 解析进程ID
		pid, convErr := strconv.Atoi(fields[4])
		if convErr != nil || pid <= 0 {
			continue
		}
		// 避免重复添加
		if _, ok := seen[pid]; ok {
			continue
		}
		seen[pid] = struct{}{}
		pids = append(pids, pid)
	}
	if err = scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan netstat output failed: %w", err)
	}
	return pids, nil
}

// killProcess 终止指定进程
// 参数:
//   - pid: 进程ID
// 返回:
//   - error: 终止失败时返回错误
func killProcess(pid int) error {
	// 不终止自身进程
	if pid == os.Getpid() {
		return nil
	}

	// 根据操作系统选择不同的终止方式
	if runtime.GOOS == "windows" {
		cmd := exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/F", "/T")
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to kill PID %d: %w: %s", pid, err, strings.TrimSpace(string(out)))
		}
		return nil
	}

	// Unix-like系统
	p, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find PID %d: %w", pid, err)
	}
	if err = p.Kill(); err != nil {
		return fmt.Errorf("failed to kill PID %d: %w", pid, err)
	}
	return nil
}
