# Auto_Pull_Git

`Auto_Pull_Git` 是一个用 Go 语言编写的强大且灵活的自动化工具，旨在简化和自动化 Git 仓库的持续集成与部署流程。它能够定期监控指定的 Git 仓库，自动拉取最新代码，执行预定义的构建命令，并根据需要重启相关服务。该工具特别适用于需要自动化部署小型服务、前端应用或任何基于 Git 仓库进行版本控制的项目。

## 核心特性

- **多仓库支持**: 轻松配置和管理多个独立的 Git 仓库，每个仓库都可以有自己的拉取、构建和部署策略。
- **灵活的认证机制**: 全面支持 HTTPS 和 SSH 两种主流的 Git 认证方式。
  - **HTTPS**: 可通过嵌入用户名和个人访问令牌 (PAT) 或密码进行认证，适用于 GitHub, GitLab, Bitbucket 等平台。
  - **SSH**: 支持通过 SSH 私钥文件进行认证，并可选择提供 SSH 密钥的密码，确保私有仓库的访问安全。
- **自动化工作流**: 实现从代码更新到服务部署的全自动化流程。
  - **智能检测**: 定时检查 Git 仓库是否有新的提交，避免不必要的构建。
  - **自动拉取**: 一旦检测到新提交，自动执行 `git pull` 或 `git clone` 操作，确保本地代码库与远程同步。
  - **自定义构建**: 支持执行任意数量的自定义构建命令（例如 `npm install`, `go build`, `docker build` 等），以适应各种项目类型。
  - **服务重启**: 构建成功后，可配置执行自定义的重启命令，实现服务的平滑更新或重新启动。
- **自更新能力**: `Auto_Pull_Git` 自身也具备从 Git 仓库拉取最新代码并进行自更新的能力，确保工具本身始终保持最新状态，无需手动干预。
- **日志记录**: 提供详细的日志输出，方便跟踪每次拉取、构建和部署的状态，快速定位问题。

## 项目结构

```
.  
├── build.go        # 负责项目的构建和重启逻辑
├── config.go       # 定义配置结构体和加载配置的函数
├── config.yaml     # 示例配置文件，用于定义仓库、认证和构建规则
├── go.mod          # Go 模块依赖管理文件
├── go.sum          # Go 模块依赖校验文件
├── main.go         # 程序入口，负责加载配置、启动定时任务和处理仓库更新
├── repo.go         # 定义仓库相关操作，如 Git clone/pull、commit 检查等
└── self.go         # 负责工具自身的自更新逻辑
```

## 快速开始

### 1. 克隆项目

首先，将 `Auto_Pull_Git` 项目克隆到您的本地机器：

```bash
git clone https://github.com/eefenaxce/Auto_Pull_Git.git
cd Auto_Pull_Git
```

### 2. 配置 `config.yaml`

`config.yaml` 是 `Auto_Pull_Git` 的核心配置文件。您需要根据您的项目需求修改此文件。以下是一个详细的配置示例和说明：

```yaml
# 全局日志级别（debug|info|warn|error）
log_level: info

# 每轮检查间隔（分钟）
interval_minutes: 1

# 仓库列表
repos:
  - name: "my-frontend-app"
    # 支持 https / ssh
    url: "https://github.com/your-org/your-frontend-app.git"
    branch: "main"

    auth:
      type: "https"        # https | ssh
      # https 方式：用户名 + PAT/Token
      username: "oauth2" # 对于 GitHub 通常是 "oauth2" 或您的用户名
      token:    "ghp_xxxxxxxxxxxxxxxxxxxx" # 您的个人访问令牌 (Personal Access Token)
      # ssh 方式：私钥路径 + 可选 passphrase
      # ssh_key: "/home/user/.ssh/id_rsa" # SSH 私钥的绝对路径
      # ssh_passphrase: "your_ssh_passphrase"  # 如果私钥有密码，请填写；如无密码可省略
      
    # 本地克隆目录（会被自动创建）
    clone_dir: "./workspace/my-frontend-app" # 仓库克隆到本地的路径
    # 解压 / 拉取后放置源码的位置
    source_dir: "./workspace/my-frontend-app/src" # 实际源码所在的子目录，构建命令将在此目录执行
    # 编译成功后二进制输出目录
    output_dir: "./workspace/my-frontend-app/dist" # 构建产物（如编译后的二进制、打包文件）的输出路径
    # 编译命令（支持多行；用数组）
    build_cmd:
      - npm install # 安装依赖
      - npm run build # 执行构建命令
    # 编译后产物名称（可选，用于日志）
    artifact_name: "frontend-dist"
    # 服务重启命令（可选，构建成功后执行）
    restart_cmd: "pm2 restart my-frontend-app" # 例如使用 pm2 重启 Node.js 应用

  - name: "my-backend-service"
    url: "git@github.com:your-org/your-backend-service.git"
    branch: "dev"
    auth:
      type: "ssh"
      ssh_key: "/root/.ssh/id_rsa_backend"
    clone_dir: "./workspace/my-backend-service"
    source_dir: "./workspace/my-backend-service"
    output_dir: "./workspace/my-backend-service/bin"
    build_cmd:
      - go mod tidy
      - go build -o my-backend-service .
    artifact_name: "my-backend-service"
    restart_cmd: "systemctl restart my-backend-service"

# 自更新专用字段（**不在 repos 内**）
self_update:
  enable: true # 是否启用自更新
  url: "https://github.com/eefenaxce/Auto_Pull_Git.git" # Auto_Pull_Git 自身的 Git 仓库 URL
  branch: "main" # 监控 Auto_Pull_Git 自身的分支
  # 以下三个目录全部用 **当前目录** (.)，表示在工具运行的当前目录进行自更新
  clone_dir: "."
  source_dir: "."
  output_dir: "."
  build_cmd:
    - go build -o Auto_Pull_Git-new . # 自更新的构建命令，生成新的二进制文件
```

