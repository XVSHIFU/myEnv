目标范围：按 docs/architecture.md 实现 CLI 首版 T00–T06。
当前任务：T03 Python 纵向闭环；状态 in_progress（2026-09-08 最新）。

当前实现概况：
- T00/T01 基础 CLI、严格配置、init 交互、帮助/离线手册与补全已实现；最终选项合同和版本范围边界继续随首版验收核对。
- T02 Node 的真实 Windows 同步/运行、无变更返回、失败保留旧代已有证据；Linux/macOS 可交叉构建，原生验证未完成。
- T03 已接通固定 uv 准备、Python 版本选择、受管解释器、新代 venv、原生锁摘要、sync/use/run、构建预检与 CLI 本次确认。真实 Windows 无依赖项目/空组闭环通过，失败后原生锁变更报告通过。真实第三方 wheel/非空组、完整配置及 workspace 输入闭包、全部构建来源许可语义仍待验证或补齐。
- T04 部分：项目 status/doctor/rollback/current 支持 Node/Python；用户 profile 未实现。T05/T06 的清理、完整恢复/租约身份、共享缓存、原生平台与性能发布门禁尚未完成。
- 首版未验收完成；以下为历史记录，旧阶段的“尚未实现”不代表当前代码状态。最新证据追加在文末。

初始 T01 阶段记录：

已完成：
- T00：完整方案副本、根 AGENTS.md、Go+Cobra 可构建入口；起点无 Git 仓库，未假定远端地址。
- T01 部分：internal/config 实现 1 MiB 输入限制、严格 YAML schema、重复键/未知字段/危险标签/别名拒绝、深度限制、Python 项目路径及链接越界检查、VCS 边界发现。
- init 静态读取 .python-version、.node-version、.nvmrc、package.json engines.node、pyproject.toml requires-python 和 dev 组；独占创建配置，已有配置不覆盖；缺失/冲突返回 NEEDS_INPUT。CLI 有状态只读入口、JSON 结果结构、init 帮助和退出码。

必要决定：
- 本地模块名 myenv；Go 1.26.6；Cobra v1.10.2。
- YAML 使用 go.yaml.in/yaml/v3 v3.0.5；TOML 使用 github.com/pelletier/go-toml/v2 v2.4.3，避免用文本匹配误读 pyproject；未引入应用框架。官方 API 来源 https://pkg.go.dev/go.yaml.in/yaml/v3 和 https://github.com/pelletier/go-toml 。新增解析库对二进制和 RSS 的影响待 T06 实测。
- GOPATH、GOCACHE、TMP/TEMP 均注入 .build 下；不修改用户 profile 或 Shell。

有效验证（2026-09-08，Windows amd64 / Go 1.26.6）：
- go test ./internal/config ./internal/cli 通过：非法 YAML、VCS 边界、init JSON、非交互缺失/冲突、已有配置字节保留、Python manifest/dev 检测。
- go build -o dist/myenv.exe ./cmd/myenv 通过；dist/myenv.exe init --help 输出参数与示例。
- go mod tidy 成功。当前相关文件 SHA256 记录 .build/t01-fingerprint.txt；本次测试覆盖的文件对应该记录。旧 root.go 验证指纹已失效。
- 沙箱内路径测试两次报 Access is denied；包括将临时目录移入工作区后的尝试。提升执行后相同配置测试通过，说明 Windows EvalSymlinks 的祖先目录访问受沙箱限制。未删减路径检查。

未完成/未验证：
- T01 尚缺 TTY 必要信息提问、完整离线手册和补全、细化 IO 错误退出码、所有选项组合的结构化错误校验。版本选择目前支持稳定数字版本，不支持预发行/命名别名及 Node hyphen range；需补齐原生范围语义核对。
- 状态目前明确标记 environment=not_checked，尚未接入活动环境；sync/use/run/doctor/rollback/clean 未实现。没有安装可用性声明。
- T02–T06 均待完成；三平台运行与性能预算尚未验证。

下一步：完成版本约束原生语义核对及 T01 交互/帮助合同，再接通 T02 Node 解析、流式制品校验、SQLite 状态及新代提交、运行与失败保护。只重测受修改影响范围。

本次接续（2026-09-08）：
- 新增 internal/config/version.go：稳定版本前缀、比较、排除、Node caret/tilde、Python compatible-release 与 OR 分支；配置验证拒绝非法范围。init 对多来源求交集，保留全部约束，不再将 3.12 与 >=3.12 误判为冲突。
- 新增版本选择/交集定向测试，Python init 集成测试同时提供 .python-version 和 requires-python，证明两者兼容时能生成配置。
- `go test ./internal/config -run 'TestConstraint|TestMerge'` 通过；随后受影响范围 `go test ./internal/config ./internal/cli` 及 `go build -o dist/myenv.exe ./cmd/myenv` 通过（Windows amd64，路径测试提升执行，数据仍隔离）。文件指纹更新到 .build/t01-fingerprint.txt。
- 本轮尝试查询官方范围规范的 web 工具连接失败；不将该项当作已核实。下一轮应核对 Python/Node 范围边界，并补相应缺项，避免仅靠当前测试宣布完整兼容。

帮助/补全接续（2026-09-08）：
- 新增 internal/cli/help.go：help manual 输出随二进制发布的离线手册；命令参考和参数表从 Cobra 命令对象生成；help <command> 输出完整命令说明；completion 支持 bash/zsh/fish/powershell。根帮助增加手册入口。
- 手册已有配置、版本范围、输出合同、源码构建/卸载和各 Shell 补全启用步骤。用户 profile、代理/镜像与恢复章节仍须随 T02–T05 的真实行为补齐，未标为首版完整手册完成。
- 验证：go build -o dist/myenv.exe ./cmd/myenv 通过；myenv -C Z:/nonexistent help manual 成功；help init 展示帮助参数及公共参数；四类补全生成成功，产物 .build/completions/；PowerShell Parser.ParseFile 无语法错误。completion powershell --json 返回结构化 USAGE_ERROR，退出 2。
- Bash/Zsh/Fish 仅生成，未在对应 Shell 执行；PowerShell 仅语法校验，未修改真实 Shell 配置。指纹 .build/help-fingerprint.txt；旧 t01 指纹中 root.go 已失效，配置范围代码未变化、既有证据继续有效。
- T01 仍 in_progress：下一步补 TTY 缺参处理及范围原生语义核对，随后进入 T02；无当前硬阻塞。

范围规范核对（2026-09-08）：
- 已读取 https://github.com/npm/node-semver 的范围定义及 https://packaging.python.org/en/latest/specifications/version-specifiers/ 的比较规则；此前网络核对缺口已解除。
- 实现 Node hyphen range，包含完整上界和部分上界、与其他约束相交；Python 比较/compatible-release 不再接受 wildcard。手册同步更新 Node 范围说明。
- `go test ./internal/config -run 'TestConstraint|TestMerge'` 通过；`go build -o dist/myenv.exe ./cmd/myenv` 通过。新增案例验证 20.1 - 22.3 包含 22.3.9 而排除 22.4.0，以及完整上界与交集限制。此次仅重测受影响范围。
- T01 尚未完成，仍需 TTY 输入与 IO/JSON 合同补齐；T02–T06 未完成。规范核对不代表支持所有原生语法（仍仅稳定数字 runtime 版本）。

错误合同接续（2026-09-08）：
- 项目发现新增 ErrNoProject，与真实文件访问错误区分；不存在的 -C 目录不再伪装为无项目成功，而是 IO_ERROR/退出 1。
- Cobra 提前遇到未知参数时，从原始参数恢复 JSON 输出意图，保证 init --invalid --json 返回单个 JSON 错误。路径参数值与 -- 分隔符不会当作输出选项。
- 新增 internal/cli/errors_test.go；`go test ./internal/cli -run TestErrorOutputContract`、`go test ./internal/config -run TestDiscoverVCSBoundary` 通过，`go build -o dist/myenv.exe ./cmd/myenv` 通过。当前改动指纹 .build/error-contract-fingerprint.txt；对应旧指纹失效。未重复无关安装测试。
- 手册补充 I/O 退出码。T01 仍缺 TTY 必要信息选择，完整参数组合还需随 run 等命令接入验证。下一步实现 init TTY 输入并进入 T02；整体首版未完成。

init 交互接续（2026-09-08）：
- InitWithInput 接受必要信息选择回调；缺失工具/版本与冲突声明可选择，已有配置不提问、不覆盖。终端输入使用有界 scanner，EOF/空输入取消。
- CLI 使用 golang.org/x/term v0.45.0 的 IsTerminal 检测输入/输出；CI、JSON、no-input 或管道禁用交互。新增 Go 官方 x/term（及 x/sys v0.47.0）避免手写各平台控制台检测；内存影响待性能门禁实测，模块版本已固定。
- `go test ./internal/config ./internal/cli` 通过；涵盖选择、冲突、已有配置不提问、EOF 及原有非交互合同。第一次 gofmt 通配符在 PowerShell 下未展开，已改为显式文件路径并完成格式化；随后 go mod tidy 与 go build -o dist/myenv.exe ./cmd/myenv 通过。
- Windows PTY 真实验证：dist/myenv.exe -C .build/tty-init init 显示提示，输入 node@22 后退出 0，生成 schema: 1/tools.node: "22"。产物 .build/tty-init/myenv.yaml；未修改用户环境。指纹 .build/init-interaction-fingerprint.txt 取代相关旧记录。
- 下一步：T01 收尾审计（选项组合与帮助语义），随后直接推进 T02 的 Node 下载、SQLite 新代提交与 run。整体首版尚未完成；不把交互通过当作环境安装已完成。

T02 后端接续（2026-09-08）：
- 新增 internal/backend/node.go：官方 index.json 解析、稳定版本数值排序、三目标制品映射、精确版本/npm/URL/SHA256/固定后端身份。元数据限制大小与超时；不依赖远端排序。来源 https://nodejs.org/dist/index.json 。
- 新增 download.go：64 KiB 缓冲流式下载到唯一 staging 文件，512 MiB 压缩包上限，下载前锁定 SHA256 校验、Sync 落盘、失败删除本次临时文件。尚未接入解包、共享缓存发布或 CLI sync。
- TestNodeResolve 通过；显式 MYENV_TEST_NETWORK=1 的 TestNodeOfficialMetadata 真实官方网络验证通过，解析 Windows Node 22.23.2/npm 10.9.8，SHA256 1177b4137ba5adaa56354ae40f1080c7450e8ae09cecb47da459d1c52ac99f97。这只证明解析，未下载/执行该制品。
- TestDownloadChecksumAndCleanup 验证内容校验与失败清理通过。第一次测试构建遇到 Windows 临时测试 exe 被占用，按方案只进行一次确认重试后通过；未改测试预期。指纹 .build/node-backend-fingerprint.txt。
- 当前 T02 in_progress；T01 基本使用已验证，完整参数组合/最终帮助仍随后续命令收尾。下一步 Node 安全解包、SQLite 单一活动引用、新代准备与失败保护，接通 sync/run；不能把后端局部实现标为 T02 完成。

T02 ZIP 解包接续（2026-09-08）：
- 新增 ExtractZIP，要求新目标目录；拒绝绝对路径、.. 越界、反斜杠/冒号、Windows 保留设备名、尾点/空格、符号链接与特殊文件；文件独占写入，大小与 CRC 检查、落盘。限制 100000 项和 2 GiB 总展开大小。
- `go test ./internal/backend -run TestExtractZIPBoundary` 通过，Windows amd64、工作区临时目录；验证恶意路径、链接、正常内容及已有目的目录拒绝。指纹 .build/extract-fingerprint.txt。
- 尚缺 Unix tar.gz 解包及合法归档链接处理、真实 Node 包解包/执行验证、SQLite 状态、新代提交与 CLI 接入。不能声明真实安装完成。下一步优先接通 Windows 真实包和 SQLite 发布链，随后扩展 Unix 解包。

真实 Node 准备门禁（2026-09-08）：
- 新增 TestNodeOfficialPrepare：显式 MYENV_TEST_PREPARE=1 + MYENV_TEST_ARTIFACTS 隔离目录，串联真实 Resolve/Download/ExtractZIP，检查 node --version 和 npm-cli.js --version，保留制品与 prepared.json 证据；普通单元测试跳过网络门禁。
- 已启动 `go test ./internal/backend -run TestNodeOfficialPrepare -v`，Windows amd64、MYENV_TEST_ARTIFACTS=.build/node-real、提升网络执行。进程 session_id=70420 最近轮询仍在运行，尚无成功/失败结果；不要重启或重复下载，下一轮先轮询同一 session。Go 测试内部 context 超时 5 分钟。当前不能宣称真实包已准备成功。
- 集成门禁指纹 .build/node-integration-fingerprint.txt。SQLite 驱动候选官方 API 已查阅 https://pkg.go.dev/modernc.org/sqlite ，尚未引入依赖或声称平台验证完成。

SQLite 接续（2026-09-08）：
- 新增 internal/state/store.go：最小 generations/active 表，FULL synchronous、外键与 busy_timeout；Publish 在同一事务插入新代并以 expectedID 条件切换当前/前代，冲突回滚。尚未接 CLI、准备完成标记或操作恢复。
- 新增 TestPublishPreservesActiveOnConflict 覆盖旧代保留、冲突回滚后重试及重新打开持久化；尚未运行，等待依赖下载。仅 gofmt 完成，指纹 .build/state-fingerprint.txt。
- 驱动选择 modernc.org/sqlite（纯 Go，避免 C 工具链依赖），API 来源 https://pkg.go.dev/modernc.org/sqlite 。体积/RSS 和三平台实际运行必须后续验证，不因驱动文档标称支持而标通过。
- `go get modernc.org/sqlite@latest` 正在 session_id=25377 运行，显示下载 v1.58.0；最近轮询未完成。下一轮先继续该句柄，不重复下载。
- Node 真实准备 session_id=70420 最近仍运行，无终态输出；.build/node-real 临时下载文件仍为 0 字节。继续观察同一进程，不凭文件大小重启。整体无完成声明。

runner 环境注入接续（2026-09-08）：
- 新增 internal/runner/environment.go：从显式 parent/env/bin 参数构建子进程环境，项目覆盖、运行时 PATH 前置、Windows 大小写去重及驱动器当前目录变量保留，不修改宿主环境。尚未启动子进程或登记租约，非 run 完成声明。
- `go test ./internal/runner` 通过（Windows amd64，隔离临时目录），验证 PATH 合并、父环境保留、非法键拒绝。指纹 .build/runner-fingerprint.txt。
- 本轮 session 70420 已终态成功：TestNodeOfficialPrepare 通过（289.46 秒），真实 Node 22.23.2 与 npm 10.9.8 均执行版本检查成功，制品保留在 .build/node-real，证据 prepared.json。不要重复下载/测试该未变化后端。session 25377 仍运行，新增输出下载 modernc.org/libc v1.75.6；下一轮先继续轮询，随后运行状态库测试。


锁模型接续（2026-09-08）：
- 新增 config.Lock/PlatformLock/RuntimeLock，记录配置摘要、平台、精确版本、固定后端、version/artifact 证据、URL/SHA256/npm；ReadLock 有文件大小上限、未知字段拒绝和单 JSON 校验；WriteNewLock 独占创建并落盘，尚不处理替换。
- Digest 对 Config JSON 求 SHA256，忽略 YAML 注释/映射顺序，版本变化会改变摘要。Python 原生输入摘要仍需 T03 接入，当前模型不宣称完整首版锁覆盖。
- `go test ./internal/config -run 'TestLock|TestConfigDigest'` 通过，验证锁往返、不覆盖已有锁、缺失制品摘要拒绝以及配置摘要稳定性。指纹 .build/lock-fingerprint.txt。
- session 25377 本轮两次轮询仍 live、无新增输出；下轮继续该依赖进程，完成后跑 state 测试并接 core sync。Node 真实准备已有通过证据，不重复下载。

状态库验证接续（2026-09-08）：
- session 25377 终态成功，modernc.org/sqlite v1.58.0 及依赖已固定到 go.mod/go.sum，不再等待该进程。
- `go test ./internal/state` 首次发现 Windows URI authority C: 错误；已将绝对路径编码为 file:///C:/...，重测通过。覆盖活动引用冲突回滚、旧代保留、后续成功发布和数据库重开持久化。
- 新增 PrepareNodeZIP/VerifyNode 生产入口：解包到最终目录，绝对路径执行 --version 校验，诊断输出最多 4 KiB；未连接 CLI。`MYENV_TEST_PREPARED_RECORD=.build/node-real/prepared.json go test ./internal/backend -run TestVerifyRetainedNode -v` 通过，复用现有真实 Node，验证正确版本及不匹配拒绝，未重复下载。
- 当前指纹 .build/state-verified-fingerprint.txt；旧 state 指纹失效。下一步 core 同步与工作区串行修改保护，接通 sync/run；Unix 解包及平台运行仍待做。整体目标未完成。

工作区串行保护（2026-09-08）：
- 新增 LockWorkspace：Windows 使用 LockFileEx，Linux/macOS 使用 flock；持有文件句柄即持有锁，进程退出由 OS 释放。不删除锁文件，避免不同进程锁住不同 inode/文件身份。等待支持 context 取消。
- `go test ./internal/state -run TestWorkspaceLockContention` Windows amd64 通过，覆盖竞争超时、释放后重新获取。Linux/macOS 代码尚未编译/运行，未标为平台通过。指纹 .build/workspace-lock-fingerprint.txt。
- 仅模块完成，尚未证明整个 sync 的跨进程串行和崩溃恢复；下一步 core 接入本锁、输入复核、Node 准备及 SQLite Publish。整体仍 T02 in_progress。

锁文件替换接续（2026-09-08）：
- 新增 ReplaceLock：先校验、同目录临时文件完整写入/Sync，再按平台替换；Windows MoveFileEx(REPLACE_EXISTING|WRITE_THROUGH)，Unix rename 后同步父目录。调用者必须持有工作区修改锁；不将文件替换宣称为和状态库跨资源原子事务。
- 扩展 TestLockEvidenceAndNoOverwrite，验证无效新锁不损坏旧锁、有效替换可读；`go test ./internal/config -run TestLockEvidenceAndNoOverwrite` Windows amd64 通过。此前先运行原有测试只证明编译及旧路径，扩展后的结果才覆盖替换行为。
- 指纹 .build/lock-replace-fingerprint.txt；Linux/macOS 替换未运行。下一步 core 将已有模块串联为同步，T02 仍未完成。

core 同步接续（2026-09-08）：
- 新增 Service.Sync，把配置读取/摘要、工作区锁、有效 Node 锁保留、locked 校验、锁写入、下载、最终目录准备、输入复核和 SQLite Publish 串联。当前 Windows Node-only 纵向实现；旧代在准备失败时不切换，健康相同摘要快速返回。
- `go test ./internal/core` 仅证明编译通过（明确输出 no test files），尚未进行核心集成验证；不能凭此标同步闭环通过。指纹 .build/core-sync-fingerprint.txt。
- 下一步复用 .build/node-real 已验证归档，通过本地 HTTP 测试服务器验证实际 Service.Sync 的首次/幂等/失败旧代保护，然后接入 CLI sync/run。当前缺少 operation 登记/完成标记、共享缓存、非 Windows 准备及完整选项，不属于首版完成状态。

核心集成与 CLI sync（2026-09-08）：
- TestSyncRetainedNode 使用本地 HTTP 提供已验证真实 Node ZIP，调用生产 Service.Sync，验证首次发布、第二次零 HTTP 请求且不换代、注入下载 503 后旧 Node --version 可执行。`go test ./internal/core -run TestSyncRetainedNode -v` 通过，10.11 秒；未重新联网下载。
- CLI sync/--locked 接入同一 core 服务，支持既有 JSON/no-input，阶段信息到 stderr，帮助从命令定义生成。go build -o dist/myenv.exe ./cmd/myenv 通过，sync --help 验证成功；后续更新帮助文案尚需最终构建时纳入。
- 仍缺 run 及 operation/共享缓存/多平台和完整 sync 选项，不能标 T02 或首版完成。下一步 run 选代、输入校验、进程执行与租约。

run 选代租约接续（2026-09-08）：
- SQLite 新增 leases 表；AcquireActive 在同一事务读取活动代并登记 supervisor PID/代引用，ReleaseLease 在执行结束后释放。不会因时间过期自动释放。
- `go test ./internal/state -run TestLeaseRetainsSelectedGeneration` Windows 通过，验证未应用时拒绝、活动代切换后原租约仍保护旧代、显式释放。
- 新增 core.SelectRun：拒绝缺环境/声明漂移（current 可显式允许），检查实际入口并返回调用者必须释放的租约；`go test ./internal/core -run '^$'` 仅编译通过，尚未验证执行链。指纹 .build/run-selection-fingerprint.txt。
- 仍待 CLI argv 直通、进程监督/信号、退出码、权限完整检查、generation 配置快照（current 不应采用后改 env）；不将选代模块当作 run 完成。后续清理必须查询 leases，并补进程身份/崩溃恢复。

环境代配置快照（2026-09-08）：
- 同步提交前独占写入 generation/config.yaml 并 Sync；SelectRun 读取该快照并与状态库摘要复核，返回应用时配置，而非当前修改后的 env。
- 扩展 TestSyncRetainedNode 验证普通选代拒绝声明漂移、current 选代保留原 env；`go test ./internal/core -run TestSyncRetainedNode -v` 通过（9.56 秒，复用真实归档，无外网下载）。指纹 .build/snapshot-fingerprint.txt。
- 仍需让无变更 sync 检查快照缺失/损坏以修复早期无快照代；CLI run 与进程监督尚未接入。下一步继续运行入口，不标 T02 完成。

runner 子进程执行（2026-09-08）：
- 新增 runner.Execute，要求绝对 executable，独立 argv/env/cwd/stdio，等待子进程结束并返回退出码；未经过 shell。context 目前使用 CommandContext 默认终止，完整平台信号转发仍需补齐。
- `go test ./internal/runner -run TestProcessNodeArguments -v` 通过：复用真实 Node，验证空格/引号/中文/--json 参数，stdin→stderr 内容与退出码 17。指纹 .build/process-fingerprint.txt。
- 下一步 CLI run 参数边界、命令查找（包括 npm launcher）、选代租约释放、信号转发；当前只是执行模块，不是 run 命令已完成。

CLI run 接续（2026-09-08）：
- 新增 run/--current，Cobra 停止在首个 command 后解析选项；选代/租约、快照 env、runner 执行接通。node 使用所选绝对入口，npm/npx 经配套 JS launcher；子进程非零退出直接返回，不打印 myEnv 错误。
- `go build -o dist/myenv.exe ./cmd/myenv`、run --help 成功；`go test ./internal/cli -run TestRunRetainedNode -v` 通过，真实 Node 验证 command 后 --json 原样传递与退出 17。指纹 .build/cli-run-fingerprint.txt。
- 尚缺完整信号监督、通用命令按子环境 PATH 查找（当前 LookPath 使用父 PATH，需修正）、权限与进程树租约清理、启动错误统一退出 1；帮助首页旧文案仍需同步。run 基本接通不等于 T02/首版完成。

子环境 PATH 查找（2026-09-08）：
- 通用 run 命令改用 runner.Lookup，读取子环境 PATH/PATHEXT，支持显式相对路径，避免父 PATH 泄漏；受管 node/npm/npx 保持直接入口。
- `go test ./internal/runner -run TestLookupChildPath` 与 `go build -o dist/myenv.exe ./cmd/myenv` 通过，覆盖子 PATH、空 PATH 不回退父环境及相对路径。指纹 .build/lookup-fingerprint.txt。
- Windows .cmd/.bat 可被找到但尚缺可靠 shell launcher，不能宣称任意脚本全部运行通过；信号监督、租约进程身份及其他既有缺口继续保留。

同步快照健康检查（2026-09-08）：
- 无变更 sync 的快速返回现在要求活动代配置快照可读且摘要匹配，缺失或修改的快照进入重建流程，解决早期无快照代无法通过 sync 修复的问题。
- `go test ./internal/core -run TestSnapshotHealth` 通过，覆盖缺失、有效、摘要变化三种状态。指纹 .build/snapshot-health-fingerprint.txt；尚未新增真实重建集成场景，原同步证据中快速返回分支因本次改动需后续针对性集成验证。
- 整体 T02 仍未完成，继续信号监督与 operation 恢复等明确缺口；不扩展首版范围。

快照重建集成验证（2026-09-08）：
- 扩展真实归档同步测试：首次同步后删除测试代 config.yaml，locked 同步重建到新代，普通选代恢复成功；随后仍验证声明漂移和下载失败旧代可用。
- `go test ./internal/core -run TestSyncRetainedNode -v` 通过，19.15 秒。复用已验证 ZIP、本地 HTTP，不联网重新下载。该证据补齐此前快照健康检查的核心集成缺口，指纹 .build/snapshot-repair-fingerprint.txt。
- 下一步仍为信号监督、operation 登记与恢复，继而 Python/用户操作/清理/发布；首版目标未缩减，当前未完成。

操作登记接续（2026-09-08）：
- 新代解包前写 operations(id,directory,digest,owner_pid,preparing)；正常错误返回登记 failed，Publish 在活动切换同一事务中登记 complete，后续失败清理不会覆盖 complete。
- `go test ./internal/state` 验证现有行为通过；新增 `go test ./internal/state -run TestOperationPublication` 覆盖完成/失败状态转换通过。指纹 .build/operation-fingerprint.txt。
- 尚缺完成文件标记、进程身份与崩溃恢复逻辑，下载仍在 operation 登记前；下一步完善完整操作生命周期，不能宣称恢复已经完成。

中断操作恢复（2026-09-08）：
- core 在持有工作区 OS 修改锁后调用 RecoverInterrupted，将遗留 preparing 操作标记 failed，不删除文件、不改变活动代。依据独占锁所有权，不依赖心跳或 PID 猜测。
- `go test ./internal/state -run TestRecoverInterruptedKeepsActive` 通过，覆盖恢复计数、保留活动引用、重复恢复无变化。指纹 .build/recovery-fingerprint.txt。
- 该测试是数据库状态级验证，真实 kill/重启及后端子进程存活尚未验证；下载登记、完成文件标记和运行租约身份仍缺。完整崩溃恢复未达到验收。

下载操作生命周期（2026-09-08）：
- BeginOperation 提前到下载之前，下载失败走同一 failed 登记；没有将未下载文件误认为准备完成。
- `go test ./internal/core -run TestDownloadFailureRegistered` 通过，本地 HTTP 503 注入后检查 SQLite 恰有一条 failed 操作；仅此前 compile-only 结果不能替代本行为证据。指纹 .build/download-operation-fingerprint.txt。
- 临时归档路径尚未记录在 operation 中，崩溃后文件归属清理仍需完善；完成标记和真实 kill 测试等缺口保留。整体未完成。

下载目录归属（2026-09-08）：
- 下载临时文件改放 .myenv/operations/<operation-id>/，目录编号对应状态库操作记录，不再散落在 .myenv 根目录；未来 clean 可按明确操作归属处理。
- 扩展下载失败测试，核对 failed 操作对应目录存在且 HTTP 503 未留下内容；`go test ./internal/core -run TestDownloadFailureRegistered` 通过。指纹 .build/operation-directory-fingerprint.txt。
- 尚未实现 clean 或真实断电/kill 恢复；完成标记与进程监督缺口仍在，整体继续开发。

准备完成标记（2026-09-08）：
- 同步在快照落盘后写 complete 摘要标记，再提交 SQLite 活动引用；run 和无变更 sync 检查标记。缺标记的旧代不再被快速视为健康。
- `go test ./internal/core -run TestSnapshotHealth` 与 `go test ./internal/cli -run TestRunRetainedNode` 通过，覆盖缺失标记拒绝、有效快照与真实 Node 执行。指纹 .build/complete-marker-fingerprint.txt。
- 尚未验证真实 kill/重启与目录级持久化平台要求；标记存在不是完整深度内容校验。下一步继续平台信号/恢复验证及后续 T03–T06。

早期跨平台构建（2026-09-08）：
- Windows 宿主设置 CGO_ENABLED=0 GOOS=linux GOARCH=amd64，`go build -o dist/myenv-linux-amd64 ./cmd/myenv` 成功，session 2310 退出 0。证明当前 SQLite/平台适配可交叉编译，不代表 Linux 运行通过；sync 仍限制 Windows。
- CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 的 `go build -o dist/myenv-darwin-arm64 ./cmd/myenv` 已启动 session 28026，接续时确认该句柄终态，不能重复启动或将编译当作运行。
- session 28026 最后轮询退出 0，macOS arm64 交叉编译成功；无未完成构建句柄。产物约 13.8 MB，Linux 约 14.0 MB（非最终发布构建）；两平台均未实际运行，仍待真实门禁。

run 缺环境错误合同（2026-09-08）：
- ENV_NOT_READY 映射退出 1，包装的 PathError 不覆盖产品错误码。command 后 --json 仍作为子参数，不改变 myEnv 输出模式。
- `go test ./internal/cli -run TestRunMissingEnvironmentExit` 通过，验证错误码/退出码、stdout 为空、缺环境时不创建 .myenv。指纹 .build/run-error-fingerprint.txt。
- 尚有平台信号、批处理 launcher、权限/租约身份、Python、用户操作和发布等工作；目标保持进行中。

子进程取消结果（2026-09-08）：
- Unix 信号终止退出码转换为 128+signal，Windows 保留平台退出码。
- `go test ./internal/runner -run TestCancelRetainedNode -v` Windows 真实 Node 通过，context 超时后进程结束、返回非零非负状态（0.26 秒），未重下载。指纹 .build/cancel-fingerprint.txt。
- Unix 分支尚未运行；仍缺 CLI 信号转发、进程树监督，不能把 context 终止测试当作完整取消验收。

CLI 状态说明与释放诊断（2026-09-08）：
- run 租约释放失败现在输出 stderr 诊断，不再静默丢弃错误；保持已经执行的子进程退出状态。
- 首页/离线手册纠正 run 尚未实现的旧文案，明确 Windows Node-only 当前范围。`go build -o dist/myenv.exe ./cmd/myenv` 成功、--help 展示实际命令列表；不为文案重复安装测试。指纹 .build/cli-current-fingerprint.txt。
- 当前仍 T02 收尾：信号监督、真实崩溃恢复、平台解包、共享缓存；T03 Python、T04 use/profile/doctor/rollback、T05 clean/租约完整身份、T06 性能发布均未完成。普通状态仍需接活动数据库，不能用已有局部测试声明首版完成。

精确锁版本校验（2026-09-08）：
- RuntimeLock.Version 要求规范三段数字精确版本，拒绝前缀、范围、v 前缀和前导零；声明范围语义不受影响。
- `go test ./internal/config -run TestLockRequiresCanonicalExactVersion` 通过。指纹 .build/lock-version-fingerprint.txt。
- 当前仍为首版开发中，原有平台/信号/Python/用户操作/发布缺口未完成。

发布操作一致性（2026-09-08）：
- Publish 在同一数据库事务内检查已有 operation 的目录、输入摘要和 preparing 状态；不匹配时拒绝发布，避免将另一组输入记为完成。
- `go test ./internal/state -run TestOperationPublication -count=1` 通过（Windows，0.179 秒），覆盖摘要不匹配拒绝后仍能正常发布，以及完成/失败状态转换。
- 无 operation 的历史调用仍兼容；此检查不替代文件完整性检查和真实崩溃恢复验证。T02 及后续 T03–T06 仍未完成。

Unix Node 归档准备（2026-09-08）：
- 新增流式 tar.gz 解包：条目数/展开大小限制、路径越界和重复条目拒绝、CRC 校验、可执行权限保留；延后创建仅指向归档内普通文件的相对符号链接，拒绝硬链接和特殊条目。
- 新增 PrepareNodeTarGZ，识别 Linux x64 与 macOS arm64 官方目录并执行既有版本验证。尚未接入 core 的平台分派，sync 仍限 Windows。
- `go test ./internal/backend -run TestExtractTarGZ -count=1 -v` Windows 通过，覆盖普通文件、越界路径、绝对/越界/缺目标链接、硬链接、重复路径、损坏 CRC 和目录不可复用；有效符号链接测试因 Windows 权限前提跳过，Unix 权限位分支尚未运行。
- `go test ./internal/backend -run '^TestPrepare' -count=1` 编译成功但没有匹配测试，仅作编译证据。真实 Unix Node 准备和执行未验证，不能作为跨平台验收。

Node 平台分派接入（2026-09-08）：
- sync 使用 runner 平台识别，Windows 走 ZIP，Linux amd64/glibc、macOS arm64 走 tar.gz；npm/npx 在 Unix 使用 lib/node_modules 下的实际入口。帮助文案明确 Unix 运行验证待完成。
- Linux 识别显式拒绝 WSL、musl 和缺少预期 glibc loader 的宿主，避免默默选择不同平台制品；此检查不承诺 glibc 版本兼容性。
- `go test ./internal/runner -run TestPlatformIdentification -count=1` 通过。设置 MYENV_TEST_PREPARED_RECORD 为保留制品记录后，`go test ./internal/core ./internal/cli -run 'TestSyncRetainedNode|TestRunRetainedNode' -count=1` Windows 通过（core 21.705 秒，cli 0.513 秒），未重新下载官方归档。
- 这补验了平台分派后的 Windows 真实同步、无变更、缺快照重建、失败保护及 run；Unix 仍缺实际运行证据，首版 T02–T06 继续进行。

三平台 Node 验证入口（2026-09-08）：
- 官方准备测试改用宿主平台与对应 ZIP/tar 准备入口；保留归档同步测试按记录生成文件名及平台元数据，并拒绝宿主不匹配的记录。CLI 真实运行测试新增 npm/npx 版本断言。
- `go test ./internal/backend ./internal/core ./internal/cli -run 'TestSyncRetainedNode|TestRunRetainedNode|TestVerifyRetainedNode' -count=1` 复用 Windows 官方制品通过（backend 0.382 秒、core 20.352 秒、cli 2.119 秒）。修改后的联网官方准备测试未重跑；未重复下载。
- 新增 scripts/verify-node.ps1、scripts/verify-node.sh 及 docs/node-validation.md，首次准备、后续复用制品。PowerShell Parser 静态语法检查通过；脚本整体未执行，bash 语法及 Unix 实际运行尚未验证。没有远端 CI 或发布操作。
- 完整目标继续保持，下一步补进程监督等功能；Unix 门禁等待对应原生宿主，不将脚本存在计作运行通过。

