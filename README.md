# myEnv

在桌面或终端里查看已有开发环境，为项目选择工具版本，并管理 Node.js / Python 的包与工具。

**Python · Node.js · Java · Go · Rust** — Windows GUI、Windows / Linux TUI 与 CLI 共用同一个环境内核。

[中文文档](https://xvshifu.github.io/myEnv/) · [GUI 入门](https://xvshifu.github.io/myEnv/guide/gui/) · [包与工具](https://xvshifu.github.io/myEnv/guide/packages/) · [更新记录](CHANGELOG.md) · [GitHub Releases](https://github.com/XVSHIFU/myEnv/releases)

<img width="1084" height="771" alt="image" src="https://github.com/user-attachments/assets/49e3414f-1a7b-47a9-a93e-326bed88333a" />


*Windows 原生界面，rc.11 隔离演示项目。截图来自实际软件，具体版本和路径以你的电脑为准。*

首版 **0.1.0**：[下载安装包与校验清单](https://github.com/XVSHIFU/myEnv/releases/tag/v0.1.0)。

## 可以用它做什么

- **看清已有环境。** 首次检查本机命令、实际路径和已安装版本，也可选择一个项目一起检查。保存后不再弹出，之后从维护页刷新。
- **按项目选择版本。** 使用 `myenv.yaml` 声明工具，预览后同步。准备成功才切换活动环境；失败保留旧代，运行命令不会暗中安装。
- **管理包与工具。** 在 GUI 中查询官方目录、选择版本、更新或卸载。支持搜索、批量选择和逐项结果，操作前明确作用位置。
- **选适合自己的入口。** GUI 提供工作台、进度和管理抽屉；TUI 用键盘完成环境选择与同步；CLI 适合脚本、远程终端和自动化。

### 三种环境各管什么

| 位置 | 含义 |
| --- | --- |
| 本机环境 | 应用启动时继承的 PATH 命令与其他已发现安装。检查不会自动接管安装，也不会改终端默认版本。 |
| myEnv 默认工具链 | 当前用户独立的一套受管版本，通过 `myenv run --global …` 使用；不会自动加入系统 PATH。 |
| 项目环境 | 当前目录自己的声明、锁和已应用环境；默认不继承 myEnv 默认工具链。 |

终端配置、虚拟环境或后来修改的 PATH，可能使终端里的 `python` 与 GUI 检查结果不同；界面会显示实际路径和检查时间。

<details>
<summary>安装与启动</summary>

在 [0.1.0 下载页](https://github.com/XVSHIFU/myEnv/releases/tag/v0.1.0) 选择适合的平台制品。

| 使用方式 | 选择的制品 | 启动 |
| --- | --- | --- |
| Windows 图形界面 | `myenv-0.1.0-windows-amd64-portable.zip` | 完整解压，双击 `myenv-gui.exe`。 |
| Windows 终端 | 同一便携包，或 `myenv-0.1.0-windows-amd64-setup.exe` | `myenv tui` 或 `myenv --help`。 |
| 原生 Linux 终端 | `myenv-linux-amd64` | 添加执行权限后运行 `./myenv-linux-amd64 tui`。 |

GUI 需要 **Microsoft Edge WebView2 Runtime**，便携包不包含离线运行时。保持 GUI 与同版本的 `myenv.exe` 在同一目录；setup 目前只安装 CLI/TUI，可加入当前用户 PATH。当前制品未签名，下载后可用随包的 `SHA256SUMS` 核对完整性。

详细步骤：[Windows 安装](docs/windows-install.md) · [文档站安装指南](https://xvshifu.github.io/myEnv/guide/installation/)。

</details>

<details>
<summary>从 GUI 开始</summary>

1. 打开软件，选择检查本机环境；需要时同时选择一个项目目录。
2. 查看本机已安装的版本，或进入项目 / myEnv 工具链，选择需要的语言与版本。
3. **保存并预览 → 核对变更 → 确认同步**。仅选中版本不会立即安装。
4. 同步完成后进入“运行命令”。底部始终保留任务状态，展开可查看阶段、结果和原始日志。

Python 下的 **uv / pip**、Node.js 下的 **npm / pnpm** 只在已检测到时显示快捷入口，点击直接进入管理抽屉；小窗口会调整入口位置。

### 包与工具管理

<img width="1084" height="771" alt="image" src="https://github.com/user-attachments/assets/71e4b253-0047-47e8-b3c7-0061c7f1ecce" />

“包与工具”按项目、全局安装位置或 Python 解释器分组。工具全部展示；包默认显示前 10 项，可展开、搜索、选择或全选筛选结果。单次最多管理同一位置的 100 个包。

- npm 支持官方关键词搜索与版本查询；PyPI 使用完整包名查询。官方未找到与网络失败分别提示。
- 期望可设为精确版本或“最新版”。最新版在预览时解析为具体版本，不会在后台自动升级。
- 安装、更新、切换与卸载都先预览，再确认执行。批量操作展示每个包的结果，失败或取消不会被描述成全部成功。
- **pip 属于具体 Python 解释器；uv 是 Astral 开发的独立工具。** 管理范围在抽屉中明确显示，同名多位置先选择位置。

GUI 包管理禁用 Node 安装脚本，Python 仅安装 wheel。myEnv 活动 Python 环境通过声明与同步更新，不能原地改包；外部包操作不会随环境代回退。复杂 workspace、未知管理器归属及独立 uv 卸载会提示交回原管理方式。

[查看完整包管理说明 →](https://xvshifu.github.io/myEnv/guide/packages/)

</details>

<details>
<summary>从终端开始</summary>

在已有 `.node-version`、`.nvmrc`、`.python-version` 或受支持项目清单的目录中：

```sh
myenv init
myenv sync
myenv run node --version
```

`init` 生成声明而不覆盖已有文件，`sync` 准备并应用环境，`run` 使用已应用版本。Python 项目最后一行可改为 `myenv run python --version`。

也可直接创建 `myenv.yaml`，只保留需要的工具，再运行 `myenv sync`：

```yaml
schema: 1
tools:
  node: "22"
  python: "3.12"
```

调整版本、核对环境、预览清理：

```sh
myenv use node@22
myenv sync --locked --no-input
myenv doctor
myenv clean --dry-run
```

`use` 等于修改声明并同步；`--locked` 不改锁。Python 受管项目依赖以 `pyproject.toml` / `uv.lock` 为准；Node 项目依赖使用项目自己的包管理器或 GUI 显式管理。

### 用键盘操作 TUI

```sh
myenv tui
myenv tui --global
```

方向键选择语言，Enter 查询与选择版本，`/` 筛选，Esc 返回。确认保存、预览和同步后可进入运行；运行前恢复终端，结束后返回界面。TUI 需要真实交互式终端，脚本使用普通 CLI 与 `--json`。

包管理抽屉目前属于 GUI，CLI/TUI 没有新增一套包管理命令。完整按键和流程见 [TUI 指南](https://xvshifu.github.io/myEnv/guide/tui/)。

更多帮助：`myenv` 查看状态，`myenv help manual` 查看离线手册，`myenv help sync` 查看命令说明。中文系统默认中文，可通过 `--lang en` 切换。

</details>

## 版本从哪里来

| 工具 | 当前接入来源 | 常见选择 |
| --- | --- | --- |
| Python | 默认 Astral CPython；Windows 可显式选择 python.org 完整 x64 ZIP | `python@3.12` |
| Node.js | Node.js 官方归档 | `node@22`，或完整版本 / 受支持范围 |
| Java | Eclipse Temurin HotSpot JDK | `java@8`、`java@17`、`java@21`；Java 8 即常说的 JDK 1.8 |
| Go | Go 官方归档 | 版本族或完整版本 |
| Rust | Rust 官方完整工具链归档 | 稳定版本、`beta`、`nightly[-YYYY-MM-DD]` |

示例不是版本上下限。可选范围取决于来源是否提供当前平台的制品；普通版本范围选择稳定版，预发布需要明确选择。查询和安装不承诺覆盖所有厂商、历史版本与变体。

```sh
myenv versions java --major 8
myenv system install java@8
myenv run --global java -version
```

Python 默认查询复用已有固定 uv 的 Astral 目录，不为查询自动安装 uv；Windows 可显式使用 `--provider python.org`。更多来源、平台和已验证样本见[支持范围](https://xvshifu.github.io/myEnv/guide/support/)。

## 当前支持范围

| 平台 | CLI / TUI | GUI |
| --- | --- | --- |
| Windows amd64 | 提供 | 提供，需要 WebView2 |
| 原生 Linux amd64 / glibc | 提供 | 暂缓 |
| WSL TUI、macOS、ARM64、musl | 不在本次支持范围 | 不提供 |

0.1.0 仍有性能预算、高 DPI、真实 IME、无 WebView2 干净机器，以及部分外部管理器写操作的验收缺口。已有功能验证不代表这些项目已通过；详见[当前记录](docs/implementation.md)和[性能缺口](docs/performance-gap.md)。

## 参与开发

CLI 使用 [go.mod](go.mod) 指定的 Go **1.26.6**；Windows GUI 还需 Node/npm 和 PowerShell 7。使用发行包无需自行安装 Go 或 Node。

```sh
go build -o dist/myenv ./cmd/myenv
```

Windows 输出名可改为 `dist/myenv.exe`。GUI 与发行包构建、依赖许可和源码证据见[构建说明](docs/build.md)；文档站单独维护于 [website/](website/README.md)。

| 目录 | 内容 |
| --- | --- |
| `cmd/` | CLI 与独立 Windows GUI 入口 |
| `internal/` | 共用内核、后端、终端界面与平台适配 |
| [docs/](docs/README.md) | 当前架构、构建与验证状态；设计和历史记录分目录保留 |
| `scripts/` | 构建、打包和受控验证脚本 |
| `website/` | 中文文档站、使用指南与真实截图 |

开发前阅读 [AGENTS.md](AGENTS.md)、[架构约定](docs/architecture.md)和[当前任务](docs/implementation.md)。缓存、临时验证和本地发行包不提交到 Git。

## Community

[linux.do](https://linux.do/) - A thriving developer community.

## 许可证

本项目采用 [MIT 许可证](LICENSE)。随制品分发的第三方组件保留各自许可，详情见[第三方声明](cmd/myenv-gui/build/THIRD_PARTY_NOTICES.txt)。