**配置注意事项**：

- `clone_dir`, `source_dir`, `output_dir` 建议使用相对路径，它们将相对于 `Auto_Pull_Git` 的运行目录。
- `build_cmd` 和 `restart_cmd` 中的命令会根据操作系统自动选择 `sh -c` (Linux/macOS) 或 `cmd /C` (Windows) 执行。
- 对于 HTTPS 认证，GitHub 推荐使用 [个人访问令牌 (PAT)](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token) 而非您的 GitHub 密码。
- 对于 SSH 认证，请确保 `ssh_key` 指向的私钥文件具有正确的权限 (例如 `chmod 600 your_ssh_key`)。

### 3. 构建 `Auto_Pull_Git`

在项目根目录执行 Go 构建命令：

```bash
go build -o Auto_Pull_Git .
```

这将在当前目录生成一个名为 `Auto_Pull_Git` (或 `Auto_Pull_Git.exe` 在 Windows 上) 的可执行文件。

### 4. 运行 `Auto_Pull_Git`

```bash
./Auto_Pull_Git # Linux/macOS
.\Auto_Pull_Git.exe # Windows PowerShell
```

程序启动后，将根据 `config.yaml` 中定义的 `interval_minutes` 定时检查并更新您的仓库。

## 常见问题 (FAQ)

### Q1: 为什么我的构建命令没有执行？

- **A1**: 请检查 `config.yaml` 中 `build_cmd` 字段的语法是否正确，确保命令是有效的 shell 命令。同时，检查 `source_dir` 是否指向了正确的目录，因为构建命令会在 `source_dir` 下执行。

### Q2: 如何处理 Git 认证失败？

- **A2**: 
  - **HTTPS**: 确认 `username` 和 `token` (或密码) 是否正确。对于 GitHub，请确保 PAT 具有访问私有仓库的权限。
  - **SSH**: 检查 `ssh_key` 路径是否正确，私钥文件是否存在且权限是否正确 (通常是 `600`)。如果私钥有密码，请确保 `ssh_passphrase` 填写正确。
  - 尝试在命令行手动执行 `git clone` 或 `git pull` 命令，使用相同的认证信息，以排除网络或凭证问题。

### Q3: `Auto_Pull_Git` 自身如何更新？

- **A3**: 如果 `self_update.enable` 设置为 `true`，`Auto_Pull_Git` 会定期检查自身仓库的更新。一旦检测到新版本，它会拉取代码，执行 `self_update.build_cmd` 中定义的构建命令（通常是重新编译自身），然后尝试热重启，用新编译的二进制文件替换当前运行的实例。这实现了无缝的工具升级。

### Q4: 如何查看详细日志？

- **A4**: 您可以通过修改 `config.yaml` 中的 `log_level` 为 `debug` 来获取更详细的日志输出。日志会直接打印到控制台。

### Q5: 可以在 Windows 上运行吗？

- **A5**: 是的，`Auto_Pull_Git` 是用 Go 语言编写的，具有良好的跨平台特性。您可以在 Windows、Linux 和 macOS 上编译和运行它。命令执行时会自动适配操作系统的 shell (cmd 或 sh)。

## 贡献

欢迎任何形式的贡献！如果您有任何功能建议、Bug 报告或代码改进，请随时提交 Issue 或 Pull Request。