Node 同步预览（2026-09-08）：
- sync --dry-run 复用现有解析逻辑，在写锁、下载和安装前返回平台、Node 锁定信息与 needs_apply；changed 始终为 false。可与 --locked 组合，缺锁或过期仍报 LOCK_OUT_OF_DATE；实际 sync 不使用预览作为执行凭证。
- 新增 state.OpenReadOnly，mode=ro 且跳过 schema 初始化；预览不创建工作区修改锁、不登记/恢复 operation、不领取租约。
- `go test ./internal/state ./internal/core -run 'TestReadOnlyStore|TestSyncRetainedNode' -count=1` 通过（Windows，state 0.170 秒，core 18.324 秒）。复用官方 Node 归档；新增断言验证首次预览不下载运行时、不创建 .myenv/myenv.lock，以及已应用预览离线且声明/锁/数据库字节不变。既有真实同步失败保护继续通过。
- `go test ./internal/cli -run TestDryRunLockedMissingLock -count=1` 通过，覆盖组合参数、JSON 错误/退出码及无状态目录副作用。
- 当前预览仍限 Node-only；Python 动态构建需求报告随 T03 完成。进程树监督、更新解析、其他用户操作和完整平台门禁等既有工作仍待实现。

显式 Node 更新解析（2026-09-08）：
- sync --update node 跳过旧 runtime 解析并按声明重新解析；--dry-run 可组合，--locked 在 CLI 与 core 两层拒绝组合。未知更新目标明确失败，Python 更新随 Python 后端实现。
- `go test ./internal/core ./internal/cli -run 'TestUpdatePreview|TestDryRunLockedMissingLock' -count=1` Windows 通过（core 0.194 秒、cli 0.208 秒）。本地元数据测试验证默认保留 22.1.0 且不联网，显式更新解析 22.9.0，预览仅请求索引/校验和，不写锁/状态，并拒绝无效参数。
- 尚未运行跨版本真实更新安装；安装继续调用既有同步准备与发布流程。同版本显式更新目前也重建环境，后续可在具备活动制品身份比较后避免。首版未完成。

活动代运行时锁快照（2026-09-08）：
- 新代发布前写入该代 myenv.lock 快照；无变更 sync 除配置摘要和完成标记外，要求当前平台 Node 的完整 RuntimeLock 与应用快照一致，避免同配置但制品锁变化时误报无需准备。缺少锁快照的早期代会重建。
- `go test ./internal/core -run 'TestRuntimeSnapshotIdentity|TestSyncRetainedNode' -count=1` Windows 通过（20.160 秒），复用官方归档。单元覆盖版本、hash、URL、npm、后端、证据、平台和摘要差异；真实集成覆盖写入后离线无变更、预览、快照修复和失败保护。
- 此快照为后续回滚保留应用时锁定信息，但尚未实现 rollback；不声称已完成深度文件完整性检查。其余首版缺口继续保留。

首页实际状态（2026-09-08）：
- 裸命令通过 core.Status 只读查询活动引用及小型输入，显示 not_ready、ready、drifted、incomplete 和下一步；移除 not_checked 占位信息。不执行 runtime、不联网、不申请租约或恢复操作。
- `go test ./internal/core ./internal/cli -run 'TestStatusReadOnly|TestRuntimeSnapshotIdentity|TestRoot' -count=1` 中 core 通过（0.274 秒），cli 无匹配测试，仅编译。随后 `go test ./internal/cli -run TestStatusJSON -count=1` 验证真实 CLI JSON 与无状态目录副作用通过。
- 核心测试覆盖四种状态、数据库字节不变和首次查询不创建 .myenv；ready 是轻量入口/快照检查，不等于运行验证或深度校验。状态性能预算未测，Python/profile 等后续功能仍未完成。

Unix 非交互进程组监督（2026-09-08）：
- runner 为非终端 stdin 的 Unix 命令设置独立进程组，context 取消向整组发送 SIGKILL，并在运行期间转发 INT/TERM/HUP。终端 stdin 保留前台组，避免 SIGTTIN；交互终端完整信号/作业控制仍需完善。
- 新增 TestCancelProcessGroup，验证 shell 后代持有输出管道时取消不能等到 sleep 自然退出，并接入 Unix 验证脚本。该测试在当前 Windows 宿主未运行。
- `go test ./internal/runner -run 'TestCancelRetainedNode' -count=1` Windows 保留真实 Node 通过（0.349 秒）；GOOS=linux GOARCH=amd64 CGO_ENABLED=0 的 `go test -c -o .build/runner-linux.test ./internal/runner` 编译成功，仅证明代码可编译。
- Windows Job Object、父进程异常退出、后代脱离进程组和租约全生命周期仍待实现/验证，不能据此宣布完整进程树监督完成。

Windows Job 进程树监督（2026-09-08）：
- 采用 CREATE_SUSPENDED 启动、AssignProcessToJobObject 纳管、恢复初始线程的顺序；纳管失败终止尚未执行的子进程。Job 使用 KILL_ON_JOB_CLOSE，context 取消终止整个 Job；直接子进程退出后用固定大小 accounting 查询等待 ActiveProcesses 为零才返回，保留调用方租约。依据 Microsoft Job Objects 文档：https://learn.microsoft.com/en-us/windows/win32/procthread/job-objects 。复用已有 x/sys，无新依赖。
- `go test ./internal/runner ./internal/cli -run 'TestCancelRetainedNode|TestRunRetainedNode' -count=1` Windows 真实 Node/npm/npx 通过（runner 0.366 秒，cli 1.732 秒）。Linux runner 测试二进制交叉编译成功，未运行。
- 初版后代等待测试失败：Node 非 detached 后代随父提前结束，未写标记；改为 detached 后代并把 stderr 写到普通文件（不靠输出管道拖延 Wait），验证父先退出、后代完成后 runner 才返回。`go test ./internal/runner -run TestWindowsJobRetainedNode -count=1 -v -timeout 20s` 最终两项通过（1.688 秒），另项验证取消能及时关闭后代持有的输出管道。已加入 Windows 验证脚本。
- 仍需真实 supervisor 异常退出测试、控制台信号转发与性能测量。Job 核算当前每 25ms 查询一次；完整进程身份/清理及 Unix 原生验证仍未完成。

Windows supervisor 强制退出验证（2026-09-08）：
- 新增独立测试 supervisor，启动真实 Node 父进程和 detached 后代；父测试取得两者进程句柄并确认仍运行后，强制结束 supervisor，再等待两句柄报告退出，验证 Job 句柄随 supervisor 退出关闭的实际行为。仅使用保留官方 Node 与临时目录，失败路径保留测试进程清理。
- `go test ./internal/runner -run '^TestWindowsJobSupervisorCrash$' -count=1 -v -timeout 20s` Windows 通过（0.390 秒）。加入 Windows 验证脚本；未重跑此前未变的同步安装测试。
- 该证据覆盖 runner supervisor 崩溃及 detached 后代终止，不覆盖 SQLite 遗留租约回收、sync 准备中崩溃、控制台信号转发或 Unix 父进程崩溃。完整目标继续进行。

直接子进程退出后的取消修复（2026-09-08）：
- 新增父进程先退出、detached 后代继承输出管道的测试。修改前 `go test ./internal/runner -run 'TestWindowsJobRetainedNode/cancel_after_parent_exit' -count=1 -v -timeout 15s` 失败，等待了 3.318 秒而非 750ms 取消。
- Windows Job 在整个监督周期独立监听 context，直到关闭监督器才停止；不依赖 CommandContext 在直接子进程退出后已结束的监听。取消后即使直接子进程原本退出 0，也不会将被取消的运行报告成功。
- `go test ./internal/runner ./internal/cli -run 'TestWindowsJobRetainedNode|TestWindowsJobSupervisorCrash|TestCancelRetainedNode|TestRunRetainedNode' -count=1 -v -timeout 20s` Windows 通过（runner 2.977 秒、cli 1.633 秒）；新增场景 0.75 秒，原后代等待、取消、supervisor 强杀和 Node/npm/npx 继续通过。
- Unix 对应取消时序仍待原生验证与完善；控制台信号、遗留租约和其他首版功能仍未完成。

Unix 后代管道取消接续（2026-09-08）：
- Unix 监督 goroutine 在直接子进程退出后仍监听 context，取消时向受管进程组发送 SIGKILL；处理一次后禁用取消 channel，避免空转。新增 shell 退出、sleep 后代保留管道的测试，要求取消在两秒内结束并返回非零。
- GOOS=linux GOARCH=amd64 与 GOOS=darwin GOARCH=arm64，CGO_ENABLED=0 分别执行 `go test -c -o .build/runner-linux.test ./internal/runner`、`go test -c -o .build/runner-darwin.test ./internal/runner`，两者编译成功。当前没有原生 Unix 执行证据；验证脚本的 TestCancelProcessGroup 匹配模式已包含新增测试。
- 不重复未受影响的 Windows 运行测试。Unix 无管道后代、终端作业控制、supervisor 崩溃与完整租约回收仍待完善，首版保持未完成。

发布前运行时锁复核（2026-09-08）：
- 同步准备结束时重新读取并比较完整运行时锁语义；锁丢失、无效或内容变化均返回 INPUT_CHANGED，不发布新代、不覆盖外部编辑。空白排版变化不视作语义变化。
- 扩展官方保留归档集成测试，在本地 HTTP 提供归档时模拟外部修改 myenv.lock，断言拒绝发布、旧活动代不变及外部修改保留。`go test ./internal/core -run TestSyncRetainedNode -count=1 -v -timeout 90s` Windows 通过（27.679 秒），复用已有归档，无官方重复下载。
- 该复核不能让任意外部编辑器与 SQLite 提交成为一个原子事务；复核之后的外部编辑仍由后续状态查询识别。Python 原生输入、控制台信号及其他首版缺口继续保留。

回退引用事务基础（2026-09-08）：
- state 新增 Previous 查询及 Rollback 条件更新，一条 SQLite UPDATE 交换 current/previous，双引用不匹配返回 INPUT_CHANGED；不修改代记录或运行租约。调用者必须持有工作区修改锁并验证回退目标。
- `go test ./internal/state -run TestRollbackReferences -count=1` Windows 通过（0.194 秒），覆盖无前代、过期引用拒绝、交换结果及现有运行租约保留原代。
- 仅状态层基础完成，core 目标完整性校验和 CLI 尚未接入，不能宣称 rollback 命令已可用。T03 Python 与 T04 用户操作闭环等整体要求保持未完成。

Node 回退命令接入（2026-09-08）：
- core.Rollback 在修改锁内检查前代配置快照、完成标记、普通 runtime 文件及平台锁快照，再调用引用交换；CLI 提供 rollback 和 JSON 结果，明确保留声明/锁并提示 run --current。
- `go test ./internal/core -run TestRollbackPreservesInputs -count=1` Windows 通过（0.295 秒）：两代本地状态夹具验证回退、声明/锁字节保留、普通选代拒绝漂移、current 选回退代及不完整目标拒绝后活动代不变。夹具未执行 runtime，不能计作真实 Node 回退运行验收。
- `go test ./internal/core ./internal/cli -run 'TestStatusJSON|TestStatusReadOnly' -count=1` 通过。仍需 CLI 回退调用与真实代集成验证、权限/路径完整性强化及 Python/profile 支持；不宣称 T04 完成。

CLI 回退后真实 Node 执行（2026-09-08）：
- 新增 TestRollbackRetainedNode：用两份配置/锁快照及状态记录共用保留官方 Node 入口，CLI rollback JSON 确认旧代被应用；声明/锁字节不变，普通 run 拒绝漂移，run --current 执行真实 Node，读取旧代环境变量并透传退出 17。
- `go test ./internal/cli -run TestRollbackRetainedNode -count=1 -v` Windows 通过（0.558 秒），未重新下载或安装。该测试验证实际 CLI 与 Node 执行，不代表两个独立安装版本之间的回退，也未验证目录搬迁或 Unix 运行。
- 已加入两平台脚本的测试选择模式。路径/权限强化、Python/profile 和其他首版要求仍未完成。

run 锁漂移检查（2026-09-08）：
- 普通 run 现在比较声明摘要、当前平台期望 runtime 锁和应用锁快照；锁缺失、无效或变化时返回 ENV_NOT_READY，不启动命令。run --current 保持使用已应用环境，不要求期望锁有效。
- `go test ./internal/cli ./internal/core -run 'TestRunRetainedNode|TestRollbackRetainedNode|TestRollbackPreservesInputs' -count=1` Windows 通过（cli 2.536 秒、core 0.549 秒）。真实 Node 测试新增仅改坏锁时普通 run 拒绝、current 仍执行保留 runtime；Node/npm/npx 与回退测试继续通过。
- 未重新下载安装；完整输入/权限检查、Python/profile 与其他首版要求继续进行。

doctor 快速诊断入口（2026-09-08）：
- doctor 复用只读状态查询，并显示 applied_node、parent_path_node、run_path_node 和 path_differs，输出明确 check_level=quick 与建议；不执行 runtime 或修改环境。当前不提供 --deep，帮助说明完整受管文件证据检查尚未实现。
- `go test ./internal/cli -run TestRunRetainedNode -count=1 -v` 初次新增断言因 Windows .EXE/.exe 字符串差异失败；改为 os.SameFile 文件身份断言后通过。测试实际验证 doctor JSON、ready 状态、实际入口及 run PATH 指向保留官方 Node，继续覆盖 Node/npm/npx 和锁漂移行为。
- Python/profile、无效配置的详细报告及 --deep 证据核对仍需补齐，不能宣称完整 doctor/T04 已完成。

Node use 编辑与同步（2026-09-08）：
- use node@<version> 在修改锁内编辑 YAML，再调用已有 Service.Sync；通过 ExpectedDigest 拒绝编辑与同步之间的声明漂移。失败说明 declaration_changed 及重试/current 路径；不另建安装流程。SetTool 保留注释与无关值，可能规范化排版，使用同步写入及平台替换。
- `go test ./internal/config ./internal/core -run 'TestSetToolPreservesComments|TestUseRetainsEditedDeclarationOnFailure' -count=1` Windows 通过（config 0.132 秒、core 0.226 秒）。覆盖注释/无关值、同版本字节不变，以及本地 metadata 503 后新声明保留、旧活动引用保留。加入 ExpectedDigest 后重跑失败路径通过。
- 首轮 `go test ./internal/config ./internal/core ./internal/cli -run '^TestStatusJSON$' -count=1` 仅 cli 有匹配测试通过，其他包仅编译。真实跨版本 use 安装及竞争时序仍待验证；Python/global/构建许可未完成，保持全首版目标。

use 部分修改的 JSON 报告（2026-09-08）：
- use 失败时 CLI 保留 UseResult 到 data，顶层 changed 反映已发生的声明修改，而 ok 仍为 false；不再把部分修改仅藏在错误文本。编辑后重读声明还会核对用户请求的版本，避免采纳刚被其他编辑替换的版本。
- `go test ./internal/cli ./internal/core -run 'TestUseFailureReportsDeclarationChange|TestUseRetainsEditedDeclarationOnFailure' -count=1` Windows 通过（cli 0.253 秒、core 0.239 秒）。CLI 用无效锁触发编辑后失败，无网络，核对 changed/declaration_changed/error 与实际新 YAML；原旧活动引用保护测试继续通过。
- 锁是否修改的精确分阶段报告仍需完善，完整 use/Python/profile 验收保持未完成。

同步锁修改结果（2026-09-08）：
- SyncResult 增加 lock_changed，在成功替换期望锁后立即记录，后续准备失败仍返回该信息；sync/use 失败 JSON 的顶层 changed 纳入锁修改，use 错误文本也显示精确 lock_changed。锁语义未变时不再重复替换文件。
- `go test ./internal/core ./internal/cli -run 'TestDownloadFailureRegistered|TestUseFailureReportsDeclarationChange|TestUseRetainsEditedDeclarationOnFailure' -count=1` Windows 通过（core 1.110 秒、cli 0.982 秒）。本地 HTTP 503 测试验证声明漂移先更新锁再失败时 lock_changed=true、文件摘要确已更新，以及再次失败时相同锁不报告修改。
- 发布成功与锁修改仍分别报告，未把锁更新视作环境已应用。真实 use 跨版本安装及其他首版缺口仍未完成。

use 真实归档成功路径（2026-09-08）：
- 保留官方归档集成新增 use node@22 无变更且不请求网络，以及 use node@精确版本改变声明、更新锁、准备新代并可被普通 run 选中；继续覆盖快照修复、下载失败和锁编辑冲突。
- `go test ./internal/core -run TestSyncRetainedNode -count=1 -v -timeout 120s` Windows 通过（39.646 秒），复用既有官方 ZIP、本地 HTTP，未从官方重复下载。测试将范围声明收窄为同一个真实 runtime 版本，不冒充跨 runtime 版本升级。
- `go test ./internal/core -run TestUseExpectedInputRejectedBeforePreparation -count=1` 通过（0.229 秒），核对 ExpectedDigest 不匹配时 INPUT_CHANGED、未改锁且未创建 .myenv。
- 真实不同 runtime 版本、Python/profile 与其他首版验收仍未完成。

Python 固定 uv 接入起点（2026-09-08）：
- 本机只读执行 `uv --version`，确认 0.11.26；读取 `uv python install --help`、`uv venv --help` 核实参数，并参照官方 https://docs.astral.sh/uv/reference/cli/ 与 https://docs.astral.sh/uv/concepts/projects/config/ 。后端常量固定 0.11.26，升级需代码版本变更。
- 新增 UV.VerifyVersion 和 CreateVenv：绝对解释器/目标路径、已有目标拒绝、--no-python-downloads/--offline/--no-project/--no-config 和显式 cache；子环境过滤 UV_*、PYTHONHOME/PYTHONPATH、外部 venv/conda 激活变量，保留代理等父环境设置。通过 runner 执行并限制输出缓冲。
- `go test ./internal/backend -run 'TestUVEnvironmentIsolation|TestUVRefusesExistingVenv' -count=1` Windows 通过（0.148 秒），仅环境与既有目录保护边界；本轮未实际创建 Python venv，也未安装依赖。
- 受管 uv 下载/可信制品校验、Python 安装解析、原生锁、groups 和构建授权均尚未接入。此入口未被 core 调用，不把系统 uv 或系统 Python 当作产品的受管后端替代。T03 仍在开发。

uv 受管 Python 安装适配（2026-09-08）：
- 只读 `uv python find --help` 核对参数。新增 InstallPython，要求已解析三段版本和绝对目录；安装显式 --install-dir/--no-bin/--no-registry，查找使用 --managed-python/--no-python-downloads/--no-project/--no-config/--offline，并校验解释器留在受管目录内，最后 -I 执行版本核对。此结果仅提供版本证据。
- `go test ./internal/backend -run 'TestPythonInstallRejectsUnresolvedRequests|TestUVEnvironmentIsolation|TestUVRefusesExistingVenv' -count=1` Windows 通过（0.153 秒），覆盖未解析/选项形请求在启动前拒绝及既有环境隔离边界。本轮未调用实际 Python 安装方法。
- 尚待受管 uv 制品供应、真实 Python 安装/venv 验证和 core/state 接入，依赖原生锁与构建策略也仍缺；不能将适配代码或系统 uv help 当作 Python 同步通过。

固定 uv 制品摘要（2026-09-08）：
- 查阅官方 https://github.com/astral-sh/uv/releases/tag/0.11.26 及其 Windows x64/Linux x64 glibc/macOS arm64 checksum 链接；将三目标 URL 与 SHA256 固定在 uv_release.go。运行时不获取 latest 或远端替换摘要。原流式下载校验提取为共用函数，新增 DownloadUV。
- `go test ./internal/backend -run 'TestDownload' -count=1 -v` 通过（0.149 秒），覆盖下载校验与清理；`go test ./internal/backend -run TestFixedUVReleases -count=1` 验证三目标版本/摘要格式和不支持平台拒绝通过。
- 本轮没有下载/解包 uv。发布页提供 GitHub Artifact Attestations，证明验证尚未接入，受管 uv 准备、Python 真实安装与原生锁/构建策略仍待完成。

uv 官方下载实测（2026-09-08）：
- 新增显式门控 TestUVOfficialDownload，已有 uv-download.json 时拒绝重复下载。Windows 首次沙箱内网络被拒；授权执行后 `go test ./internal/backend -run TestUVOfficialDownload -count=1 -v -timeout 6m` 通过（20.935 秒），固定 SHA256 校验成功，记录 .build/uv-real/uv-download.json，归档 artifact-download-1705022050。
- `gh attestation verify <保留归档> --repo astral-sh/uv --source-ref refs/tags/0.11.26 --format json` 沙箱内无法初始化 Sigstore verifier；授权执行后终态失败：期望 tag 引用，但证明的 SourceRepositoryRef 是 refs/heads/main。输出文件不是通过证据，归档未执行。
- 下一步核对官方发布提交与证明源码 digest 的关系，使用准确身份约束验证，不能简单取消来源限制并宣称通过。所有下载/验证进程均已退出，无活动句柄，后续复用归档。

uv 来源证明与受管入口通过（2026-09-08）：
- `gh api repos/astral-sh/uv/commits/0.11.26 --jq .sha` 得到发布提交 396ef7ce44e343095f3a5a5e8bc4b0b670ed0f5b；授权执行 `gh attestation verify <保留归档> --repo astral-sh/uv --source-digest 396ef7ce44e343095f3a5a5e8bc4b0b670ed0f5b --format json` 退出 0。检查保存的 verificationResult，证明源码/签名摘要同为该发布提交，工作流为 .github/workflows/release.yml@refs/heads/main，包含固定 Windows ZIP SHA256。来源约束从错误的 tag ref 改为精确发布提交，不是取消来源校验。
- 验证结果 .build/uv-real/attestation-verification.json。新增 PrepareUV 在执行前重新流式校验内置 SHA256、使用既有解包、核对固定版本。`go test ./internal/backend -run TestUVRetainedPrepare -count=1 -v -timeout 60s` Windows 通过（1.905 秒），实际执行受管 uv 0.11.26，入口保存在 .build/uv-real/uv-prepared.json；没有重复下载。
- Unix 解包入口尚未实测。制品来源证明本轮为发布验证命令，尚未封装完整自动分发流水线；下一步用此受管 uv 验证隔离 Python 安装与 venv，再接依赖/状态流程。

真实受管 Python 与 venv（2026-09-08）：
- 新增显式 TestPythonManagedPrepare，使用保留且已验证的 uv，调用 InstallPython 与 CreateVenv；安装/缓存仅位于 .build/python-real，--no-bin/--no-registry。授权执行 `go test ./internal/backend -run TestPythonManagedPrepare -count=1 -v -timeout 6m` Windows 通过（47.826 秒），Python 3.12.13 与 venv 版本均核对成功。
- 制品记录 .build/python-real/python-prepared.json。另通过保留 venv 解释器 -I 检查 sys.prefix/base_prefix/executable，确认分别指向 venv-20260908T124119.399501900、受管 runtimes/cpython-3.12.13-windows-x86_64-none 与该 venv/Scripts/python.exe，isolated=1。首次诊断命令引号写法导致 SyntaxError，改用 dict(...) 后退出 0；不影响前述 Go 集成测试。
- 没有安装项目依赖或执行构建；证据仅为 Windows 受管解释器与独立 venv，Python core/state 发布、runtime 范围解析、uv.lock/groups/构建授权仍未完成。后续复用保留运行时与 uv，不重复下载。

固定 uv 的 Python 范围解析（2026-09-08）：
- 核对受管 uv python list --help 与 JSON 输出；初次探查遗漏 --cache-dir，默认用户缓存访问被拒，随后显式使用 .build/python-real/cache 成功。生产 ResolvePython 强制绝对 cache，--only-downloads/--all-versions/--offline/--no-python-downloads/--no-config，过滤 CPython default 与宿主架构，从固定内置清单选声明范围内最大稳定版本。
- 返回 key/version/url/platform/backend 与 evidence=version，不把 URL 可见误报为强制制品锁。`go test ./internal/backend -run TestPythonRetainedResolve -count=1 -v` Windows 真实受管 uv 通过（0.778 秒），3.12 与 >=3.12.12,<3.13 均解析为 3.12.13，无 Python 下载。
- Unix 清单筛选未原生验证；Python core/state 接入、依赖锁/groups/构建策略仍待完成。

venv 身份校验（2026-09-08）：
- CreateVenv 成功后执行 VerifyVenv，用 -I 读取 prefix/base_prefix，以目录身份核对目标 venv 并拒绝基础解释器。避免版本相同但实际入口不属于新代时误判准备成功。
- `go test ./internal/backend -run TestRetainedVenvIdentity -count=1 -v` Windows 复用真实受管 Python/venv 通过（0.706 秒），覆盖正确 venv、基础 Python 冒充 venv 与不同目标目录拒绝；未下载或重新创建环境。
- 尚未重新运行完整创建流程，现有证明针对新身份校验函数及已保留环境。Unix 和 Python 发布/依赖流程仍待完成。

Python 原生输入摘要（2026-09-08）：
- ReadPythonInputs 在工作区路径边界内有界读取 pyproject.toml、uv.lock、可选 uv.toml；记录独立 SHA256 及聚合摘要，区分缺失/空文件。locked 模式缺 uv.lock 返回 LOCK_OUT_OF_DATE。此处只读证据，不解析或执行构建元数据。
- `go test ./internal/config -run TestPythonInputDigest -count=1` 初次因 Windows 祖先路径解析 Access denied 失败；授权相同测试后通过（0.194 秒）。覆盖三项变化、缺锁/locked 和越界项目拒绝，均在工作区测试目录。
- Python workspace 多成员输入闭包尚未覆盖，不能把当前三文件摘要当作完整多项目依赖证据；后续接入原生锁和发布时继续扩展。core/state Python 同步仍未完成。

uv 原生锁与同步入口（2026-09-08）：
- 核对 uv lock/sync help 后新增 SyncPythonProject：固定绝对解释器和 UV_PROJECT_ENVIRONMENT，先 uv lock（locked 时 --check），再 uv sync --locked；显式禁用默认 groups、传指定 groups、复制依赖文件并禁止额外 Python 下载。未允许构建时 --no-build 加第一方配置预检，存在构建/动态/workspace/本地来源结构先返回 NEEDS_INPUT。
- `go test ./internal/backend -run TestPythonProjectRetainedSync -count=1 -v -timeout 30s` Windows 真实受管 uv/Python 通过（2.665 秒）。隔离新 venv，无依赖非打包项目、空 dev 组，生成 uv.lock 后 locked 再同步字节不变；加入 build-system 后无授权拒绝。没有下载安装依赖或执行构建后端。
- 构建预检仍需覆盖父 workspace 发现、PEP 508 本地引用和配置闭包，并避免误判无关同名键；尚未接 core，不可作为完整安全许可门禁。真实 wheel/授权构建依赖测试、Python 发布与完整首版仍未完成。

构建预检配置作用域（2026-09-08）：
- 预检定位到 build-system、project.dynamic、tool.uv package/workspace/sources，以及项目/可选/组依赖引用；uv.toml 按 uv 设置解释。不再因为无关工具配置中的 path/dynamic/package 键触发授权。PEP 508 @ 引用暂保守要求构建许可，避免本地引用绕过前置检查。
- `go test ./internal/backend -run 'TestFirstPartyBuildLocations|TestPythonProjectRetainedSync' -count=1` Windows 通过（2.616 秒）：配置位置、注册表依赖/索引、不相关键、动态元数据、路径和直接引用等单元检查，以及保留真实 uv/Python 的原生同步回归。
- 父 workspace 和配置发现闭包仍未覆盖；纯 wheel URL 直接引用目前也会要求许可，后续须细化。许可门禁和 Python core 发布尚未完成。

父 Python workspace 预检（2026-09-08）：
- 无构建许可时，在启动 uv 前有界读取项目祖先目录的 pyproject.toml；遇到 tool.uv.workspace 返回 NEEDS_INPUT，避免只检查子项目而漏掉 uv 可能发现的父 workspace。只沿祖先路径读取，不扫描成员目录。
- `go test ./internal/backend -run 'TestParentWorkspaceBuildPreflight|TestPythonProjectRetainedSync' -count=1` Windows 通过（2.618 秒），覆盖父 workspace 不得绕过许可，独立项目真实原生锁/同步仍通过。
- 当前对祖先 workspace 保守拒绝，尚未计算 members/exclude 的精确归属及完整成员输入摘要；用户配置来源闭包、wheel URL 细化与 Python 发布仍需完成。此预检不是完整安全沙箱。

第一方构建许可实测（2026-09-08）：
- 新增本地 PEP 517 测试后端，导入时写测试目录标记，使用 Python 标准库生成最小 wheel；无外部构建依赖。先不允许构建，断言 NEEDS_INPUT 且标记不存在；随后 AllowBuild=true，实际锁定、构建、安装并在目标 venv 导入验证 VALUE=42。
- `go test ./internal/backend -run TestPythonBuildPermissionRetained -count=1 -v -timeout 30s` Windows 真实保留 uv/Python 离线通过（3.005 秒），未修改用户环境或重复下载安装解释器。
- 证据覆盖这一明确第一方后端的拒绝/授权执行，不覆盖所有依赖来源、配置闭包、TTY 单次确认或 core 发布；全首版仍未完成。

Python 代状态与租约入口（2026-09-08）：
- Generation 增加 PythonExecutable，以可选 generation_python 表保存并在发布同一事务提交；支持 Python-only 与 Node/Python 混合记录。Active/Previous 和选代租约均读取 Python 入口，租约仍在选代同一事务登记。只读打开旧 Node 数据库不创建新表，保持旧数据可读。
- `go test ./internal/state -count=1` 原状态回归通过（0.594 秒）；新增 `go test ./internal/state -run 'TestPythonGenerationPublication|TestReadOnlyLegacyNodeDatabase' -count=1` 通过（0.251 秒），覆盖 Python/混合记录、冲突后正常发布、重开持久化、租约完整入口与旧库兼容。
- 此轮仅状态层支持，core/CLI 仍按 Node 路径运行；Python 同步发布/运行入口尚待接入，不宣称 Python 命令已可用。

Python run 入口（2026-09-08）：
- SelectRun 比较当前平台完整 runtime 锁快照，并按应用声明验证 Node/Python 入口；Python 可使用 venv 链接。CLI 支持 python/python3，PATH 前置 Python 与 Node 的实际目录，Python-only 环境不再加入空 Node 的相对目录。
- `go test ./internal/cli ./internal/core -run 'TestRunRetainedNode|TestRollbackRetainedNode|TestRollbackPreservesInputs' -count=1` Windows 通过（cli 2.313 秒、core 0.313 秒）。新增 `go test ./internal/cli -run TestRunRetainedPython -count=1 -v` 首次因 Python 本地输出编码与 UTF-8 断言不同失败；测试显式传 -X utf8 后通过，真实保留 venv 透传中文/空格参数和退出 17。
- 测试状态记录为夹具，尚未由 Python sync 发布；原生依赖输入漂移、Python 状态/doctor/rollback 和同步发布仍需接通。

Python 状态、诊断与回滚入口（2026-09-08）：
- 状态按应用声明验证 Node/Python 入口，支持 Python-only 代，并比较当前平台完整工具锁；缺失应用锁标记 incomplete。Python venv 入口允许符号链接。rollback 按目标声明检查完整工具锁和入口，保留当前声明与锁。
- doctor JSON 与文本增加 Python 的应用、父 PATH 和运行 PATH 路径；只把非空入口目录加入 PATH，path_differs 对实际受管工具计算。
- `go test ./internal/core ./internal/cli -run 'TestStatusReadOnly|TestRollbackPreservesInputs|TestRunRetainedPython|TestRunRetainedNode|TestRollbackRetainedNode' -count=1`：cli Windows 真实保留 Node/Python 测试通过（2.802 秒）；core 因新增测试多余括号编译失败。修正后 `go test ./internal/core -run 'TestStatusReadOnly|TestRollbackPreservesInputs' -count=1` 通过（0.522 秒），覆盖 Node/Python 回滚保留输入、普通 run 拒绝漂移与 current 选代；回滚为状态夹具，不代表真实 Python sync 发布。
- Python sync、原生依赖输入漂移、完整构建许可闭包与跨平台原生验证仍待完成；首版目标尚未完成。

T03 Python core/CLI 同步发布（2026-09-08）：
- Sync 复用 Node 修改锁、operations、新代最终目录和 SQLite 单一活动引用；支持 Python 与 Node 工具选择，Python 委托固定 uv 安装解释器并在新代创建 venv。不移动受管解释器或 venv，不修改用户 profile。受管 uv 使用固定 SHA 校验/解包，验证后以临时文件 rename 发布入口；集成可注入保留制品和解释器目录，生产默认下载固定发行。
- PlatformLock 增加可选 PythonInputs，校验所需摘要；记录 pyproject.toml、uv.lock、uv.toml。普通 sync 更新必要原生锁，locked 要求已有摘要匹配并委托 uv 检查，最终再次复核配置/锁/原生输入。status 和普通 run 检测原生漂移；current/rollback 检查应用工具锁完整性。
- 无变更 sync 直接返回，不启动 uv。dry-run 不安装后端，缺少 Python 解析时明确显示待解析范围，不伪称已有精确版本。已锁 Python 后端与内置固定版本不符时要求显式 --update python，不静默升级后端。
- use 接入 Python/混合声明和同一同步选项。sync/use --allow-build 传递本次许可；已知第一方构建在后端准备之前预检，TTY 提供一次确认，JSON/no-input/CI 不询问。第三方需要构建时的精确分类及 TTY 重试仍待完善。
- SyncResult 增加 native_lock_changed，失败 JSON/文本区分期望锁变化与活动代未发布；Python 准备/同步失败映射 SYNC_FAILED。帮助与离线手册更新实际范围，仍注明平台和 workspace 限制。

