# myEnv GUI / TUI 选型与实施建议

调研及可行性复核：2026-09-10。本轮重点复核两项主选框架与运行边界，核对公开 `main`（`a7a574272d4d95d8b35999a7833a163b1be710a3`）、官方发布/API 与相关问题源码；Windows 本地工作树可能已有后续修复，实施时以本地状态为准。本轮未实现界面、未做 GUI/TUI 实机原型或性能测试。

**结论：选型有条件可行，可以进入一次真实业务原型；暂不足以承诺完整界面可发布。** 保留 Wails v2 与 Bubble Tea v2，优先验证 TUI 信号/进程所有权、GUI 后台子进程和取消收尾。无需再扩大候选比较，也不应为增加界面重写环境内核。

## 1. 推荐组合

**保留 Go 内核和 Cobra；Windows GUI 首选 Wails v2 + React/TypeScript，Windows/Linux TUI 首选 Bubble Tea v2。先复用本地已完成的 CLI 修复与验证，再用一个真实业务小原型确认关键风险；GUI/TUI 开发不以抹掉既有性能失败为前提。** 这是综合现有代码、维护成本和交互需求的判断，不是框架性能排名。

| 平台 | 产品目标 | 本轮建议的首批验收范围 |
| --- | --- | --- |
| Windows | 完整 GUI、TUI、现有 CLI | Windows 11 x64 主验收；Windows 10 x64 单列兼容验收及具体系统构建号 |
| Linux | 完整 TUI、现有 CLI | 原生 x86_64/glibc，先沿用 Kali 实测环境，再补目标发行版 |
| macOS | 不纳入开发与发布 | 不安排界面、打包或验收任务 |

