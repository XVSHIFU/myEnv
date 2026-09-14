# 构建发行制品

使用 `go.mod` 指定的 Go 1.26.6。以下命令构建 `0.1.0`；构建成功不代表平台、性能和发布验收已经完成，具体交付状态与证据见 [implementation.md](implementation.md)。

使用已构建的软件无需准备这些开发依赖。Windows 用户安装与首次操作见 [Windows 安装与入门](windows-install.md)，网站文档的本地构建见 [website/README.md](../website/README.md)。

## 构建 CLI/TUI

在 PowerShell 中运行：

```powershell
./scripts/build-release.ps1 -Version 0.1.0
```

默认构建 Windows amd64 与 Linux amd64 的 CLI/TUI，输出到 `dist/release`；脚本不提供 macOS 或 Linux GUI 目标。只构建本次需要的目标：

```powershell
./scripts/build-release.ps1 -Version 0.1.0 -Targets windows-amd64
```

可用 `-Version 0.1.0` 指定写入二进制的版本，用 `-OutputDirectory` 指定产物目录。脚本只构建和生成清单，不发布、安装或修改 PATH。所选目标全部成功后才写入本次清单；目录内其他已有文件不列入清单。

## 构建与打包 Windows GUI

构建 Windows GUI 需要 Windows PowerShell 7、Node/npm 和锁定的前端依赖，`-GUI` 同时要求选择 Windows CLI 目标：

```powershell
./scripts/build-release.ps1 -Version 0.1.0 -GUI
./scripts/package-windows.ps1
```

脚本执行 `npm ci --ignore-scripts` 与前端构建，以 `gui,desktop,production` 标签构建 GUI，再注入原生图标；GUI 资源不进入 CLI。打包脚本使用 Windows .NET Framework 编译器，输出包含 GUI/CLI 的便携 ZIP、仅安装 CLI/TUI 的 setup 和分发包摘要。GUI 运行仍需 WebView2，便携包不捆绑离线运行时，当前制品未签名。

myEnv 自身使用根目录 [MIT 许可证](../LICENSE)。便携 ZIP 内和打包输出目录均提供 `LICENSE`、`THIRD_PARTY_NOTICES.txt` 与 `third-party-manifest.json`。setup 内嵌并在实际安装目录写入前两份许可原文，安装记录校验其摘要；升级支持旧安装记录，卸载只删除已确认由安装程序拥有的文件。第三方许可独立保留，不由项目的 MIT 许可证替代。

默认构建目录为 `dist/release`，默认打包目录为 `dist/packages`；两处 `SHA256SUMS` 各自覆盖本次输出，打包目录包含 ZIP、setup 和三份许可相关文件的摘要。公开发行时，最终清单须同时覆盖实际提供的二进制、ZIP、setup 和许可文件，并明确对应源码版本；不能仅用打包目录的摘要替代完整制品清单。

## 制品、源码与校验

固定构建参数为 `CGO_ENABLED=0`、`-trimpath`、`-buildvcs=false` 和 `-ldflags="-s -w -X main.version=<版本>"`。构建失败立即报错，结束时恢复脚本调用前的 GOOS、GOARCH 和 CGO_ENABLED。调试信息被移除，需要调试时使用普通 `go build` 另建调试制品。

每次生成：

- 所选目标的 `myenv-<目标>` 二进制，Windows 带 `.exe` 后缀。
- `build-manifest.json`：版本、Go 编译器、参数、目标、文件大小、SHA-256 和源码输入证据。
- `SHA256SUMS`：本次所选制品的摘要清单。

manifest 的 `source` 记录基准提交 `base_commit`、工作区是否有未提交改动 `working_tree_modified`，以及实际构建输入的逐文件摘要和总摘要 `input_sha256`。输入涵盖 `cmd`、`internal`、`scripts`、`go.mod`、`go.sum` 与根目录 `LICENSE`，包含生成的前端嵌入资源，排除依赖缓存；构建前后输入变化会使构建失败。Windows 打包还核对根 `LICENSE` 与该次构建清单一致；缺少证据或内容改变时须重新构建。有未提交改动的候选不能仅凭基准提交复现，交付时还须提供对应源码快照及其摘要。

校验清单只记录文件摘要，不是签名或来源认证。性能报告必须引用实际受测制品的摘要；不同构建不能直接沿用旧报告作为门禁通过证据。

代码 CI 在 Windows/Linux 上运行已有离线接口、包目录和核心包管理定向测试；包管理的真实联网生命周期测试保持显式启用，不纳入日常 CI。Windows GUI CI 单独验证前端、生产标签构建和图标资源，不会创建 Release 或上传发行包。工作流通过与公开发布是独立步骤。

所有选定目标先构建到输出目录下的独立 `.build-<随机ID>` 暂存目录，摘要和清单也先写入该目录。某个目标编译失败时保留此前正式产物，部分构建文件留在暂存目录供检查。重复目标只构建一次。

全部编译和清单生成成功后，脚本逐文件替换所选制品，最后替换摘要清单和 manifest。这个步骤不是跨文件原子事务；复制失败、文件占用或进程中断可能留下不完整更新，应重新运行构建，并在脚本成功结束后核对清单中的全部摘要。不要在脚本运行过程中分发输出目录。

## 记录性能样本

Windows PowerShell 7 可用以下工具记录单个 CLI 进程的峰值工作集：

```powershell
./scripts/measure-memory-windows.ps1 -Output .build/perf/memory-version.json
./scripts/measure-memory-windows.ps1 -CommandArguments '--help' -Output .build/perf/memory-help.json
```

默认使用 `dist/release/myenv-windows-amd64.exe`、每个命令运行20次；可指定 `-Executable` 和 `-Samples`。报告保存全部样本、最大值、命令参数和二进制摘要，不覆盖已有报告。它运行给定命令，不要将会修改数据的命令当作无副作用测量。

测量 `sync --locked --no-input --json` 时，可传 `-ExpectedGeneration <已应用代ID>`，工具会逐次检查成功、无变化和代 ID 一致；不符合时不会生成成功报告。该检查只用于同步 JSON 输出，读取缓冲上限为64 Ki字符。

计数使用 Windows [GetProcessMemoryInfo](https://learn.microsoft.com/en-us/windows/win32/api/psapi/nf-psapi-getprocessmemoryinfo) 的 [PeakWorkingSetSize](https://learn.microsoft.com/en-us/windows/win32/api/psapi/ns-psapi-process_memory_counters)，保持进程句柄直到读取结束。它不包含测量宿主或子进程树，不测延迟，也不清空文件缓存。

状态命令内存测量也可使用 `-StatusResponse -ExpectedGeneration <代ID>`，配合 `-CommandArguments @('-C', '<隔离项目目录>', '--json')`。脚本逐次核验 JSON 为 ready、changed=false 且代ID匹配；缺少 ExpectedGeneration 时拒绝启动。该模式仍只测 CLI 父进程峰值工作集，不测子进程树，也不能替代延迟门禁。真实混合测试通过 MYENV_TEST_MIXED_STATUS_MEMORY 单独启用这一模式。