本轮验证：
- 初次 `go test ./internal/core -run TestSyncRetainedPython -count=1 -v -timeout 60s` 因 Windows 祖先路径权限 Access denied 失败；相同隔离测试获准执行后通过（7.487 秒）。随后加入受管 uv 制品准备/cache 与 use Python，复用 .build 的官方制品/解释器通过（8.604 秒），没有重新下载解释器。
- `go test ./internal/core -run TestSyncRetainedNode -count=1 -v -timeout 90s` 真实 Windows 保留 Node ZIP 完整同步回归通过（36.315 秒）。通用同步修改未破坏首次、无变更、锁保护、use 与准备失败保留旧代。
- `go test ./... -count=1 -timeout 90s` Windows 全部常规测试通过，真实制品测试没有配置门控时跳过；不把这些跳过算作平台证据。
- `go test ./internal/core ./internal/cli -run 'TestSyncRetainedPython|TestRunRetainedPython|TestRunRetainedNode|TestRollbackRetainedNode' -count=1 -v -timeout 60s` 设置保留 Python、uv archive、Node record 后通过（core 10.305 秒、cli 2.696 秒）。Python 真实同步后运行验证版本；无变更用不存在的注入后端证明不启动 uv；原生漂移、locked 拒绝、use、构建拒绝、rollback、缺组导致 uv.lock 改变但活动代不变均覆盖。仅空 dev 组/无外部依赖，不代表非空依赖安装验收。
- 加入 SYNC_FAILED 与 JSON/no-input 构建拒绝后，`go test ./internal/core ./internal/cli -run 'TestSyncRetainedPython|TestPythonBuildNoninteractive|TestBuildPrompt' -count=1 -v -timeout 60s` 通过（core 8.036 秒、cli 0.293 秒）。输入 yes 不被非交互模式消费为授权；未创建锁、后端或代。
- `go build -o dist/myenv.exe ./cmd/myenv` 通过；CGO_ENABLED=0、GOOS=linux/GOARCH=amd64 与 GOOS=darwin/GOARCH=arm64 分别构建 dist/myenv-linux-amd64 和 dist/myenv-darwin-arm64 通过。后两者只交叉构建，未执行。

下一步：验证真实 wheel 与非空依赖组、混合 Node/Python 同步和本次授权构建的 core/CLI 路径；补齐 uv 配置发现/原生 workspace 输入闭包、第三方构建需要许可的分类与交互重试。随后继续用户 profile、清理/恢复、跨平台原生矩阵及性能门禁；不将这一纵向通过当作首版完成。

T03 依赖组、混合代与 core 构建许可验证（2026-09-08）：
- 新增 TestPythonWheelGroupsRetained：本地 HTTP 简单索引提供三份可安装测试 wheel（不是生产假后端），由真实保留 uv 解析、下载、校验与安装。基础依赖和显式 qa 组可导入，uv 默认 dev 组不可导入；取消 qa 后新代只保留基础依赖，旧代仍可导入 qa。uv.lock 包含各 wheel SHA256，组选项变化不改原生锁；关闭索引后 locked 无变更同步通过。
- `go test ./internal/core -run TestPythonWheelGroupsRetained -count=1 -v -timeout 60s` Windows 通过（6.599 秒）。仅本地测试 wheel/索引，无 PyPI 下载；不将此证据当作公共索引或任意第三方包兼容性证明。
- 新增 TestMixedRuntimeSyncRetained：保留 Node 官方制品通过本地服务获取，保留受管 Python 创建 venv；locked 同步在一个最终代发布两个入口，各自真实执行并核对版本，status ready。关闭 Node 服务且将注入 uv 设为不可用后，locked 无变更仍返回同一代。
- `go test ./internal/core -run TestMixedRuntimeSyncRetained -count=1 -v -timeout 60s` Windows 通过（11.434 秒）；无外部重下载、无用户环境修改。
- 新增 TestPythonCoreBuildPermissionRetained：核心服务回调拒绝后无构建标记；允许时只确认一次，真实执行本地 PEP 517 后端，生成/安装 wheel 后导入 VALUE=42；健康同步不再询问，后续输入改变的新调用仍需独立许可。
- `go test ./internal/core -run TestPythonCoreBuildPermissionRetained -count=1 -v -timeout 60s` Windows 通过（4.452 秒）。测试后端代码显式限定在隔离临时项目中，不依赖外部构建包。以上命令因已知 Windows 路径祖先 metadata 限制获准运行，未改沙箱配置。
- 本轮新增验收测试，未修改生产实现；复用上一轮常规回归和二进制构建证据，不重复无关测试。仍缺第三方 sdist 构建失败的 NEEDS_INPUT 分类/TTY 重试、uv 配置与 workspace 输入闭包、用户 profile、清理恢复及平台/性能门禁；首版继续 in_progress。

第三方源码分发构建许可（2026-09-08）：
- 使用本地简单索引与自定义源码 tar.gz（静态 PKG-INFO、无外部构建依赖）复现真实 uv 行为。初次普通 sync 的 lock 阶段因 no-build 返回解析失败，原实现只报 SYNC_FAILED；标记证明未运行测试构建后端。
- 固定 uv 的明确“building from source is disabled ... --no-build”提示现在映射 config.NeedsInput。core 在尚无本次构建许可时调用确认回调一次；同意后复核声明摘要、myenv.lock 和原生输入，再在尚未发布的新代重试。输入在确认期间改变则 INPUT_CHANGED，不执行构建。JSON/no-input 无回调，沿既有错误合同返回 NEEDS_INPUT。
- 进一步验证空 wheel 缓存的 --locked 首次安装，发现 sync 阶段使用另一条明确提示“marked as --no-build but has no binary distribution”；已增加对应分类。仅匹配固定版本的明确禁用构建诊断，不把普通解析冲突、网络失败或后端构建失败当作许可问题。
- 查阅真实受管 uv --help 确认 --color never 支持，lock/sync 固定无颜色输出，避免诊断匹配受终端颜色设置影响。
- `go test ./internal/core ./internal/backend -run 'TestPythonSourceBuildPermissionRetained|TestSourceBuildDisabledDiagnostic' -count=1 -v -timeout 60s`：加入普通源码分类/授权重试后通过（core 7.777 秒、backend 0.062 秒）。此前仅源码测试如预期因未分类失败，未降低断言。
- `go test ./internal/core ./internal/backend ./internal/cli -run 'TestPythonSourceBuildPermissionRetained|TestPythonCoreBuildPermissionRetained|TestPythonWheelGroupsRetained|TestSourceBuildDisabledDiagnostic|TestPythonBuildNoninteractive' -count=1 -v -timeout 90s` 通过（core 19.571 秒、backend 0.085 秒、cli 0.286 秒），覆盖输入确认时修改的拒绝与已有第一方/wheel/非交互路径。
- 新增 locked 空缓存场景初次失败，准确暴露不同诊断；修复后再次执行上述 core/backend 两测试命令通过（core 15.587 秒、backend 0.077 秒）。普通与 locked 拒绝时构建标记均不存在；普通许可后实际导入 VALUE=73；locked 同意时仅确认一次并成功发布，两份锁与原始字节一致。
- 所有真实调用使用保留 uv/Python、本地测试索引及隔离临时目录；Windows 祖先目录元数据权限按已有证据获准运行，无外部包下载或用户 profile 修改。`go build -o dist/myenv.exe ./cmd/myenv` 通过，更新 Windows 开发二进制；本轮未重建或运行 Unix 目标。
- 下一步仍是 uv 配置发现及 workspace/native 输入闭包，然后推进用户 profile、清理恢复与平台/性能门禁。当前证据针对固定 uv 两种明确错误与本地测试源码包，不宣称覆盖任意第三方后端；完整首版未完成。

Python 显式项目配置策略（2026-09-08）：
- 官方配置说明 https://docs.astral.sh/uv/concepts/configuration-files/ 确认默认发现并合并项目/用户/系统设置，--config-file 替代自动发现；保留 uv sync --help 确认该参数。固定 0.11.26 的官方 crates/uv-settings/src/lib.rs 通过 gh api 只读核对了 validate_uv_toml 的项目字段限制及配置 origin 行为。
- 新增 projectUVConfig：优先读取项目 uv.toml，否则从 pyproject.toml 的 tool.uv 提取通用设置；无设置时提供空配置。uv 的项目专属字段（workspace/sources/groups/package 等）保留在原生 pyproject.toml，不复制进独立配置。lock/sync 显式传 --config-file。
- 配置快照以 0600 临时文件放在 Python 项目目录，维持相对设置的原生 origin，调用结束删除；不改原始项目文件。普通结束的清理已有测试，强杀后临时文件回收尚须 T05 覆盖。
- PythonInputs 增加 config_policy=project-only-v1，参与聚合摘要及完整输入比较；旧空策略可读取，新策略可使旧准备记录被识别为不同输入。帮助/手册说明项目配置边界。
- 初次将整个 tool.uv 表转成独立配置时，真实 wheel 测试揭示 default-groups 不允许出现在 uv.toml；未修改其组断言，而是核对固定源码并保留项目专属字段在原生位置。修正后 HTTP wheel 组测试通过（7.19 秒）。
- 新增本地 uv.toml/相对 find-links 路径场景：关闭 pyproject 中 HTTP 索引，uv.toml 的 no-index 和 ./wheels 仍成功安装依赖，证明优先级及 origin 正确。原 HTTP hash 断言初次用于本地目录时失败；检查真实 uv.lock 显示本地目录仅记录 registry/path，故测试分开验证路径证据，HTTP SHA256 断言保留，不把本地路径当作制品哈希。
- `go test ./internal/core ./internal/backend ./internal/config -run 'TestPythonWheelLocalConfigRetained|TestPythonProjectRetainedSync|TestPythonCoreBuildPermissionRetained|TestPythonInputDigest|TestPythonLockInputs' -count=1 -v -timeout 60s` Windows 通过（core 13.674 秒、backend 4.287 秒、config 0.261 秒）。涵盖相对本地配置、默认组隔离、无 tool.uv 时忽略无效父 uv.toml、第一方构建和原生锁摘要。
- `go test ./internal/core ./internal/config -run 'TestPythonSourceBuildPermissionRetained|TestPythonInputDigest' -count=1 -v -timeout 60s` 通过（core 15.922 秒、config 0.156 秒），验证新策略下源码构建拒绝/授权、确认时输入变化、locked 两锁字节保持及策略摘要变化。
- 真实调用全部复用保留 uv/Python、本地 wheel/source fixtures；已知 Windows 祖先元数据限制按之前证据获准运行，没有修改用户或系统 uv 配置。`go build -o dist/myenv.exe ./cmd/myenv` 通过；Unix 未在本轮重建或运行。
- 尚未完成：原生 workspace 成员及父 workspace 的完整输入闭包、相对本地包内容变化的策略、强杀临时配置清理；之后仍有 profile、清理恢复及平台/性能门禁。首版目标保持 in_progress。

本地 Python 来源输入跟踪（2026-09-08）：
- PythonInputs 增加 related_sha256。读取 tool.uv.sources 中显式 path，递归记录本地目录的 pyproject.toml、setup.cfg、setup.py、uv.toml（含缺失状态）；直接本地制品采用固定缓冲 SHA256。不遍历业务代码目录，不执行元数据。路径必须留在当前工作区；图最多 64 路径、元数据累计 8 MiB、制品累计 512 MiB，循环依赖去重。
- 关联摘要参与原生锁、status/普通 run 的完整输入比较，以及同步准备前后/构建许可期间的 manifest 复核。已有锁没有关联摘要时不能与当前有本地来源的输入相等。原 Python 输入测试改用有效 TOML 做 manifest 变更，因为收集声明的本地来源现在需要静态解析 TOML。
- `go test ./internal/config -run 'TestPythonRelatedInputs|TestPythonLocalArchiveDigest|TestPythonInputDigest|TestPythonLockInputs' -count=1 -v` Windows 通过（0.706 秒）：递归/cycle 稳定、传递依赖 setup.cfg 增加、无关业务代码不扫描、越界拒绝、直接文件内容变化及原摘要合同。
- `go test ./internal/core -run 'TestPythonLocalSourceRetained|TestPythonCoreBuildPermissionRetained' -count=1 -v -timeout 60s` 复用真实 uv/Python 通过（9.015 秒）。新本地 path 依赖实际构建、安装并导入 VALUE=42，无变更不再询问；只变更依赖自身的 pyproject.toml，status drifted、普通 run 拒绝、locked 返回 LOCK_OUT_OF_DATE，后续普通同步仍须本次构建许可。
- 将制品上限改为图的累计上限后，`go test ./... -count=1 -timeout 90s` 所有常规包通过（真实制品门控未启用的测试仍跳过）。`go build -o dist/myenv.exe ./cmd/myenv` 通过。测试获准访问 Windows 祖先路径元数据，写入仍限隔离工作区；未重复下载解释器或改变用户环境。
- 当前只覆盖显式 tool.uv.sources path 的工作区内来源；workspace members/exclude/glob、父 workspace 的有效根、PEP 508 file 引用及外部本地路径策略仍需完成。动态构建后端和任意业务代码变化不在这一小输入摘要范围。大本地制品对状态/运行延迟的影响尚未测量，不能宣称达到性能预算。首版目标仍在进行中。

workspace 根的成员输入闭包（2026-09-08）：
- 新增 Python workspace members/exclude 展开，将匹配成员的 manifest/setup/uv.toml 纳入 related_sha256；复用本地来源图的去重和累计输入边界。成员新增、删除和内容变更均能影响摘要，排除成员不读取元数据。
- 新依赖 github.com/bmatcuk/doublestar/v4 固定 v4.10.0；先核对官方 Go API（https://pkg.go.dev/github.com/bmatcuk/doublestar/v4）与 `go list -m -json github.com/bmatcuk/doublestar/v4@latest` 发布记录（2026-01-25），再执行固定版本 go get。必要性是复用成熟的递归 glob 匹配，不自行发明 ** 解析。
- GlobWalk 仅使用受约束 fs.FS：每次路径访问经 Within，目录读取有界，整次相关输入收集共享 4096 条目/操作预算；开启 WithFailOnIOErrors，使超限和访问失败不能静默变成空成员集。成员/排除模式数组各最多 64 项，关联图仍受总 64 路径限制。不声明无限大型 monorepo 支持。
- `go test ./internal/config ./internal/core -run 'TestPythonWorkspaceInputs|TestPythonWorkspacePatternBoundaries|TestPythonWorkspaceSourceRetained' -count=1 -v -timeout 60s` Windows 通过（config 0.608 秒、core 5.012 秒）。覆盖 packages/*、递归 nested/**/pkg、exclude、成员 manifest 改变/新增、非法/越界模式和读取预算。
- 真实 workspace fixture 复用保留 uv/Python，以 workspace=true 引用成员；本次许可后构建、安装并导入 VALUE=42。健康同步不重复确认，只改成员 pyproject.toml 后 status drifted、普通 run 拒绝、locked 返回 LOCK_OUT_OF_DATE。
- `go mod tidy` 完成（同时补齐已有 SQLite 等依赖的测试模块缓存）；`go test ./... -count=1 -timeout 90s` 常规包全部通过；`go build -o dist/myenv.exe ./cmd/myenv` 通过。真实制品门控未开启的测试未计为运行证据。Windows 祖先元数据访问按既有隔离测试依据获准，没有修改用户环境。
- 此轮支持的是 Python 配置指向 workspace 根时的成员收集；从成员路径定位父 workspace 的共享锁根与有效配置仍待接通。PEP 508 file 引用、跨工作区来源策略、强杀临时文件回收、profile 及跨平台/性能门禁也仍未完成。首版目标保持 in_progress。

父 workspace 与成员目录入口（2026-09-08）：
- PythonWorkspaceRoot/PythonNativeWorkspaceRoot 沿已知项目的祖先有界读取 manifest，并按 members/exclude 判断归属；不为定位一个已知成员扫描目录。限制 64 层祖先、8 MiB 元数据；父 workspace 必须位于 myEnv 工作区内。被 exclude 的项目保持独立；非成员给出修正 members/exclude 的错误。
- PythonInputs 增加 workspace_root/workspace_sha256，分别记录有效父根和父 manifest。成员自己的 pyproject 摘要保留；uv.lock/uv.toml 从有效父根读取，related 摘要覆盖整个根的成员，因而兄弟成员及父声明变化也能识别。锁校验要求根/摘要同时存在且格式有效，准备/许可复核加入这些字段。
- PythonProjectRequest 增加独立 ConfigProject：cwd 仍为选择的成员，显式 uv 配置来自有效 workspace 根。预检复用成员归属判断，不再因被排除的独立项目恰有父 workspace 就要求构建许可。
- `go test ./internal/config ./internal/core ./internal/backend -run 'TestPythonWorkspaceSharedInputs|TestPythonWorkspaceMemberRetained|TestParentWorkspaceBuildPreflight' -count=1 -v -timeout 60s` Windows 通过（config 0.802 秒、core 9.542 秒、backend 0.106 秒）。真实成员配置指向 dependency，实际构建/安装并导入，uv.lock 仅在父根生成；故意无效的成员 uv.toml 未覆盖父配置。输入测试覆盖兄弟与父 manifest 改变。
- `go test ./internal/config ./internal/backend -run 'TestPythonWorkspaceSharedInputs|TestPythonProjectRetainedSync' -count=1 -v -timeout 40s` 通过（config 0.648 秒、backend 3.661 秒），证明被排除项目的真实锁/同步独立，父根超出 myEnv 边界时拒绝采用。
- `go test ./... -count=1 -timeout 90s` 常规包全部通过（真实门控未启用时仍跳过对应测试）。Windows 构建通过；CGO_ENABLED=0 的 linux/amd64 与 darwin/arm64 交叉构建通过，未原生运行。手册措辞更新后再次 Windows 构建通过；Unix 二进制仍是同轮代码、手册更新前版本。
- 真实测试复用保留 uv/Python，构建后端为限定在隔离项目内的显式测试代码，无外部依赖下载或用户配置修改；Windows 路径 metadata 按已有依据获准访问。
- 尚需处理 PEP 508 file 引用、外部本地路径策略、更多 glob/符号链接兼容边界与性能成本；profile、强杀/清理恢复及原生平台/性能发布门禁仍未完成。首版目标继续 in_progress。

PEP 508 本地 file 引用输入（2026-09-08）：
- 关联来源收集扩展到 project.dependencies、optional-dependencies、dependency-groups，以及 uv 的 dev/override/constraint dependencies 中的 file: 直接引用。解析 URL 编码路径和 Windows 驱动器，忽略 URL 后的 marker 条件但保留其对应来源，以覆盖原生锁的多环境输入。
- 引用复用原有有界本地来源图、metadata/制品摘要与工作区检查；远程 HTTP 引用不当作本地路径。远程 file authority、查询字符串与无效路径明确拒绝，不转换成宿主网络路径。
- `go test ./internal/config -run 'TestPythonFileReferenceInputs|TestPythonFileReferenceBoundary' -count=1 -v` Windows 通过（0.611 秒），覆盖编码空格、普通/可选/组依赖、依赖 manifest 变化、越界与远程引用区分。
- `go test ./internal/core -run TestPythonFileSourceRetained -count=1 -v -timeout 60s` 复用真实 uv/Python 通过（8.692 秒）。显式 file URL 指向带空格的本地依赖目录；本次许可后构建、安装并导入 VALUE=42；无变更返回，随后只改依赖 manifest 即 status drifted、普通 run 拒绝、locked 返回 LOCK_OUT_OF_DATE。
- `go test ./internal/config -count=1` 配置回归通过（3.317 秒），`go build -o dist/myenv.exe ./cmd/myenv` 通过。已通过且未受修改影响的 Node/runner 等测试复用此前记录。Windows 路径 metadata 依据既有隔离测试获准访问，未下载解释器或修改用户配置。
- 第一版用户 profile、清理/强杀恢复、更多 URL/glob/符号链接边界、原生平台与性能门禁仍未完成；保持完整目标 in_progress。

用户 profile 配置基础（2026-09-08）：
- 新增 ProfilePath/LoadProfile/ParseProfile。默认路径采用 os.UserConfigDir 下的 myenv/profile.yaml；路径解析只读，不创建文件/状态。显式配置目录参数供隔离集成使用，必须为绝对路径，不混用项目 cwd。
- profile 复用严格 YAML 与工具约束校验，并只接受 schema/tools。即使 env/python 是 null 或空映射也拒绝这些项目字段；重复键和未知字段继续拒绝。读取明确的 profile.yaml，不回退或合并相邻 myenv.yaml。
- `go test ./internal/config -run TestProfileIsolationAndValidation -count=1 -v` Windows 沙箱内通过（0.165 秒），验证路径无创建副作用、声明隔离、项目字段/重复键拒绝、相对用户目录拒绝；只使用隔离临时目录，没有触及真实用户 profile。
- 此轮为配置层基础；--global 尚未暴露，用户默认环境的同步/运行/编辑及状态存储仍需接入现有 core 事务。没有将这一基础验证宣称为全局功能完成。首版保持 in_progress。

用户 profile 核心事务与 CLI（2026-09-08）：
- Service 增加显式 profile 上下文和隔离用户配置目录注入，use/sync/run/doctor/rollback 复用已有代准备、活动引用、租约和回滚实现。profile.yaml 仅含 schema/tools；profile.lock 与 .myenv-profile 独立于相邻项目的 myenv.lock/.myenv，不继承项目声明。
- use --global 可首次创建声明；已有声明编辑保持严格 profile 校验。CLI 仅指定五个命令接受 --global；run 的子命令后同名参数原样透传，执行 cwd 保持调用目录。status/doctor JSON 增加 scope，profile 下一步提示保留 --global，手册说明路径与当前能力。
- 核心真实测试先通过首次创建、健康无操作、声明切换和回滚（4.050 秒）。先前 CLI 调用输出在上下文边界丢失，检查确认无运行中的 go/cli.test 进程，未将未知结果记为通过。
- 提示与手册修改后执行 `go test ./internal/cli ./internal/core -run 'TestGlobalProfileCLIRetained|TestProfilePythonRetained' -count=1 -v -timeout 60s`，Windows 通过（cli 5.014 秒，core 4.214 秒）。复用已有真实 uv/Python；覆盖 exit 17、子参数 --global、locked/no-op、doctor、回滚后普通 run 拒绝与 --current 成功、相邻或无效项目声明不被采用、项目状态不被创建。
- `go test ./... -count=1 -timeout 90s` 常规回归全部通过；未开启真实制品门控的测试仍跳过。`go build -o dist/myenv.exe ./cmd/myenv` 通过，Unix 本轮未重建或原生运行。测试使用注入的临时用户目录，按已知 Windows 路径元数据限制获准运行，没有修改真实用户 profile 或下载解释器。
- 未完成项包括 profile 准备失败的专门集成证据、最终用户数据/cache 布局、显式 Shell PATH 接入、强杀恢复/clean/租约身份及原生平台与性能门禁。完整 T00–T06 目标继续进行，当前不是发布候选。

profile 失败保护与持锁进程强杀证据（2026-09-08）：
- 扩展 TestProfilePythonRetained：已有真实 Python 活动代下编辑为不同精确版本，注入不存在的 uv 使同步失败；验证请求声明保留、Changed=false、status 为 profile/drifted、--current 仍选择原代并实际执行 Python 输出 42。
- use 失败提示改为 active environment unchanged，适用于首次失败尚无旧代的情况；重试命令经作用域转换，profile 显示 myenv sync --global 和 myenv run --global --current。
- `go test ./internal/core ./internal/cli -run 'TestProfilePythonRetained|TestUse' -count=1 -v -timeout 60s` Windows 通过（core 3.968 秒，cli 0.248 秒），使用保留 uv/Python 与隔离用户目录，按既有 Windows 祖先元数据限制获准执行。
- 新增 TestRecoveryKilledOwner：独立测试进程持真实 OS modification lock，提交 preparing 操作后通过 stdout 就绪握手；父进程确认锁竞争超时，再强杀该已知子进程并 Wait 确认退出。随后重新取锁、打开数据库并恢复，验证仅一条操作转 failed、活动引用保持、重复恢复为零。
- `go test ./internal/state -run TestRecoveryKilledOwner -count=1 -v -timeout 25s` Windows 沙箱内通过（0.428 秒）。这是存储/OS 锁边界的真实强杀证据，不代表完整 Sync 安装途中强杀、子树租约或 clean 已获验证。
- `go build -o dist/myenv.exe ./cmd/myenv` 通过。未重复运行上一轮全量常规回归；修改影响范围内用例已通过。Unix 本轮未运行或重建。
- 下一步仍包括清理候选的事务保护与 deleting 状态、运行进程身份及存活判定、完整同步强杀恢复；用户数据布局、Shell 接入、deep doctor 和原生平台/性能发布门禁也尚未完成。完整目标保持进行中。

清理事务保护基础（2026-09-08）：
- 新增 state.CleanCandidates/MarkDeleting/FinishDeleting：候选按 ID 分页（每页最多 256），排除当前代、前一代、任何租约和按 ID/目录关联的 preparing 操作。MarkDeleting 在写语句中重新检查条件与候选目录，不能用过期 dry-run 结果直接删除。
- deleting_generations 持久保存删除预约，文件删除失败或进程中断后可再次预约同一项继续处理。FinishDeleting 只接受仍满足保护条件的预约项，在事务中清除 Python 入口、标记与代记录；不改变活动引用。
- SQLite 触发器阻止将 deleting 代重新放入活动/前代引用、插入租约或在其目录开始 preparing 操作。既有数据库通过 CREATE IF NOT EXISTS 增量初始化；只读打开路径不迁移。
- TestCleanReservationProtectsReferences 覆盖四代、活跃租约、目录关联 preparing 操作、过期目录拒绝、第二数据库连接观察持久标记与拒绝重新引用、受保护项拒绝 finalize、Python 入口事务删除以及失败操作释放候选资格。测试仅操作临时数据库，没有执行文件系统递归删除。
- 首次 `go test ./internal/state -count=1 -timeout 30s` 通过（1.344 秒）。补充 preparing 入口触发器后，`go test ./internal/state ./internal/core ./internal/cli -run 'TestClean|TestPublish|TestRollback|TestLease|TestRecovery|TestProfilePythonRetained|TestGlobalProfileCLIRetained' -count=1 -v -timeout 60s` Windows 通过（state 1.040 秒、core 5.662 秒、cli 5.765 秒）。真实 profile 测试复用保留 Python/uv；Node 门控回滚测试未启用，明确跳过。
- 测试所需 Windows 祖先元数据访问按现有隔离验证依据获准；未修改真实用户环境。Windows 开发构建通过，Unix 本轮未重建/运行。
- 此处是清理的数据库合同，尚未暴露 clean CLI，也尚未执行受验证路径的文件删除/占用统计。崩溃遗留租约目前仍保守保护，进程身份与释放判定待完成；失败准备目录与缓存清理也须接入。完整目标继续进行。

受管旧代的文件清理与核心入口（2026-09-08）：
- 新增 Service.Clean，以 128 项分页读取候选，通过回调逐项报告，避免历史代数量导致无界结果缓存。dry-run 使用只读数据库且不初始化状态；执行模式持有 modification lock，重新预约 deleting 后才删除文件并完成数据库事务。
- 数据库路径仅允许等于 generations/<id> 的直接子目录；拒绝空、点、父路径与路径分隔符。使用 os.OpenRoot(project) 和受限 OpenRoot(generations) 获取目录句柄，Root.RemoveAll 执行删除。采用本机 Go 标准库文档核对 Root API；首次 go doc 未设置项目缓存，触发的模块查询被网络沙箱拒绝，未增加依赖，随后恢复项目 GOPATH/GOCACHE。
- 大小统计按目录每批 128 项读取，最多 100 万条目/128 层，检查上下文取消，不跟随符号链接；报告普通文件逻辑字节，不声称是磁盘实际分配块。删除中断保留数据库预约，下次显式清理可继续；旧代目录已缺失时可完成元数据删除。
- `go test ./internal/core -run TestCleanFilesBoundary -count=1 -v` Windows 沙箱内通过（0.188 秒），覆盖非法路径/取消拒绝、实际删除、邻近目录与外部文件保留、重复删除。符号链接创建因当前账户缺少权限失败并明确记录，该分支未获运行证据。核心入口接入后编译并运行该测试通过（0.178 秒），没有新增符号链接证据。
- `go test ./internal/core -run TestCleanRetainsAppliedAndLeased -count=1 -v -timeout 30s` Windows 沙箱内通过（0.339 秒）：四代真实临时文件，dry-run 不删除；执行仅删无引用代，活动/前代/有租约旧代保留；释放租约后可删；再次清理无变化。该测试为文件/数据库合同，fixture 不是运行时安装证据。
- clean CLI 尚未接通；失败准备目录、共享缓存与进程死亡后的租约回收仍待实现。符号链接竞态、挂载点及目录身份/权限仍需进一步目标平台验证，不能将这一核心实现视为完整清理验收。未重建发布产物，完整目标保持进行中。

clean CLI 与部分失败输出（2026-09-08）：
- 接入项目 clean/--dry-run/--json，命令帮助与离线手册说明当前清理范围。按方案不接受 --global；当前只清理无引用已发布旧代，缓存、失败准备和崩溃遗留租约仍待接入。
- JSON 逐项流式写入一个文档，末尾包含 summary、ok、changed、error；不收集无界候选列表，也不创建报告临时文件。核心失败仍闭合 JSON 并返回 CLEAN_FAILED/exit 1，给出重试 clean --dry-run 提示。输出设备失败不承诺完整文档。
- CleanResult 增加 Changed：成功获得删除预约后即为 true，因为数据库已发生持久修改，且 RemoveAll 失败可能已部分删除文件。Removed 仍仅计完整 RemoveAll 成功的代，避免将部分删除误报为完整删除。
- `go test ./internal/cli ./internal/core -run 'TestCleanCLIResult|TestCleanRetainsAppliedAndLeased' -count=1 -v -timeout 30s` Windows 沙箱内通过（cli 0.935 秒、core 0.599 秒）。CLI 测试覆盖预览/删除成功、先删除一项再遇到不合法数据库目录的部分失败；验证单个 JSON 文档、exit/changed/removed、活动/前代保留和外部文件未被删除。
- `go test ./internal/cli -count=1 -timeout 45s` 中除 TestPythonBuildNoninteractive/TestPythonProjectDetection 的已知 Windows 祖先 metadata Access denied 外，常规测试通过；真实制品门控未启用的测试跳过。未将这一受限批次记录为全通过。
- 获准后只重跑两个受限用例：`go test ./internal/cli -run 'TestPythonBuildNoninteractive|TestPythonProjectDetection' -count=1 -v -timeout 30s` 通过（0.462 秒）。`go build -o dist/myenv.exe ./cmd/myenv` 通过；未下载运行时或修改用户环境。Unix 本轮未重建/运行。
- 下一步继续清理失败准备/缓存与崩溃遗留租约；文件身份/挂载点/符号链接场景、完整同步强杀、deep doctor、数据布局和平台/性能门禁仍未验收，完整目标保持进行中。

清理 ID 规范与删除后恢复（2026-09-08）：
- 审查发现此前 generationName 仅限制为单层目录名，未排除 Windows 目录别名拼写。现按 Sync 的真实生成规则严格接受 16 随机字节对应的 32 位小写十六进制 ID；路径仍须精确等于 generations/<id>。removeGeneration 同样独立执行该校验。
- 拒绝尾随点/空格、大写、设备名称、8.3 形态以及错误长度，避免数据库中的非规范名字成为删除权限。清理测试 fixtures 改为生产格式 ID，没有放宽原有保护预期。
- `go test ./internal/core ./internal/cli -run 'TestClean' -count=1 -v -timeout 30s` Windows 沙箱内通过（core 0.720 秒、cli 0.993 秒）。覆盖规范 ID 下的实际删除、非规范拒绝、租约/活动/前代保护及 CLI 部分失败输出；符号链接 fixture 仍因权限不足未执行对应分支。
- 新增 TestCleanResumeAfterFileRemoval：预约 deleting，实际删除隔离旧代目录，关闭数据库而不 finalize，再经 Service.Clean 只读预览（候选一项、字节零、无修改）及执行完成元数据收尾；再次预览无候选。模拟的是文件已删而数据库未完成的中断持久状态，不声称是真实进程强杀。
- `go test ./internal/core -run TestCleanResumeAfterFileRemoval -count=1 -v -timeout 30s` 通过（0.312 秒），Windows 开发构建通过。未下载工具、修改真实用户环境或重跑无关完整安装；Unix 本轮未重建/运行。
- 缓存、失败准备目录与遗留租约回收仍待完成；完整同步/清理强杀、文件对象身份及原生平台/性能门禁尚待验收，完整目标继续进行。

失败准备目录清理（2026-09-08）：
- state 新增有界 FailedCandidates 与持久 deleting_operations 预约/完成事务，仅选择 failed 且按 ID/目录均无已发布代的操作。preparing 不参与清理；预约期间数据库触发器拒绝发布复用目录的代或修改操作记录。
- Service.Clean 在旧代之后流式处理失败准备；ID 仍要求生产规范格式，目录仍须精确属于 generations。清理同时覆盖 generations/<id> 与 operations/<id>，后者路径由受验证 ID 推导。两个位置都删除成功后才移除操作元数据，缺失目录视为已清理；报告 kind=failed_preparation，大小合计两处普通文件逻辑字节。
- 新增 TestCleanFailedPreparation：隔离目录内保留真实临时文件，验证六字节预览、不修改、持久预约拒绝复用目录发布、执行移除两处失败文件与操作记录、preparing 文件仍存在、再次无候选。未将此 fixture 计作真实后端安装或强杀测试。
- `go test ./internal/core ./internal/cli ./internal/state -run 'TestClean|TestPublish|TestRecover' -count=1 -timeout 30s` Windows 沙箱内通过（core 1.318 秒、cli 1.212 秒、state 0.971 秒）。核心接入时的 TestClean 子集也通过（core 0.673 秒、state 0.345 秒），最终批次覆盖新增失败准备场景。
- 命令帮助与手册同步说明已支持失败准备清理。Windows 开发构建通过；未下载后端、触及真实用户环境或在 Unix 原生运行。
- clean 不把 preparing 自动判定死亡；现有 sync 恢复可将中断操作转 failed，完整清理强杀验证仍待完成。共享缓存、崩溃遗留租约及文件对象身份/平台性能门禁仍在范围内，完整目标保持进行中。

