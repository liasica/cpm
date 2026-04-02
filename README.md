# cpm - Claude Profile Manager

[中文文档](README_zh.md)

Quickly switch between multiple Claude accounts on the macOS desktop app while keeping Claude Code sessions, MCP configs, and other settings shared across all profiles. Also supports running multiple accounts simultaneously.

## Install

```bash
go install .
```

Requires `$GOPATH/bin` (default `~/go/bin`) in your `$PATH`.

## Usage

### 1. Save your first account

Make sure you're logged into Account A in the Claude desktop app, then run:

```bash
cpm add personal
```

### 2. Save another account

In the Claude desktop app, **log out of Account A and log into Account B**, then run:

```bash
cpm add work
```

Repeat for as many accounts as you need.

### 3. Switch between accounts

```bash
cpm switch personal
cpm switch work
```

The app will be automatically closed before switching and reopened after.

### 4. Run multiple accounts simultaneously

```bash
cpm open work      # launch a second Claude instance with the "work" profile
cpm close work     # close that instance
```

The main Claude app keeps running as your current profile. Each `cpm open` instance gets its own window with MCP config and app settings synced from the main app automatically.

### Commands

| Command | Alias | Description |
|---------|-------|-------------|
| `cpm` | | Show all profiles |
| `cpm add <name>` | | Save current login as a profile |
| `cpm switch <name>` | `cpm sw` | Switch to a profile (single instance) |
| `cpm open <name>` | | Launch an additional instance (multi-instance) |
| `cpm close <name>` | | Close an additional instance |
| `cpm list` | `cpm ls` | List all profiles |
| `cpm current` | | Print current profile name |
| `cpm rename <old> <new>` | | Rename a profile |
| `cpm remove <name>` | `cpm rm` | Delete a profile |

### Flags

- `cpm switch <name> --no-restart` — don't relaunch Claude after switching

## How It Works

The Claude desktop app is an [Electron](https://www.electronjs.org/) application. Like all Electron apps, it stores its data under `~/Library/Application Support/Claude/` on macOS. This directory contains both **authentication state** (who you're logged in as) and **local application data** (your sessions, settings, window layout, etc.).

### `cpm switch` — single-instance mode

cpm exploits the separation between auth and app data. When you run `cpm add <name>`, it snapshots only the authentication-related files into `~/.config/cpm/profiles/<name>/`. When you `cpm switch`, it:

1. **Quits Claude** gracefully via AppleScript (falls back to `pkill` on timeout)
2. **Saves** the current auth files back to the active profile
3. **Restores** the target profile's auth files into the Claude data directory
4. **Relaunches** Claude (unless `--no-restart` is set)

Everything except the auth files stays untouched, so your Claude Code sessions, MCP tool setup, and window layout carry over seamlessly.

### `cpm open` — multi-instance mode

Electron supports launching additional instances via the `--user-data-dir` flag, which points the new process at a completely separate data directory. When you run `cpm open <name>`, it:

1. **Creates** an isolated data directory at `~/.config/cpm/instances/<name>/`
2. **Copies** the profile's auth files (Cookies, Local Storage, Session Storage) into the instance directory
3. **Syncs** shared config files (`claude_desktop_config.json`, `config.json`) from the main Claude data directory so MCP servers and app preferences are available
4. **Launches** a new Claude process with `open -na Claude --args --user-data-dir=<instance-dir>`

Each instance is a fully independent Electron process with its own window and its own data. The main Claude app is unaffected.

`cpm close <name>` finds the instance's processes (via `pgrep -f user-data-dir=<path>`) and sends `SIGTERM` for graceful shutdown, escalating to `SIGKILL` if it doesn't exit within 6 seconds.

### What gets switched vs. shared

| Switched (per-account) | Purpose | Shared (across all accounts) | Purpose |
|---|---|---|---|
| `Cookies` | Session cookies (SQLite) — the core login credential | `claude-code-sessions/` | Claude Code conversation history |
| `Cookies-journal` | SQLite WAL journal for Cookies | `claude_desktop_config.json` | MCP server configuration |
| `Local Storage/` | Supplementary auth tokens (LevelDB) | `config.json` | App preferences |
| `Session Storage/` | Ephemeral session data (LevelDB) | `window-state.json` | Window size and position |
| | | `Cache/`, `GPUCache/`, etc. | Chromium caches |
| | | Everything else | Crash reports, extensions, etc. |

### File layout

```
~/.config/cpm/
├── state.json                # tracks which profile is active
├── profiles/                 # auth file snapshots (used by switch & open)
│   ├── personal/
│   │   ├── Cookies           # SQLite database with session cookies
│   │   ├── Cookies-journal
│   │   ├── Local Storage/    # LevelDB — auth tokens
│   │   └── Session Storage/  # LevelDB — ephemeral session data
│   └── work/
│       └── ...
└── instances/                # full data dirs for multi-instance mode
    └── work/
        ├── Cookies           # copied from profiles/work/
        ├── Local Storage/    # copied from profiles/work/
        ├── claude_desktop_config.json  # synced from main Claude dir
        ├── config.json                 # synced from main Claude dir
        └── ...               # Electron creates remaining files at runtime
```

### `switch` vs. `open` — when to use which

| | `switch` | `open` |
|---|---|---|
| Simultaneous logins | No — one account at a time | Yes — each in its own window |
| Shares Claude Code sessions | Yes | No (isolated data dir) |
| Shares MCP config | Yes (same data dir) | Yes (synced on launch) |
| Shares window state | Yes | No |
| Resource usage | Single process | Additional Electron process per instance |

**Rule of thumb:** use `switch` for your daily driver and `open` when you need two accounts visible side by side.

## Safety

**There is no risk of account suspension.** Here's why:

1. **Local file operations only** — cpm copies files within `~/Library/Application Support/Claude/`. This is identical to manually logging out and back in, just without retyping your password
2. **No reverse engineering or patching** — the Claude binary is never modified
3. **No API forgery or automation** — no requests are forged; each session is a real, authenticated login
4. **Same mechanism as browser profiles** — Chrome and Firefox isolate accounts the same way (separate cookie stores). This is a widely accepted, standard practice
5. **Anthropic's servers see normal sessions** — after a switch, the app presents a valid session cookie just like any regular login

**Keep in mind:**

- Each account must independently comply with Anthropic's [Terms of Service](https://www.anthropic.com/terms)
- Do not use this tool to share your account with others (account sharing violates the ToS)
- This tool is intended for switching between **your own** multiple accounts on a single machine
