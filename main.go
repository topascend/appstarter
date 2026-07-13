package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// App 配置文件中的应用结构
type App struct {
	Name    string `json:"name"`    // 应用名称
	Path    string `json:"path"`    // 可执行文件路径（.exe / .bat / .cmd）
	WorkDir string `json:"workdir"` // 工作目录（可选）
}

const (
	serviceName = "AppStarterService" // 服务名称
	logFile     = "appstarter.log"    // 日志文件
	configFile  = "appstarter.json"   // 配置文件
	startDelay  = 3 * time.Second     // 前台模式下逐个启动间隔
)

var (
	apps          []App       // 所有应用
	logger        *log.Logger // 日志记录器
	logFileHandle *os.File    // 日志文件句柄
)

// 初始化日志：同时写入文件和标准输出（前台模式）
func initLogger(foreground bool) error {
	var writers []io.Writer
	if foreground {
		writers = append(writers, os.Stdout)
	}
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("无法打开日志文件: %v", err)
	}
	logFileHandle = file
	writers = append(writers, file)
	multiWriter := io.MultiWriter(writers...)
	logger = log.New(multiWriter, "", log.LstdFlags)
	return nil
}

// 加载配置文件
func loadConfig() error {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("读取 %s 失败: %v", configFile, err)
	}
	if err := json.Unmarshal(data, &apps); err != nil {
		return fmt.Errorf("解析 %s 失败: %v", configFile, err)
	}
	if len(apps) == 0 {
		return fmt.Errorf("%s 中没有定义任何应用", configFile)
	}
	return nil
}

// 启动单个应用，返回 *exec.Cmd
var outputMu sync.Mutex

func pipeOutput(prefix string, r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		outputMu.Lock()
		logger.Printf("[%s] %s", prefix, scanner.Text())
		outputMu.Unlock()
	}
}

func startApp(app App, foreground bool) (*exec.Cmd, error) {
	workDir := filepath.Clean(app.WorkDir)
	if workDir == "." {
		var err error
		workDir, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("获取当前目录失败: %v", err)
		}
	}
	cmdPath := filepath.Clean(app.Path)
	ext := strings.ToLower(filepath.Ext(cmdPath))
	var cmd *exec.Cmd
	if ext == ".bat" || ext == ".cmd" {
		cmd = exec.Command("cmd", "/c", cmdPath)
	} else {
		cmd = exec.Command(cmdPath)
	}
	cmd.Dir = workDir

	if foreground {
		stdout, _ := cmd.StdoutPipe()
		stderr, _ := cmd.StderrPipe()
		go pipeOutput(app.Name, stdout)
		go pipeOutput(app.Name, stderr)
	}
	cmd.Stdin = nil

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("启动失败: %v", err)
	}
	logger.Printf("启动应用: %s, 路径: %s, 工作目录: %s, PID: %d",
		app.Name, cmdPath, workDir, cmd.Process.Pid)
	return cmd, nil
}

// 启动全部应用，返回所有 *exec.Cmd 切片
func startAllApps(foreground bool) ([]*exec.Cmd, error) {
	var cmds []*exec.Cmd
	for i, app := range apps {
		if i > 0 && foreground {
			time.Sleep(startDelay)
		}
		cmd, err := startApp(app, foreground)
		if err != nil {
			logger.Printf("启动 %s 失败: %v", app.Name, err)
			continue
		}
		cmds = append(cmds, cmd)
	}
	if len(cmds) == 0 {
		return nil, fmt.Errorf("没有成功启动任何应用")
	}
	return cmds, nil
}

// 终止所有进程
func killAll(cmds []*exec.Cmd) {
	for _, cmd := range cmds {
		if cmd.Process != nil {
			if err := cmd.Process.Kill(); err != nil {
				logger.Printf("终止 PID %d 失败: %v", cmd.Process.Pid, err)
			} else {
				logger.Printf("已终止 PID %d", cmd.Process.Pid)
			}
		}
	}
}

// ============ 服务模式 ============

type serviceHandler struct {
	cmds []*exec.Cmd
}