已发布旧代的操作目录收尾（2026-09-08）：
- 检查发现旧代清理此前仅删除 generations/<id>，成功同步的 operations/<id> 目录与 complete 操作行会残留。现在候选逻辑字节合计生成目录与操作目录；删除预约后依次删除两处，均成功后才 finalize。
- FinishDeleting 在现有事务中一并删除 ID/生成目录精确匹配且状态 complete 的操作行，不扩大到任意同名或不同目录的记录。下载目录仍由已验证规范 ID 在受限目录句柄下推导，不接受数据库提供的任意删除路径。
- 扩展 TestCleanResumeAfterFileRemoval：三代都经 BeginOperation/Publish 形成真实完成记录，分别留下七字节下载文件；预约旧代并只删除生成目录后关闭数据库。重新 clean 预览仍报告七字节；执行清理下载目录与完成记录，活动/前代下载文件保留；再次预览无候选。通过重新插入已清理操作 ID 验证完成行已移除，新记录保持 preparing。
- `go test ./internal/core ./internal/state ./internal/cli -run 'TestClean' -count=1 -timeout 30s` Windows 沙箱内通过（core 1.008 秒、state 0.273 秒、cli 0.841 秒）。Windows 开发构建通过。未重复安装后端或修改用户环境，Unix 本轮未重建/运行。
- 该变更处理已跟踪旧代的操作残留，不是共享 uv 缓存回收；历史已删除代留下的孤立 complete 记录仍需迁移/清理策略。崩溃租约、完整强杀场景、用户数据布局与原生平台/性能验收继续待办，完整目标保持进行中。

历史孤立完成操作兼容清理（2026-09-08）：
- 准备残留候选扩展为 failed 或 complete，但仍要求按操作 ID 和生成目录均不存在关联已发布代。preparing 保持排除；完成记录不会仅因时间旧或不在 active 引用而被当作孤立对象。
- 状态 API 改名 PreparationCandidates/MarkPreparationDeleting/FinishPreparationDeleting，返回原始操作状态；核心将历史孤立 complete 记录报告为 kind=orphaned_operation，失败记录保持 failed_preparation。两类复用相同预约、发布排斥、规范路径与两目录收尾事务。
- 新增 TestCleanHistoricalCompletedOperation，直接构造早期清理可能留下的 complete 行与下载文件（生成目录已缺失）。预览和执行均仅选择真正孤立项；另一个记录即使 ID 与已发布代不同，只要目录仍被引用就保留。验证剩余操作行数量和受保护下载文件。
- `go test ./internal/core ./internal/state ./internal/cli -run 'TestClean' -count=1 -timeout 30s` Windows 沙箱内通过（core 1.247 秒、state 0.294 秒、cli 0.987 秒）。未下载后端或触及真实用户环境。
- CGO_ENABLED=0 下 Windows 开发构建、GOOS=linux GOARCH=amd64 与 GOOS=darwin GOARCH=arm64 交叉构建全部通过。Unix 仅编译证据，未原生执行清理、符号链接或进程行为。
- 共享缓存、遗留租约与进程身份、完整同步/清理强杀、用户目录布局、deep doctor 及原生平台/性能验收仍未完成。完整目标保持进行中。

租约监督进程启动身份（2026-09-08）：
- 新增 lease_identity 附属表，与活动代选择和 leases 登记在同一事务写入；外键 ON DELETE CASCADE 随租约释放删除。既有无身份租约仍保留，不猜测或自动回收。
- Windows 使用 GetProcessTimes 创建 FILETIME；Linux 使用有界读取的 /proc/self/stat 启动 tick 与 boot_id；macOS 使用 SysctlKinfoProc 的 P_starttime。平台代码依据已固定 x/sys 本地声明与类型实现，无新增依赖。身份读取失败则拒绝登记，避免产生无法识别的新租约。
- ReleaseLease 同时匹配随机租约 ID、PID 和启动身份，不会仅因 PID 相同而释放不同启动身份的租约。尚未实现其他 PID 身份查询或监督子树死亡判定，不能宣称遗留租约可自动清理。
- `go test ./internal/state -count=1 -timeout 30s` Windows 通过（1.647 秒）。加入释放身份检查后，`go test ./internal/state ./internal/core -run 'TestLease|TestClean|TestProfilePythonRetained' -count=1 -timeout 40s` 通过（state 0.511 秒、core 5.185 秒）；真实 profile 复用已有 Python/uv，隔离用户目录，Windows 祖先 metadata 按已有依据获准访问。
- 补充测试模拟同 PID/不同启动身份，确认 ReleaseLease 保留该行，恢复正确身份后租约及附属身份行一起删除；`go test ./internal/state -run TestLeaseRetainsSelectedGeneration -count=1` 通过（0.257 秒）。
- Windows 构建及 Linux amd64/macOS arm64 交叉构建全部通过（CGO_ENABLED=0）。Unix 启动身份读取尚无原生运行证据，不能计为平台验收完成。
- 遗留租约回收、共享缓存、进程子树与强杀完整场景、性能成本及最终发布门禁仍待完成，完整目标保持进行中。

按 PID 读取启动身份与退出证据（2026-09-08）：
- 身份读取扩展为 processIdentity(pid)，自进程登记通过相同函数读取。Windows 以 query-limited + synchronize 打开目标句柄，零等待检查退出，再从同一句柄读取创建时间，避免查询过程 PID 重用造成拼接身份。不存在 PID/已退出句柄才返回 ErrProcessGone，权限错误原样返回。
- Linux 按已验证正 PID 有界读取 /proc/<pid>/stat，缺失路径返回 ErrProcessGone；继续结合 boot_id 和启动 tick。Darwin 使用指定 PID 的 sysctl；当前没有把 EIO 等模糊失败视为死亡，进程消失识别仍待原生确认。非法 PID 不视为已观察的死亡进程。
- TestProcessIdentityLifecycle 启动专用测试子进程，由子进程报告自身份；父进程查询两次须相等，随后只强杀该已知子进程并 Wait 确认结束，再查询退出证据。Windows 沙箱内通过。
- `go test ./internal/state -run 'TestProcessIdentityLifecycle|TestLeaseRetainsSelectedGeneration' -count=1 -v -timeout 20s` 通过（0.382 秒）；最初身份查询接入时租约测试也通过（0.260 秒）。Windows 构建、Linux amd64/macOS arm64 交叉构建均通过，CGO_ENABLED=0；Unix 尚未原生执行。
- 当前没有据此自动删除租约：监督进程死亡不单独证明其整个子树已经结束。后续须接通进程树证据与保守未知状态，才可回收遗留租约。共享缓存、完整强杀/平台/性能验收等完整目标事项仍待完成。

Windows 租约与命名进程树关联（2026-09-08）：
- SelectRun 将随机租约 ID 传至 runner.Process.TreeID；Windows 将其作为 Local\myEnv-lease-<id> Job Object 名称。未携带 TreeID 的内部运行仍使用匿名 Job；Unix 保持原有进程组行为，未声称已有可查询持久树身份。
- 仅接受 32 位小写十六进制名称后缀。创建命名 Job 时保留 CreateJobObjectW 的 last-error，遇到 ERROR_ALREADY_EXISTS 即关闭新获得的句柄并拒绝运行，不复用或重新配置已有对象。现有 x/sys 包装会在非零句柄时丢弃该 last-error，因此此处使用固定系统 DLL 的 API 调用。
- 先核实 Microsoft CreateJobObjectW 文档：https://learn.microsoft.com/en-us/windows/win32/api/jobapi2/nf-jobapi2-createjobobjectw ，确认名称碰撞返回既有对象、Local 会话命名空间与销毁条件。未添加依赖。
- `go test ./internal/runner ./internal/cli -run 'TestNamedJobOwnership|TestGlobalProfileCLIRetained' -count=1 -v -timeout 40s` Windows 通过（runner 0.458 秒，cli 7.054 秒）。覆盖命名碰撞拒绝、非法 ID 拒绝及真实 Python profile CLI 运行，隔离目录，复用已有制品；路径 metadata 依据既有记录获准。
- 真实 Node 子孙等待用例改用命名 Job；`go test ./internal/runner -run 'TestWindowsJobRetainedNode/wait_for_descendant' -count=1 -v -timeout 30s` 沙箱内通过（0.993 秒），直接子进程提前退出后仍等待独立子孙写完标记。其他已通过且未改动的监督场景未重复运行。
- Windows 构建与 Linux amd64/macOS arm64 交叉构建通过，CGO_ENABLED=0。Unix 未原生运行。
- 尚未根据 Job 名称自动回收遗留租约；必须补齐会话身份、只在监督进程确认退出后查询树状态，以及跨权限/跨会话/崩溃窗口的保守处理。完整目标继续进行。

Windows 会话身份与 Job 存在性查询（2026-09-08）：
- Windows 启动身份升级为 windows-v2:<session>:<creation FILETIME>，通过 ProcessIdToSessionId 记录 Local Job 所在会话；保留精确进程创建时间。旧格式记录没有会话证据，不应由后续回收逻辑猜测归属。
- runner.JobTreeGone 在查询前要求当前会话等于记录会话，严格校验随机 TreeID，仅用 OpenJobObjectW 的 JOB_OBJECT_QUERY 权限查询。现有对象一律保护（包括当前零进程的对象），只有 ERROR_FILE_NOT_FOUND 返回已缺失；权限/跨会话等错误返回未知，不视为死亡。
- 该查询的前提明确为匹配的监督进程已确认死亡。尚未接入自动租约删除，因此没有将查询能力宣称为回收完成。
- `go test ./internal/state ./internal/runner -run 'TestProcessIdentityLifecycle|TestLeaseRetainsSelectedGeneration|TestNamedJobOwnership' -count=1 -v -timeout 20s` Windows 沙箱内通过（state 0.334 秒，runner 0.074 秒）。覆盖新身份的自报/外查一致、退出证据、租约释放以及 Job 存在/缺失/跨会话拒绝与名称碰撞。
- Windows 开发构建通过。本轮仅改 Windows 平台行为与对应测试，Unix 构建证据复用上轮，未声称原生运行。没有修改用户环境或下载依赖。
- 下一步将身份、会话与树状态合并到有界租约审查及条件删除；仍须覆盖监督进程强杀、遗留旧格式保留和真实运行中保护。完整目标继续进行。

Windows 遗留租约执行回收（2026-09-08）：
- state 提供 128/256 有界租约分页（兼容缺少身份表的旧数据库）与按 ID/代/PID/启动身份全部匹配的条件删除。原进程不存在或 PID 对应不同启动身份才作为监督进程退出证据；访问错误保持未知。
- Windows clean 执行模式安装租约审查：只接受规范 windows-v2 会话/创建时间及规范随机租约 ID；监督进程退出且同会话 Job 不存在才删除匹配租约。缺少/旧格式身份保留，跨会话和观察错误保留并增加 unknown_leases；成功数 reported 为 recovered_leases。非 Windows 未安装回收实现，继续保守保护。
- TestCleanKilledWindowsSupervisor 使用真实独立监督测试进程获取租约，再经 runner 创建命名 Job 并启动持续运行子进程，stdout 就绪握手后父进程发布两个新代。活进程情况下 clean 不回收/不删；强杀已知监督进程并 Wait、等待 Job 消失后 clean 回收一条租约并删除旧代。未启动真实 Python/Node 后端，验证的是实际 Windows 进程/Job/SQLite/文件清理链。
- 单独 `go test ./internal/core -run TestCleanKilledWindowsSupervisor -count=1 -v -timeout 25s` 通过（0.566 秒）。收紧身份规范并更新帮助后 `go test ./internal/core ./internal/state ./internal/cli -run 'TestClean|TestLease' -count=1 -timeout 30s` 通过（core 1.923 秒、state 0.554 秒、cli 1.173 秒）。Windows 开发构建通过，全部在隔离工作区沙箱中执行。
- 明确未完成：dry-run 目前不模拟租约回收，因此会遗漏执行 clean 回收租约后才出现的候选，下一步须补齐这一预览一致性合同；不能把当前阶段称为 clean 完整验收。旧租约、跨平台树身份、共享缓存、性能与发布门禁仍待完成，完整目标继续进行。

租约回收的只读预览一致性（2026-09-08）：
- dry-run 现在也按平台审查租约退出证据，但只累计 recoverable_leases，不删除任何租约。执行结果区分 recoverable_leases/recovered_leases/unknown_leases，文本与 JSON 均报告。
- state 新增 PreviewCandidates（保留活动/前代/preparing 排除，交由调用者审查租约）与 GenerationLeaseRecords 分页。核心逐代、逐页核对所有租约，只有每条均可回收时才列出该旧代；任一存活、旧格式或未知租约继续保护。无无限集合、无临时数据库写入；正式执行仍先条件回收并重新做数据库删除预约。
- 初步 `go test ./internal/core ./internal/state -run 'TestClean|TestLease' -count=1 -timeout 30s` 通过（core 1.230 秒，state 0.470 秒）。扩展真实强杀测试后 `go test ./internal/core ./internal/cli -run 'TestCleanKilledWindowsSupervisor|TestCleanCLIResult' -count=1 -v -timeout 30s` 通过（core 0.594 秒，cli 0.705 秒）。
- TestCleanKilledWindowsSupervisor 同时验证存活时预览零候选、强杀并确认树退出后预览一条可回收租约/一个旧代；预览前后数据库字节完全一致，原租约仍在；随后真实执行删除的 ID 与预览一致。预览仍是观察时快照，不冻结外部进程或并发修改，执行必须重新判断。
- Windows 构建与 Linux amd64/macOS arm64 交叉构建本轮执行，Unix 仍仅编译证据。未安装后端或触及真实用户环境。
- 上轮记录的 dry-run 遗漏回收后候选问题已由本轮实现及真实测试补齐。共享缓存、Unix 进程树回收、跨平台文件身份、deep doctor、用户数据布局及性能/发布验收仍待完成，完整目标继续进行。

多租约保护与分页查询（2026-09-08）：
- 扩展真实 Windows 强杀测试，在被清理旧代上同时保留父进程活租约和子监督进程租约。强杀子监督后预览报告一条可回收租约但零候选；释放父进程租约后才出现旧代候选，随后清理与预览仍一致。
- 新增 leases(generation_id,id) 索引，将按代查询从可选 OR 条件改为明确 generation_id 条件，保证分页可利用该索引；全租约分页继续按主键 ID。
- 新增 TestLeaseRecoveryObservationAndPaging：一项一页遍历四条租约、不重复/遗漏、按代隔离、缺失身份的旧租约不能条件删除，以及 PID/代/身份不匹配的观察均不能删除。条件删除本身只验证数据库行，OS 进程/树授权仍由核心处理，测试不混淆两者。
- `go test ./internal/core ./internal/state -run 'TestCleanKilledWindowsSupervisor|TestLease' -count=1 -v -timeout 30s` 通过（core 0.560 秒、state 0.203 秒）；补充状态边界用例后 `go test ./internal/state -count=1 -timeout 30s` 全部通过（1.980 秒）。Windows 开发构建通过。
- 查询性能路径修改后运行一次固定基准：`go test ./internal/state -run '^$' -bench BenchmarkGenerationLeasePage -benchtime=100x -benchmem -count=1` 通过；Windows amd64、i7-10850H，10000 条租约中读取目标代 128 条，100 次均值 647312 ns/op、36514 B/op、712 allocs/op。设置写入不计时。此为热数据库分页微基准，不是 run/status p95、冷启动或整树 RSS 验收，不据此宣称达标。
- 全部使用隔离临时状态，无用户环境或运行时下载。Unix 本轮未运行/重建。共享缓存、Unix 树身份、deep doctor、用户目录与正式性能/平台门禁继续待办，完整目标保持进行中。

生成父目录缺失后的清理恢复（2026-09-08）：
- 修复已发布旧代清理在整个 generations 父目录缺失时提前失败的问题。生成目录与下载目录统一通过受限父句柄访问，缺失父目录视为该处文件已清理；仍先校验规范 ID/精确数据库路径，执行时仍重新预约并条件完成事务。
- 不创建缺失父目录；访问拒绝或其他非缺失错误继续返回，不能将所有 I/O 错误吞作已删除。活动/前代元数据继续由数据库候选规则保护，即使对应文件也被外部删除，clean 不改变这些引用。
- 新增 TestCleanMissingGenerationParent，构造三代数据库状态但不创建两个文件父目录，验证只读预览一项零字节、执行完成旧代元数据收尾、两个父目录仍缺失、活动/前代记录保留和再次无候选。
- `go test ./internal/core ./internal/cli -run 'TestClean' -count=1 -timeout 30s` Windows 沙箱内通过（core 1.823 秒、cli 0.958 秒），覆盖新场景与已有失败准备/租约/真实强杀/CLI 输出回归。Windows 开发构建通过；未修改用户环境或下载后端，Unix 本轮未重建/运行。
- 共享缓存、Unix 子树回收、deep doctor、用户数据布局、文件身份与完整性能/平台门禁仍未完成，完整目标保持进行中。

共享存储路径基础（2026-09-08）：
- 按方案第 8 节重新核对当前布局：Python runtime、固定 uv backend 和 uv cache 目前仍位于项目 work 下，尚未满足平台用户数据/缓存目录及跨项目复用。迁移不能只替换路径，还需要共享安装串行点、缓存所有者操作及测试目录注入，当前不宣称布局已达标。
- 新增 config.ResolveUserStorage 与 UserStorage：数据目录 Windows 采用 LOCALAPPDATA、Linux 采用 XDG_DATA_HOME 或 ~/.local/share、macOS 采用 ~/Library/Application Support；缓存目录复用 os.UserCacheDir。结果在根下追加 myenv，要求绝对路径。明确注入 OS-root 参数供隔离集成，解析只读不创建目录。
- TestUserStorageResolution 验证显式 data/cache 隔离、无创建副作用、相对路径拒绝、三平台数据根选择、缺失 LOCALAPPDATA/home 的错误。平台选择通过注入环境函数测试，不覆盖真实环境变量。
- `go test ./internal/config -run TestUserStorageResolution -count=1 -v` Windows 沙箱内通过（0.130 秒）。未运行后端、移动已有代或改写真实用户目录。此轮为路径解析基础，尚未接入 Service，既有开发二进制行为不变，未重复构建。
- 下一步接入内部存储依赖与隔离测试，并为共享 uv/backend/runtime 安装补齐跨项目锁，再迁移默认布局。完整目标中的共享缓存、Unix 树回收、deep doctor 与平台/性能门禁仍在进行。

可注入共享存储与 uv 安装锁（2026-09-08）：
- Service 新增 Storage（应用级 Data/Cache 路径），统一 storagePaths 校验绝对路径；显式注入后 uv backend、Python managed runtime 默认目录及 uv 解析/venv/project sync 缓存均采用共享目录。未提供 Storage 暂时保留项目布局，待默认迁移与全面隔离测试接通，不宣称默认布局已满足方案。
- managedUV 在 backends 目录取得按固定 uv 版本/平台的 OS 文件锁，再检查入口、准备和发布；不同项目共享同一串行点。Python 显式共享存储下增加按平台/精确版本的安装锁，目前覆盖安装和 venv 准备阶段；项目修改锁继续独立，固定锁顺序为项目再共享资源。
- TestSharedUVRetained 并发运行两个独立 Service/项目上下文，复用保留官方 uv archive，经实际摘要校验、解包和版本验证，只产生一份后端并返回同一可执行文件；后续传入不存在 archive 仍成功复用；不创建项目目录。`go test ./internal/core -run TestSharedUVRetained -count=1 -v -timeout 40s` 通过（3.553 秒）。随后将测试路径断言改为直接验证返回 executable 位于共享 Data 下，未另行重复真实解包。
- TestProfilePythonRetained 改为显式隔离 Data/Cache；仍使用保留 PythonDirectory 避免重新下载安装解释器。因此验证新缓存与锁路径、真实 venv/运行/失败回退，但不是解释器首次下载到共享目录的证据。`go test ./internal/core -run TestProfilePythonRetained -count=1 -v -timeout 40s` 通过（4.219 秒）。
- 两项实际后端测试按既有 Windows 祖先 metadata 依据获准，未下载运行时或触及真实用户存储。Windows 开发构建通过；Unix 本轮未重建/运行。
- 下一步迁移默认存储，给现有 CLI/集成注入独立用户数据路径；还需共享 Node 制品、缓存所有者清理、跨项目引用保护与平台/性能门禁。完整目标保持进行中。

CLI 默认采用用户共享 Python 存储（2026-09-08）：
- sync/use 通过 runtimeService 构造核心服务，默认解析平台用户 Data/Cache；uv backend、Python runtime 与 uv cache 采用上轮共享路径及锁实现。help/status/run 不构造该安装服务，不因只读命令解析或初始化存储。
- 私有 execute 的隔离 userConfigDirectory 现在同时派生 test-storage/data 与 cache，测试不覆盖 HOME 或真实用户环境。项目与 profile 使用相同共享资源，但各自声明、环境代和状态引用仍分离。
- TestRuntimeServiceStorage 检查默认映射与 ResolveUserStorage 一致、项目/profile 共享同一路径、构造无目录创建、相对注入拒绝。TestGlobalProfileCLIRetained 的准备也改用 CLI 相同服务工厂，在隔离路径下验证真实同步、run、回滚与 no-op。解释器仍显式复用保留 PythonDirectory，未重复下载到默认用户目录。
- `go test ./internal/cli -run 'TestErrorOutputContract|TestUse|TestCleanCLIResult' -count=1 -timeout 30s` 通过（1.230 秒）。`go test ./internal/cli -run 'TestRuntimeServiceStorage|TestGlobalProfileCLIRetained|TestPythonBuildNoninteractive' -count=1 -v -timeout 40s` 通过（6.049 秒），真实运行按既有 Windows 祖先 metadata 依据获准，只写隔离目录。
- 离线手册说明各平台存储位置及旧环境保持原路径。Windows 构建和 Linux amd64/macOS arm64 交叉构建通过（CGO_ENABLED=0），Unix 未原生运行。
- Core Service 未显式提供 Storage 时仍保留旧项目布局作为内部迁移兼容路径；正式 CLI 安装入口已显式传入用户存储。已有健康代 no-op 不强制迁移；下一次必要准备按新布局使用共享资源，旧代及其解释器不自动移动/删除。
- 共享 Node 制品、缓存所有者清理、共享运行时引用管理、Unix 树回收和最终平台/性能验收仍在目标范围内，完整目标保持进行中。

Node 按摘要共享制品缓存（2026-09-08）：
- 显式 Storage 下的 Node sync 改用 user Cache/node-archives/<sha256>，按摘要 OS 文件锁串行下载/读取；同制品只发布一次缓存文件。下载复用现有 512 MiB 上限、流式摘要和 flush，再同目录 rename 发布。无 Storage 的内部兼容路径仍直接下载到操作目录。
- 每次从共享缓存复制到操作专属暂存文件，固定 64 KiB 缓冲、512 MiB 上限、取消检查及完整摘要复核，flush 后才交给原有解包/版本验证。复制失败清理暂存文件；缓存损坏明确 CHECKSUM_MISMATCH，不静默采用或覆盖。项目副本不与共享缓存硬链接，原有操作 defer cleanup 继续仅删自己的副本。
- `go test ./internal/core -run TestNodeArchiveSharedCache -count=1 -v -timeout 30s` 通过（0.258 秒）：本地传输 fixture 并发两次仅一次 HTTP 请求、独立项目副本、修改副本不影响缓存、损坏缓存拒绝且无未验证暂存残留。该 fixture 不是运行时安装证据。
- `go test ./internal/core -run TestNodeCacheRetainedArchive -count=1 -v -timeout 60s` 通过（12.537 秒），复用保留官方 Node 22.23.2 ZIP，由本地 HTTP 提供字节，首次写缓存后关闭服务器；再次离线命中并复制，PrepareNodeZIP 完成真实解包及 Node/npm 验证。按已知 Windows 祖先 metadata 依据获准，仅隔离目录，无外网下载或真实用户缓存写入。
- 离线手册更新共享 Node 制品说明，Windows 开发构建通过；Unix 本轮未重建/运行。缓存锁当前覆盖校验复制，后续清理应使用同一锁，不能直接绕过删除。
- 未完成：共享缓存目录的所有权/路径加强、损坏缓存显式修复与手动清理、跨项目运行时引用管理及完整平台/性能验收。完整目标保持进行中。

共享 Python 安装锁范围（2026-09-08）：
- 将共享解释器安装提取为 installSharedPython，资源锁仅覆盖 uv InstallPython；返回解释器路径后即释放，再在各项目最终目录创建 venv。避免同版本的独立项目因本地 venv 准备而持有共享安装锁。项目修改锁和 uv 自身缓存协调保持原有职责。
- 提取时首次编译发现局部 err 作用域未声明，已改为 if 短声明。修复后 `go test ./internal/core -run TestProfilePythonRetained -count=1 -v -timeout 40s` 通过（4.781 秒），覆盖真实 profile venv、版本声明改变/回滚/失败保留。
- 新增 TestSharedPythonConcurrentRetained：两个 Service 使用相同 Data/Cache 与保留解释器，并发创建两个项目的真实 venv；验证返回同一 base Python、不同 venv 入口且两者存在。`go test ./internal/core -run TestSharedPythonConcurrentRetained -count=1 -v -timeout 40s` 通过（3.143 秒）。此为并发正确性证据，不作为吞吐量提升或性能预算达标声明。
- 真实调用复用已保留 uv/Python，不重新下载解释器；均使用隔离临时 Data/Cache。Windows 祖先 metadata 按已有依据获准，Windows 开发构建通过，Unix 本轮未重建/运行。
- 共享缓存清理、共享解释器引用管理、Unix 树回收、deep doctor 与完整平台/性能验收仍待完成，完整目标保持进行中。

Node 缓存显式清理与损坏恢复（2026-09-08）：
- 新增 CleanNodeCache 与 clean --cache node（替代本次项目代清理，不要求项目发现），支持 --dry-run/--json。目录按 128 条目分批读取，仅接受规范 64 位小写十六进制文件名；其他临时/锁/无关文件保持不变。非普通文件拒绝，不递归删除外部目标。
- 正式删除使用与 nodeArchive 相同的摘要 OS 锁，锁取得后重新读取类型/大小，在受限 Root 下删除单个缓存文件。缓存是传输副本，项目暂存与应用文件不使用硬链接，清理不移除解释器或已应用代。预览不取会创建文件的锁、不改存储，执行仍重新核对。
- 扩展 TestNodeArchiveSharedCache：损坏缓存先被同步拒绝，预览仍保留，显式缓存清理后可再次从本地传输源下载并校验。新增 TestCleanNodeCacheCLI 检查隔离 namespace 内 JSON/changed、预览不删、删除缓存而保留无关文件。
- 最终 `go test ./internal/core ./internal/cli -run 'TestNodeArchiveSharedCache|TestCleanNodeCacheCLI|TestCleanCLIResult' -count=1 -timeout 30s` Windows 沙箱内通过（core 0.358 秒、cli 0.844 秒）。核心新增时同名缓存用例也通过（0.251 秒），CLI 接入时已有清理 JSON 回归通过（0.804 秒）。本次传输 fixture 不是后端安装证据，复用此前真实 Node 缓存/解包验证。
- 帮助更新显式 Node 缓存清理用法，Windows 开发构建通过。未写真实用户缓存或下载外部工具；Unix 本轮未原生运行/重建。
- 仍需验证缓存删除与实际下载持锁时的竞争、uv 所有者缓存清理、共享资源路径/权限与跨项目解释器引用；完整平台性能验收继续待办，目标保持进行中。

Node 缓存清理持锁保护证据（2026-09-08）：
- 新增 TestNodeCacheCleanLockProtection，使用实际 state.LockWorkspace 持有缓存摘要锁；执行 CleanNodeCache 在 150 ms 上下文期限内不能删除，必须返回 DeadlineExceeded，Changed=false/Removed=0，缓存内容保持不变。
- 同时验证持锁期间只读预览仍可报告候选且不修改；释放锁后执行可删除缓存，并保留摘要 .lock 文件，不制造不同锁身份。此为真实 OS 锁/文件删除边界测试，锁由测试模拟缓存使用者持有，不将其宣称为下载流中断或进程强杀场景。
- `go test ./internal/core -run TestNodeCacheCleanLockProtection -count=1 -v -timeout 20s` Windows 沙箱内通过（0.371 秒）。生产实现未改动，未重复运行真实 Node 安装、未重建二进制；沿用上轮缓存修复/CLI 与构建证据。
- 本轮补齐缓存清理与持锁使用者的等待/取消合同。uv 缓存所有者清理、共享解释器引用管理、路径/权限与最终平台/性能门禁继续待办，完整目标保持进行中。

uv 所有者缓存清理后端（2026-09-08）：
- 直接读取保留 uv 0.11.26 的 `cache clean --help`，确认显式 --cache-dir 与 --force 的含义；选择不传 --force，保留 uv 的使用中检查。没有猜测 dry-run 支持，实际 help 中没有该选项。
- 新增 backend.UV.CleanCache：要求绝对目录，缺失缓存直接无操作，拒绝文件/链接；验证固定 uv 后通过 runner 执行 cache clean --cache-dir ... --no-config --offline --no-python-downloads --no-progress --color never，子进程输出摘要限制 8192 字节。执行后的错误返回 attempted=true，避免部分清理失败被误报为完全无修改。
- TestUVCleanCacheRetained 用真实保留 uv 清理隔离 archive-v0 fixture，确认缓存文件被删除、同级 runtime fixture 字节保持；缺失缓存不初始化，普通文件不能当作缓存清理。此为真实 uv 命令证据，不是依赖安装或 in-use 锁冲突测试。
- `go test ./internal/backend -run TestUVCleanCacheRetained -count=1 -v -timeout 30s` Windows 沙箱内通过（0.484 秒），没有外网下载或真实用户缓存修改。生产入口尚未调用此新增后端方法，未重复构建。
- 下一步接入 clean --cache uv 的现有后端定位与只读预览，不能为清理隐式下载安装 uv；之后补充使用中缓存行为及完整平台/性能验收。完整目标保持进行中。

uv 缓存清理 CLI 与预览（2026-09-08）：
- 新增 CleanUVCache 与 clean --cache uv，使用应用 Cache/uv 路径；缺失缓存直接无操作，拒绝根链接/普通文件。预览按有界目录统计逻辑字节，不调用 uv 或初始化后端，报告 uv_cache 整体候选。
- 执行只读取 Data/backends 中已有固定 uv 入口（或内部显式 UV 注入），校验入口留在后端目录内，再调用上轮 UV.CleanCache。不使用 managedUV 安装路径，不为清理下载安装后端；版本验证、离线、禁用配置发现及不传 --force 的规则保留。
- attempted 已进入后端调用时保守标记 Changed；失败仍输出单个 CLEAN_FAILED JSON 及清理专用重试提示。逻辑字节为清理前观察，后端和其他进程可能使实际回收量不同，不声称为精确释放磁盘块。
- `go test ./internal/cli -run 'TestCleanCLIResult|TestCleanNodeCacheCLI' -count=1 -timeout 30s` 通过（0.830 秒）。`go test ./internal/cli -run TestUVCleanCLIRetained -count=1 -v -timeout 30s` 通过（2.356 秒）：预览无需 uv，缺失后端执行失败且无 Data 创建/缓存删除；复制保留官方 uv 到隔离 Data 并写受管入口后真实 CLI 清理成功，后端文件保留。
- 真实测试复用保留 uv，按既有 Windows 祖先 metadata 依据获准，仅临时 namespace，无外网下载或真实用户缓存改动。帮助更新 node/uv 选择，Windows 开发构建通过；Unix 本轮未运行/重建。
- 使用中 uv 缓存冲突、项目 venv 在缓存清理后仍可运行、共享解释器引用及完整平台/性能门禁仍需验证，完整目标保持进行中。

uv 缓存清理后活动与前代可用（2026-09-08）：
- 扩展既有真实 wheel/group fixture，新增 TestPythonWheelCacheCleanupRetained，使用隔离共享 Data/Cache，保留固定 uv/Python，从本地 HTTP 索引安装实际可导入 wheel，准备两个组选择不同的代。
- 关闭包索引后先预览，确认缓存非空且预览无修改，再经 CleanUVCache 调用真实 uv 清理。随后当前代与前一代分别重新导入 myenv_base/myenv_qa，并保持未选组不存在；SelectRun 返回原活动代，最后 sync --locked 在索引关闭、缓存清理后仍 no-op。
- `go test ./internal/core -run TestPythonWheelCacheCleanupRetained -count=1 -v -timeout 60s` Windows 通过（11.420 秒）。测试按既有祖先 metadata 依据获准，全部缓存/项目在隔离临时目录，无外网运行时下载或真实用户配置修改。
- 此轮只新增相关验证，没有生产代码变更，未重复构建或运行已通过的其他真实组选择用例。该证据覆盖缓存清理不破坏 copy 安装的活动/前代依赖，不代表 uv 正在使用缓存时的冲突或 Unix 原生行为已验证。
- uv 使用中清理、共享解释器引用、路径/权限、Unix 树回收、deep doctor 及完整平台/性能门禁仍待完成，完整目标保持进行中。

清理取消与部分结果（2026-09-08）：
- 项目清理、Node 缓存清理和 uv 缓存清理入口先检查 ctx.Err，已取消请求不再解析/访问后续资源。Node 缓存在取得摘要锁并完成元数据检查后、实际 Remove 前再次检查取消。
- 新增 TestNodeCacheCleanupCancelBetweenItems：两个缓存对象，第一条成功报告后取消上下文，验证仅一项被删、下一项保留、结果 Changed=true/Removed=1 并返回 context.Canceled；同一已取消上下文再次执行保持无修改。
- `go test ./internal/core -run 'TestNodeCacheCleanupCancelBetweenItems|TestNodeCacheCleanLockProtection|TestCleanMissingGenerationParent' -count=1 -v -timeout 20s` Windows 沙箱内通过（0.480 秒），同时复核持锁超时和缺失父目录边界。Windows 开发构建通过，未下载后端或修改真实用户缓存。
- 取消检查无法撤销已完成删除，结果保留部分进度；不将检查点之间的 OS 调用声称为可原子取消。uv 使用中清理、共享资源引用、Unix 行为、deep doctor 与正式性能/平台门禁仍待完成，完整目标保持进行中。

