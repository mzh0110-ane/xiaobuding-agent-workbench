# 小布丁 Xiaobuding

小布丁是面向 Windows 的中文多 Agent 协作工作台，建立在 [Kandev](https://github.com/kdlbs/kandev) 之上。它将项目调查、Codex / DeepSeek Harness / ZCode 的任务分工、实际检查、独立审查、证据留存、自动返工与人工验收组织在同一工作流程中。

## 当前能力

- 使用本机已有的 Codex ChatGPT 登录及原生配置；可配置 DeepSeek Harness 和 ZCode 原生连接。可用模型、思考强度和技能以本机原生工具及实际调用结果为准。
- 可以先只读调查，再针对具体项目开启修改；普通项目文件夹可创建本地 Git 存档。修改先在工作副本中进行，由用户验收后应用。
- 每批最多进行三次自动实施尝试；后续可以继续开启新批次。任务记录、证据、验证与 Codex 独立审查用于辅助核对，但不能保证模型永不出错。
- 中文界面支持持续提问、文件附件、运行记录、模型安排和任务结果查看。大项目、原生技能与附件支持有实际环境和格式边界。

## 部署状态

本仓库是源码发布版本，**不是一键安装包**。当前的集成和测试主要面向 Windows；不同系统、Codex 更新和第三方 Agent 版本可能需要适配。Kandev 完整源码和上游许可一并保留；其原始说明见 [README.upstream.md](README.upstream.md)。

### 准备

安装 Git、Go（版本以 [apps/backend/go.mod](apps/backend/go.mod) 为准）、支持 CGO 的 C 编译器，以及已登录的 Codex CLI 或桌面应用。要使用 Harness 或 ZCode，还需要分别安装并配置它们自己的原生工具；本项目不会提供或复制其账号凭据。请确保本机可正常运行所选模型。

### 构建后端

在 PowerShell 中，从本仓库根目录运行：

```powershell
New-Item -ItemType Directory -Force -Path dist | Out-Null
Push-Location apps/backend
$env:CGO_ENABLED = '1'
go build -tags fts5 -o ../../dist/kandev.exe ./cmd/kandev
go build -tags fts5 -o ../../dist/agentctl.exe ./cmd/agentctl
Pop-Location
```

`agentctl.exe` 需要和 `kandev.exe` 放在同一目录。Windows 的 CGO 编译器、依赖下载和 Kandev 其他运行组件需自行准备；如果构建失败，请根据本机 Go 和编译器输出排查。以上命令尚非针对所有环境验证的一键部署流程。

### 创建本机配置并运行

1. 将 [examples/reliable-settings.example.json](examples/reliable-settings.example.json) 复制到一个专用目录，文件名改为 `reliable-settings.json`。
2. 把示例里的路径改为本机**绝对路径**；`codex_path` 必须指向可执行的 Codex 程序，`codex_home` 及连接里的 `home` 指向已登录的 Codex 用户目录。模型名改为当前账号实际可用的模型。
3. 在同一 PowerShell 中设置 `KANDEV_HOME_DIR` 为该配置文件所在目录，设置 `KANDEV_SERVER_HOST=127.0.0.1` 和 `KANDEV_SERVER_PORT=38429`，再启动 `dist/kandev.exe __backend`，通过浏览器访问 `http://127.0.0.1:38429/reliable`。如果端口已被占用，请使用其他本地端口。

配置文件是本机私有数据；不要提交登录文件、API 密钥、任务数据库或本地工作副本。示例只配置 Codex，其他 Agent 需要按现有适配器及原生工具设置；配置字段见 [connections.go](apps/backend/internal/reliable/connections.go)。实际执行前检查本机权限和工具状态。修改实施阶段默认允许配置的原生工具获得完全访问权限；只读调查使用对应的只读策略。请只在信任的本机环境使用。

## 测试

后端包的针对性测试：

```powershell
Push-Location apps/backend
go test -tags fts5 ./internal/reliable
Pop-Location
```

可选的 ZCode 原生联调测试需要自行设置 `XIAOBUDING_ZCODE_EXECUTABLE`、`XIAOBUDING_ZCODE_ENTRY`、`XIAOBUDING_ZCODE_HOME`、`XIAOBUDING_ZCODE_PROVIDER` 和 `XIAOBUDING_ZCODE_MODEL`，并显式启用相应测试开关；这些值只放在本机环境中。完整测试所需的软件、账号、网络及资源请根据实际测试输出准备，跳过不代表通过。

## 许可证和素材

本项目包含 Kandev 及其衍生代码，按 [GNU AGPL-3.0](LICENSE) 发布，并保留原有版权和许可声明。上游项目：<https://github.com/kdlbs/kandev>。

Codex、DeepSeek、ZCode 的名称和图标是各自权利人的标识；头像和背景为本项目附带的素材。代码的开源许可不自动授予商标或第三方素材的再使用权，请自行核实适用权限。