# 构建开发制品

使用 `go.mod` 指定的 Go 1.26.6。当前制品仍为开发版本；构建成功不代表平台、性能和发布验收已经完成，具体证据见 [implementation.md](implementation.md)。

在 PowerShell 中运行：

```powershell
./scripts/build-release.ps1
```

默认构建 Windows amd64、Linux amd64、macOS arm64，输出到 `dist/release`。只构建本次需要的目标：

```powershell
./scripts/build-release.ps1 -Targets windows-amd64
```

可用 `-Version 0.1.0-dev` 指定写入二进制的版本，用 `-OutputDirectory` 指定产物目录。脚本只构建和生成清单，不发布、安装或修改 PATH。所选目标全部成功后才写入本次清单；目录内其他已有文件不列入清单。

固定构建参数为 `CGO_ENABLED=0`、`-trimpath`、`-buildvcs=false` 和 `-ldflags="-s -w -X main.version=<版本>"`。构建失败立即报错，结束时恢复脚本调用前的 GOOS、GOARCH 和 CGO_ENABLED。调试信息被移除，需要调试时使用普通 `go build` 另建调试制品。

每次生成：

- 所选目标的 `myenv-<目标>` 二进制，Windows 带 `.exe` 后缀。
- `build-manifest.json`：版本、Go 编译器、参数、目标、文件大小和 SHA-256。
- `SHA256SUMS`：本次所选制品的摘要清单。

校验清单只记录文件摘要，不是签名或来源认证。性能报告必须引用实际受测制品的摘要；不同构建不能直接沿用旧报告作为门禁通过证据。

所有选定目标先构建到输出目录下的独立 `.build-<随机ID>` 暂存目录，摘要和清单也先写入该目录。某个目标编译失败时保留此前正式产物，部分构建文件留在暂存目录供检查。重复目标只构建一次。

全部编译和清单生成成功后，脚本逐文件替换所选制品，最后替换摘要清单和 manifest。这个步骤不是跨文件原子事务；复制失败、文件占用或进程中断可能留下不完整更新，应重新运行构建，并在脚本成功结束后核对清单中的全部摘要。不要在脚本运行过程中分发输出目录。

Windows PowerShell 7 可用以下工具记录单个 CLI 进程的峰值工作集：

```powershell
./scripts/measure-memory-windows.ps1 -Output .build/perf/memory-version.json
./scripts/measure-memory-windows.ps1 -CommandArguments '--help' -Output .build/perf/memory-help.json
```

默认使用 `dist/release/myenv-windows-amd64.exe`、每个命令运行20次；可指定 `-Executable` 和 `-Samples`。报告保存全部样本、最大值、命令参数和二进制摘要，不覆盖已有报告。它运行给定命令，不要将会修改数据的命令当作无副作用测量。

测量 `sync --locked --no-input --json` 时，可传 `-ExpectedGeneration <已应用代ID>`，工具会逐次检查成功、无变化和代 ID 一致；不符合时不会生成成功报告。该检查只用于同步 JSON 输出，读取缓冲上限为64 Ki字符。

计数使用 Windows [GetProcessMemoryInfo](https://learn.microsoft.com/en-us/windows/win32/api/psapi/nf-psapi-getprocessmemoryinfo) 的 [PeakWorkingSetSize](https://learn.microsoft.com/en-us/windows/win32/api/psapi/ns-psapi-process_memory_counters)，保持进程句柄直到读取结束。它不包含测量宿主或子进程树，不测延迟，也不清空文件缓存。

状态命令内存测量也可使用 `-StatusResponse -ExpectedGeneration <代ID>`，配合 `-CommandArguments @('-C', '<隔离项目目录>', '--json')`。脚本逐次核验 JSON 为 ready、changed=false 且代ID匹配；缺少 ExpectedGeneration 时拒绝启动。该模式仍只测 CLI 父进程峰值工作集，不测子进程树，也不能替代延迟门禁。真实混合测试通过 MYENV_TEST_MIXED_STATUS_MEMORY 单独启用这一模式。