深度诊断的环境代内容摘要基础（2026-09-08）：
- 核对 Doctor 目前只有 quick 检查，快照中没有受管文件内容基线；不能靠当前版本号检查冒充 doctor --deep。新增 config.GenerationDigest 供后续发布记录/深度复核使用，尚未接入 Sync 或 CLI。
- 摘要记录规范相对路径、文件类型、权限位、普通文件 SHA256 与符号链接目标文本，不跟随链接到外部解释器。忽略 Python __pycache__ 目录及根 completion/evidence 标记；配置/锁等普通文件仍参与摘要。外部共享运行时内容需要其所有者单独的证据，不在该目录摘要内。
- 文件读使用 64 KiB 缓冲；单目录最多 4096 条目、总百万条目/128 层/8 GiB 内容，并检查取消。子项排序后流式编码到摘要，输出常量大小，不缓存全目录文件内容。此功能只面向准备/深度检查，普通 status/run 尚未增加扫描。
- TestGenerationDigestChanges 验证稳定基线、字节码缓存不造成漂移、同大小文件内容变化与新增文件能改变摘要、取消请求拒绝。`go test ./internal/config -run TestGenerationDigestChanges -count=1 -v -timeout 20s` Windows 沙箱内通过（0.136 秒）。只使用临时小文件，无用户环境修改。
- 本轮为内容证据基础，doctor --deep 尚未提供；后续需将基线绑定到发布事务、处理旧代缺失证据、补齐外部运行时/权限范围及真实制品性能测量。未重复构建未调用的新函数，完整目标保持进行中。

发布内容基线与只读深度诊断（2026-09-08）：
- generation_evidence 保存 generation-tree-v1 摘要；PublishWithEvidence 将基线、代记录、活动引用及操作完成状态在同一事务发布，外键随代删除。旧表或旧代缺失证据返回 missing，不当作匹配。
- Sync 完成快照/锁/标记后计算摘要，再核对声明、期望锁及 Python 输入，变化拒绝发布。初次编译重复短声明失败，修正赋值后通过。无变化同步和普通 status/run 不增加内容扫描。
- doctor --deep 输出 matched/mismatch/missing/unavailable、检查级别及范围；内容不同标记 content_changed。只读检查不创建基线、不取得租约、不修复文件；扫描失败或活动引用变化不报告匹配。这不是并发写入下的原子文件系统快照，不验证来源、外部解释器内容或 ACL。
- go test ./internal/core ./internal/state -run 'TestEvidencePublicationAtomic|TestProfilePythonRetained' -count=1 -v -timeout 40s 通过（core 4.476 秒、state 0.315 秒），覆盖发布绑定、CAS 失败不泄露基线和真实 profile 同步。
- go test ./internal/core ./internal/state ./internal/cli -run 'TestDoctorDeepMissingEvidence|TestEvidencePublicationAtomic|TestStatusReadOnly' -count=1 -timeout 30s 首次 core 失败：旧代 fixture 缺少必需运行时入口；state 通过 0.283 秒，cli 无匹配测试。修正 fixture 后 go test ./internal/core -run 'TestDoctorDeepMissingEvidence|TestProfilePythonRetained' -count=1 -v -timeout 40s 通过（4.984 秒）：无状态不初始化、旧代不写数据库、真实 Python 基线匹配、新增文件检出、quick 保持快速检查。真实测试按既有 Windows metadata 依据获准，仅使用隔离目录及保留制品。
- go test ./internal/cli -run TestDoctorDeepJSON -count=1 -v -timeout 20s 通过（0.213 秒），验证单 JSON、deep/missing、Changed=false 和无状态写入。帮助/手册更新，go build -o dist/myenv.exe ./cmd/myenv 通过；Unix 本轮未重建或原生运行。
- 待办：真实大型环境摘要成本、变化后的显式重建、共享运行时证据、路径/权限加强、uv 使用中清理、共享解释器引用、Unix/性能/发布门禁。完整目标保持进行中。

深度诊断后的显式重建（2026-09-08）：
- 上轮已产生代码、测试和构建证据，属于 progress。本轮核对方案快速路径约束，发现内容变化后普通 sync 仍可能 no-op，缺少安全恢复入口。新增 sync --rebuild / SyncRequest.Rebuild，仅跳过活动代快速复用，沿用新代准备及单一发布流程；支持 --locked、--dry-run 和 --global。
- doctor 内容不匹配时建议 sync --rebuild --locked，profile 提示含 --global。重建不会直接修改旧代、重装共享基础解释器或删除共享缓存；不能据此声称能修复外部解释器。帮助及离线手册说明边界。
- 扩展真实 TestProfilePythonRetained：新增文件引起 deep mismatch 后，以缺失 uv 验证 dry-run 仍可预览 NeedsApply、执行失败保留活动引用；恢复保留 uv 后真实重建，新 ID、锁字节不变、旧代修改保留、新代无额外文件且 deep matched。后续版本切换/回退验证继续以重建代为前代。
- go test ./internal/core -run TestProfilePythonRetained -count=1 -v -timeout 40s 通过（7.448 秒）。使用隔离目录及保留 Python/uv，按既有 Windows 祖先 metadata 依据获准，无真实用户存储修改或外网下载。随后仅移除测试中先删除再写回相同 fixture 的冗余步骤，没有生产变更或改变观察输入，不重复安装验证。
- go test ./internal/cli -run 'TestRebuildRequiresMatchingLock|TestDoctorDeepJSON' -count=1 -v -timeout 20s 通过（0.259 秒），确认 flag 接入、缺失锁被 --locked 拒绝且 dry-run 无状态创建，deep JSON 合同保持。go build -o dist/myenv.exe ./cmd/myenv 通过。
- Unix 原生、实际 Node 重建、native Python 项目重建和大环境摘要成本尚未本轮验证；共享资源引用、外部运行时证据、权限/路径、uv 使用中缓存及最终性能/发布门禁仍待完成，完整目标保持进行中。

实际 Python 依赖的缓存清理与锁定重建（2026-09-08）：
- 上轮重建实现和真实 profile 验证属于 progress。本轮扩展既有 wheel/group fixture，新增 TestPythonWheelRebuildRetained，使用项目 uv.toml 相对本地 wheel 源，HTTP 已关闭，共享缓存经真实 uv 清理。
- 在真实环境中通过模块 __file__ 定位已安装 myenv_base，转换为代内相对路径并经 Within 检查后修改其源文件；doctor --deep 必须报告 mismatch。随后 sync --rebuild --locked 从保留本地 wheel 重建，验证新 ID、实际导入恢复、组选择保持、旧代被修改的文件不动、另一保留代仍可导入。
- 重建前后 myenv.lock 和 uv.lock 字节一致，LockChanged/NativeLockChanged 均 false；执行导入后新代 deep 仍 matched，覆盖 Python 字节码生成不造成漂移。
- go test ./internal/core -run TestPythonWheelRebuildRetained -count=1 -v -timeout 60s 首次失败（9.908 秒）：测试误将绝对路径传给只接受相对路径的 Within；修正为 filepath.Rel 后同命令通过（16.412 秒）。两次都在隔离项目/共享缓存目录，复用已保留 uv/Python，按既有 Windows 祖先 metadata 依据获准；没有外部运行时下载或真实用户配置修改。
- 本轮只增加受影响集成验证，生产代码未变，沿用上一轮 Windows 构建，不重复构建或运行无关测试。该证据是本地 wheel 依赖重建，不代表索引不可达且无任何依赖来源时仍可重建，也不证明共享基础解释器重装或 Unix 原生行为。
- 大环境摘要成本、Node 重建、共享运行时证据和引用、权限/路径、uv 使用中缓存与完整平台/性能/发布门禁继续待办，完整目标保持进行中。

锁文件重复 JSON 字段拒绝（2026-09-08）：
- 上轮实际 wheel 重建验证属于 progress。本轮检查 ReadLock：标准 JSON struct 解码默认会合并/覆盖重复字段，DisallowUnknownFields 不拒绝重复。新增解码前的 token 检查，逐对象记录已见键，拒绝重复键（含转义后同名），不同对象独立计数；限制 64 层嵌套并拒绝多顶层值。输入仍由 ReadInput 限制大小，未引入依赖。
- go test ./internal/config -run 'TestLock|TestJSONKey' -count=1 -timeout 30s 通过（0.451 秒）。新增证据覆盖顶层 schema、平台映射、运行时 version、Unicode 转义重复、数组内对象重复、独立对象同名合法、截断/多值/过深输入。此检查针对相同解码键，不声称已解决 encoding/json 的大小写字段别名兼容行为。
- go build -o dist/myenv.exe ./cmd/myenv 通过，未运行后端或修改真实用户环境。额外 token 遍历对锁读取的开销仍需纳入最终状态/run/no-op 性能测量；Unix 本轮未原生运行。
- 共享运行时证据/引用、路径权限、uv 使用中清理、Node 重建和完整平台/性能/发布门禁仍待完成，完整目标保持进行中。

锁字段大小写歧义关闭（2026-09-08）：
- 上轮重复字段拒绝属于 progress。本轮补齐 encoding/json 对 struct 字段大小写不敏感的漏洞：锁对象键必须采用规范小写 ASCII 字母、数字、下划线或连字符，包括平台/tool 映射键；非规范字段不再作为别名接受。现有写入器产生的固定字段、平台标识和工具名符合此规则，字符串值中的大小写/Unicode 不改变。
- go test ./internal/config -run 'TestLock|TestJSONKey' -count=1 -timeout 30s 通过（0.391 秒）。新增测试覆盖 Schema、运行时 Version、Python 输入大写字段、Unicode 转义大写和 Unicode 长 s 别名；确认 Unicode 项目路径和大小写 URL 值保持合法。
- go build -o dist/myenv.exe ./cmd/myenv 通过，无后端调用和真实用户配置写入。仅手写非规范字段名的旧锁会被拒绝，自动生成锁无需迁移。输入读取性能仍待最终测量。
- 全部架构目标尚未完成：共享资源引用/证据、平台进程树行为、权限/路径、缓存并发、性能与发布验收继续待办。

锁解析与 Windows 启动性能初测（2026-09-08）：
- 上轮严格锁键实现属于 progress。本轮新增 BenchmarkReadLock，固定三平台、每平台 Node/Python 和原生 Python 输入摘要，分别测内存 strict_keys 与文件读取/解码/Validate；不将微基准均值当作命令 p95。
- go test ./internal/config -run '^$' -bench BenchmarkReadLock -benchmem -benchtime 500x -count=1 -timeout 30s 通过（0.864 秒）。Windows amd64 / i7-10850H：strict_keys 101941 ns/op、14914 B/op、654 allocs/op；file_decode_validate 1223943 ns/op、35976 B/op、805 allocs/op。本轮只加 benchmark，生产二进制未变，未重建。
- PowerShell 使用 Stopwatch 启动 dist/myenv.exe --version 与 --help 并排空 stdout，各 51 次，首个观察单报，其后 50 次排序取 nearest-rank p95。制品 SHA256 FA99C7346E31F4651FAAD29ECD8E96A79D29D9142DED51E95C1523678A8D48FB。结果存 .build/perf/startup-windows.json。
- --version 首次观察 116.6503 ms、中位 85.6764 ms、p95 109.741 ms；--help 首次观察 111.7815 ms、中位 71.0482 ms、p95 96.0864 ms。包含 PowerShell 启动/管道开销，未做 OS 缓存清空，因此首次观察不称作冷缓存基准。数据明显需要继续定位启动开销，不能宣称性能门禁通过。
- 下一步以直接进程启动测量区分 shell/管道开销，再补齐实际项目 status/run/no-op、RSS/子进程树和冷缓存。完整平台及架构目标保持进行中。

直接启动与最小 Go 对照（2026-09-08）：
- 上轮性能测量属于 progress。本轮新增 scripts/measure-startup.ps1：Process.Start/UseShellExecute=false/CreateNoWindow，异步排空 stdout/stderr，10 秒超时仅终止本次测量子树，保存全部样本、制品摘要、OS、首个观察和 nearest-rank 中位/p95。默认每命令 50 个正式样本，首个另报，不声称 OS 冷缓存。
- .\scripts\measure-startup.ps1 实际通过，结果 .build/perf/startup-direct-windows.json。既有 Windows myenv 制品 SHA256 FA99C7346E31F4651FAAD29ECD8E96A79D29D9142DED51E95C1523678A8D48FB：version 中位 141.1231 ms/p95 176.6137 ms；help 中位 94.2394 ms/p95 123.1062 ms。没有降低 30 ms 预算，当前观察不达标。
- 临时设置 GODEBUG=inittrace=1 执行该制品 --version 后恢复环境，记录 .build/perf/inittrace-windows.txt。单次样本最后包初始化约 @17 ms，sqlite/lib 自身 0.52 ms，config 0.57 ms；不能据单次初始化输出把其余耗时全部归因于加载器或宿主。
- 在 .build/perf/control/main.go 建立仅 fmt.Println 的最小 Go 对照，go build -o .build/perf/control.exe ./.build/perf/control 通过；用相同脚本 -Executable .build/perf/control.exe -Output .build/perf/startup-control-windows.json 测量通过。对照不解析参数，只在两个批次输出固定短文本：第一批中位 70.4981 ms/p95 87.3194 ms，第二批中位 56.2641 ms/p95 80.9602 ms。对照自身仍高于预算，说明测量包含显著宿主/进程启动成本；不同批次有波动，不将 p95 相减当作 myEnv 精确附加开销。
- 本轮未修改生产实现或重建 myenv。后续需要交错对照/参考环境与真实 status/run/no-op、RSS 证据，定位后优化，不以这组受当前宿主影响的数据宣布性能目标不适用。全部架构目标保持进行中。

Windows CLI 启动中的 Cobra Explorer 检测（2026-09-08）：
- 上轮启动对照属于 progress。本轮新增 BenchmarkInformationalCLI，将 Execute 命令构建/输出隔离于包初始化和进程创建。go test ./internal/cli -run '^$' -bench BenchmarkInformationalCLI -benchmem -benchtime 500x -count=1 -timeout 30s 首次通过（37.239 秒，基准阶段不受普通测试 timeout 限制）：version 42232352 ns/op、23746 B/op、192 allocs/op；help 31555947 ns/op、31700 B/op、322 allocs/op。
- 据本地固定 Cobra 1.10.2 command_win.go/cobra.go 源码，preExecHook 每次调用 StartedByExplorer，默认 Explorer 启动还会暂停/退出。使用依赖明确提供的禁用接口，在 Windows 专属 startup_windows.go 的 init 中设置 MousetrapHelpText 为空，避免每条 CLI 命令执行此检测，也符合显式输入控制。不改第三方模块或宿主设置。
- 同一基准修复后通过（0.299 秒）：version 31753 ns/op、22578 B/op、189 allocs/op；help 56052 ns/op、30517 B/op、319 allocs/op。显著减少进程内执行开销；go build -o dist/myenv.exe ./cmd/myenv 通过。
- .\scripts\measure-startup.ps1 -Output .build/perf/startup-no-mousetrap-windows.json 通过，完整进程各 50 次：version 中位 92.4699 ms/p95 150.6437 ms，help 中位 84.9252 ms/p95 111.0421 ms；新制品 SHA256 EA82A978113BB005A7AB580BBF270DA26C1CD872F1B043F45DA2417BE4140B0C。完整启动仍未达到 30 ms 预算，批次波动不能用单纯差值归因，保留全部样本。
- 下一步继续参考环境启动与 RSS、实际项目 status/run/no-op 验证。本轮修复作用于 Windows，未声称 Unix 已验证或完整性能门禁通过；全部架构目标保持进行中。

真实 Node/npm 内容证据与共享缓存重建（2026-09-08）：
- 上轮 Windows CLI 启动修复属于 progress。本轮提取已有 TestSyncRetainedNode fixture 入口，新增 TestNodeRebuildRetained 分支；普通旧场景保持原流程，新分支使用隔离显式共享 Data/Cache。
- 复用已保留官方 Node 22.23.2 ZIP，由本地 HTTP 服务原始字节，经原有摘要验证/真实解包/运行时校验发布。首次 doctor --deep matched，单次用时 9.7247274 秒；是本宿主单次观察，不是 p95、冷缓存或大项目通用性能声明。
- 修改实际 npm-cli.js（追加注释），deep 检出 mismatch；关闭服务后 sync --rebuild --locked 从共享摘要缓存完成新代发布，锁字节不变，旧 npm 文件保持修改，新代 deep matched，下载计数仍为 1。后端 Prepare 原有 Node/npm 验证仍执行，不用假可执行入口替代。
- go test ./internal/core -run TestNodeRebuildRetained -count=1 -v -timeout 90s 通过（68.722 秒）。Windows 祖先 metadata 按既有依据获准，隔离存储，无外网制品下载或真实用户缓存写入。未重复既有长场景，生产代码未变、沿用上轮构建。
- 真实 Node 扫描成本显著，后续应分析有界 Root 访问和文件数量成本，保持普通 status/run/no-op 不扫描。共享解释器证据/引用、权限路径、uv 使用中清理和完整 Unix/性能/发布门禁仍待完成，完整目标保持进行中。

诊断在已取消请求下停止读取（2026-09-08）：
- 上轮真实 Node 重建属于 progress。本轮在 Status 入口和 GenerationDigest OpenRoot 前检查 ctx.Err；Doctor 在 Status 后、DoctorDeep 在快速诊断后再检查取消，避免无代/无基线分支掩盖此前已发生的取消。
- go test ./internal/core ./internal/config -run 'TestCanceledDiagnosisBeforeDiscovery|TestDoctorDeepMissingEvidence|TestGenerationDigestChanges' -count=1 -v -timeout 30s 通过（core 0.298 秒、config 0.080 秒）。新增用不存在路径证明已取消请求优先返回 context.Canceled，而非发现/打开路径错误；兼容只读无基线和内容摘要测试。
- go build -o dist/myenv.exe ./cmd/myenv 首次因临时 a.out.exe 被其他进程占用失败；未删除进程/缓存，同命令重试通过。本轮未修改 CLI 信号接入或新增取消 JSON 错误码，不声称完整 Ctrl-C 诊断流程已验证。
- 仍需完成 CLI 取消一致性、共享运行时引用/证据、平台进程树、权限路径、缓存并发和最终性能/发布门禁；完整目标保持进行中。

CLI 操作取消上下文与诊断错误（2026-09-08）：
- 上轮诊断入口取消检查属于 progress。本轮新增 ExecuteContext，既有 Execute/隔离 execute 保持兼容，Cobra 通过 ExecuteContext 传递调用方上下文。非 run 的操作进入 PersistentPreRun 时注册 os.Interrupt NotifyContext，命令返回后 Stop；帮助/版本不执行该 hook，run 保留 runner 原有信号职责。
- 可识别的 context.Canceled/DeadlineExceeded 经普通根错误路径返回 CANCELED、退出码 130，并提示检查当前状态而非声称全部回滚。现有错误部分结果/Changed 保持原处理；clean 的专属输出和被后端转换失去 error wrapping 的错误仍需另行核对，不声称所有命令取消输出已经统一。
- go test ./internal/cli -run 'TestDoctorCanceledJSON|TestDoctorDeepJSON|TestRebuildRequiresMatchingLock' -count=1 -v -timeout 30s 通过（0.221 秒）。新测试用已取消上下文和不存在路径验证一个完整 JSON、CANCELED、130、Changed=false、stderr 空。go build -o dist/myenv.exe ./cmd/myenv 通过。
- 本轮为上下文/API 与 os.Interrupt 接入，未实际发送 Windows 控制台 Ctrl-C，未验证 Unix SIGTERM/SIGHUP 或 run 信号流程；这些仍是验收缺口。共享资源、权限路径、缓存并发、平台与性能发布门禁继续待办，完整目标保持进行中。

清理取消的单 JSON 与部分进度（2026-09-08）：
- 上轮 CLI 取消上下文属于 progress。本轮 clean 专属流式输出识别 context.Canceled/DeadlineExceeded，保留 summary/items/Changed，错误代码 CANCELED、退出码 130；普通 CLEAN_FAILED 继续为 1，stdout 写入失败仍按 I/O 失败处理。
- 新增 TestCleanCanceledPartialJSON，隔离 namespace 中放置两个 Node 缓存对象，在第一条真实删除记录写入输出时取消调用方上下文。通过完整 JSON 解码检查仅一条记录、Removed=1、Changed=true、CANCELED、130、stderr 空；后续只读预览确认另一个对象仍保留。
- go test ./internal/cli -run 'TestCleanCanceledPartialJSON|TestCleanCLIResult|TestDoctorCanceledJSON' -count=1 -v -timeout 30s 通过（0.555 秒），复核普通成功/部分失败和诊断取消。go build -o dist/myenv.exe ./cmd/myenv 通过，无后端下载或真实用户缓存修改。
- 此为 CLI 上下文取消与实际单文件删除边界证据，不是 OS 控制台信号或 uv 进程取消验证。后端转换后的取消识别、平台信号、共享资源/权限/性能与最终发布验收继续待办，完整目标保持进行中。

后端子进程取消错误保留（2026-09-08）：
- 上轮清理部分取消属于 progress。本轮发现 runner 为用户命令返回退出码，后端直接使用时取消可能被转成普通非零后端错误。新增 executeBackend 包装：先等待 runner 完成子树监督，再检查调用方 ctx.Err，返回可识别取消；不改变用户 run 的 runner 退出码合同。
- uv 版本验证/venv/cache clean、Python 安装/查找/解析/版本与 venv 身份验证、原生项目 lock/sync 的执行点统一调用包装层，检查这些调用点后确认错误直接向上传递。缓存清理仍保留 attempted=true 的部分修改语义。
- go test ./internal/backend -run TestBackendProcessCancellation -count=1 -v -timeout 15s 通过（0.253 秒）。实际启动当前测试可执行文件，子进程输出 ready 后由输出接收端取消上下文，等待其结束后要求 errors.Is(context.Canceled)，不是伪造 runner 结果。仅本地 helper，不是实际 uv 长安装取消证据。
- go build -o dist/myenv.exe ./cmd/myenv 通过。普通未取消调用原样透传结果，本轮未重跑不受该条件影响的全部安装；后续仍需真实 uv/控制台信号取消与进程树、上层包装错误审计。全部架构目标保持进行中。

真实 uv 原生同步取消与旧代保留（2026-09-08）：
- 上轮后端取消包装属于 progress。本轮核对 Sync 的 Python 准备/项目同步失败路径使用 %w，保留 errors.Is 语义；新增 TestPythonSyncCancellationRetained 验证完整服务路径。
- 使用保留 uv/Python 在隔离项目先完成 tools-only 活动代，再加入 package=false 项目及仅本地索引依赖。真实 uv 请求索引时，服务端触发上下文取消并等待连接结束。要求确实观察到请求、errors.Is(context.Canceled)、Changed=false；run --current 的核心选择仍指向初始代，临时 .myenv-uv-config-*.toml 无残留。
- go test ./internal/core -run TestPythonSyncCancellationRetained -count=1 -v -timeout 40s 通过（5.879 秒），按既有 Windows 祖先 metadata 依据获准，未下载外部运行时/包或写真实用户配置。此为实际 uv 网络等待取消，不是构建 hook 取消或控制台信号测试；仅验证活动引用/选择保留，未重新执行旧代程序。
- 本轮只新增集成验证，生产代码未变，沿用已有 Windows 构建。完整信号/进程树、共享资源引用证据、权限路径、缓存并发和最终平台性能发布门禁仍待完成，完整目标保持进行中。

非 run 操作的 Unix 终止信号集合（2026-09-08）：
- 上轮真实 uv 取消验证属于 progress。本轮将 CLI 操作信号集合移到平台文件：Windows 继续 os.Interrupt；非 Windows 增加 SIGTERM/SIGHUP，使准备/诊断/清理等非 run 命令进入已有上下文取消与返回清理路径。run 保留 runner 原有信号转发，未更改其退出码合同。
- go test ./internal/cli -run 'TestDoctorCanceledJSON|TestCleanCanceledPartialJSON' -count=1 -timeout 30s Windows 通过（0.207 秒）；go build -o dist/myenv.exe ./cmd/myenv 通过。
- 在本 Windows 宿主设置 GOOS=linux/GOARCH=amd64 后 go build -o dist/myenv-linux-amd64 ./cmd/myenv 通过；GOOS=darwin/GOARCH=arm64 后 go build -o dist/myenv-darwin-arm64 ./cmd/myenv 通过。仅命令环境内设置，没有修改用户环境。二者是跨编译证据，不是原生信号/运行时验证。
- 完整 SIGTERM/SIGHUP 原生行为、Windows 控制台事件、共享资源证据/引用、权限路径、缓存并发与性能发布门禁仍待完成，完整目标保持进行中。

发现并启用现有 WSL Linux 验证环境（2026-09-08）：
- 上轮 Unix 信号集合/跨编译属于 progress。本轮 Get-Command 确认 wsl.exe/bash.exe 已存在，wsl.exe --list --quiet 沙箱内 E_ACCESSDENIED；只读枚举申请获准后成功，已安装 Ubuntu。未安装或修改发行版配置。
- wsl.exe -d Ubuntu -- sh -lc 'uname -a; command -v go; /mnt/c/Users/worker/Documents/myEnv/dist/myenv-linux-amd64 --version' 获准执行通过：Linux 6.18.33.2-microsoft-standard-WSL2 x86_64，Go 路径 /usr/bin/go，当前 myenv Linux 制品输出 0.1.0-dev。WSL 可用于后续 Linux 内核行为验证，不能再将全部 Unix 执行视为不可用；macOS 与非 WSL Linux 仍需区分。
- Windows 设置 GOOS=linux/GOARCH=amd64，go test -c -o .build/runner-linux.test ./internal/runner 通过。首次 WSL 调用测试参数未整体引用，被传成未知 -test，未执行测试；修正参数后 wsl.exe -d Ubuntu -- /mnt/c/Users/worker/Documents/myEnv/.build/runner-linux.test '-test.run=TestCancelProcessGroup' '-test.v' '-test.timeout=20s' 通过。
- TestCancelProcessGroup 与 TestCancelProcessGroupAfterParentExit 各 0.30 秒，实际 Linux shell/进程组执行，覆盖取消及父进程提前退出后子孙管道不挂起。测试仅操作自身隔离子进程，宿主 WSL 访问按已确认权限获准；不是 Windows 模拟/仅编译证据。
- 下一步利用此现有 WSL 补齐 Linux 进程/租约/信号和性能验证，仍不能声称 macOS、终端交互、主管崩溃后全树回收或完整平台门禁已通过。全部架构目标保持进行中。

WSL Linux 状态层与清理验证（2026-09-08）：
- 上轮确认 WSL 可用及进程组测试属于 progress。本轮在 Windows 设 GOOS=linux/GOARCH=amd64，分别 go test -c -o .build/state-linux.test ./internal/state、go test -c -o .build/core-linux.test ./internal/core，均编译通过。
- wsl.exe -d Ubuntu -- /mnt/c/Users/worker/Documents/myEnv/.build/state-linux.test '-test.v' '-test.timeout=30s' 全部 14 项通过且无跳过：删除保留标记、持锁主管被杀恢复、证据原子发布、进程身份生命周期、租约分页/条件回收、租约保留、文件锁争用、操作/Python 发布、旧数据库只读兼容、只读存储、中断恢复、回退及发布冲突。测试数据使用 Linux 临时目录，不将 DrvFS 上的测试制品路径误当作数据库测试目录。
- wsl.exe -d Ubuntu -- /mnt/c/Users/worker/Documents/myEnv/.build/core-linux.test '-test.run=TestClean' '-test.v' '-test.timeout=30s' 六项通过：失败准备、文件边界、缺失父目录、旧完成操作、删除后恢复、活动/租约保留。TestCleanFilesBoundary 没有 symlink fixture unavailable 日志，实际创建外部链接并验证删除不波及外部保留文件；这是本次新增的 Linux 链接边界证据。
- WSL 调用按已确认访问限制获准，只运行项目自身测试、仅处理测试文件和子进程。没有生产代码变更或重复此前已通过的 Linux runner 测试。
- 这些结果不证明 Linux 自动死租约回收已实现（当前保守保留），也不涵盖原生 macOS、终端交互、挂载点攻击或完整性能门禁。下一步继续 Linux 平台缺口，全部架构目标保持进行中。

WSL CLI 实际信号取消（2026-09-08）：
- 上轮 WSL 状态/清理测试属于 progress。本轮新增 Unix 专属 TestOperationSignalCancellation。父测试持有隔离项目 modify.lock，启动真实 CLI 测试子进程执行 clean --json；读取只会在 PersistentPreRun 注册信号上下文之后输出的 JSON 前缀，再分别发送 SIGINT、SIGTERM、SIGHUP，避免靠固定 sleep 猜测注册完成。
- 三个子场景要求子进程退出码 130、一个完整 JSON、CANCELED、Changed=false、stderr 空。持锁阻止清理成功走完；超时只杀测试自己启动的子进程。测试没有模拟信号 handler 或直接 cancel 代替 OS 信号。
- GOOS=linux/GOARCH=amd64 下 go test -c -o .build/cli-linux.test ./internal/cli 通过。wsl.exe -d Ubuntu -- /mnt/c/Users/worker/Documents/myEnv/.build/cli-linux.test '-test.run=TestOperationSignalCancellation' '-test.v' '-test.timeout=40s' 通过（合计 0.09 秒），interrupt/terminated/hangup 均通过。WSL 操作按现有权限获准，仅测试目录/子进程，无真实用户修改。
- 本轮只有测试新增，未重建未改变的生产制品。证据覆盖信号上下文注册后的清理路径，不声称操作注册前窗口、Windows 控制台、uv 安装中的 OS 信号或 Unix run 全子孙回收已验证。完整目标保持进行中。

WSL Linux 启动位置对照（2026-09-08）：
- 上轮实际 Unix 信号测试属于 progress。本轮新增 scripts/measure-startup.py，以 Python subprocess 无 shell 执行帮助/版本，stdout 丢弃、stderr 捕获，每组首个观察另报/50 次正式样本，保存样本、SHA256、平台、文件系统设备及 nearest-rank 中位/p95。另复制同一制品到脚本自己的 TemporaryDirectory，结束自动清理，不修改安装路径。
- wsl.exe -d Ubuntu -- python3 /mnt/c/Users/worker/Documents/myEnv/scripts/measure-startup.py /mnt/c/Users/worker/Documents/myEnv/dist/myenv-linux-amd64 /mnt/c/Users/worker/Documents/myEnv/.build/perf/startup-wsl-linux.json 获准执行通过。制品 SHA256 B8FFB372065B16221B906E35263CB95428540B50F6D6D0FA3A095711E46A2114。
- 原位置 version 中位 56.5754 ms/p95 64.3675 ms，help 中位 7.057 ms/p95 57.106401 ms；临时复制 version 中位 3.0541 ms/p95 5.8931 ms，help 中位 3.2115 ms/p95 5.8437 ms。不同位置差异显著，报告保留全部样本，不将不同批次相减声称精确成本。
- 后续只读 wsl stat -f -c '%T %n' 确认 /tmp 为 tmpfs，工作区 dist 为 v9fs。临时目录并非本地 SSD，因此虽本次延迟低于 30 ms，不能将 tmpfs 数字作为方案要求的本地 SSD 性能门禁通过；还缺 RSS、冷缓存、实际状态/run/no-op 和参考硬件存储测量。
- 本轮生产制品未改，未重建；全部架构目标保持进行中。

WSL 帮助/版本峰值 RSS（2026-09-08）：
- 上轮启动位置对照属于 progress。本轮只读核对现有 /usr/bin/time --version/--help：GNU Time、%M 为最大常驻内存 KB，-o 独立结果文件。新增 scripts/measure-memory-linux.py，逐次运行单个帮助/版本命令，临时复制制品、10 次各自的峰值 RSS，保存样本和制品摘要；不把 Python 进程或同时子树内存混入指标。
- wsl.exe -d Ubuntu -- python3 /mnt/c/Users/worker/Documents/myEnv/scripts/measure-memory-linux.py /mnt/c/Users/worker/Documents/myEnv/dist/myenv-linux-amd64 /mnt/c/Users/worker/Documents/myEnv/.build/perf/memory-wsl-linux.json 获准运行通过。相同 Linux 制品 SHA256 B8FFB372065B16221B906E35263CB95428540B50F6D6D0FA3A095711E46A2114：version 10 次最大峰值 7796 KiB，help 7924 KiB，均低于 32 MiB。
- 这是 WSL Linux 单进程信息命令、tmpfs 临时制品的内存观测，不覆盖安装/用户子进程树、Windows/macOS、冷缓存或最终本地 SSD 延迟门禁。所有临时文件限脚本自己的 TemporaryDirectory，报告位于工作区 .build/perf/memory-wsl-linux.json，无用户环境修改。
- 本轮仅新增测量工具与证据，生产未改/未重建。后续需实际状态/run/no-op 与支持平台完整性能、共享资源和恢复门禁；完整目标保持进行中。

WSL 配置与后端文件边界验证（2026-09-08）：
- 上轮 Linux RSS 测量属于 progress。本轮交叉编译 .build/config-linux.test（go test -c -o .build/config-linux.test ./internal/config），WSL 执行 '-test.v' '-test.timeout=40s' 全部配置测试通过且无跳过：严格 YAML/JSON、锁读写、项目/profile 隔离、Python file 引用/归档/相关元数据/workspace 边界、用户存储及版本约束。使用 Linux 临时文件系统，非仅 Windows 兼容证明。
- go test -c -o .build/backend-linux.test ./internal/backend 通过。首次 WSL 默认 shell 将正则竖线解释为管道，两段被当命令且失败，没有把该次当作测试通过。改用 --exec 绕过 shell 后：wsl.exe -d Ubuntu --exec /mnt/c/Users/worker/Documents/myEnv/.build/backend-linux.test '-test.run=TestExtract|TestBackendProcessCancellation|TestDownloadChecksumAndCleanup' '-test.v' '-test.timeout=30s' 全部通过。
- 后端证据覆盖真实 Linux tar 普通文件/合法符号链接、越界/绝对/缺失链接/硬链接/重复/CRC 拒绝，ZIP 名称边界，HTTP fixture 摘要失败清理和真实测试子进程取消。没有官方制品下载或真实用户环境修改。今后含 shell 元字符的 WSL 参数优先 --exec。
- 本轮没有生产变更，不重复重建产品或已通过的状态/清理测试。实际 Linux Node/Python 安装、主管崩溃全树处理、共享资源引用/权限和完整平台性能发布门禁仍待完成，完整目标保持进行中。

