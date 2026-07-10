# AppStarter - 批量应用启动工具（Windows 服务版）
### 一款用 Go 编写的 Windows 工具，可一次性启动配置文件中定义的所有应用程序（.exe 和 .bat/.cmd），并支持前台运行或安装为 Windows 服务实现开机自启。所有启动/停止事件均记录日志，便于运维追踪。

### 功能亮点
#### 📦 批量启动 – 读取 appStart.json，同时启动所有已配置的应用。

#### 🔧 支持多种可执行文件 – 直接运行 .exe，通过 cmd /c 调用 .bat / .cmd。

#### 📋 日志记录 – 所有事件（启动、PID、终止）写入 appstarter.log，带时间戳。

#### 🖥️ 前台模式 – 方便调试，实时显示子进程输出，按 Ctrl+C 一键终止全部。

#### ⚙️ Windows 服务模式 – 安装为系统服务，开机自启，后台静默运行。

#### 🔄 简单管理 – 通过 install / uninstall 命令轻松安装/卸载服务。

#### 🚦 退出码传递 – 前台模式下，子进程异常终止时，工具返回对应退出码。

### 快速开始
#### 1. 编译
#### 在 Windows 环境下，确保已安装 Go 1.16+，并下载依赖包：

```
go build -o appstarter.exe starter.go
```

### 2. 准备配置文件
#### 在程序同目录下创建 appStart.json，示例内容如下：
```
json
[
  {
    "name": "Notepad",
    "path": "C:\\Windows\\notepad.exe",
    "workdir": "C:\\"
  },
  {
    "name": "MyBackend",
    "path": "D:\\Services\\backend.exe",
    "workdir": "D:\\Services"
  },
  {
    "name": "InitScript",
    "path": "C:\\scripts\\setup.bat",
    "workdir": "C:\\scripts"
  }
]
```
name – 应用名称（仅用于日志标识，不要求唯一）。

path – 可执行文件的绝对路径（推荐）或相对路径（相对于程序启动目录）。

workdir – 程序启动后的工作目录（可选，留空则使用程序当前目录）。

### 3. 运行
#### 🔹 前台运行（调试）
```
appstarter.exe run
```
或直接

```
appstarter.exe
```
此时所有应用同时启动，输出打印到控制台。按 Ctrl+C 会终止所有子进程并退出。

### 🔹 安装为 Windows 服务（需管理员权限）
以管理员身份运行命令提示符
```
appstarter.exe install
```
成功后，服务 AppStarterService 将创建并设为“自动”启动类型。
启动服务：
```
net start AppStarterService
```

停止服务：
```
net stop AppStarterService
```
服务模式下，所有子进程在后台运行，不显示控制台输出，日志记录到 appstarter.log。

### 🔹 卸载服务（需管理员权限）
```
appstarter.exe uninstall
```

### 日志文件
程序运行时会自动在当前目录生成 appstarter.log，记录内容包括：

启动应用的时间、名称、路径、工作目录、PID。

启动失败的错误信息。

收到终止信号及终止进程的 PID。

服务安装/卸载结果。

日志采用追加模式，不会覆盖历史记录。

命令详解
命令	说明
```
run 或省略参数	前台运行模式，可调试查看子进程输出
install	安装 Windows 服务（需管理员权限）
uninstall	卸载已安装的服务（需管理员权限）
```
注意：若程序以服务方式启动（由 SCM 调用），会自动进入服务模式，无需手动指定。

配置文件详解
文件名必须为 appStart.json，且与可执行文件在同一目录。

支持 JSON 数组格式，每个元素包含三个字段。

字段说明：

name：字符串，便于识别。

path：字符串，支持反斜杠转义（\\）或正斜杠 /。

workdir：字符串，为空时默认使用程序启动目录（服务模式下为 C:\Windows\System32，建议明确指定）。

常见问题（FAQ）
Q：为什么服务启动后，我的应用没有运行？
A：请检查：

配置文件路径是否正确（是否在程序目录）。

path 是否使用绝对路径，服务启动目录可能与预期不同。

查看 appstarter.log 中的错误信息。

Q：如何让服务随系统启动而自动运行？
A：安装服务时已默认设为“自动”启动，无需额外设置。若需延迟启动，可在服务管理器中修改。

Q：服务模式下，子进程的输出去哪儿了？
A：默认丢弃。如需保留子进程日志，可修改代码重定向到单独文件（或直接让子程序自身记录日志）。

Q：想终止某个单独的子进程怎么办？
A：目前工具只支持统一终止所有子进程（Ctrl+C 或停止服务）。如需精细控制，建议使用任务管理器。

Q：可以添加启动参数吗？
A：当前版本不支持为每个应用单独配置参数。如有需要，可在 path 字段中包含参数（如 "C:\\app.exe -flag"），但需注意引号转义。更优雅的方式是后续扩展。

Q：安装服务时提示“拒绝访问”？
A：请确保以管理员身份运行命令提示符。

开发与贡献
本项目专注于 Windows 平台，使用 Go 标准库及 golang.org/x/sys/windows 实现服务管理。
欢迎提交 Issue 或 PR，提出改进建议（如增加进程守护、参数传递等）。

许可证
MIT License