框架支持 ARM64 等平台，不代表 myEnv 已支持。当前运行时选择仍限制 WSL、musl 和多数其他架构，本轮不自动扩大这些范围。Windows Server、传统控制台也不能仅凭编译通过就列为“完整界面已支持”。[当前平台判断](https://github.com/XVSHIFU/myEnv/blob/main/internal/runner/platform.go)

附件中“不要 WebView2、兼容 Win7、Hello World 小于 10 MB/50 MB”是原帖作者的条件，**不是你的项目新增约束**；不因此排除 Wails，也不因此把项目换成其他语言。

## 2. GUI 候选：主线只选一套

| 候选 | 与 myEnv 的匹配和代价 | 决定 |
| --- | --- | --- |
| **[Wails v2 + React/TS](https://wails.io/docs/howdoesitwork/)** | 通过绑定调用 Go，能够直接复用核心；现有文档站已用 React/TS。Windows 运行需要 WebView2 | **首选**；按正式 release 固定版本 |
| **Wails v3** | 增加新能力，但本次核实仍为 `v3.0.0-beta.19` 预发布 | 不作为当前正式产品基础 |
| **[Fyne 2.8.1](https://github.com/fyne-io/fyne/blob/v2.8.1/CHANGELOG.md)** | Go 编写界面、不需要 WebView2；[构建](https://docs.fyne.io/started/quick/)需要 C 编译器/图形相关依赖，中文输入细节和新无障碍实现需验收 | “坚持 Go 且要去掉 WebView2”时的备选 |
| **[C# WPF](https://learn.microsoft.com/en-us/dotnet/desktop/wpf/overview/)** | Windows 桌面控件、绑定和验证成熟；需增加 C#/XAML、.NET 分发和 Go 进程通信 | “原生桌面交互优先”时的备选 |
| **[Tauri 2](https://v2.tauri.app/develop/sidecar/)** | 可将 Go CLI 打包为 sidecar；会形成 TS + Rust 壳 + Go 三套构建链，同样使用 Windows WebView2 | 当前没有足够收益替代 Wails |
| **WinUI 3 / WinForms** | 都能完成任务；前者增加 [Windows App SDK 部署组合](https://learn.microsoft.com/en-us/windows/apps/package-and-deploy/deploy-overview)，后者适合简单表单与设计器开发 | 不再与 WPF 同时保留为实施路线 |
| **[Gio](https://github.com/gioui/gio) / [Go Walk](https://github.com/lxn/walk)** | Gio 更偏自绘交互；Walk 提供 Windows 控件，但本次不足以确认其现代桌面适配与维护风险 | 不作为主线 |
| **[Qt](https://doc.qt.io/qt-6/windows.html)** | 能力完整，但需要额外界面语言/绑定与打包体系，跨平台 GUI 的收益在本项目用不上 | 不作为主线 |

本次 GitHub 正式 release 核实到 **Wails v2.14.0**；官网文档页头为 v2.15.0，不把文档页头当作已发布证明。开工固定可取得的正式 tag，不照抄 `@latest` 或混用 v2/v3 文档。[v2.14.0](https://github.com/wailsapp/wails/releases/tag/v2.14.0)、[v3 beta.19](https://github.com/wailsapp/wails/releases/tag/v3.0.0-beta.19)、[Go/TS 绑定](https://wails.io/docs/howdoesitwork/)

Electron、Flutter、Python GUI 也不进入小原型：当前没有需要引入其运行时和界面生态的新需求。

**不选 Wails 的触发条件要具体：**目标机器不能部署 WebView2，或真实页面在确定的内存预算内无法运行，再选择 Fyne/WPF 中匹配限制的一套；不要同时开发三套 GUI。WPF 也不能凭“原生”保证更省内存，更不能默认可无条件裁剪/AOT。[.NET 裁剪限制](https://learn.microsoft.com/en-us/dotnet/core/deploying/trimming/incompatibilities)

## 3. TUI 候选与当前风险

| 候选 | 判断 |
| --- | --- |
| **Bubble Tea v2 + Bubbles + Lip Gloss** | 首选。Go 内事件和状态驱动模型适合版本选择、异步任务和进度，界面逻辑容易与 core 分离 |
| **tview + tcell** | 表格、树、表单和弹窗较直接；若 Bubble Tea 的目标终端或子进程恢复问题阻塞，采用这一备选 |
| **Ratatui** | Rust 项目合适；myEnv 使用它会增加语言和 Go 通信边界 |
| **Textual** | Python 项目合适；本项目管理 Python 环境，给管理工具再增加 Python 应用分发不划算 |

本次正式发布核实到 Bubble Tea **v2.0.9**、Bubbles **v2.2.1**、Lip Gloss **v2.0.6**。新代码统一使用 `charm.land/bubbletea/v2`、`charm.land/bubbles/v2`、`charm.land/lipgloss/v2`，不要拼接旧版教程。备选 tview **v0.42.0** 沿用其 **tcell/v2** 依赖，不自行套用 tcell/v3 文档。[Bubble Tea](https://github.com/charmbracelet/bubbletea/releases/tag/v2.0.9)、[Bubbles](https://github.com/charmbracelet/bubbles)、[Lip Gloss](https://github.com/charmbracelet/lipgloss)、[tview](https://pkg.go.dev/github.com/rivo/tview@v0.42.0)

补充候选的原始资料：[Ratatui](https://github.com/ratatui/ratatui)、[Textual](https://textual.textualize.io/getting_started/)。实际 TUI 可从 Bubble Tea 官方列出的 gh-dash 学习筛选列表和键位提示；不把其他工具的流行度当作性能证明。[官方使用案例](https://github.com/charmbracelet/bubbletea#bubble-tea-in-the-wild)

有一个值得先验证的具体风险：Bubble Tea [#1778](https://github.com/charmbracelet/bubbletea/issues/1778) 提供了 Linux 上短命令/启动失败后终端恢复冻结的复现，报告涉及 v1.3.10 与 v2.0.9，不能简单降回 v1。Windows 粘贴 [#1712](https://github.com/charmbracelet/bubbletea/issues/1712) 和中文显示 [#1777](https://github.com/charmbracelet/bubbletea/issues/1777) 的环境/归因尚不完整，只作为定向验证线索。

TUI 首批以 **Windows Terminal、原生 Linux 终端**为主要目标。PowerShell/Bash 是 Shell，Windows Terminal/传统控制台是终端宿主，必须分别记录。中文宽度、粘贴、组合字符、缩放终端和返回 TUI 不能仅看截图。若 #1778 可复现，候选退路为“完整退出旧 Program→运行→重建全新 Program”，不要复用已停止的渲染器或增加固定 sleep。重建只能针对渲染器恢复风险，不能证明信号和进程树正确；运行交接按第 5 节验证。框架恢复仍失败才考虑 tview；若问题在 myEnv 的进程边界，换 TUI 框架也不能解决。程序化复制还依赖终端对 OSC52 的支持，应提供手工复制退路。[剪贴板 API](https://pkg.go.dev/charm.land/bubbletea/v2#SetClipboard)

## 4. 社区与实际产品带来的取舍

| 来源 | 能借鉴什么 | 不能据此推出什么 |
| --- | --- | --- |
| [Tiny RDM](https://github.com/tiny-craft/tiny-rdm)（中文生态、Wails v2） | 连接/对象列表、详情、分段加载、操作日志；证明 Go + Web 界面可承载管理工具 | “轻量”宣传不等于 myEnv 的实测内存；它用 Vue，不要求本项目换 Vue |
| [V2EX：WinForms 壁纸工具作者分享](https://www.v2ex.com/t/1150815) | 作者采用 Core/Desktop 分离；报告 AI 改设计器和单文件首启的具体问题 | 个案不能变成所有 WinForms 的性能结论，系统支持以官方矩阵为准 |
| [WinUI 官方及开发者讨论](https://github.com/microsoft/microsoft-ui-xaml/discussions/11096) | 官方仍投入性能改进，实际开发者体验有差异，应看具体页面与发布方式 | File Explorer 中 WinUI 部分的优化比例不能套到所有 GUI |
| [Fyne 输入法 #618](https://github.com/fyne-io/fyne/issues/618)、[读屏 #1093](https://github.com/fyne-io/fyne/issues/1093) | 中文输入候选框定位仍有开放问题；无障碍有新进展但需验证 | 不能说 Fyne 完全不支持中文，也不能说已成熟解决所有 Windows 输入/读屏问题 |
| [UniGetUI](https://github.com/Devolutions/UniGetUI) | 包/版本筛选、详情、队列与失败日志的交互 | 当前仓库已使用 Avalonia 发布路径，不能再当作当前 WinUI 3 样板 |

附件对应的 [LINUX DO 原帖](https://linux.do/t/topic/2851165) 主要说明小工具选型会受分发、输入法和维护体验影响。本文没有采用争论中的主观语言排名、AI 语料能力判断和不可比的 Hello World 数字。

## 5. 贴合当前仓库的最小架构

当前 [core.Service](https://github.com/XVSHIFU/myEnv/blob/a7a574272d4d95d8b35999a7833a163b1be710a3/internal/core/sync.go) 已不打印终端文本；[SyncPhase](https://github.com/XVSHIFU/myEnv/blob/main/internal/core/progress.go) 已提供同步阶段回调。无需重写内核，但不能直接将含 `context`、`ConfirmBuild`、`Progress` 函数的请求结构绑定到前端。需提取 [CLI storage.go](https://github.com/XVSHIFU/myEnv/blob/main/internal/cli/storage.go) 的镜像/证书/存储初始化，以及 CLI 层的运行组装与错误分类；使用普通可序列化请求、结果连接界面。前端沿用已有 [React/TS 技术栈](https://github.com/XVSHIFU/myEnv/blob/main/website/package.json)，单独使用 Vite 静态构建，不带入网站的 SSR 或服务器运行时。

| 入口/边界 | 具体安排 |
| --- | --- |
| `myenv` / 原有子命令 | Cobra 保留，裸命令仍显示状态；JSON、退出码和脚本行为保持原合同 |
| `myenv tui` | 显式进入 TUI；非 TTY 清楚提示使用 CLI，不自动打开全屏界面 |
| `myenv-gui.exe` | Windows 独立图形入口；Go 绑定直接调用共享业务服务 |
| 共享适配层 | 按实际需要提取配置、结构化结果、错误、任务取消和事件；不新建通用插件平台或守护进程 |
| 界面任务 | 只提供启动、取消、查询当前结果及构建确认；后台执行核心用例，UI 不直接写 SQLite 或管理租约 |

关键约束：

- **普通 CLI 不初始化界面。** Wails 与静态资源只进入 GUI 构建。TUI 不得在包初始化时读数据库、扫描目录或启动循环；若引入后普通命令出现实际回归，再拆可选 TUI 可执行文件。
- **任务与事件有界。** Go 侧以 task ID 保存有限数量的取消函数、状态和结果；首次实现单个写任务，读取不阻塞界面，不做通用队列平台。Progress 回调只投递/合并状态并立即返回，事件带任务 ID；切换项目或重建界面后主动读取结果，不能因错过事件丢掉终态。构建确认只作用于当前任务，可被取消，不在 UI 线程阻塞。阶段未知时不显示虚假百分比，日志限制大小。版本搜索优先过滤已有结果，远端查询可取消，旧请求不能覆盖新选择；空闲时不持续动画或轮询数据库。
- **生命周期明确。** 每项操作使用独立 context，取消后显示“正在收尾”，核心返回后才成为终态。Wails 使用 `OnBeforeClose` 暂停关闭，选择“取消后关闭”时异步等待；`OnShutdown` 已在前端销毁后，不能再依赖弹窗或事件完成确认。TUI 实例的退出不应自动取消随后运行的用户命令。界面崩溃仍使用已有恢复，不承诺后台安装自动继续。[Wails 生命周期](https://wails.io/docs/reference/options/#onbeforeclose)
- **TUI 运行优先保持同进程所有权。** 完整结束 Tea Program 并恢复终端后，在同一个 myEnv 进程调用提取后的共享 Run 用例，保留 `SelectRun → runner.Execute → Finish`，结束后重建界面。交接时停用 TUI/Cobra 的取消监听，再由 runner 管理运行信号，不能重复转发；不要把已取消的 TUI context 传给 run。直接嵌套调用 Cobra 不是共享业务接口。[Bubble Tea 执行 API](https://pkg.go.dev/charm.land/bubbletea/v2@v2.0.9#ExecProcess) 仅负责终端交接，不负责 myEnv 租约。
- **额外 run 子进程尚未证明等价。** 当前 Linux [launcher](https://github.com/XVSHIFU/myEnv/blob/a7a574272d4d95d8b35999a7833a163b1be710a3/internal/runner/launcher_linux.go) 只暂停当前调用进程，监督器通过它持有的 lifetime 管道识别死亡。外层 TUI 再启动 run 后，只强杀外层不会自动关闭内层的管道；Ctrl-Z/fg/bg 也可能改变。这是源码确认的机制差异，实际后果待原型验证。若仍用子进程方案，须补齐外层死亡、暂停和信号交接，不能写成现成可靠退路。
- **GUI 交互运行交给终端。** 打开随包同版本、绝对路径的 CLI，使用参数数组并验证 Windows 参数保真；不拼接任意 Shell 字符串。`wt.exe` 可转发到既有终端，启动器退出不代表用户命令已完成，界面只显示“已交给终端”，退出状态由终端内 CLI 负责。没有 Windows Terminal 时提供新控制台退路；关闭 GUI 不自动关闭已交接的用户终端。完整管理功能不要求内嵌终端。[终端启动参数](https://learn.microsoft.com/en-us/windows/terminal/command-line-arguments)
- **GUI 后台命令不闪窗。** 当前版本核验等路径会启动控制台程序；后台下载/核验与交互 run 必须区分启动模式，不能统一隐藏窗口。新增 Windows 参数不得覆盖现有挂起创建、句柄白名单和 Job 关联。原型中验证 uv/版本检查不弹黑框、交互程序仍能输入和 Ctrl-C。[Wails 子进程说明](https://wails.io/docs/guides/windows/#spawning-other-programs)
- **Linux 监督器保留。** GUI/TUI 不改变 SIGKILL 后回收、SIGSTOP 期间截止时间、完成回执和 run/clean 协调；Windows 保留现有 Job Objects 机制。若候选实现增加运行代理进程，其开销和生命周期也必须计入，不能排除后再声称达标。
- **三个入口语义一致。** 每个任务固定目录、作用域与设置快照，不能通过 `os.Chdir` 或 `os.Setenv` 切换不同任务的上下文。 `use` 仍按当前语义编辑并同步；回退不恢复代码；预览是快照，执行时重新校验相关输入，发生变化返回界面重新确认。复用核心锁和保护，不能只把按钮置灰当作并发控制。

Windows WebView2 的在线包先检测 runtime，缺少时走微软引导安装；离线包提供完整 Standalone Installer。Wails 的嵌入 bootstrapper 不是离线运行时，不能宣传为“无依赖单文件”。GUI 静态资源随包提供，核心管理功能不依赖远程网页/CDN。首版“便携”只表示 myEnv 无需安装，仍需机器已有 WebView2；WebView 用户数据单列到用户应用数据目录，不把程序放在 U 盘等同于所有数据随行。[Wails Windows 分发](https://wails.io/docs/guides/windows/)、[微软运行时分发](https://learn.microsoft.com/en-us/microsoft-edge/webview2/concepts/distribution)

## 6. “完整 GUI/TUI”的界面范围

两端共用业务用例与结果，界面组件各自实现。GUI 用导航、列表和详情；TUI 用方向键/Tab、Enter、Esc 和始终可见的帮助。不把全部 CLI 参数平铺成表单。

| 页面/区域 | 覆盖现有能力 |
| --- | --- |
| 项目 / 当前用户默认环境 | 打开项目、初始化、状态、工具列表；明确区分项目与用户默认，避免把 `system` 误写为系统管理员安装 |
| 工具与版本 | Node/Python/Java/Go/Rust 的来源、版本查询/筛选、选择与同步；从现有 Catalog 获取，不硬编码 Java 17 等少数版本 |
| 操作与结果 | 变更预览、构建确认、同步进度、取消、错误与下一步；失败保留原环境的说明 |
| 诊断与恢复 | doctor/deep、当前与保留前代、回退、清理预览及确认；不为“历史”新增完整时间线 |
| 设置与运行 | 已有镜像/证书/存储设置、打开终端运行、复制命令；仅添加现有能力实际需要的设置 |

首个可用版本只完成项目/用户默认环境→选版本→sync→run→doctor；回退、清理与外部管理在后续步骤补齐后，才称“完整 GUI/TUI”。能力对应表放在现有 implementation.md，避免两套界面同时铺开所有页面。Python 当前官网发布页与可安装 Astral 目录不一致，界面必须区分来源及可安装性，不能给发布页统一显示安装按钮。高级构建/脚本细节仍由简短帮助解释，不增加 Agent 控制台、插件市场、常驻托盘和无关设置。

## 7. 给 Codex 的实施顺序

| 步骤 | 交付与停止条件 |
| --- | --- |
| **A：一次风险小原型** | TUI：中文列表与输入，短命令/启动失败/交互程序返回，Ctrl-C、Windows Ctrl-Break 和 Linux Ctrl-Z/fg/bg；运行期间强杀/暂停用户启动的 myEnv 进程，检查既有回执与保护。GUI：同一列表、后台不闪窗、取消、关闭收尾，中文路径/DPI、WebView2 缺失与离线安装。保持隔离目录和真实 core；框架可用但进程语义失败时只修边界，不重新比较全部框架 |
| **B：共享服务 + TUI 闭环** | 只提取初始化、Run、结构化错误和最小任务接口；完成打开项目→选版本→sync→run→doctor。同进程运行通过原型门禁后固定接口，复用现有 runner 用例，只补界面边界验证 |
| **C：Windows GUI 闭环** | 接同一业务合同与取消行为；完成独立入口和安装/便携分发，不重复实现版本解析或状态管理 |
| **D：补齐管理入口并验收** | 按能力对应表补回退、清理、用户默认和已有外部管理；对最终制品做一次功能与性能对照，记录真实未覆盖项 |

原型只证明选型风险，不冒充完整产品。B/C 可在接口固定后按文件归属并行；下载缓存和验证记录共用。本次只评估，不据此修改 AGENTS.md 或启动完整开发。用户明确启动 GUI/TUI 阶段时，再将 AGENTS.md 的“只做 CLI”更新为新范围，并保留可靠性和测试去重规则；无需复制新的大型 Agent 手册。[现有 AGENTS.md](https://github.com/XVSHIFU/myEnv/blob/main/AGENTS.md)

性能口径维持诚实：CLI 原预算不放宽、已有超标不抹掉；GUI 与 TUI 单列首次可交互时间、空闲/任务中 CPU 和同时内存占用。GUI 要统计 Go 宿主与相关 WebView2 进程组；Windows 同时报工作集与 Private Bytes，Linux 保留聚合 RSS 并补 PSS/私有页。不能拿 exe 包大小代替内存，也不能用 GUI 预算覆盖 CLI 成绩。[当前性能记录](https://github.com/XVSHIFU/myEnv/blob/main/docs/performance-gap.md)、[WebView2 进程模型](https://learn.microsoft.com/en-us/microsoft-edge/webview2/concepts/process-model)

原型开始前固定目标机器、构建模式、业务场景和采样口径：先显示窗口不等于可交互，空闲须在界面稳定后采样。GUI/TUI 的数值预算结合一次原型数据和用户可接受目标确定，进入 B/C 前固定，不边开发边放宽。当前没有依据宣称哪套方案能满足 32/50 MiB。CLI 已知超预算无需靠重测证明，但 UI 接入不得降低其目标或退化原行为。只做一次足够定位问题的采样和一次有实际修改的前后对照；相同制品、平台与场景已通过的结果复用，不反复全量安装、压测或测试所有候选。

后续文档只保留必要变更：更新原支持矩阵与界面入口，在 `docs/implementation.md` 记录一次选型、有效验证与未决项。此文件作为决策依据，无需再拆成调研报告、日报、任务总结和多份同义方案。