WSL 上官方 Linux Node 制品兼容验证（2026-09-08）：
- 上轮配置/后端边界测试属于 progress。本轮尝试 TestNodeOfficialPrepare，立即因 runner.Platform 明确拒绝 WSL 失败（0.00 秒）。方案要求 WSL 分别识别，保留此产品限制，没有修改平台判断来通过测试。
- 提取 Node 实际准备测试公共实现，新增 TestNodeOfficialPrepareWSL：仍需 MYENV_TEST_PREPARE=1，明确检查 Linux amd64、Microsoft 内核标识及 glibc loader，输出兼容性限定，再显式选择官方 linux-amd64-glibc 制品。它是测试入口，不会让 CLI 隐式支持 WSL。
- Linux 交叉编译 .build/backend-linux.test 通过。wsl.exe -d Ubuntu --exec env MYENV_TEST_PREPARE=1 MYENV_TEST_ARTIFACTS=/mnt/c/Users/worker/Documents/myEnv/.build/node-linux-real /mnt/c/Users/worker/Documents/myEnv/.build/backend-linux.test '-test.run=^TestNodeOfficialPrepareWSL$' '-test.v' '-test.timeout=6m' 获准运行通过（133.31 秒）。官方解析/下载/摘要校验、tar 解包及真实 Node/npm 版本验证均完成：Node 22.23.2、npm 10.9.8。
- 制品保留 .build/node-linux-real/artifact-download-2211632472（56851233 字节），入口/官方 URL/SHA 等写 .build/node-linux-real/prepared.json；后续使用该记录，不重复下载。解包位于 v9fs 挂载工作区，耗时不是网络单独时间，也不是本地 SSD 安装性能门禁。
- 测试仅专用目录，未修改系统 Node/PATH/用户配置。该证据属于 WSL 对官方 Linux 制品的兼容性，不代表原生 Linux 首发平台或 WSL 产品支持已经验收。全部架构目标保持进行中。

保留 Linux Node 的 runner 执行合同（2026-09-08）：
- 上轮官方 Linux Node WSL 兼容准备属于 progress。本轮复用 .build/node-linux-real/prepared.json，未重复解析/下载/解包。
- wsl.exe -d Ubuntu --exec env MYENV_TEST_PREPARED_RECORD=/mnt/c/Users/worker/Documents/myEnv/.build/node-linux-real/prepared.json /mnt/c/Users/worker/Documents/myEnv/.build/runner-linux.test '-test.run=TestProcessNodeArguments|TestCancelRetainedNode' '-test.v' '-test.timeout=20s' 获准执行通过。
- TestProcessNodeArguments（0.39 秒）用真实 Node 验证空格/引号/中文/--json 参数不变，stdin 字节经 Node 写回 stderr，stdout JSON 保持，退出码 17 原样返回。TestCancelRetainedNode（0.26 秒）启动持续运行 Node，250 ms 上下文期限后返回非零非负退出码，调用方无普通执行错误。
- 这是 WSL 下 runner 对 Linux Node 的执行证据，不绕过或改变 CLI 对 WSL 的平台拒绝，不代表原生 Linux 整体支持或交互终端/完整主管崩溃子孙回收。测试仅临时脚本和自身子进程，无生产变更，未重建未改变的 runner 制品。
- 后续需补齐平台支持矩阵、共享资源与恢复、真实命令性能和发布验收；完整目标保持进行中。

真实 Linux Node 的 SIGTERM 转发（2026-09-08）：
- 上轮 Node 参数/stdio/取消属于 progress。本轮新增 Unix 专属 TestForwardSignalRetainedNode，父测试启动运行器主管 helper，主管使用真实保留 Node 注册 SIGTERM handler 并输出 ready，父测试随后仅向该主管发送 SIGTERM。
- Node handler 自行 exit(23)，要求主管转发信号并原样返回 23，stderr 空。与先前上下文强制取消不同，该场景验证子命令处理信号后的自选退出码，未用 shell 替代 Node。
- Linux 交叉编译 .build/runner-linux.test 通过。wsl.exe -d Ubuntu --exec env MYENV_TEST_PREPARED_RECORD=/mnt/c/Users/worker/Documents/myEnv/.build/node-linux-real/prepared.json /mnt/c/Users/worker/Documents/myEnv/.build/runner-linux.test '-test.run=^TestForwardSignalRetainedNode$' '-test.v' '-test.timeout=20s' 获准执行通过（0.52 秒）。只操作测试自己启动的进程，复用制品，无下载或用户配置修改。
- 本轮只新增测试，生产未改。该证据不覆盖信号注册前的启动窗口、交互终端、主管被 SIGKILL 后脱离子孙树或 macOS 原生行为；完整目标保持进行中。

Unix 子进程创建前的信号注册窗口（2026-09-08）：
- 上轮真实 Node SIGTERM 转发属于 progress。本轮发现 prepareSupervision 直到子进程启动后的 Start 才 signal.Notify，主管在创建窗口可能按默认动作退出。将注册移到 prepareSupervision，保持有界 8 项通道，Start 后转发排队信号；exec.Start 失败时 Close 也注销信号，不遗留通知注册。
- 新增 TestSignalBufferedBeforeChildStart：另设测试观察通道确保真实 SIGTERM 已送达当前测试进程，再启动 shell 子进程，要求其收到排队 SIGTERM 并返回 143，不能退化到超时 SIGKILL。只向测试自身发送信号，未触及其他进程。
- Linux 编译 runner-linux.test 后，WSL --exec 携带保留 Node 记录执行 '-test.run=TestSignalBufferedBeforeChildStart|TestForwardSignalRetainedNode|TestCancelProcessGroup' '-test.v' '-test.timeout=20s' 全部通过：新窗口用例 0.00 秒、真实 Node 0.36 秒、两个进程组取消各 0.30 秒。
- GOOS=linux/amd64 的 dist/myenv-linux-amd64 与 GOOS=darwin/arm64 的 dist/myenv-darwin-arm64 构建通过。Windows 平台代码未变，未重建；macOS 仅编译，尚无原生信号证据。
- 此修复覆盖 prepareSupervision 注册后的创建窗口，不涵盖整个 CLI 启动前段、终端交互或 SIGKILL 后全树回收。完整目标保持进行中。

发现 Unix 无继承管道子孙的提前返回缺陷（2026-09-08）：
- 上轮创建窗口信号修复属于 progress。本轮重新核对 T05 运行中对象不得误删要求：Unix supervision.Finish 为空，cmd.Wait 仅因继承输出管道的复制 goroutine 才偶然等待部分子孙；子孙关闭管道后会在仍使用环境时提前返回，CLI defer Release 随即释放租约。
- 新增 TestWaitDescendantWithoutInheritedPipes：shell 启动 0.3 秒后写 finished 的子孙，子孙 stdin/stdout/stderr 都指向 /dev/null，父 shell 立即退出。要求 Execute 返回时 finished 已存在。失败路径也等待短子孙自行结束再清理临时目录。
- Linux runner 测试制品编译通过；wsl.exe -d Ubuntu --exec /mnt/c/Users/worker/Documents/myEnv/.build/runner-linux.test '-test.run=TestWaitDescendantWithoutInheritedPipes' '-test.v' '-test.timeout=10s' 实际失败（0.31 秒）：Execute 已返回而 finished 尚不存在。测试保持失败，没有跳过或降低预期；这是新发现的生产缺陷，不是环境限制。
- 下一步必须实现 Unix 子孙生命周期证明及相应租约持有策略。只检查主管 PID 或只等待输出管道不满足要求；单纯 kill(-pgid,0) 还需处理僵尸、进程组脱离与 PID 重用，不能据此宣称完整安全。Linux 可用 WSL 做真实验证，macOS 仍需平台实现/证据。
- 当前 Unix runner 回归套件有上述明确失败项，产品未改/未重建，不能发布或标记 T05 完成。完整目标保持进行中。

Linux 独立 subreaper 基础验证（2026-09-08）：
- 上轮提前返回失败回归属于 progress，本轮不以 PID/进程组扫描代替子孙生命周期证明。核对 Linux man-pages PR_SET_CHILD_SUBREAPER：https://www.man7.org/linux/man-pages/man2/PR_SET_CHILD_SUBREAPER.2const.html ，确认孤儿重归最近存活 subreaper，可 wait 获取退出；该属性作用于进程，必须隔离于共享 CLI/并发后端执行。
- 新增 Linux reapDescendants，仅供专用主管进程：启用 PR_SET_CHILD_SUBREAPER 后启动唯一命令，Wait4(-1) 处理 EINTR，直到 ECHILD，并记录原始 leader 退出状态。调用约束明确禁止无关子进程、并发 exec.Cmd.Wait 和 pipe goroutine；暂未接入生产 Execute，不把原缺陷声称为已修复。
- TestDedicatedSubreaperDetachedDescendant 在独立测试主管进程调用该函数：shell 退出17，短子孙 setsid 脱离会话/组、stdin/out/err 指向 /dev/null，0.3 秒后写文件。要求返回前文件存在，最终仍为 leader 退出码17。
- Linux 编译 runner-linux.test 通过；wsl.exe -d Ubuntu --exec /mnt/c/Users/worker/Documents/myEnv/.build/runner-linux.test '-test.run=^TestDedicatedSubreaperDetachedDescendant$' '-test.v' '-test.timeout=10s' 获准通过（0.32 秒）。仅测试主管自身 prctl 和测试临时目录，无系统服务/用户环境修改。
- 下一步接入专用主管的受限启动协议、信号/取消、启动失败以及租约释放/主管异常死亡策略，再让 TestWaitDescendantWithoutInheritedPipes 通过。该既有回归仍是明确未修复失败项；未重建未使用新函数的产品，完整目标保持进行中。

独立 Linux subreaper 的取消与完成证明（2026-09-08）：
- 上轮独立 subreaper 基础验证属于 progress。本轮新增 reapDescendantsContext，返回原 leader 状态、Complete 布尔值和错误。正常等待用 SIGCHLD 唤醒/WNOHANG，不扫描系统进程；取消时只枚举主管各线程的直接子进程并 SIGKILL，再持续 reap 新接管的孤儿，直到 ECHILD 才 Complete=true。
- 核对 https://www.man7.org/linux/man-pages/man5/proc_tid_children.5.html：children 列表在并发退出时可能漏项，因此不把空列表当作完成。取消期 25 ms 有界等待后重试，最终仍以 Wait4/ECHILD 为准。线程条目上限4096、每个 children 文件上限1 MiB；同一专用主管、单一 wait goroutine，在发信号期间不 reap 这些直接子进程，避免其 PID 在该窗口重用。此函数仍禁止共享进程/无关子进程/并发 wait。
- 扩展独立测试主管：原正常 setsid 子孙等待保留退出码17；新增取消场景，确认脱离子孙确实已写 started，再验证300 ms期限取消后 Complete=true、DeadlineExceeded，并保留原 leader17。Linux 编译 runner-linux.test 后，wsl --exec 运行 '-test.run=^TestDedicatedSubreaper' '-test.v' '-test.timeout=10s' 通过（正常0.32秒、取消0.31秒）。
- 测试仅专用主管自己的子进程，WSL 执行获准。生产 Execute 尚未接入，原 TestWaitDescendantWithoutInheritedPipes 仍未修复；下一步主管启动协议、信号转发和父主管异常/租约生命周期。未重建未调用新函数的产品，完整目标保持进行中。

Linux 独立主管的受限启动协议（2026-09-08）：
- 上轮独立 reaper 取消证明属于 progress。本轮新增 launcher_linux.go，使用同一可执行文件的内部主管入口，fd3 请求/fd4 结果，用户 stdin/out/err 独立直通。请求上限1 MiB、响应读取上限64 KiB，ready 握手后才向专用主管发 SIGUSR1 取消，等待完整 Complete 结果及主管退出。
- 主管在执行用户程序前对协议 fd 设置 close-on-exec，避免子程序继承结果管道；输入要求绝对 executable。reap 完成回报原 leader 状态，取消和普通错误分开；异常 EOF、主管非零退出或 Complete=false 均不提供树结束证明。超大请求在启动前拒绝。
- 新增 TestLinuxSupervisorTransport：真实 shell stdin/stdout 字节直通、无协议污染、父命令 exit17 后仍等关闭管道的子孙完成；setsid 长子孙取消后得到 Complete=true/Canceled。TestLinuxSupervisorFailureProof 仅让测试 shell 终止自己的主管，要求 Complete=false/error；大请求则安全启动前拒绝。
- Linux runner 测试编译通过，wsl.exe -d Ubuntu --exec /mnt/c/Users/worker/Documents/myEnv/.build/runner-linux.test '-test.run=TestLinuxSupervisor' '-test.v' '-test.timeout=15s' 获准执行通过（Transport 0.33 秒、FailureProof 0.01 秒）。仅测试主管/子孙和临时目录，无用户配置修改。
- 尚未由生产 Execute 选择此启动器，原提前返回回归仍待修复；下一步正常信号转发、主管父死亡与 CLI 租约保留策略，再切换调用路径。未重建产品，完整目标保持进行中。

Linux 生产运行器接入独立主管（2026-09-08）：
- 在此前受限协议基础上，父运行器和专用主管在启动用户程序前注册 INT/TERM/HUP，父进程转交信号，专用 reaper 向其直接拥有的子进程转发。新增 TestLinuxSupervisorSignalNode，真实保留 Node 的 SIGTERM handler 返回23。切换 Linux Execute 到专用主管，非 Linux 继续原平台路径；未修改 WSL 产品支持限制。
- Execute 新增 ErrTreeUnconfirmed 标识，主管异常退出/无完整结束证明时返回。RunEnvironment 提供 RetainLease（只关闭数据库连接），CLI run 在该错误下保留租约，正常完成仍释放。Linux 死租约仍采用保守保留策略。该 CLI 分支已编译，尚缺完整 SelectRun→主管死亡→clean 的端到端验证。
- 上轮测试结果在上下文边界被截断，未视为通过。本轮获准用 wsl --exec ps 检查已无测试进程，再重跑受影响 runner 测试并保留 .build/linux-supervisor-validation.txt：TestCancelRetainedNode、TestLinuxSupervisorSignalNode/Transport/FailureProof、TestProcessNodeArguments、TestDedicatedSubreaperDetachedDescendant/Cancellation、TestForwardSignalRetainedNode、TestWaitDescendantWithoutInheritedPipes、TestCancelProcessGroup/AfterParentExit 全部通过。原提前返回回归现于 Linux 通过（0.31秒）；macOS 同类问题仍未解决。
- backend.executeBackend 使用 errors.Join 保留 runner 错误与 ctx.Err，避免取消掩盖进程树未确认。新增 TestBackendCancellationKeepsTreeUnconfirmed：shell 杀自己的主管后输出触发取消，要求同时 errors.Is(Canceled) 和 errors.Is(ErrTreeUnconfirmed)。重新交叉编译 backend/runner 测试后，WSL 执行该测试与 TestBackendProcessCancellation 通过（0.02/0.03秒）；新增 TestExecuteLinuxUnconfirmedTree 通过（0.01秒）。仅操作测试自身进程，无下载或用户环境修改。
- Windows go test ./internal/runner ./internal/backend -run 'TestBackendProcessCancellation|TestProcessArguments|TestExecute' -count=1：backend取消测试通过（0.299秒），runner未匹配测试，不记为Windows运行器新增覆盖。go build 三个 dist 制品 windows/amd64、linux/amd64、darwin/arm64 均退出0；macOS只编译。
- 尚有重要缺口：安装操作的 FailOperation/RecoverInterrupted 在主管异常后缺少持久树保护；父 CLI 死亡时的主管清理；交互终端重复信号、普通信号对子孙的完整转发；启动握手取消边界；macOS生命周期实现。不能因本轮 runner 回归通过而宣称 T05 完成或发布安全。现有性能结果对应旧制品，未对新主管额外进程开销给出门禁通过结论。完整目标继续进行，下一步优先安装操作异常后的持久清理保护与租约端到端验证。

准备操作的持久进程树保护（2026-09-08）：
- 上轮 Linux Execute 接入属于 progress。本轮检查发现 core.Sync 无条件 defer FailOperation，后续 RecoverInterrupted 也把所有 preparing 转为 failed；修改锁释放不能证明子孙结束，可能使仍使用准备目录的进程失去清理保护。
- 新增 operation_tree_holds 附表，无需重建 operations。BeginGuardedOperation 在同一 SQLite 事务中创建操作和保护记录，生产 Sync 在创建准备目录/启动本代后端前使用该入口。FailOperation 与 RecoverInterrupted 都排除有保护记录的操作，使它们保持 preparing，现有候选/删除预留规则继续保护相关目录。
- 确认全部已启动进程树结束的普通失败由 FailGuardedOperation 原子解除记录并转 failed；Sync 遇到 ErrTreeUnconfirmed 保留记录，panic/未完成返回也不解除。正常发布在同一活动引用事务中删除保护记录；失败 CAS 不丢失记录。退出处理失败使用 errors.Join 传播。旧无保护操作沿用旧恢复行为，历史操作缺少树证明的问题未由本轮追溯解决。
- 新增 TestGuardedOperationRecovery（关闭/重开数据库、普通失败/恢复不能越过保护、删除预留拒绝、确认结束后可清理）和 TestGuardedPublicationAtomicity（失败 CAS 保留/成功发布解除）。扩展 TestRecoveryKilledOwner：真实 OS 锁持有子进程同时提交普通与受保护记录后被 Kill；重新取得锁只恢复普通记录，受保护记录仍 preparing 且不可预留删除。
- Windows go test ./internal/state -count=1 全部通过（2.568秒）。Linux 交叉编译 .build/state-linux.test 后，WSL --exec '-test.v' '-test.timeout=40s' 全部16项通过，无跳过；真实 Kill 恢复0.17秒。测试只杀自己的 helper，不影响无关进程。
- 扩展 TestPythonSyncCancellationRetained 检查初次成功与后续真实取消后保护记录总数0、failed操作1，旧活动代不变。获准 Windows 复用 .build/python-real/python-prepared.json 执行 go test ./internal/core -run 'TestPythonSyncCancellationRetained|TestDownloadFailureRegistered' -count=1 -v，真实取消4.76秒、下载失败0.19秒均通过，无重复下载/用户环境写入。
- TestCleanFailedPreparation 改为真实持久保护记录：确认失败一项、未知一项，持锁 RecoverInterrupted 不解除未知项。Linux 编译 core-linux.test 后，WSL 单测通过（0.04秒）；预览与实际清理只处理确认失败目录，未知项 generations/operations 内 partial 文件均保留。
- Windows amd64、Linux amd64、Darwin arm64 三个 dist 产品构建通过。该保护目前保守持久保留，尚无可靠树结束证明后的自动回收；共享运行时准备/操作前期解析、历史记录、macOS完整监督、父主管死亡和终端信号仍待实现/验证。本轮不声称完整 Sync→主管死亡→clean 端到端或T05已完成，完整目标保持进行中。

Linux 调用进程死亡与握手前取消（2026-09-08）：
- 上轮持久操作保护属于 progress。本轮在 Linux 专用主管协议增加 fd5 生命周期管道，只有调用进程保留写端。主管对 fd3/4/5 设置 close-on-exec，子命令不继承协议端；父进程 SIGKILL/正常关闭写端使主管观察 EOF，取消自己的 reaper 上下文并等待全部拥有子孙结束。未用 PID 存活判断或父线程死亡信号替代归属证明。
- 上下文取消也关闭该写端，监视 goroutine 在主管 Start 后、读取 ready 前启动；不再依赖 ready 后 SIGUSR1。普通 INT/TERM/HUP 仍经原握手后转发路径。EOF 是持久可观察事件，早于主管初始化也不会丢失；不承诺取消与 child.Start 竞态中完全不创建子进程，而是最终取消并 reap。
- 新增 TestLinuxSupervisorParentDeath：测试调用 helper 使用生产 Execute 启动 shell 及 setsid 脱离会话的 sleep，等待两者 PID 文件且确认仍在，再 Kill 仅调用 helper。要求一秒内两个 /proc 条目均消失（含僵尸已被 reap），fixture 自身五秒到期，失败路径不以可能重用的 PID 强杀。WSL 实测0.02秒通过。
- 新增 TestLinuxSupervisorLifetimeClosedBeforeHandshake：在专用主管启动之前关闭生命周期写端，要求仍取得 Complete/Canceled 结果，WSL 0.01秒通过。该用例直接检验协议早期 EOF，不以短超时碰运气。
- 先接父死亡通道后，受影响 runner 测试全部通过；随后改上下文取消复用管道，重新编译 runner/backend Linux 测试制品。获准 WSL --exec 携带保留 Node 记录执行 '-test.run=TestLinuxSupervisor|TestCancelRetainedNode|TestCancelProcessGroup|TestForwardSignalRetainedNode' '-test.v' '-test.timeout=25s' 全部通过（含新两项、真实Node退出23、协议失败、脱离子孙取消）；backend '-test.run=TestBackendCancellationKeepsTreeUnconfirmed|TestBackendProcessCancellation' '-test.v' '-test.timeout=15s' 两项通过（0.01/0.02秒）。无下载或用户环境修改。
- GOOS=linux GOARCH=amd64 go build -o dist/myenv-linux-amd64 ./cmd/myenv 通过，最后仅新增测试未改生产。Windows/macOS生产代码未变，不重复构建。工作区仍无 .git。
- 此变更处理调用进程消失且专用主管仍可运行的情况，不覆盖主管自身也被杀、内核不可中断任务或交互终端完整信号传播。父死亡后结束证明无法返回，租约/操作持久记录仍保守保留；自动回收、共享资源与历史记录保护、macOS及完整发布门禁继续待完成。完整目标保持进行中。

主管异常→持久运行租约→实际清理保护（2026-09-08）：
- 上轮父死亡管道属于 progress。本轮把 CLI 内的租约保留判定集中到 core.RunEnvironment.Finish(runErr)，内部构造 leasedRunEnvironment 绑定真实 state.Lease 和数据库连接。ErrTreeUnconfirmed 仅关闭连接，其他结果释放租约后关闭；SelectRun 使用同一构造，CLI defer 调用 Finish。保留 Release 供未执行/既有内部调用使用，减少调用方重复实现异常判定。
- 新增 Linux TestRunFinishProtectsUnconfirmedTree 两个真实执行分支。隔离数据库发布4代，租约选择最老一代，另外两代作为 current/previous 对照。正常 shell exit17 后 Finish 释放租约，预览/实际 clean 删除两项无引用旧代。
- 异常分支 shell 启动 setsid 短子孙、关闭其标准管道并等待 started，再杀自己的专用主管。生产 runner 返回 ErrTreeUnconfirmed；Finish 后另开只读连接确认租约仍有1条；预览和实际 clean 仅删除另一个无引用旧代。0.5秒后短子孙从受保护代读取 payload 写 survived，要求内容确为 retained。测试失败也等待短 fixture 自行完成，不向可能重用的孤儿 PID 发信号。
- Linux 交叉编译 core-linux.test 后，获准 WSL --exec '-test.run=^TestRunFinishProtectsUnconfirmedTree$' '-test.v' '-test.timeout=15s' 通过：正常0.06秒、异常0.54秒。此证据贯通生产 runner、租约收尾和实际 clean；入口使用内部真实状态 fixture，未调用 WSL 被产品明确拒绝的 SelectRun，不声称整个 WSL CLI 已受支持。
- Windows 获准复用 .build/node-real/prepared.json 与 .build/python-real/python-prepared.json，go test ./internal/cli -run 'TestRunRetainedNode|TestRunRetainedPython|TestRunMissingEnvironmentExit' -count=1 -v 全通过：真实Node1.99秒、Python0.42秒、缺失环境0.00秒。仅临时项目，未下载/安装或修改用户环境。
- 三个 dist 产品 windows/amd64、linux/amd64、darwin/arm64 构建通过。自动回收仍缺持久可验证结束证明，主管本身异常时保守保留；交互终端、macOS、共享资源及完整T06门禁仍待完成。完整目标保持进行中。

Linux 子孙结束凭据通道基础（2026-09-08）：
- 上轮真实租约→clean 异常保护属于 progress。本轮检查当前 leaseReclaimable：Windows 检查进程出生身份和命名Job，Linux仍未安装回收判定。仅 PID 消失不证明子孙结束，继续保留现有保守策略，没有凭空开启自动删除。
- runner.Process 新增可选 TreeCompletion（独占、初始空普通文件和32字节小写hex绑定令牌）。Linux调用端启动前检查文件/令牌，专用主管通过 fd6 获取文件、内部请求获取令牌，fd6 close-on-exec，用户子进程不继承。reaper Complete=true后 WriteAt 写固定版本前缀+令牌并 Sync；父死亡后仍可完成写入。未获得树结束证明不写。调用者负责文件生命周期和启动前持久绑定，目前 core.SelectRun 尚未传该字段。
- 新增 TestLinuxSupervisorCompletionReceipt：正常 shell exit17且不能访问 /proc/self/fd/6，凭据内容精确匹配；再次使用非空凭据被启动前拒绝。另一个分支 shell Kill 自己主管，要求 Complete=false/error且文件仍空。
- 扩展 TestLinuxSupervisorParentDeath：调用helper创建凭据，父测试确认子孙存活时文件为空；Kill调用helper后要求 leader/setsid子孙均消失且凭据含正确令牌。WSL实测0.02秒，说明主管在失去父进程后仍写出结束证明；不等同于断电持久性测试。
- Linux交叉编译 runner-linux.test 通过。获准 WSL --exec 携带保留Node记录运行 '-test.run=TestLinuxSupervisor|TestCancelRetainedNode|TestForwardSignalRetainedNode' '-test.v' '-test.timeout=25s' 全通过：凭据正常/主管被杀各0.01秒、真实Node信号0.45/0.33秒、握手前EOF0.00秒、传输0.32秒，取消0.26秒。只测试自身进程与临时凭据，无下载或用户环境修改。
- 三平台产品 windows/amd64、linux/amd64、darwin/arm64 构建通过。凭据通道当前是未接入核心租约的基础能力，不能声称Linux自动回收已完成。下一步必须在启动前原子绑定租约和随机令牌、使用受限文件路径验证完整凭据，并结合原主管出生身份、CAS删除和凭据清理；主管本身异常/无凭据仍保守保留。完整目标保持进行中。

Linux 运行租约凭据绑定与自动回收（2026-09-08）：
- 上轮凭据通道基础属于 progress。本轮将其接入 SelectRun/CLI：Linux 验证所选代后，在工作目录 lease-receipts/<随机租约ID>.complete 独占创建空文件，crypto/rand 生成32字节令牌，state.BindLeaseCompletion 提交绑定后才返回可运行对象；CLI 把 Completion 传给 runner。其他平台沿原监督方式，不创建该文件。
- 新增 lease_completion 附表（租约外键级联），单条 INSERT SELECT 同时核对当前 PID 与出生身份；不能绑定他人/缺失租约或覆盖已有令牌。LeaseRecords 兼容旧数据库无附表，返回空令牌。DeleteObservedLease CAS 额外核对令牌，避免观察后新增绑定仍被旧观察删除。TestLeaseCompletionBindingCAS 验证单赋值、陈旧观察拒绝、正确观察删除及附表级联。
- Linux 清理验证租约ID、linux:bootUUID:startticks身份和令牌格式，先确认原主管已退出，再用 os.Root 锚定工作目录读取凭据；O_NOFOLLOW 拒绝叶符号链接，O_NONBLOCK 避免FIFO阻塞，fstat只接受普通文件，最多读取预期长度+1，要求版本前缀/完整令牌/末尾换行精确相同。旧无令牌、缺失/错误/未完成凭据不会使环境可回收。预览不修改；实际 CAS 移除租约后删除其凭据。正常 Finish 删除租约与凭据；未知树只关闭文件和连接，保留两者。
- 新增 TestCleanCompletedLinuxOrphan：独立调用helper绑定真实租约并用生产runner启动sleep，父测试确认活主管不可回收后Kill调用helper；专用主管结束子树并写凭据，清理预览识别1个可回收租约、2个无引用旧代，实际回收1条租约/删除2代/移除凭据，current/previous文件保留。空/截断/多余字节/错误令牌均拒绝；额外真实符号链接/FIFO替换验证不会被接受或阻塞。最终WSL通过0.08秒。
- TestRunFinishProtectsUnconfirmedTree 现在也创建绑定并传Completion：正常exit17的Finish移除凭据；主管自身被杀时空凭据与租约保留，实际clean仍跳过旧代，短子孙读文件成功。最终WSL正常0.03秒、异常0.53秒通过。
- Windows go test ./internal/state -count=1 全部通过（加入CAS测试后2.582秒）；Linux交叉编译state/core测试后，WSL state完整套件全部通过，无跳过，涵盖旧数据库、绑定CAS、真实锁持有进程Kill、清理/发布/恢复；core受影响CleanFailedPreparation、CleanRetainsAppliedAndLeased及两个新/扩展用例均通过。新增文件替换断言后仅重跑两项改变的core测试通过。
- Windows获准 go test ./internal/core -run '^TestCleanKilledWindowsSupervisor$' -count=1 -v 通过（0.43秒），确认共享CAS更改未破坏原Job回收。三平台dist windows/amd64、linux/amd64、darwin/arm64 构建通过。测试全在隔离目录/自身子进程，未下载或修改用户环境。
- 此证据贯通Linux内部租约配置、生产runner、父死亡与实际clean；WSL仍被产品平台入口拒绝，没有声称支持WSL CLI。自动回收只适用于带绑定且完整凭据的运行租约；准备操作tree hold、历史无凭据租约、主管自身异常后未结束子孙仍保守保留。凭据创建/删除崩溃窗口可能留下孤立元数据文件，后续需收敛；没有断电/全文件系统攻击模型或新run启动性能门禁证明。终端信号、macOS、共享资源及T06完整门禁继续待完成，完整目标保持进行中。

孤立 Linux 租约凭据清理（2026-09-08）：
- 上轮凭据绑定/回收属于 progress。本轮处理已删除租约但凭据 unlink 尚未完成时崩溃留下的文件。清理只处理专用 lease-receipts 目录下精确32位小写hex ID+.complete 的普通文件，不递归、不跟随叶链接；128条一批，累计100万条上限，每批/项检查上下文，不读取凭据正文。未知名称、目录、符号链接和FIFO保留。
- 新增 state.LeaseExists 供预览；实际 RemoveUnleasedReceipt 在同一 SQLite 写事务中通过无内容变化 UPDATE 取得写锁，确认该ID无租约后才执行单次 os.Root.Remove。写锁直到unlink完成才释放，避免仅查询不存在后与新登记竞态；没有数据库业务状态需要提交，回滚释放锁。回调约束不访问Store、不递归删除。仍有任何租约记录（包括无绑定/未知身份）的文件都不清理。
- core.Clean 在租约回收后清理孤立凭据，报告 orphaned_lease_receipt、实际字节、候选与删除计数。预览无修改；已删除一项后取消保留Changed/Removed，后续调用处理剩余项。其他平台此扫描为空，未扩展受管目录外范围。
- 新增 TestReceiptRemovalExcludesConcurrentLease，第二SQLite连接busy_timeout设为20ms，在unlink回调内插入同名租约必失败，释放写锁后完全相同插入成功；已有租约再次清理不得调用unlink。Windows通过0.16秒，交叉编译后WSL通过0.05秒。
- 新增 TestCleanOrphanedLeaseReceipts：3个真实孤立文件、1个仍登记租约文件、未知文件、真实symlink/FIFO；预览只列3项18字节，实际第一项后取消返回Canceled/Removed1/Changed，继续删除剩余2项12字节，所有受保护/未知对象均保留，最终预览候选0。WSL0.02秒通过。
- Linux core测试制品重新编译，获准WSL执行新测试及受影响 TestCleanCompletedLinuxOrphan（0.08秒）、TestRunFinishProtectsUnconfirmedTree（正常0.03秒/未知0.53秒），全部通过。三平台dist windows/amd64、linux/amd64、darwin/arm64 构建通过。测试只专用目录/自身进程，无下载或用户环境修改。
- 范围明确：若创建凭据后绑定前崩溃但数据库租约仍存在，本轮仍保守保留；不把“无绑定”误当无子孙证明。准备操作自动回收、终端信号、macOS、共享资源及完整性能/发布门禁继续待完成，完整目标保持进行中。

Linux 非终端进程组重复信号回归（2026-09-08）：
- 上轮孤立凭据清理属于 progress。本轮检查新Linux主管发现遗漏旧 prepareSupervision 的非终端 Setpgid 隔离，调用者、专用主管和用户命令都在同一组。组广播信号会直接到用户命令，还会分别被两级主管重复转发。
- 提取真实Node信号测试公共路径，新增 TestLinuxGroupSignalRetainedNode：只给测试调用helper创建独立进程组，待Node ready后仅向该组发一次SIGTERM；Node在100ms窗口计数，恰好一次返回23，多次返回90+计数。输入为非终端，测试不冒充真实PTY/Ctrl-C验证。
- 修复前Linux编译通过，获准WSL执行 '-test.run=^TestLinuxGroupSignalRetainedNode$' '-test.count=5' '-test.v' '-test.timeout=15s' 五次全部失败，均exit93，证明一次组信号被处理三次。仅对自身测试组发信号，无其他进程影响。
- executeLinuxSupervisor 现在在非终端stdin时为专用主管设置 Setpgid=true，与调用者组隔离，由调用者转发一次。stdin为TTY时保留前台组，避免直接分组导致SIGTTIN；该分支仍需明确终端前台交接和恢复，尚未修复其重复信号问题。
- 重新编译runner-linux.test并构建dist/myenv-linux-amd64后，完全相同5次回归全部通过（0.48–0.63秒）。随后WSL受影响 TestLinuxSupervisor*、TestForwardSignalRetainedNode、TestCancelProcessGroup/AfterParentExit、TestCancelRetainedNode、TestProcessNodeArguments 全通过：真实Node参数0.39秒、直接信号0.41秒、父死亡0.02秒、取消各0.26/0.30秒，凭据正常/主管死亡均0.01秒。复用保留LinuxNode，无下载或用户环境修改。
- 生产改动仅Linux，未重复构建Windows/macOS；最后只澄清代码注释，制品行为与最终生产逻辑相同。交互TTY/完整作业控制、准备操作回收、macOS、共享资源与发布性能门禁仍待完成，完整目标保持进行中。

