# cpm - Claude Profile Manager

[English](README.md)

快速切换 Claude 桌面应用的多账户登录，同时保持 Claude Code 会话、MCP 配置等数据共享。支持多账户同时在线。

支持 **macOS**、**Linux** 和 **Windows**。

## 安装

**一键安装**（macOS / Linux）：

```bash
curl -fsSL https://raw.githubusercontent.com/liasica/cpm/master/install.sh | sh
```

**从源码安装**（需要 Go）：

```bash
go install github.com/liasica/cpm/cmd/cpm@latest
```

**手动下载**：从 [Releases](https://github.com/liasica/cpm/releases) 页面下载对应平台的二进制文件。

## 使用流程

### 1. 保存第一个账户

在 Claude 桌面应用中确认已登录账户 A，然后执行：

```bash
cpm add personal
```

### 2. 保存第二个账户

在 Claude 桌面应用中**登出账户 A，登录账户 B**，然后执行：

```bash
cpm add work
```

后续需要更多账户，重复此步骤即可。

### 3. 切换账户

```bash
cpm switch personal   # 切换到 personal
cpm switch work       # 切换到 work
```

切换时会自动关闭 Claude → 替换认证文件 → 重新启动。

### 4. 多账户同时在线（双开）

```bash
cpm open work      # 以 work profile 启动第二个 Claude 实例
cpm close work     # 关闭该实例
```

主 Claude 应用保持当前 profile 不变。每个 `cpm open` 的实例都有独立窗口，MCP 配置和应用设置会自动从主应用同步。

### 命令一览

| 命令 | 别名 | 说明 |
|------|------|------|
| `cpm` | | 显示所有 profiles |
| `cpm add <name>` | | 保存当前登录为 profile |
| `cpm switch <name>` | `cpm sw` | 切换到指定 profile（单实例模式） |
| `cpm open <name>` | | 启动额外的 Claude 实例（多实例模式） |
| `cpm close <name>` | | 关闭额外的 Claude 实例 |
| `cpm list` | `cpm ls` | 列出所有 profiles |
| `cpm current` | | 显示当前 profile 名称 |
| `cpm rename <old> <new>` | | 重命名 profile |
| `cpm remove <name>` | `cpm rm` | 删除 profile |

### 选项

- `cpm switch <name> --no-restart`：切换后不自动重启 Claude

## 工作原理

Claude 桌面应用是一个 [Electron](https://www.electronjs.org/) 应用。和所有 Electron 应用一样，它将数据存储在 macOS 的 `~/Library/Application Support/Claude/` 目录下。该目录同时包含**认证状态**（当前登录的账户）和**本地应用数据**（会话记录、配置、窗口布局等）。

### `cpm switch` —— 单实例模式

cpm 利用了认证数据和应用数据的分离。执行 `cpm add <name>` 时，它只将认证相关的文件快照到 `~/.config/cpm/profiles/<name>/`。执行 `cpm switch` 时：

1. **关闭 Claude** —— 通过 AppleScript 优雅退出（超时后回退到 `pkill`）
2. **回写当前认证文件** —— 将当前 Claude 目录中的认证文件保存到当前活跃的 profile
3. **恢复目标认证文件** —— 将目标 profile 的认证文件复制回 Claude 数据目录
4. **重新启动 Claude** —— 除非指定了 `--no-restart`

认证文件以外的所有内容保持不变，所以 Claude Code 会话、MCP 工具配置和窗口布局可以无缝延续。

### `cpm open` —— 多实例模式

Electron 支持通过 `--user-data-dir` 参数启动额外的实例，新进程会使用一个完全独立的数据目录。执行 `cpm open <name>` 时：

1. **创建**独立的数据目录 `~/.config/cpm/instances/<name>/`
2. **复制** profile 的认证文件（Cookies、Local Storage、Session Storage）到实例目录
3. **同步**共享配置（`claude_desktop_config.json`、`config.json`）从主 Claude 数据目录，确保 MCP 服务器和应用偏好可用
4. **启动**新的 Claude 进程：`open -na Claude --args --user-data-dir=<实例目录>`

每个实例都是独立的 Electron 进程，拥有自己的窗口和数据。主 Claude 应用不受影响。

`cpm close <name>` 通过 `pgrep -f user-data-dir=<路径>` 找到实例进程，发送 `SIGTERM` 优雅退出，6 秒内未退出则升级为 `SIGKILL`。

### 切换与共享的文件

| 切换（按账户隔离） | 用途 | 共享（所有账户通用） | 用途 |
|---|---|---|---|
| `Cookies` | Session Cookies（SQLite）—— 核心登录凭证 | `claude-code-sessions/` | Claude Code 对话历史 |
| `Cookies-journal` | Cookies 的 SQLite WAL 日志 | `claude_desktop_config.json` | MCP 服务器配置 |
| `Local Storage/` | 补充认证 token（LevelDB） | `config.json` | 应用偏好设置 |
| `Session Storage/` | 临时会话数据（LevelDB） | `window-state.json` | 窗口大小和位置 |
| | | `Cache/`、`GPUCache/` 等 | Chromium 缓存 |
| | | 其他所有文件 | 崩溃报告、扩展等 |

### 文件布局

```
~/.config/cpm/
├── state.json                # 记录当前活跃的 profile
├── profiles/                 # 认证文件快照（switch 和 open 共用）
│   ├── personal/
│   │   ├── Cookies           # SQLite 数据库，存储 session cookies
│   │   ├── Cookies-journal
│   │   ├── Local Storage/    # LevelDB —— 认证 token
│   │   └── Session Storage/  # LevelDB —— 临时会话数据
│   └── work/
│       └── ...
└── instances/                # 多实例模式的完整数据目录
    └── work/
        ├── Cookies           # 从 profiles/work/ 复制
        ├── Local Storage/    # 从 profiles/work/ 复制
        ├── claude_desktop_config.json  # 从主 Claude 目录同步
        ├── config.json                 # 从主 Claude 目录同步
        └── ...               # Electron 运行时自动生成其余文件
```

### `switch` 与 `open` 的对比

| | `switch` | `open` |
|---|---|---|
| 同时在线 | 不行，一次一个 | 可以，各有独立窗口 |
| 共享 Claude Code 会话 | 共享 | 不共享（独立数据目录） |
| 共享 MCP 配置 | 共享（同一数据目录） | 共享（启动时同步） |
| 共享窗口状态 | 共享 | 不共享 |
| 资源占用 | 单进程 | 每个实例一个额外的 Electron 进程 |

**经验法则：** 日常使用 `switch`，需要两个账户并排显示时用 `open`。

## 安全性说明

**没有封号风险。** 原因如下：

1. **纯本地文件操作** —— cpm 只在 `~/Library/Application Support/Claude/` 内复制文件，等同于手动登出再登入，只是省去了重新输入密码的步骤
2. **不涉及逆向工程** —— Claude 应用本体没有被修改
3. **不涉及 API 伪造或自动化** —— 没有伪造请求，每个 session 都是真实的已认证登录
4. **和浏览器多 Profile 机制相同** —— Chrome/Firefox 隔离账户也是同样的方式（独立 Cookie 存储），这是被广泛接受的标准做法
5. **Anthropic 服务端看到的是正常 session** —— 切换后，应用呈现的是一个有效的 session cookie，和普通登录完全一致

**注意事项：**

- 每个账户仍须独立遵守 Anthropic 的[使用条款](https://www.anthropic.com/terms)
- 不要用此工具将账户共享给他人（账户共享违反条款）
- 此工具仅用于在**同一个人的多个账户**之间快速切换