func (s *serviceHandler) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	changes <- svc.Status{State: svc.StartPending}

	// 启动所有应用
	cmds, err := startAllApps(false)
	if err != nil {
		logger.Printf("服务启动应用失败: %v", err)
		changes <- svc.Status{State: svc.Stopped, Win32ExitCode: 1}
		return false, 1
	}
	s.cmds = cmds
	logger.Printf("服务已启动，共运行 %d 个进程", len(cmds))

	changes <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}

	// 等待控制命令
	for {
		c := <-r
		switch c.Cmd {
		case svc.Stop, svc.Shutdown:
			logger.Printf("收到停止信号，正在终止所有子进程...")
			changes <- svc.Status{State: svc.StopPending}
			killAll(s.cmds)
			changes <- svc.Status{State: svc.Stopped}
			return false, 0
		default:
			// 忽略其他命令
		}
	}
}

func runService() error {
	err := svc.Run(serviceName, &serviceHandler{})
	if err != nil {
		return fmt.Errorf("服务运行失败: %v", err)
	}
	return nil
}

// ============ 前台运行模式 ============

func runForeground() {
	logger.Printf("前台模式启动，加载配置...")
	if err := loadConfig(); err != nil {
		logger.Fatalf("加载配置失败: %v", err)
	}
	logger.Printf("共加载 %d 个应用", len(apps))

	cmds, err := startAllApps(true)
	if err != nil {
		logger.Fatalf("启动应用失败: %v", err)
	}
	logger.Printf("所有应用已启动，按 Ctrl+C 终止")

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	logger.Printf("收到中断信号，正在终止所有子进程...")
	killAll(cmds)
	logger.Printf("已退出")
}

// ============ 服务安装/卸载 ============

func installService() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取可执行文件路径失败: %v", err)
	}
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("连接服务管理器失败: %v", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err == nil {
		s.Close()
		return fmt.Errorf("服务 %s 已存在", serviceName)
	}
	s, err = m.CreateService(serviceName, exePath, mgr.Config{
		DisplayName: "应用批量启动服务",
		StartType:   mgr.StartAutomatic,
	})
	if err != nil {
		return fmt.Errorf("创建服务失败: %v", err)
	}
	defer s.Close()
	logger.Printf("服务 %s 安装成功，请通过服务管理器启动或使用 'net start %s'", serviceName, serviceName)
	return nil
}

func uninstallService() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("连接服务管理器失败: %v", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("服务 %s 不存在", serviceName)
	}
	defer s.Close()
	if err = s.Delete(); err != nil {
		return fmt.Errorf("删除服务失败: %v", err)
	}
	logger.Printf("服务 %s 已卸载", serviceName)
	return nil
}

// ============ 主函数 ============

func main() {
	// 解析命令行参数（第一个参数为动作）
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "用法:\n")
		fmt.Fprintf(os.Stderr, "  %s install    安装为 Windows 服务\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s uninstall  卸载服务\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s run        前台运行（默认）\n", os.Args[0])
	}
	var action string
	if len(os.Args) > 1 {
		action = strings.ToLower(os.Args[1])
	} else {
		action = "run"
	}

	// 初始化日志（服务模式下无前台输出）
	foreground := (action == "run" || action == "")
	if err := initLogger(foreground); err != nil {
		fmt.Fprintf(os.Stderr, "初始化日志失败: %v\n", err)
		os.Exit(1)
	}
	defer logFileHandle.Close()

	switch action {
	case "install":
		if err := installService(); err != nil {
			logger.Fatalf("安装服务失败: %v", err)
		}
	case "uninstall":
		if err := uninstallService(); err != nil {
			logger.Fatalf("卸载服务失败: %v", err)
		}
	case "run", "":
		// 检查是否以服务方式启动（通常不会，但以防万一）
		if b, _ := svc.IsWindowsService(); b {
			// 如果以服务方式运行，则进入服务模式
			if err := runService(); err != nil {
				logger.Fatalf("服务运行失败: %v", err)
			}
			return
		}
		runForeground()
	default:
		logger.Fatalf("未知动作: %s，支持: install, uninstall, run", action)
	}
}