Linux 真实终端前台交接与 Ctrl-C（2026-09-08）：
- 上轮非终端组信号修复属于 progress。本轮核对本机 Go1.26.6 syscall/exec_linux.go 的 SysProcAttr.Foreground/Ctty：Foreground隐含Setpgid，子进程在exec前TIOCSPGRP，并在该调用后恢复信号mask以避免SIGTTOU。核对 Linux man-pages https://man7.org/linux/man-pages/man3/tcsetpgrp.3.html：设置前台组要求相同控制终端/会话，后台调用需忽略或阻塞SIGTTOU。
- 新增 TestLinuxTerminalInputAndInterrupt，Python pty.fork 创建真实控制终端/会话，仅测试依赖Python、不引入产品依赖。Go调用helper通过生产Execute运行保留Node，Node读取hello后输出input-ok，PTY写入真实Ctrl-C字节0x03，Node100ms内计数，要求恰好一次返回23；调用者验证TIOCGPGRP恢复为自身组，再读取again并退出23。PTY输出限制64KiB、阶段超时4秒，失败仅Kill自身helper。
- 修复前该WSL回归明确失败（0.85秒）：PTY已完成hello/input-ok，Ctrl-C后Node exit93，证明真实终端也重复三次；未把该失败归为环境问题。
- Linux请求增加TerminalGroup。调用方stdin为TTY时要求自己拥有前台，记录组ID；专用主管独立Setpgid，不接收终端对用户前台组的广播。主管启动用户命令用Foreground=true/Ctty=0，新用户组获得前台。reaper返回后恢复保存的调用者组；若终端已被shell等切到其他组则不覆盖。恢复时专用主管忽略SIGTTOU，此后不再启动用户程序，不改变用户子进程的继承信号处置。
- 扩展同一PTY用例：正常Node退出后、缺失可执行文件启动失败后、100ms上下文取消sleep后均检查前台恢复，最终调用者还能读取终端。对于Start在fork交接后exec失败且Process=nil的场景，也恢复原调用者组。
- 重新编译runner-linux.test后，获准WSL初次修复测试通过：真实PTY0.48秒、非终端组信号0.48秒、直接转发0.35秒、父死亡0.02秒。增加启动失败/取消断言后，最终PTY通过0.63秒；受影响 TestLinuxSupervisor*、TestCancelProcessGroup/AfterParentExit、TestCancelRetainedNode、TestProcessNodeArguments 全通过，复用Node无下载/用户环境修改。dist/myenv-linux-amd64 构建通过；非Linux生产未变，未重复构建。
- 当前stdin为TTY而调用者不在前台时明确返回错误，尚未实现后台作业接入、Ctrl-Z/SIGCONT暂停恢复及所有主管异常/外部前台切换竞态。WSL PTY证据不等于原生Linux全终端或macOS/Windows支持验收。准备操作回收、共享资源和完整T06门禁仍待完成，完整目标保持进行中。

Linux run 组件开销观测（2026-09-08）：
- 上轮PTY输入/Ctrl-C恢复属于 progress。本轮按方案第10节区分完整run门禁与组件诊断：完整门禁要求固定发布构建、受支持本地SSD、相同用户命令基线，包含项目检查/活动读取/租约登记，p95≤50ms。以下没有绕过WSL产品平台拒绝，不声称完整门禁通过。
- 新增门控 TestMeasureLinuxRunComponents，MYENV_TEST_RUN_TIMINGS 指定专用新报告路径才运行。直接 /bin/true、独立runner主管、state.Open/AcquireActive+真实凭据绑定+runner+Finish 三种方法各51次、每轮轮换顺序、首样本另列、剩余50样本按nearest-rank给median/p95，同时保存每轮与直接执行基线之差。使用真实SQLite/进程/凭据Sync，无模拟数据；不含CLI进程启动、配置/锁检查、PATH查找和完整RSS。
- Linux交叉编译core-linux.test通过，获准WSL运行门控用例（1.40秒）写 .build/perf/run-components-linux-v9fs.json。测试制品SHA256 7f639342e9ba751cdae80af6572e0e8b9d6603cdaddf75ff45cdfa318c94978d，内核6.18.33.2-microsoft-standard-WSL2，制品FS 0x1021997(v9fs)，临时状态FS 0x1021994(tmpfs)。直接执行p95 1.9209ms，主管12.5321ms，租约+主管14.1016ms；成对附加开销p95分别10.538201/13.0402ms，租约+主管附加中位10.4086ms。
- 新增 scripts/measure-run-components-linux.py，在脚本自己的TemporaryDirectory复制同一测试二进制，运行同一门控测试，不覆盖已有报告。获准WSL执行通过（测试1.21秒），写 .build/perf/run-components-linux-temporary.json；SHA完全相同，制品/状态均tmpfs。直接执行p95 2.0273ms，主管12.8120ms，租约+主管17.3982ms；成对附加p95 11.2408/15.5828ms，租约+主管附加中位10.0711ms。
- 两次有限观测没有显示临时复制带来稳定尾延迟改善，不据此归因为挂载加载瓶颈或做对象池/缓存重构。tmpfs不是本地SSD，测试二进制不是最终发布CLI，首样本不是冷缓存证明。完整样本/方法/内核/文件系统/哈希保留供后续对照。需继续采集完整CLI路径、参考平台SSD、冷缓存和myEnv自身/整树峰值RSS。
- 本轮只新增测试测量与脚本，生产未变，不重复构建产品或运行不受影响的功能套件。全部临时状态专用目录、报告在工作区，无下载/用户环境修改。准备操作回收、后台作业/暂停恢复、macOS、共享资源与完整发布门禁仍待完成，完整目标保持进行中。

Windows 完整 run 性能失败与分段诊断（2026-09-08）：
- 上轮Linux组件观测属于 progress。本轮提取已有保留Node运行fixture构造，新增门控 TestMeasureFullRunCLI。使用实际dist/myenv-windows-amd64.exe、隔离myenv.yaml/应用快照/锁/SQLite、保留官方Node22.23.2，实际完整 `-C <fixture> run node -e ''` 与同一Node直接 `-e ''` 比较，另列进程内ExecuteContext；三种各51次轮换、首样本另列、50样本nearest-rank和成对差值，原始数据完整保存，不下载/安装或改真实项目。
- 获准运行 go test ./internal/cli -run '^TestMeasureFullRunCLI$' -count=1 -v -timeout=3m 完成（47.45秒），生成 .build/perf/run-full-cli-windows.json。测量测试成功不等于性能验收通过：完整CLI成对附加中位226.0114ms、p95 321.1844ms，进程内CLI附加中位149.0656ms、p95 237.58ms，明确未达方案50ms预算。直接Node中位168.2837ms/p95 242.1733ms；完整CLI总中位392.8675ms/p95 498.188ms。
- 制品SHA256 4f05d9d958c0d8bbbc0624df7047849031cbcf46f1f147e773f35e4528221781。只读宿主信息记录 .build/perf/run-full-cli-windows-host.json：Windows11专业工作站版10.0.26200，i7-10850H/12逻辑处理器，约16GiB物理内存，C盘NTFS Fixed，存储设备报告KIOXIA NVMe SSD（BusType RAID）。这些是本宿主观测，未控制其他负载/清除冷缓存，不将首样本声称冷启动。myEnv自身/子树RSS尚未测。
- 为定位失败新增 TestMeasureRunStages（SelectRun、runner含用户Node、Finish各51次），第一组未传stdio时选择中位14.3486ms/p95 17.8567ms、执行168.0472/217.4707ms、收尾8.415/10.3268ms；记录 .build/perf/run-stages-windows.json。注意该组stdio与CLI不同，不用它支持管道瓶颈归因。
- 随后把分段stdin/stdout/stderr改为与进程内CLI相同的空reader/discard/buffer，重新运行改变的测量（10.35秒），记录 .build/perf/run-stages-windows-pipes.json：选择中位14.1152ms/p95 18.6886ms，执行174.4762/203.1916ms，收尾8.0993/9.8933ms。未观察到stdio统一导致显著变化；分段结果尚不能解释完整测量全部差额，不能把各阶段p95相加或根据不同时间段测量直接归因。
- 下一步应采集执行追踪/系统调用等待证据，再决定是否修改启动/监督/状态路径；未降低预算或进行无证据优化。本轮生产未改、不重复构建。性能失败仍开放，Linux完整命令、macOS、准备操作回收、共享资源及完整发布门禁待完成，完整目标保持进行中。

Windows run 追踪定位与线程枚举优化（2026-09-08）：
- 上轮完整run性能失败属于 progress。本轮首个 -trace 相对未加引号参数执行了测量但没有生成trace文件，报告 run-full-cli-windows-trace.json 只能当计时数据，不能作为追踪证据。发现Go安装缺少trace/pprof工具，从本机Go1.26.6源码 go build -o .build/trace.exe cmd/trace 与对应cmd/pprof成功，无下载/新增产品依赖。
- 用带引号绝对路径的短用例确认trace-smoke.out存在（26744字节）；随后相同完整测量传带引号绝对 -trace 路径成功（29.16秒），.build/perf/run-cli-windows.trace 6959180字节，另存 run-full-cli-windows-trace-verified.json。带追踪数据只作诊断，不替换无追踪性能验收。首次pprof未加引号的focus被PowerShell拆分，重新引用完整focus后正确过滤ExecuteContext。
- .build/trace.exe -pprof=syscall 生成 run-cli-windows-syscall.pprof，pprof -top -cum '-focus=myenv/internal/cli.ExecuteContext' 摘要存 run-cli-windows-syscall-summary.txt。51次进程内CLI中resumeInitialThread累计系统调用等待约4.3426秒，其中Thread32Next约3.8319秒；当前实现为每个挂起子进程扫描全系统线程，定位到与宿主线程数量相关的明显开销。该累计值不是用户命令耗时或整套运行墙钟百分比。
- 核对微软官方 PssCaptureSnapshot/PssWalkSnapshot/PSS_THREAD_ENTRY/PSS_CAPTURE_FLAGS/PssFreeSnapshot/PssWalkMarkerCreate/Free 文档：https://learn.microsoft.com/en-us/windows/win32/api/processsnapshot/nf-processsnapshot-psscapturesnapshot 、https://learn.microsoft.com/en-us/windows/win32/api/processsnapshot/ns-processsnapshot-pss_thread_entry 、https://learn.microsoft.com/en-us/windows/win32/api/processsnapshot/ne-processsnapshot-pss_capture_flags 。接口Windows8.1起提供；本轮只用PSS_CAPTURE_THREADS(0x80)，不捕获内存页、上下文或无关进程。
- 新增 thread_snapshot_windows.go，替换Toolhelp全局遍历：使用已有子进程句柄（增加QUERY_INFORMATION权限）捕获该挂起进程的线程ID，要求唯一初始线程且归属PID匹配，加入Job的原顺序保持，ResumeThread要求原挂起计数1。PSS返回值按DWORD错误码处理，快照/遍历标记用专用Free接口回收并传播错误；失败继续沿原路径终止测试/用户自己启动的子进程，不启动无保护程序。没有引入第三方依赖或改变宿主权限。
- Windows runner完整测试获准通过（3.028秒），复用真实Node，涵盖参数、取消、子孙等待/父退出后取消、命名Job与主管崩溃。新增 TestSnapshotThreadEntryABI 确认amd64结构120字节、关键offset匹配；单测通过。dist/myenv-windows-amd64.exe重建成功，SHA256 cc44bde13df7570a62ab515711c7a952d8c80c79d2466eb845162e3a39ab8865；非Windows生产未变，不重复构建。
- 无trace重测完整51次三路对照成功（23.01秒），保存 run-full-cli-windows-process-snapshot.json：完整CLI成对附加中位64.7757ms/p95 136.5865ms；进程内CLI22.1678/50.7585ms。完整CLI总p95 250.1297ms，直接Node177.9777ms。较此前改善，但宿主负载/基线本身有变化，不把全部差值归因于单一修改；已实际移除全系统O(线程数)枚举。完整CLI仍未达到50ms门禁，性能失败保持开放。
- 受影响实际集成获准通过：CLI保留Python0.54秒、Node2.24秒；core Windows死主管回收0.62秒、真实Python同步取消3.41秒。所有用例为专用临时目录和保留制品，自身进程，无下载/用户环境修改。
- 下一步继续分析剩余可执行文件启动/状态落盘开销及最终发布构建，不能因进程内接近预算而宣称完整门禁通过。准备操作回收、暂停/后台作业、macOS、共享资源及其他T06验收仍待完成，完整目标保持进行中。

Windows 精简构建对照与固定构建脚本（2026-09-08）：
- 上轮PSS线程恢复优化属于 progress。本轮核对本机go help build/go tool link -h，使用-trimpath移除源目录路径、-s/-w移除符号表/DWARF，构建 .build/perf/myenv-windows-release.exe。当前调试制品18590208字节，对照12698112字节，未改业务逻辑或降低状态持久性/进程监督要求。
- 计时测试增加 MYENV_TEST_RUN_CLI_ONLY=1，只测改变的外部制品和直接Node基线，不重跑未改变的进程内CLI路径。获准Windows复用官方Node和隔离fixture运行51次对照，TestMeasureFullRunCLI完成（12.57秒），报告 .build/perf/run-full-cli-windows-stripped.json：完整run附加中位60.8789ms/p95 78.465ms，制品SHA256 cd9cd799774f393dc19ba6c0245adeac072121b82bdcd56ef5150c035d97dcb5。较前一次尾延迟更低，但不是同一时刻严格A/B，不能将全部差异归因于调试信息；仍明确未达50ms门禁。
- 新增 scripts/build-release.ps1 固定本地构建配方：CGO_ENABLED=0、-trimpath、-buildvcs=false、-s -w及main.version；默认三个目标，可选子集，默认0.1.0-dev，输出独立dist/release，不替换已有调试制品。校验版本参数，逐目标检查退出码，最后生成build-manifest.json/SHA256SUMS，恢复GOOS/GOARCH/CGO_ENABLED，不发布/安装/改PATH。
- 实际执行 ./scripts/build-release.ps1 -Targets windows-amd64 成功：dist/release/myenv-windows-amd64.exe 12698112字节，SHA256 c25848043b16b9a06a98da85651af61725a019f3192f1a0a614ec84c24750e43，--version输出0.1.0-dev。manifest记录go1.26.6 windows/amd64和完整参数。它与前述实验制品的构建元数据/摘要不同，不把实验性能报告直接当该制品门禁通过。脚本的Linux/macOS目标本轮未执行；未声称跨平台精简制品已验证或位级可复现已验收。
- 新增docs/build.md记录使用方式、产物和摘要/签名区别。没有依赖更新、网络下载或用户环境修改。下一步继续分析Windows状态落盘/启动剩余开销并测明确最终候选制品，完整性能、准备操作回收、后台作业、macOS、共享资源及其他发布验收仍待完成，完整目标保持进行中。

Run 选择阶段去除重复读取（2026-09-08）：
- SelectRun 对一次选择只调用一次 runner.Platform、读取一次已应用 myenv.lock，供期望锁对比与已应用快照健康检查共用。活动代租约、声明摘要、完成标记、快照摘要、工具入口和进程树完成协议均保留；没有改 SQLite schema 初始化或 synchronous(FULL)。generation 发布后作为不可变对象使用，重复读取不构成原子防篡改保证。
- 已读取 AGENTS.md、相关架构和前轮证据；git status --short 确认仍无 Git 仓库，无法提供 git diff。gofmt 和 go test -c -o .build/cli.test.exe ./internal/cli 成功。
- 受影响 Windows 集成测试获准使用保留制品和隔离项目：TestRunRetainedPython 0.29秒、TestRunMissingEnvironmentExit 0.01秒、TestRunRetainedNode 1.39秒全部通过，含既有漂移/current 检查。首次启动因未引用 -test.v 被 PowerShell 拆分而未运行测试，引用参数后重试成功。
- 本轮未重测性能，不将减少一次读取等同于达到50ms门禁；最终发布候选需重新构建并测量。数据库迁移路径保持，准备操作回收、其他平台与完整首版验收仍待推进。

固定 Windows 制品的完整 run / 启动计时（2026-09-08）：
- 上轮为 progress（生产去重与真实后端验证）。本轮检查当前构建脚本、计时实现、架构第10节及实现记录；git status --short 仍确认无仓库。
- 使用工作区 GOPATH/GOCACHE/TMP，执行 ./scripts/build-release.ps1 -Targets windows-amd64 成功。dist/release/myenv-windows-amd64.exe 为12698112字节，SHA256 67cb737e0a42d471798f7d2b78cf0cc59718e74b73e0a7c2bc5bcd03966fd898；manifest及SHA256SUMS已同步，--version实际输出0.1.0-dev。
- 首次 TestMeasureFullRunCLI 在默认 AppData/Local/Temp 下外部CLI返回 attempt to write a readonly database (8)，未形成报告。随后显式设 TMP/TEMP=.build/tmp，用相同制品/保留Node完成原51次三路测试（17.47秒）。这是目录改变后的成功，不是默认Temp问题已定位或修复；后者原因仍未证实。
- 有效报告 .build/perf/run-full-cli-windows-single-lock.json 绑定上述SHA：完整CLI成对附加中位59.4216ms、p95 73.8679ms，进程内中位24.4148ms、p95 33.0176ms。完整run的50ms门禁仍失败；进程内数字不能替代完整产品验收，也不能从不同时刻的尾延迟证明小改动收益。
- 实际执行 ./scripts/measure-startup.ps1 -Executable ./dist/release/myenv-windows-amd64.exe -Output ./.build/perf/startup-release-single-lock-windows.json。50个热样本（首个排除）版本中位53.2112ms/p95 84.5437ms；帮助53.6597/75.2774ms，均未达30ms。首次观测分别69.4644/61.2118ms，不是驱逐缓存后的冷启动。该轮在默认沙箱执行，完整run使用获准执行上下文，不能把不同测量路径直接相减。
- 使用GODEBUG=inittrace=1运行同制品--version进行独立诊断，控制台首轮net包初始化约12ms；另存 .build/perf/startup-release-single-lock-inittrace.txt 的诊断运行整体包初始化约19ms。诊断带stderr输出且非统计测量，不能解释全部启动差额；没有改Go标准库、跳过初始化或削弱状态写入。下一步需分离进程启动/宿主上下文与程序内部成本，并继续准备操作恢复、原生平台及RSS等开放验收。

启动执行上下文对照与 run 状态库错误分类（2026-09-08）：
- 上轮为 progress（固定制品性能证据）。本轮重新读取AGENTS.md、当前脚本与记录；git status仍无仓库。对同一SHA 67cb737e0a42d471798f7d2b78cf0cc59718e74b73e0a7c2bc5bcd03966fd898，获准执行 scripts/measure-startup.ps1，输出 .build/perf/startup-release-single-lock-windows-approved.json：版本中位53.8421ms/p95 84.1803ms，帮助55.364/77.9304ms。与上轮默认沙箱测量接近，暂无证据认为沙箱造成主要延迟；仍未达到30ms预算。这不是严格同时A/B，不扩展为普遍性能结论。
- 根据实际数据库只读故障诊断，检查架构退出码合同（执行/环境错误1，用法/配置错误2）。SelectRun原来直接返回SQLite打开/初始化错误，CLI无法识别而默认USAGE_ERROR/2。现在将state.Open错误包装为带数据库路径的os.PathError，复用现有IO_ERROR/1映射，并保留底层错误链；不改声明错误、ENV_NOT_READY或子进程退出状态。
- 新增TestRunCorruptStateExit，使用隔离临时目录和真实损坏SQLite文件，验证启动前IO_ERROR/1、诊断包含数据库路径、stdout为空且损坏文件保持原内容。无需Node安装或启动。gofmt后执行 go test ./internal/cli '-run=^TestRun(CorruptStateExit|MissingEnvironmentExit)$' -count=1 通过（0.175秒）。此前正常Node/Python执行成功路径未变，不重复全量集成。
- 本轮修复的是错误分类，不声称默认Temp只读根因已解决。dist/release仍为修复前已测量制品；错误路径修改尚未重建发布制品。Windows延迟/RSS、原生其他平台和准备操作恢复等完整验收仍开放。

Run 租约登记失败分类与清理错误保留（2026-09-08）：
- 上轮为 progress（数据库打开错误分类及回归证据）。检查当前run、租约状态与架构退出码合同后，确认租约收尾已有stderr诊断且子进程退出码按合同透传，本轮未改变这一行为。
- state.AcquireActive对缺少活动代提供ErrNoAppliedGeneration哨兵，保留原ENV_NOT_READY文字；SelectRun只对其余租约登记失败包装带数据库路径的os.PathError，使其进入IO_ERROR/1而不是USAGE_ERROR/2。配置校验失败后的租约释放错误通过errors.Join保留，不再静默丢弃。
- 新增TestRunLeaseFailureExit：空活动引用仍报ENV_NOT_READY/1；隔离SQLite触发器拒绝真实lease INSERT时报IO_ERROR/1、保留底层错误，且无残留租约。测试在选择活动代阶段失败，不会运行占位的Node路径，不冒充真实后端集成。state既有租约测试同时断言哨兵错误。
- 首次联合测试state通过（0.291秒）；新增CLI fixture因缺少NodeExecutable违反已有Publish前置条件失败，补全记录字段后CLI三个相关测试通过（0.465秒）。未放宽生产校验。实际命令：go test ./internal/state ./internal/cli '-run=^(TestLeaseRetainsSelectedGeneration|TestRunLeaseFailureExit|TestRunCorruptStateExit|TestRunMissingEnvironmentExit)$' -count=1；修正fixture后仅重跑受影响CLI子集。
- 性能和完整T00–T06验收仍未完成；本轮未重新构建发布制品或重复性能测试。

源码入口与用户操作说明（2026-09-08）：
- 上轮为 progress（租约错误处理和测试）。本轮读取现有build文档、离线manual与实际命令定义，确认此前不存在README.md。
- 新增README.md，覆盖固定配方Windows构建、绝对路径调用、显式安装/PATH、最小Node+Python声明、init/sync/run、锁定自动化、use/doctor/rollback/current、profile与补全、清理和仅卸载命令的步骤。明确项目数据/声明/锁不在回滚范围、共享解释器不能盲删、WSL仍拒绝、原生Linux/macOS和性能门禁未验收。
- 对照当前root.go/run.go/clean.go/help.go核对参数和行为，检查四个本地文档链接存在。仅新增文档，没有执行示例中的安装、同步、清理或卸载，也没有把文档示例当新增集成验证。构建/真实后端行为复用既有记录。
- 完整数据卸载、共享资源引用、原生平台与性能验收仍开放，README不替代这些实现。

准备子进程结束后的崩溃恢复窗口（2026-09-08）：
- 上轮为 progress（README交付）。本轮读取AGENTS.md、Sync后端/发布顺序、状态保护与既有崩溃测试，发现所有后端已成功返回后的快照/内容摘要/发布阶段仍持有不可恢复tree hold。
- 新增state.ConfirmOperationTreesDone，要求准备中的当前owner记录匹配且恰好删除一个hold；缺失、其他owner或重复确认均失败。调用者必须持工作区修改锁，确认全部子进程树结束，并保证不再启动后端。操作仍为preparing，正常clean不会因移除hold就把它当失败候选。
- Sync在所有Node/Python后端调用成功完成后、纯本地校验与发布之前，持久化确认。仅Windows amd64和Linux amd64 glibc路径启用；macOS后代完成保证尚不充分，保留hold。正常失败仍由既有FailGuardedOperation收尾；运行中/未知进程树的崩溃仍保守保护，未宣称这部分自动恢复已实现。
- 新增owner/重复确认/活跃preparing不可清理状态测试；扩展TestRecoveryKilledOwner真实子进程持锁/被杀测试，确认settled操作在重新获锁后恢复为failed，uncertain仍preparing且不可预留删除，活动代保持。go test ./internal/state '-run=^(TestConfirmOperationTreesDoneOwnership|TestGuardedOperationRecovery|TestGuardedPublicationAtomicity|TestRecoveryKilledOwner)$' -count=1通过（0.865秒）。
- 重编译core测试，获准在工作区隔离临时项目、保留官方uv/Python上执行TestSyncRetainedPython（5.43秒）与TestPythonSyncCancellationRetained（2.88秒），均通过。没有下载或修改用户环境。新增确认阶段的原生Linux/macOS端到端未运行，发布制品未重建。
- 已缩小准备操作永久保留窗口；后台仍存活时的receipt恢复、共享运行时引用与性能/平台门禁继续开放。

Clean直接恢复明确完成的准备操作（2026-09-08）：
- 上轮为progress（后端结束后的恢复边界）。检查Clean发现它不调用sync的恢复流程；不能简单把历史无hold记录都当安全对象。本轮新增operation_tree_completed表，ConfirmOperationTreesDone在同一事务中删除hold并插入明确完成记录，失败回滚恢复保护。
- Clean实际执行先持工作区修改锁，只把带完成标记且无hold的preparing记录改为failed，再沿既有有界候选/删除预留/目录检查执行。结果新增recovered_preparations，恢复造成持久变化时Changed为true。未知hold和无标记历史preparing不新增恢复权限。
- 只读预览增加有界PreviewPreparationCandidates，旧数据库缺少完成表时退回原查询，不初始化schema；带标记的候选允许预览，真正清理重新获得锁和核对状态，预览期间若原owner发布则不会删除已发布代。README和clean命令帮助说明预览性质。
- TestCleanConfirmedPreparation覆盖三种记录：明确完成、未知hold、历史无标记。预览只列第一项且不改准备状态；实际恢复1项并删除，其他目录保留。相关core测试通过0.547秒；state四个针对性测试通过0.915秒，因schema/事务变化再运行state完整测试通过3.041秒，含既有只读和崩溃验证。
- 本轮未运行新的原生Linux/macOS或后端集成、未构建发布制品。仍无Git仓库。完整目标保持进行中，进程仍存活/主管同时丢失时的准备恢复和其他平台、性能验收仍开放。

准备恢复的并发与旧状态预览验证（2026-09-08）：
- 上轮为progress（clean完成标记恢复路径）。本轮对当前实现补证，未改变生产语义或重复无关后端测试。
- TestCleanConfirmedPreparation增加真实工作区锁竞争：原操作持锁时实际Clean以100ms截止返回DeadlineExceeded，Changed=false且RecoveredPreparations=0；释放锁后既有预览/清理继续成功。证明持有完成标记不允许绕过仍活跃的修改者。
- 新增TestCompletedPreparationPublishedAfterPreview：先取得完成准备候选，随后发布同一代，再尝试恢复及使用陈旧候选预留删除，均不得触碰发布对象；活动引用一致。另验证after游标不会重复返回完成候选。Windows state/core相关测试分别通过0.285/0.458秒。
- 新增TestPreparationPreviewWithoutCompletionTable，专用SQLite数据库移除新增表模拟旧schema，通过OpenReadOnly预览仅列failed，保留历史preparing，查询确认未重建完成表。针对性测试通过0.281秒。该测试在隔离数据库中执行DROP TABLE，不操作用户数据库。
- 实际运行 go test ./internal/state ./internal/core '-run=^(TestCompletedPreparationPublishedAfterPreview|TestCleanConfirmedPreparation)$' -count=1，以及 go test ./internal/state '-run=^TestPreparationPreviewWithoutCompletionTable$' -count=1；均通过。本轮仅新增/扩展测试，未重建发布制品或声称原生其他平台已验证。完整验收仍开放。

统一sync与clean的历史准备恢复权限（2026-09-08）：
- 上轮为progress（并发/兼容性证据）。本轮发现Sync调用的RecoverInterrupted仍将所有无hold的preparing标failed，与Clean新增的显式完成证据规则不一致；OS锁释放不能证明旧主管的子孙退出。
- RecoverInterrupted现复用RecoverCompletedPreparations，只恢复明确完成且无hold的操作。保留函数入口，避免维护两份恢复SQL。历史无标记记录继续preparing；没有通过修改schema给历史记录补造完成证明。
- 相应测试按安全合同修正：原RecoverInterrupted测试加入实际确认的settled记录，保持恢复计数1，并新增历史interrupted仍preparing断言；真实KilledOwner测试只允许settled恢复，旧无标记interrupted和uncertain均保留。此前允许历史无hold恢复的测试预期反映旧实现，本次收紧是修复删除权限，不是放宽测试以通过。
- Windows针对性state/core测试通过0.813/0.685秒，命令 go test ./internal/state ./internal/core '-run=^(TestRecoverInterruptedKeepsActive|TestRecoveryKilledOwner|TestGuardedOperationRecovery|TestCleanConfirmedPreparation|TestCleanFailedPreparation)$' -count=1。重新构建core测试并获准复用保留uv/Python执行TestSyncRetainedPython，通过5.68秒，覆盖正常/无变更/失败保留及回滚。
- 无下载或真实用户环境修改；发布制品未重建。旧无证据记录尚无自动安全回收办法，完整进程存活期恢复、共享资源、平台与性能验收仍待完成。

三个目标固定配方制品更新（2026-09-08）：
- 上轮为progress（恢复权限统一）。本轮检查当前构建脚本、Go模块和记录，git status仍无仓库。设置工作区GOPATH/GOCACHE/TMP后实际执行 ./scripts/build-release.ps1，默认三个目标全部成功；持续观察同一构建session至exit0，无重启或将等待误报结束。
- dist/release当前Windows amd64：12711424字节，SHA256 8db910f36fdd6acaa75d949cacdb927ccac9e25154ee0efcf6b0942530747a68；Linux amd64：12357794字节，e2a5939b39b2e5b7e6e3d1c1350cb90bf1464e6c3a45e717ef86b7cd34d21fd1；macOS arm64：11898082字节，d2795b37886b1bf8243af4a71d45a8875493b77a5c2e19ccf97512ddef5421ff。
- 逐项重新计算SHA256/文件大小，与build-manifest.json和SHA256SUMS核对一致。go version -m读取三份二进制，确认go1.26.6、目标GOOS/GOARCH、CGO_ENABLED=0、trimpath。Windows实际--version输出0.1.0-dev；help clean包含完成准备恢复与预览重查说明。
- 这是本地开发制品更新，不是发布或签名；未改PATH/安装位置。Linux/macOS只证明交叉编译与元数据，不证明原生运行。检查supervise_unix.go仍存在macOS后代完成保证缺口，未隐去限制。新制品未重测性能/RSS，不能沿用旧摘要的性能数据作为验收结果；完整目标仍进行中。

构建失败保留上一组开发制品（2026-09-08）：
- 上轮为progress（三目标制品更新）。本轮检查脚本发现后续目标编译失败前会覆盖前面目标，导致正式目录混合新旧产物。改为每次唯一暂存目录，全部目标和清单生成后再替换正式文件，目标去重，成功后只删除空暂存目录；失败保留部分文件供检查。不引入递归删除。
- 真实Windows构建到.build/release-staging-check成功，摘要与上轮8db910f...一致。随后仅在本次PowerShell进程中使用go函数测试替身：version及第一个Windows目标调用真实Go构建0.1.1-dev，第二个Linux目标注入exit1。脚本抛出预期错误，逐项比较此前正式二进制、manifest、SHA256SUMS的摘要全部未变。没有把注入的Linux失败当真实编译结果；正式dist/release未改。
- 文档说明编译失败的保护范围和最终逐文件替换的非原子限制，未声称掉电/磁盘故障下整个产物集合原子更新。暂存失败目录仍保留在专用验证输出下。完整首版平台和性能验收仍开放。

Windows环境变量大小写冲突（2026-09-08）：
- 上轮为progress（构建失败保护）。本轮检查架构配置/凭据约定和当前环境构造代码：已有无效变量诊断不输出值，但Windows项目env中的Path/PATH由Go map遍历顺序决定最终覆盖值，存在非确定性。
- config.Parse在Windows按当前runner使用的大小写归一化规则拒绝重复键，在写状态/运行后端之前报配置错误；runner.Environment也拒绝调用者直接提供的冲突映射，保护非配置入口。继承父环境仍按原顺序最后值优先，项目覆盖父环境保持；POSIX大小写不同键继续共存。错误仅含键，不包含值。README补充约定。
- 新增配置与runner冲突测试，断言精确诊断无测试私有值；runner同时验证POSIX两个值保留。原EnvironmentIsolation通过。首次联测runner通过0.133秒，config原ParseBoundaries受沙箱祖先元数据Access denied限制；获准重跑相同隔离config测试通过0.105秒。没有后端启动、网络下载或用户环境修改。
- 实际命令 go test ./internal/config ./internal/runner '-run=^(TestEnvironmentCaseConflict|TestParseBoundaries|TestEnvironmentProjectCaseConflict|TestEnvironmentIsolation)$' -count=1，及获准config子集重跑。未重新构建发布制品；代理/镜像专题、完整平台与性能等验收仍待推进。

HTTP请求失败诊断隐藏URL凭据（2026-09-08）：
- 上轮为progress（Windows环境变量确定性）。本轮检查HTTP下载路径发现Client.Do返回的url.Error被直接格式化，可能包含URL查询令牌和嵌套代理诊断。
- 新增requestFailure包装请求创建与发送错误，Error只输出固定的无URL提示，并区分取消/超时；Unwrap保留底层错误链供errors.Is/As使用。适用于共享制品下载和Node元数据请求，不更改请求认证、TLS校验或下载内容。
- TestRequestFailureDoesNotExposeCredentials使用本地Transport替身提供源URL和代理URL测试凭据，验证输出精确为无凭据诊断且底层原因可识别；覆盖取消、截止时间及含无效转义的URL解析错误。与既有真实本地HTTP校验和/部分文件清理测试一起运行，go test ./internal/backend '-run=^(TestRequestFailureDoesNotExposeCredentials|TestDownloadChecksumAndCleanup)$' -count=1通过0.147秒。
- 未访问外部服务或使用真实凭据。这里只覆盖HTTP请求创建/发送错误；没有声称后端子进程输出、响应正文错误、全部日志或锁文件已统一脱敏。完整代理/镜像用户配置和发布验收继续开放；发布制品未重建。

Windows单进程峰值工作集测量（2026-09-08）：
- 上轮为progress（HTTP错误诊断）。本轮核对微软GetProcessMemoryInfo/PROCESS_MEMORY_COUNTERS官方文档，新增scripts/measure-memory-windows.ps1，使用原始进程句柄跨退出读取PeakWorkingSetSize（字节），不以PID重新查找。初始探针API成功返回非零峰值，退出后current接近零，未把current误当peak。
- 脚本默认20次--version，可传命令参数；stdout/stderr异步流向Stream.Null，不缓存用户输出，stdin关闭，10秒超时终止自己启动的树，句柄最终Dispose。保存全部峰值、最大值、OS、参数与实际SHA，CreateNew拒绝覆盖报告。只测CLI自身，不包含测量宿主/子树，不宣称延迟或冷缓存结果。
- 在Windows实际执行默认20次version、20次help。报告.build/perf/memory-version-windows.json最大9392128字节(8.95703125MiB)，memory-help-windows.json最大9490432字节(9.05078125MiB)，均绑定当前制品8db910f36fdd6acaa75d949cacdb927ccac9e25154ee0efcf6b0942530747a68。此制品早于最近环境冲突/HTTP诊断修改；不将结果套用于未构建的新源码。
- 两项观测峰值低于32MiB；完整性能门禁未通过，状态/run/no-op sync与子进程树内存仍待测。首次脚本意外输出VoidTaskResult，已将异步等待结果赋给null并单样本smoke验证输出恢复简洁，报告不受该格式问题影响。docs/build.md新增调用方式和官方计数口径链接；没有后端下载或用户环境修改。

完整run的Windows CLI自身内存（2026-09-08）：
- 上轮为progress（峰值工作集测量）。新增Windows限定的TestMeasureFullRunMemory，复用prepareRetainedNodeRun的隔离已应用Node fixture，使用绝对路径测量脚本与CLI，通过环境变量传递路径、固定PowerShell代码传递argv，路径不会拼接进shell源码。
- 测量实际外部CLI的 -C <fixture> run node -e '' 共20次，使用保留官方Node22.23.2，不隐式安装。报告包含完整argv、固定制品摘要和每次CLI峰值；Node及整个子树内存明确排除。
- 首轮实际测量完成并生成memory-full-run-windows.json，最大12853248字节，但测试报告读取因PowerShell数值12853248.0不能反序列化成uint64而失败；修正读取为float64后重建测试，获准复测完整入口通过4.98秒。验证报告.build/perf/memory-full-run-windows-verified.json，最大12.2461MiB，SHA256 8db910f36fdd6acaa75d949cacdb927ccac9e25154ee0efcf6b0942530747a68。
- gate：MYENV_TEST_PREPARED_RECORD指保留Node记录；MYENV_TEST_MEMORY_SCRIPT指scripts/measure-memory-windows.ps1；MYENV_TEST_CLI_EXECUTABLE指固定制品；MYENV_TEST_RUN_MEMORY_REPORT为新报告路径。实际 .build/cli.test.exe '-test.run=^TestMeasureFullRunMemory$' '-test.v' '-test.timeout=2m'，TMP/TEMP使用.build/tmp。
- 此制品的Node-only完整run自身峰值观测低于32MiB；不能推广到任意配置，也不能当子进程树内存或新源码制品验收。状态/no-op混合sync/子树内存、延迟和原生平台仍待完成。没有外部下载或用户环境修改。

真实混合runtime项目完整CLI无变更同步计时（2026-09-08）：
- 上轮为progress（run自身内存）。本轮为TestMixedRuntimeSyncRetained加入可选measureMixedNoopCLI，复用真实保留Node/uv/Python建立项目，原测试验证两工具版本和状态后关闭本地Node下载服务；计时阶段调用实际外部CLI而不是进程内Service。
- 每次 -C <fixture> sync --locked --no-input --json，解析输出确认ok、changed/lock_changed/native_lock_changed均false且generation ID始终一致。51次，首次单列，后50次nearest-rank p95；摘要绑定实际受测制品，输出CreateNew不覆盖。
- 获准执行 .build/core.test.exe '-test.run=^TestMixedRuntimeSyncRetained$' '-test.v' '-test.timeout=4m' 完整通过21.55秒（含建立和清理fixture）。MYENV_TEST_MIXED_NOOP_TIMINGS=.build/perf/mixed-noop-cli-windows.json，CLI=dist/release/myenv-windows-amd64.exe，保留Node/Python记录与TMP/TEMP=.build/tmp。
- 完整CLI无变更同步中位49.2804ms、p95 54.8544ms，低于200ms预算。报告限定为Node+Python工具声明且不含Python项目依赖的隔离项目；不能外推到任意依赖规模或完整发布验收。制品仍为8db910f36fdd6acaa75d949cacdb927ccac9e25154ee0efcf6b0942530747a68，初次环境准备耗时不混入计时。
- 无外部下载或用户环境修改。该场景自身/子树内存尚未测，复杂Python项目和其他平台仍待验证，完整目标保持进行中。

带原生Python锁和依赖的混合项目计时（2026-09-08）：
- 上轮为progress（仅工具声明的混合同步计时）。本轮扩展共享fixture，新增TestMixedProjectSyncRetained：Node+Python、pyproject.toml、uv生成原生锁和一个本地纯Python wheel，实际导入myenv_probe并检查VALUE，未用假安装结果代替后端。
- 首次准备受既有构建许可预检拒绝（0.21秒），未改许可规则。fixture准备现显式AllowBuild=true，仅授权本次测试的自建项目/本地wheel；项目package=false，无自定义构建后端。无变更检查和外部CLI计时仍不传--allow-build，成功返回证明无需再次准备/授权。
- 关闭本机Node/wheel服务、将进程内UV改成不可用路径后验证no-op，再运行固定外部CLI 51次sync --locked --no-input --json。每次检查未改声明锁/原生锁/代ID。真实测试完整通过26.78秒（包括建立、导入检查和清理），计时报告.build/perf/mixed-project-noop-cli-windows.json：中位124.1584ms、p95 136.1979ms，低于200ms预算。
- 此结果限定一个小型纯Python依赖的Node+Python项目，计时不含首次准备；SHA绑定固定8db910f36fdd6acaa75d949cacdb927ccac9e25154ee0efcf6b0942530747a68。报告scope已按是否含Python项目分别记录，避免与上轮仅runtime数据混淆。未重跑上轮场景。
- 实际获准 .build/core.test.exe '-test.run=^TestMixedProjectSyncRetained$' '-test.v' '-test.timeout=4m'，复用保留Node/Python记录、工作区TMP及MYENV_TEST_MIXED_NOOP_TIMINGS新报告路径。没有外部下载或用户环境修改。混合项目内存、原生其他平台和剩余完整验收仍开放。

小型混合Python项目无变更同步内存（2026-09-08）：
- 上轮为progress（带wheel依赖混合项目延迟）。本轮内存脚本新增ExpectedGeneration模式，逐次解析sync JSON核对ok/changed/lock_changed/native_lock_changed和代ID；使用64Ki字符上限、4Ki缓冲异步读取，超限继续排空后报错，不无界缓存stdout。普通命令仍直接排空输出。
- 新增measureMixedNoopMemory，经MYENV_TEST_MIXED_NOOP_MEMORY单独启用，复用真实混合项目与本地wheel fixture；计时gate未设置，不重复51次延迟测量。使用固定脚本代码/环境变量传路径，报告绑定实际CLI摘要与期望代ID。
- 获准TestMixedProjectSyncRetained完整通过25.46秒（包括准备/校验/清理），20次完整外部sync的myEnv自身最大峰值12.4570MiB，报告.build/perf/mixed-project-noop-memory-windows.json。低于小型无变更sync的64MiB预算；仍只包含CLI自身，未测整个子进程树。
- 受测制品继续为8db910f36fdd6acaa75d949cacdb927ccac9e25154ee0efcf6b0942530747a68。实际运行 .build/core.test.exe '-test.run=^TestMixedProjectSyncRetained$' '-test.v' '-test.timeout=4m'，配置保留Node/Python、MYENV_TEST_MEMORY_SCRIPT、MYENV_TEST_CLI_EXECUTABLE与专用TMP。没有外部下载或用户环境修改。完整平台/延迟/子树内存和其他首版交付仍开放。

同步过程阶段反馈（2026-09-08）：
- 上轮为progress（混合项目内存）。本轮确认CLI已有Checking environment即时提示，但后续准备/发布阶段没有更新。新增core.SyncPhase与可选同步Progress回调，由CLI映射固定无敏感值文本并输出stderr，不让核心服务直接打印。
- Sync在取得工作区锁前、版本解析、uv准备、Node/Python准备、依赖同步、文件验证与发布前发出阶段；UseRequest转发同一回调。无变更快速返回只经历锁阶段，不误报下载/安装；dry-run不宣称已进入实际准备。无计时器、后台goroutine或进度守护进程，回调要求及时返回且不修改状态。
- 扩展真实TestSyncRetainedPython，断言首次准备阶段顺序和no-op只有等待锁阶段。获准复用uv/Python隔离测试通过5.42秒。CLI受影响JSON/错误基础回归通过0.574秒：go test ./internal/cli '-run=^(TestDryRunLockedMissingLock|TestCleanCLIResult|TestRunMissingEnvironmentExit)$' -count=1；core测试重新编译后执行对应真实用例。
- README说明stderr阶段提示与stdout JSON边界。未重建制品或将旧制品的性能数据套到新增输出路径；首条阶段反馈100ms预算仍需测量，其他完整首版验收仍开放。

新阶段反馈制品首条提示计时（2026-09-08）：
- 上轮为progress（阶段回调与真实验证）。本轮以固定配方构建.build/perf/progress-release/myenv-windows-amd64.exe，12719616字节，SHA256 87c5a6b0af61e951503dbef4f244961850c8449b8a50ad11fa22b64c6566b4a7，独立于旧dist/release测量制品。
- 混合CLI计时器新增首条完整stderr行到达时间，起点为exec.Run前，验证Checking environment前缀。自定义Writer不嵌入bytes.Buffer，避免io.Copy通过继承ReadFrom绕过计时Write。记录全部51个反馈样本、首个、热p95及最大值；不以终端渲染时间命名。
- 获准真实带wheel混合项目TestMixedProjectSyncRetained通过26.94秒（含fixture）。报告.build/perf/mixed-project-progress-windows.json：首条反馈热p95 39.4287ms、全部样本最大54.5523ms；完整无变更sync中位127.0549ms、p95 137.9182ms，仍低于200ms预算。每次JSON结果确认无变更。
- 本次测量为完整无变更命令的初始检查提示，代码位置先于Sync核心检查/下载；这些样本均低于100ms，但没有测终端渲染、冷缓存或首次实际下载流程，不能把本结果当全部准备反馈验收。内存等旧摘要报告不套用于该新制品。
- 实际构建 ./scripts/build-release.ps1 -Targets windows-amd64 -OutputDirectory .build/perf/progress-release；重编译core测试，设置新CLI路径与MYENV_TEST_MIXED_NOOP_TIMINGS，再执行既有真实混合项目用例。无外部下载或用户环境修改，完整目标仍进行中。

状态快照读取去重（2026-09-08）：
- 上轮为progress（新制品首条反馈计时）。检查Status发现先ReadSnapshot，再snapshotHealthy内部重复ReadSnapshot；Python快照解析还包含项目路径校验。这不是状态必需的两份独立证据。
- 新增readHealthySnapshot，依次验证完成标记、读取配置和摘要，成功返回已验证配置。Status复用返回值检查入口与平台锁，snapshotHealthy作为既有布尔包装供sync/rollback使用。不缓存跨命令状态，不跳过标记/摘要/入口，也未给Status加写入或租约。
- go test ./internal/core '-run=^(TestSnapshotHealth|TestStatusReadOnly|TestRuntimeSnapshotIdentity)$' -count=1通过0.345秒。重新构建core测试，获准复用保留uv/Python执行TestSyncRetainedPython，通过5.65秒，包含真实Python状态、no-op与回滚。
- 未重复性能测量或声称50ms状态预算已通过；新源码尚未重建发布制品。完整性能和平台/资源恢复验收仍开放，无外部下载或用户环境修改。

小型混合项目完整状态命令计时（2026-09-08）：
- 上轮为progress（状态快照去重与回归）。本轮新增measureMixedStatusCLI，经MYENV_TEST_MIXED_STATUS_TIMINGS单独启用，复用真实混合项目而不重复no-op/内存测量。51次 -C <fixture> --json，逐次验证ok、ready、活动代ID一致，另核对前后state.db SHA相同；计时包括外部进程启动与项目检查。
- 固定配方构建.build/perf/status-release/myenv-windows-amd64.exe成功，12720128字节，SHA256 8bc637c96a7b4acc7be7a0c721331ff90def47d6e439b0d72b06fbe9459986d7。它包含上轮快照读取修改，不沿用旧制品的性能结论。
- 获准TestMixedProjectSyncRetained通过24.76秒（含fixture准备/实际wheel导入/清理），报告.build/perf/mixed-project-status-windows.json：完整状态中位85.8952ms，p95 92.0212ms，明确超过50ms预算。测试通过只证明测量和行为断言成功，不是性能门禁通过。
- 实际 ./scripts/build-release.ps1 -Targets windows-amd64 -OutputDirectory .build/perf/status-release；重新编译core测试后配置保留Node/Python记录和新报告路径，执行该真实混合项目用例。无外部下载/用户环境修改，数据库字节保持不变。
- 后续需定位状态中Python项目输入/路径校验和启动的具体成本，不通过省略校验或调整预算掩盖失败。完整平台与其他首版验收仍开放。

Python项目根路径重复解析优化（2026-09-08）：
- 上轮为progress（完整状态性能失败证据）。本轮检查Within发现python.project为'.'时，先EvalSymlinks(root)，再对相同已解析路径重复遍历。现在目标等于已解析根时直接返回，仍执行相对路径/卷名/词法边界检查以及第一次完整根解析；其他子路径仍走原符号链接祖先验证。
- 新增TestWithinRootAndSymlinkBoundary，覆盖'.'、'./'、折叠到根的路径、词法逃逸、不存在子目录，以及实际符号链接逃逸/缺失后代/作为根的符号链接。不是跨调用缓存，也不声称两次检查构成的旧TOCTOU问题被完全解决。
- Windows获准配置/文件引用/workspace边界测试通过；新增真实symlink子项因宿主缺少创建符号链接权限明确SKIP，未标为通过。随后交叉构建config-linux.test，在现有WSL Ubuntu组件环境运行同一子集，含真实symlink子项全部通过。产品WSL拒绝逻辑未修改。
- 实际命令使用config测试二进制 '-test.run=^(TestWithinRootAndSymlinkBoundary|TestParseBoundaries|TestPythonFileReferenceBoundary|TestPythonWorkspacePatternBoundaries)$' '-test.v'；WSL通过--exec启动工作区二进制。仅隔离临时目录，无下载或用户环境修改。
- 新制品尚未重建/性能重测，不能据读取次数减少宣称状态50ms预算通过。Windows原生链接权限验证、其他平台和完整验收继续开放。

根路径去重后的完整状态复测（2026-09-08）：
- 上轮为progress（Within优化与Windows/WSL边界验证）。本轮读取前轮记录并重新编译core测试，固定配方构建.build/perf/root-path-release/myenv-windows-amd64.exe成功，12720128字节，SHA256 97d9d33fdcb4786f9e7158dde50dec6ef6c1cf99eb79264850272637215eb8b7。
- 获准复用保留运行时与本地wheel隔离项目，执行TestMixedProjectSyncRetained，只启用MYENV_TEST_MIXED_STATUS_TIMINGS。完整通过25.42秒，51次状态输出均ready/同一代，数据库摘要不变。报告.build/perf/mixed-project-status-root-path-windows.json，中位80.8778ms、p95 87.3273ms，仍明确未达50ms。
- 相比前次85.8952/92.0212ms略低，但两轮非同时严格A/B，不能将全部差异归于根路径去重。没有降低门禁或省略输入检查。代码检索显示ReadPythonInputs逐个输入文件、相关引用及workspace检查还重复调用Within，后续应定位这些路径的真实成本与安全可复用边界。
- 实际构建 ./scripts/build-release.ps1 -Targets windows-amd64 -OutputDirectory .build/perf/root-path-release；测量用例 '-test.run=^TestMixedProjectSyncRetained$' '-test.v' '-test.timeout=4m'。无外部下载或用户环境修改，完整首版验收仍开放。

单次Python输入检查复用已读清单（2026-09-08）：
- 上轮为progress（根路径优化复测仍超预算）。本轮检查ReadPythonInputs与pythonRelatedDigest发现顶层pyproject.toml/uv.toml在同次检查中重复边界解析/读取，且若读取间被编辑，主摘要和相关依赖分析可能来自不同内容。
- ReadPythonInputs向本次相关依赖分析传递最多三份已读元数据（项目清单、不同workspace根清单、uv.toml，单文件仍1MiB上限）；可选文件缺失也作为本次观察复用。uv.lock不额外保留；相关依赖的其他路径继续原有Within和读取，不缓存整个依赖图。每次ReadPythonInputs都重新读取磁盘，发布前既有重查保持。
- Windows获准配置测试全集exit0，覆盖原生输入变化、引用/workspace边界及锁等；Within真实symlink子项仍因宿主权限SKIP（此前WSL真实链接证据复用）。重新编译core测试并获准TestSyncRetainedPython通过5.23秒，包含实际Python同步、文件漂移、失败保留和回滚。无下载或用户环境修改。
- 本轮未重建发布制品或性能重测，不声称50ms门禁通过；也不声称取得整棵依赖图的文件系统原子快照。继续保留完整验收开放项。

输入复用后的完整状态复测（2026-09-08）：
- 上轮为progress（单次元数据复用及配置/真实Python验证）。本轮固定配方构建.build/perf/input-reuse-release/myenv-windows-amd64.exe，12723200字节，SHA256 3c1e921700e69f4d801db17e93bae6ce1c2f7388e1f8b78893a72546534c1283。
- 获准TestMixedProjectSyncRetained通过23.98秒；仅启用MYENV_TEST_MIXED_STATUS_TIMINGS，51次完整状态调用保持ready/同一代，数据库字节不变。报告.build/perf/mixed-project-status-input-reuse-windows.json，中位71.5753ms、p95 80.0633ms，仍未达到50ms预算。
- 相比上次80.8778/87.3273ms有所下降，但不是同一时刻A/B，不把差值全部归因于元数据复用。下一步应采集内部状态检查系统调用/阶段证据，区分文件路径、SQLite和启动成本；不继续凭推测减少校验。
- 实际 ./scripts/build-release.ps1 -Targets windows-amd64 -OutputDirectory .build/perf/input-reuse-release，使用上轮已重建的core测试与新制品路径执行既有真实混合项目用例。无外部下载或用户环境修改。完整平台/资源与性能验收仍开放。

状态系统调用等待定位（2026-09-08）：
- 上轮为progress（输入复用复测仍超预算）。本轮新增可选traceMixedStatus，仅在真实混合fixture准备结束后对51次进程内Service.Status启用runtime/trace，每次核验ready及代ID；独立MYENV_TEST_STATUS_TRACE新文件，不重跑外部计时，不将插桩运行当延迟门禁。
- 获准TestMixedProjectSyncRetained通过21.81秒；.build/perf/mixed-status-input-reuse.trace实际406480字节。复用本机Go工具提取syscall pprof，保存mixed-status-input-reuse-syscall.pprof与mixed-status-input-reuse-summary.txt。没有重新下载工具或运行时。
- syscall profile Type=delay：Status累计1547.14ms（51次）；ReadPythonInputs1113.98ms，Within1069.29ms，EvalSymlinks1068.59ms，FindFirstFile764.80ms。Within约占全部1548.93ms系统调用等待的69.03%，不是完整CLI墙钟百分比；不把嵌套累计值相加。
- 另按internal/state聚焦，累计57.68ms，仅约3.72%的系统调用等待，其中OpenReadOnly28.15ms。该证据不支持把SQLite持久性设定当当前状态主瓶颈，下一步优先研究重复Windows路径规范化与根边界校验的安全复用。
- 实际通过trace.exe -pprof=syscall及pprof.exe -top -cum查看；profile提示无main binary filename，但带完整Go栈并可按Status/state筛选。完整目标保持进行中，未更改产品校验或性能预算。
句柄路径解析独立实验（2026-09-08）：
- 沿用上轮状态trace证据，新增Windows专用、默认跳过的TestWindowsHandlePathExperiment（MYENV_TEST_HANDLE_PATH=1）。实验函数仅存在于_test.go，未替换生产Within。使用已有x/sys/windows的CreateFile（零数据访问、共享读写删除、目录兼容标志）和GetFinalPathNameByHandle；路径缓冲上限32768字符，仅接受DOS盘符或UNC结果，缺失路径保留os.PathError。
- 读取本机Go1.26.6 src/os/file_windows.go的normaliseLinkPath作为既有调用参考；微软官方API页面与搜索两次请求均连接失败，未声称在线文档已核实。Git状态检查再次确认工作区不是Git仓库。
- 实际命令：gofmt -w internal/config/path_resolution_windows_test.go；设置工作区GOPATH/GOCACHE/TMP及MYENV_TEST_HANDLE_PATH=1后 go test ./internal/config -run '^TestWindowsHandlePathExperiment$' -v。首次沙箱运行现有EvalSymlinks返回Access is denied；同一隔离实验获准运行通过0.348秒。现有目录、文件、大小写变体与EvalSymlinks结果完全一致，缺失文件错误分类通过；真实symlink创建仍因权限不足SKIP。
- 100组成对、交替顺序的本地既有文件调用（首组剔除）：句柄解析平均136.081微秒，EvalSymlinks平均2.237858毫秒。仅诊断单一路径，不是完整CLI p95，不据此声称状态50ms通过。
- 生产接入仍需核实真实junction/symlink、共享盘/访问权限及特殊路径语义；本轮无生产行为变化、无发布制品重建。上一轮完整状态p95 80.0633ms仍是最近有效门禁结果，完整目标继续进行。

Windows连接点边界修复（2026-09-08）：
- 上轮为progress（句柄解析独立实验）。本轮扩展真实目录连接点与超过550字符路径验证：长路径通过，但连接点暴露现有缺陷——本机EvalSymlinks返回连接点原路径，Within接受指向工作区外的junction及junction/missing/child。随后单独运行连接点子项记录两者均返回nil错误，确认不是仅字符串大小写差异。
- 将句柄解析接入config.Within的根和现有祖先检查，Windows通过最终句柄路径跟随连接点；其他平台仍使用filepath.EvalSymlinks。未知路径命名空间、API错误均返回错误，不回退到可能漏掉连接点的解析。保留相对路径词法检查和缺失路径逐祖先检查，不增加跨调用缓存。
- TestWindowsWorkspacePath现在默认执行正确性检查，计时部分仍由MYENV_TEST_HANDLE_PATH=1单独启用。连接点由固定PowerShell命令通过环境变量传递自有临时路径创建，清理只os.Remove连接点条目；验证越界及缺失子路径拒绝、连接点作为工作区根允许。符号链接因创建权限仍SKIP，未冒充通过；共享盘与ACL语义仍待真实验证。
- 获准执行工作区GOPATH/GOCACHE/TMP下 go test ./internal/config（启用实验计时）通过1.438秒；复用MYENV_TEST_PYTHON_RECORD运行 go test ./internal/core -run '^TestSyncRetainedPython$' -v，通过4.84秒，真实同步/漂移/失败保留/回滚证据有效。随后仅将已通过测试改为默认运行并重命名，计时仍可选；没有重复整个测试。
- 本轮优先完成实验发现的实际边界缺陷。尚未重建固定发布制品或测完整状态p95，不能据微基准声称50ms门禁通过。目标保持进行中，下一步以新构建执行真实混合项目状态测量并补平台语义验证。

句柄解析接入后的完整状态复测（2026-09-08）：
- 上轮为progress（真实连接点越界修复及配置/Python验证）。本轮检查实际代码与测量入口，固定配方构建 .build/perf/handle-path-release/myenv-windows-amd64.exe 成功：12720640字节，SHA256 3ce779149538ee419fd7e898e67bc422f32cd42e02f3810d9564004b1f382c4b；重新编译core测试以包含当前生产代码。
- 获准复用保留Node/Python与本地wheel执行TestMixedProjectSyncRetained，通过22.53秒。仅启用MYENV_TEST_MIXED_STATUS_TIMINGS，51次完整CLI状态均ready/同一代，数据库字节摘要不变；报告 .build/perf/mixed-project-status-handle-path-windows.json：首次77.9096ms、后50次中位49.4198ms、p95 56.1215ms，明确仍未达到50ms预算。
- 前次完整状态中位71.5753ms/p95 80.0633ms；两次不是同一时刻严格A/B，不把差值全部归因于句柄解析。不重复微基准或全量配置测试。本轮未测新制品RSS/冷缓存，既有其他SHA的RSS不能充作新制品完整资源门禁证据。
- 实际命令 ./scripts/build-release.ps1 -Targets windows-amd64 -OutputDirectory .build/perf/handle-path-release；go test -c -o .build/core.test.exe ./internal/core；测试二进制 '-test.run=^TestMixedProjectSyncRetained$' '-test.v' '-test.timeout=4m'。进程session 80077轮询到PASS后结束，无重启或重复测量。
- 配置模块以GOOS=linux GOARCH=amd64与GOOS=darwin GOARCH=arm64、CGO_ENABLED=0分别交叉编译测试二进制成功，证明新的非Windows适配文件可编译，未声称原生运行通过。完整目标继续进行，后续应结合新路径成本定位剩余启动/状态开销，并补新制品资源及平台语义验证。

统一启动测量入口（2026-09-08）：
- 上轮为progress（新制品完整状态复测56.1215ms仍超预算）。本轮核实旧启动脚本使用.NET Process.Start/WaitForExit，状态与run用Go os/exec；为减少跨测量宿主的混淆，新增默认跳过的TestMeasureInformationalCLI，MYENV_TEST_INFORMATIONAL_TIMINGS开启。
- 新入口复用固定CLI制品，帮助/版本交替顺序各51次，分别剔除首次，保留50原始样本并计算nearest-rank p95；包含进程启动、输出管道排空与退出，10秒每调用上限；核验成功、无stderr、帮助含Usage且临时项目目录无新状态文件。报告O_EXCL防覆盖，并记录实际二进制SHA256。此测试不证明没有任何用户目录访问或网络系统调用。
- 实际设置工作区GOPATH/GOCACHE/TMP、MYENV_TEST_CLI_EXECUTABLE=.build/perf/handle-path-release/myenv-windows-amd64.exe和新报告路径，执行 go test ./internal/cli -run '^TestMeasureInformationalCLI$' -v，通过4.42秒。报告 .build/perf/startup-handle-path-go-windows.json，SHA与上轮3ce779149538ee419fd7e898e67bc422f32cd42e02f3810d9564004b1f382c4b一致。
- 帮助中位39.6842ms/p95 61.6851ms，版本39.4748ms/57.9605ms，均仍失败30ms预算。旧PowerShell数据与本轮制品、时段和宿主不同，不能据差值认定.NET是唯一原因，也不以中位替代p95。
- 阅读既有inittrace显示net初始化约10ms、总包初始化约19ms；这是旧制品的诊断证据，不能当成本轮完整墙钟分解。尚未修改初始化架构或降低功能检查；下一步需区分新制品进程加载、包初始化与CLI本身成本，避免靠调整测量方式宣称达标。完整目标保持进行中。

状态资源验证与启动初始化核实（2026-09-08）：
- 上轮为progress（统一启动测量证实帮助/版本仍超预算）。本轮读取本机Go1.26.6 net/fd_windows.go与internal/poll/fd_windows.go：net.init直接调用InitWSA，后者执行WSAStartup及通知模式检查，早于CLI参数处理。因此仅延后backend.NewNode/http.Client创建不会移除此初始化成本；没有修改工具链、引入辅助产品二进制或宣称已经优化。
- 扩展现有Windows内存测量脚本StatusResponse模式，复用有界JSON读取和进程退出后GetProcessMemoryInfo峰值工作集查询；逐次验证ready、changed=false、同一代。新增MYENV_TEST_MIXED_STATUS_MEMORY独立混合fixture钩子，共用原内存辅助逻辑，未重跑延迟测量。脚本补充StatusResponse必须提供ExpectedGeneration的参数检查；docs/build.md记录用途与范围。
- 实际gofmt并go test -c -o .build/core.test.exe ./internal/core；获准复用保留Node/Python和自有本地wheel运行 '-test.run=^TestMixedProjectSyncRetained$' '-test.v' '-test.timeout=4m'，通过24.18秒。session48859确认exit0，无新下载或用户环境修改。
- .build/perf/mixed-project-status-memory-handle-path-windows.json记录20次完整状态调用，最大CLI峰值11.6992MiB，制品为既有handle-path-release（SHA 3ce779149538ee419fd7e898e67bc422f32cd42e02f3810d9564004b1f382c4b）；本fixture父进程低于32MiB。未测子进程树、冷缓存或原生其他平台，不将此标为完整资源门禁通过。新增参数缺失保护是测量结束附近的简单输入检查，未单独重跑整套fixture。
- 当前完整状态p95仍56.1215ms，帮助/版本仍超30ms；完整目标保持进行中。剩余启动优化需要可维护的依赖/加载方案，不能用绕过检查或更换测量口径解决。

连接点修复调用链覆盖（2026-09-08）：
- 上轮为progress（状态内存实测与初始化路径核实）。本轮检索internal全部EvalSymlinks引用，生产中仅剩非Windows适配；检查ReadPythonInputs、pythonRelatedDigest、PythonWorkspaceRoot及workspaceGlobFS，确认需要验证的是具体调用链，而不只是Within单函数。
- 扩展TestWindowsWorkspacePath/junction，复用同一真实隔离连接点及工作区外有效pyproject，分别通过ReadPythonInputs检查项目目录、tool.uv.sources本地路径、直接workspace member及递归junction/**成员模式。四项均断言明确的escapes workspace错误，避免把解析错误或静默忽略误当边界保护。
- 实际gofmt后，在工作区GOPATH/GOCACHE/TMP下获准 go test ./internal/config -run '^TestWindowsWorkspacePath/junction$' -v，通过0.811秒（用例0.71秒），四项逐一PASS。仅自有临时目录/连接点，无下载、依赖代码执行或用户环境修改；连接点条目仍使用os.Remove单项清理。
- 未发现需要追加的生产修复，未重建制品或重复延迟/RSS测试；已有同SHA证据保持有效。此证据覆盖静态连接点配置，不证明并发替换路径时的原子隔离，也不覆盖共享盘/ACL或宿主无法创建的符号链接。完整目标仍进行中。

悬空链接边界修复（2026-09-08）：
- 上轮为progress（连接点调用链覆盖）。本轮新增真实连接点目标目录被删除后的检查，原代码明确失败：Within(root,"junction")返回路径且nil错误。原因是解析缺失目标后一路回退到工作区根，把仍存在的连接点误当普通缺失目录。
- Within在每次not-exist解析失败后先Lstat当前探测路径；条目存在但无法解析时返回错误，其他非缺失错误也直接传播，只有路径条目确实缺失才继续向父目录回退。普通新建目录仍允许；没有缓存或忽略未知错误。此修复同时适用于悬空符号链接。
- Windows TestWindowsWorkspacePath/junction增加删除自有外部目标空目录后拒绝junction和junction/missing/child；通用TestWithinRootAndSymlinkBoundary增加对应悬空symlink场景。删除仅对明确的自有测试文件及空目录使用os.Remove，没有递归删除或跨shell文件操作。
- 实际gofmt后获准 go test ./internal/config，通过1.224秒；GOOS=linux GOARCH=amd64 CGO_ENABLED=0交叉编译config-linux.test，再通过现有WSL Ubuntu --exec运行 '-test.run=^TestWithinRootAndSymlinkBoundary$' '-test.v'，真实symlink及悬空场景PASS。Windows原生symlink仍受权限限制，WSL仅为组件证据，不代表原生Linux产品验收。
- 随后复用MYENV_TEST_PYTHON_RECORD执行 go test ./internal/core -run '^TestSyncRetainedPython$' -v，通过4.79秒。未重建发布制品或重测性能；增加缺失路径Lstat影响相关热路径，旧SHA延迟不能当成新代码门禁。并发替换路径的原子隔离仍未由此修复证明，完整目标保持进行中。

悬空链接修复后的状态延迟与内存（2026-09-08）：
- 上轮为progress（悬空连接点/符号链接修复及Windows、WSL、真实Python验证）。本轮固定配方构建 .build/perf/dangling-path-release/myenv-windows-amd64.exe：12720640字节，SHA256 63d30b02743f0455938b2d2214fe6901b48f7b3b9541a4521d044a96b527845b，并重新编译core测试。
- 获准复用保留运行时/本地wheel，在同一TestMixedProjectSyncRetained隔离fixture中顺序开启状态计时与内存钩子，最终26.18秒PASS。51次计时调用均ready/同一代，state.db字节摘要不变；随后20次内存调用逐一校验ready/changed=false及代ID。未启用其他计时、trace或下载。
- .build/perf/mixed-project-status-dangling-path-windows.json：中位51.5402ms，p95 57.4922ms，仍失败50ms门禁；.build/perf/mixed-project-status-memory-dangling-path-windows.json：CLI父进程最高11.6719MiB，低于32MiB。两报告对应同一新SHA，不包含子进程树或冷缓存。
- 实际 ./scripts/build-release.ps1 -Targets windows-amd64 -OutputDirectory .build/perf/dangling-path-release；go test -c -o .build/core.test.exe ./internal/core；'-test.run=^TestMixedProjectSyncRetained$' '-test.v' '-test.timeout=4m'。session63107轮询到exit0，未因等待重新启动测试。Git检查仍为非Git工作区。
- 前次状态p95 56.1215ms，与本次并非同时A/B，不将1.37ms差值全部归因于新增Lstat。正确性检查继续保留；完整启动/运行性能、冷缓存/进程树资源及原生其他平台验收仍开放，目标保持进行中。
