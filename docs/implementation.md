目标：按 [architecture.md](architecture.md) 完成 myEnv CLI 首版 T00–T06。状态：in_progress，2026-09-09。用户明确要求跳过 macOS，并授权通过 SSH 使用 Kali VMware 虚拟机开展原生 Linux 验证；macOS 实现、签名与原生验收不再阻塞本次交付。Windows/Linux 的既定预算仍保留。

2026-09-09 GitHub Pages 部署完成：工作流 34336974835 build/deploy 均成功；https://xvshifu.github.io/myEnv/guide/support/ 返回 HTTP 200，包含 Temurin 与 rc.1。修复干净 CI 缺少本地 hosting.json 的两处依赖，Sites 插件仅在本地宿主配置存在时启用，保留 GitHub Pages 静态导出。

2026-09-09 GitHub 与文档更新：按用户授权初始化仓库并推送 XVSHIFU/myEnv main，配置 GitHub Pages workflow。文档同步 rc.1 工具链/来源/系统管理/预览版/颜色进度，保留用户安装原图。Sites 构建脚本因 Windows npm 入口解析失败，改用现有 npm 绝对入口构建 /myEnv 前缀，11 页、262 内部引用、tsc 通过。缓存、trial、dist、站点宿主元数据未入库；暂存内容凭据模式扫描无匹配。性能差距汇总见 performance-gap.md，未重测或宣称当前 RC 达标。

2026-09-09 发布整理：用户 dev.5 试用反馈正常；将当前源码构建为本地 0.1.0-rc.1（Windows/Linux amd64），打包 Windows setup/portable，更新根 README、新增 CHANGELOG 与 release-readiness 检查单。Windows --version 通过，安装器内嵌 CLI SHA256 与独立制品一致，dist/0.1.0-rc.1 提供全套摘要。源码仅文档变化，复用 dev.5 定向回归，不重复全量测试。Linux 此候选仅交叉构建，未补原生验收。正式发布仍受原性能门槛、旧文档站、缺少 GitHub 目标与源码许可证选择阻塞；未公开发布、未创建仓库或修改旧试用记录。

2026-09-09 dev.5 显示层：依据用户 dev.4 测试通过反馈，添加 TTY 自动颜色和单行阶段动画，复用 x/term 与 x/sys，无新依赖。NO_COLOR/TERM=dumb 禁用装饰，JSON/no-input/verbose 不启用动画；run 不包装子程序输出。CLI System/Run/Locale/Language/Help/InventoryOutput/Sync/Use/Cancel 定向测试及 Progress 测试、go vet 通过；Windows PTY 已观察到 ANSI 彩色列表。制品 trial/v0.1.0-dev.5；原试用记录未修改。真实下载时动画与 Ctrl-C 视觉验收待试用，Linux 原生未重测，性能预算不变。

2026-09-09 dev.4 终端输出：完成其他 CLI 调研、分组摘要与 --details、myenv list 快捷入口、安装成功和 run 缺参的完整示例。CLI 定向回归与 go vet 通过；Windows 新二进制 list 实机查询成功。试用目录 trial/v0.1.0-dev.4，保留旧制品与图片。范围、依据及限制见 [terminal-output-design.md](terminal-output-design.md)。原性能验收仍开放。

2026-09-09 dev.3 SDK 修复与试用交付：已生成 trial/v0.1.0-dev.3（CLI、Windows 安装器/便携包、中文手册、五个示例及默认只读测试脚本）。Windows 托管 Rust 默认 LLD 的 rustc/Cargo 编译与退出通过；未放宽 Job/租约。Python 新增官网 Windows 完整 ZIP 来源，来源切换和失败保留旧环境通过。Python RC、Java 日期 EA、Node nightly、Go RC 在 Windows 实装运行通过，Rust 日期 nightly 在原生 Kali 的安装/编译/Cargo 通过。外部管理器入口默认计划，核验身份/成员；uv 修复/移除真实隔离测试通过，rustup 仅实际只读计划，Conda 协议夹具与 base/版本 pin 保护通过（未安装 Anaconda）。C/C++ 只检测与指引。相关定向测试和 vet 通过；最终二进制运行及安装器内嵌摘要匹配。Windows/Linux SHA 为 eb6880e1… / 4aac0b75…；详见 [dev.3 交付记录](sdk-dev3-delivery.md)。Linux 官网 Python 源码构建、未知外部安装器、rustup 原子强制修复及完整历史/变体矩阵仍不支持，不宣称全部来源无条件接管。原性能验收仍开放；未公开发布，旧试用记录和图片未改写。下一步收集新 trial README 与日志中的用户测试结果。

2026-09-09 系统管理与 SDK 扩展（候选实现，未发布）：Java/Go/Rust 官方归档后端、五种工具版本查询、系统发现与当前用户默认环境 install/upgrade/repair/remove/clean 已接入。未安装 Anaconda/C/C++，未修改真实默认环境、旧试用包或用户截图。Windows Go 1.26.6 与 Temurin jdk-21.0.12.1+1、Kali 三种 SDK 均完成真实安装/noop/deep/最小编译运行。Windows Rust 安装及版本验证通过，默认 MSVC 编译结束后 vctip 留在 Job 导致等待，尚未解决；遥测环境变量诊断失败，显式自带 rust-lld 编译与产物运行通过，不据此标记默认流程通过。下载零字节与内存输出捕获的早期推断已否定。定向测试/vet 通过，新增 SDK 约束合并补测通过。外部管理器修改、Python 官方安装切换、全渠道版本安装、Windows Rust 默认兼容及完整新生命周期矩阵仍未完成；新命令本地化也未完全收口。详细命令、证据、范围与下一步见 [SDK 候选记录](sdk-expansion-delivery.md)。原性能要求与租约/Windows Job 架构保留，无正式发布、无 Git 提交。

2026-09-09 开发环境小调研：查阅官方开发者调查、大学课程及发行安装文档，形成 [环境范围与官方来源建议](environment-research.md)。建议优先 Python/Node/JDK/C/C++，区分托管安装、系统前置依赖与项目库；明确现有 Python 来自 Astral 构建，新增后端尚未实施。读取 config 工具白名单确认仅 node/python；git status --short 仍报告非 Git 仓库。本轮仅文档修改，未安装环境、运行程序测试或发布；原性能目标不变，下一步供用户审阅后确定新增环境范围。


2026-09-09 使用文档站交付：读取用户新版测试记录并逐张查看五张重要截图，原件和 README 保留；图片复制到 website 并登记 SHA、日期和版本，用于安装演示及语言切换记录。用户证据确认实际安装/PATH/语言切换成功；help 简介仍漏翻译，已记录。新增 10 篇导航式中文指南、静态导出及手动 GitHub Pages 工作流，根路径/仓库前缀产物与 263 个内部引用检查通过，TypeScript 检查通过。尚无仓库地址，未公开部署。Java 当前无安装或管理后端，只能在已准备项目中执行 PATH 上的外部 java；本轮未安装 Java 或修改 CLI。详见 [文档站交付与限制](documentation-site-delivery.md)，其中保留框架导出兼容修正与构建依赖审计开放项。原性能目标不变。

2026-09-09 已执行中文体验与安装方案，交付 `trial/v0.1.0-dev.2`：中文帮助/手册/正常输出、可切换语言、status 共用裸命令入口、独立当前用户 Windows 安装向导和便携 ZIP。保留旧版与用户试用记录。Windows/Kali 各 35 项定向测试通过，最终制品运行/中文透传/重定向验证通过；安装器隔离文件与模拟设置测试、真实表单离屏预览及内嵌 CLI 摘要核验通过，未实际写用户 PATH 或注册表。一次既有启动流程前后对照仍超预算，未重测 run/RSS，未关闭原性能目标。完整证据、限制及制品 SHA 见 [本轮交付记录](localization-delivery.md)。构建命令为 `scripts/build-release.ps1 -Targets windows-amd64,linux-amd64 -Version 0.1.0-dev.2 -OutputDirectory .build/localized-release`，打包命令 `scripts/package-windows.ps1 -ReleaseDirectory .build/localized-release -OutputDirectory .build/localized-packages`。未签名、未公开发布，当前目录仍非 Git 仓库。下一步根据用户新版安装/试用反馈处理实际问题。

2026-09-09 中文体验调研：读取用户保留的 trial/README.md 输出，Node/Python 中文参数均正常；CLI 帮助、手册、阶段和错误建议目前主要为硬编码英文。核查 CLI 与本地 Cobra 模板接口，形成 [中文体验方案](localization-proposal.md)，区分面向人的中文、稳定机器输出与子程序原文。实际执行 `./trial/myenv.exe -C trial/node-demo` 成功显示 ready，`./trial/myenv.exe --help` 确认没有 status 子命令；此前试用说明写错入口，方案建议补同处理函数的 status 命令。此轮仅调研与文档，没有修改程序、替换制品或改写用户试用记录，没有运行 sync/clean 或全量测试。Git 状态检查仍报告不是仓库。下一步为用户确认范围后实施本地化，原性能开放项不变。

2026-09-09 用户试用交付：建立 `trial/`，复制已验证 Windows 开发版为 `trial/myenv.exe`，提供 Node 22.23.2 与 Python 3.12.13 的声明文件、无第三方依赖示例及简易 PowerShell 测试说明。`Get-FileHash trial/myenv.exe -Algorithm SHA256` 匹配 416d5f46…，实际 `./trial/myenv.exe --version` 返回 0.1.0-dev，`help init` 核对初始化入口。没有预先执行 init/sync 或下载，留给用户体验；没有修改 PATH 或替换性能候选。当前目录依旧不是 Git 仓库。本轮仅交付试用文件，复用既有功能验证，性能开放项不变。

2026-09-09 本轮限定评估已完成：补齐 Linux run 阶段计时；测试专用单进程 subreaper/PDEATHSIG 在 SIGKILL 后后代与回执、SIGSTOP 期间 deadline 两项不能等价，保留独立监督器。发现重复 schema 初始化，源码保留 Linux OpenForRun 快路径候选，租约/FULL/删除保护不变，Windows 仍走原 Open 与 Job 架构。定向可靠性与真实 run/clean 重叠通过。只做了一次既有端到端流程前后对照，p95 118.0288→170.2873 ms，未证明收益；没有重跑挑样或替换已验证发行候选。补齐每进程及聚合 RSS/PSS/私有页，保留采样非原子与峰值局限。详细阶段、SHA、原始记录与决定见 [run 阶段与合并评估](run-stage-evaluation.md)。Linux 默认及 runtrace 构建、Windows 默认 vet 均通过。本轮限定任务完成不等于原首版性能门槛完成。

2026-09-09 最新接续：已在用户授权的原生 Kali 完成真实 Node/Python、镜像/独立 CA、混合项目、失败保留与回滚、Bash/zsh/fish 最终制品接入，并修复跨平台测试夹具。Linux run 的租约与恢复令牌合并为一次 FULL 事务，原子提交/失败回滚与未知子树保护已验证。最新 Windows/Linux 候选为 `.build/kali-validated-release`（后续仅补测试和文档）；SHA 分别 416d5f46…、75ede63a…。完整本轮证据及原始日志索引见 [Windows/Kali 验证](kali-validation.md)。

原目标仍未全部完成：最新固定批次 Windows 启动/status/run 与 Kali run 延迟超预算；Kali 聚合 myEnv RSS 采样也超 32 MiB。没有降低预算或持久化设置。暂停态 Go 手工取消回调的限制已写入 README/手册，不能称任意时序全部关闭。开发版是否接受实测性能已询问用户，尚未据此改动验收要求。下文的 native-job-release 与“原生机器缺失”等均为本轮之前的历史记录，以本段及链接的新报告为准。

本轮之前的历史源码与制品记录（最新以页首为准）：
- 最新候选 .build/native-job-release 包含Windows创建时Job关联、原生I/O与已关闭文件拒绝、共享Writer顺序及系统下限手册。Go1.26.6/CGO0/trimpath/buildvcs=false/去符号；三份manifest/大小/SHA256SUMS独立一致。Windows 12834304 bytes，SHA 62c9b7dc1819526caa9f2832c960d6cb5f168d908352c0c538b1a1fc8b94f6fe；Linux 12476578 bytes，SHA 4f9a06bf7eab9b96c3f030a36c3551299a19f02d49d83bb03ce045325b916855；Darwin 11965714 bytes，SHA 743bd4427cc2f4642c8a383645b3980ab85fe70b90271014479a26d3dcb99288。Windows实际版本/manual通过，PowerShellAppliedProfile显式指向该制品执行shell-init/run、实际全局Python验证PASS11.81秒；准备夹具仍由当前测试CLI完成。Linux候选实际监督器截止/回执三项已通过WSL组件验证，Darwin仅交叉构建。Windows固定性能批次已完成：noop及run父进程内存通过，启动/status/run延迟失败；旧候选报告不适用于新SHA。未发布或写用户PATH。
- 上一候选为 .build/console-protocol-release，包含Windows真实控制台信号、继承管道下独立Job监督与Linux协议状态校验。Go1.26.6/CGO0/trimpath/buildvcs=false/去符号，三份制品大小、manifest与SHA256SUMS独立一致；未发布。Windows 12817408 bytes，SHA 13ca08939dd0202daafadb4d98154f3df341340e3e575226345ebf5c007588b2，实际版本/manual冒烟通过。Linux 12476578 bytes，SHA 824719e69b05a37762ba260cd1774e9ab15f49b9138a960e9d16272bfe01309b，实际候选监督器运行中截止/已过期/只读回执三场景经WSL通过0.82秒，仍非原生产品验收。Darwin 11965714 bytes，SHA 5914f3bc932b1877586640979e8ea28d9d8fc73cb90bfe9f6f86be819307ec9a，仅交叉编译。最新候选已完成Windows run、help/version、noop/status及run父进程内存采样，结果见后文新候选记录；以下旧候选数据不迁移到新SHA。
- Go 1.26.6、Cobra、CGO0；cli/config/core/backend/state/runner。没有Git仓库，不虚构提交或远端。可用环境为Windows amd64与WSL Ubuntu；产品仍拒绝WSL，原生Linux/macOS验收缺失。
- 上一组三平台构建为 .build/protection-diagnostics-release（早于Windows真实Ctrl-C修复），包含通用Unix完成判断、doctor运行保护诊断及全局提示/手册修复。Go1.26.6、CGO0、trimpath、buildvcs=false、去符号；manifest、尺寸和SHA256SUMS已独立核验，未发布或写用户PATH。下列三条均为该旧候选的历史证据。
- Windows 12813312 bytes，SHA a2a939242daa1ac65f6b3583f6e93e66abc50f64af355982b5dd5bc2c6115b75；最终制品版本/manual冒烟及项目/全局doctor文本和JSON保护诊断通过，manual-smoke.txt保留手册输出。当前SHA已完成Windows固定批次采样，见下表；启动/status/run仍超预算。
- Linux 12476578 bytes，SHA 7b0f1a796758f24febfe39a442e5692fdf5a96a29c86e2023ae9752c7803cd5b；当前制品经WSL内部监督器截止/回执三项验证通过，日志deadline-supervisor-test.txt关联该SHA；仍非原生产品运行验收。
- Darwin 11965698 bytes，SHA 555c208fce0f72b339ad5569a82e3417f3977344cb6369bd4cf4de69f6427d55；只交叉编译，原生未运行，完整监督仍未实现。

已实现的主要合同：
- init/sync/use/run、用户profile、status/doctor/--deep、rollback、run --current、clean/缓存清理、shell-init、manual及completion已接通。Windows真实Node/Python、本地wheel混合项目和失败保留旧代已有证据；不能据子集宣布T00–T06全部完成。
- sync使用最终目录准备不可变新代，SQLite单一活动引用提交；运行租约及准备hold保护对象。安装命令使用用户Data/Cache，内部Service.Storage=nil兼容布局不是CLI存储合同。项目代与共享缓存无硬链接复用。
- BeginTrackedOperation在所有后端启动前登记；历史无标记操作不自动回填/解除。Windows命名Job及出生身份/session、Linux精确完成令牌用于恢复；未知子树保留ErrTreeUnconfirmed及租约。
- 恢复按128条分页；最终事务重新比较身份、引用、hold及子树状态。ReleaseLease要求本人记录恰好删除一条。CLI收尾失败不覆盖已完成子进程退出码。
- Windows真实首次启动前、版本探测、uv请求和升级同步强杀恢复已有证据。旧活动代可经run --current继续运行；未知历史操作仍保留。
- Windows路径边界拒绝越界/悬空junction；真实符号链接受宿主权限限制。Linux回执最终叶使用openat NOFOLLOW/NONBLOCK，不能退回不安全的叶打开方式。
- 镜像URL限制、独立CA客户端、固定uv制品摘要已接通；HTTP请求/响应体错误使用安全文本并保留cause，实际截断/取消清理通过。SSL_CERT_DIR不是受支持的Go客户端配置。
- TTY无参数use、Windows init/use/构建Ctrl-C、Linux提示独立非阻塞读取及信号通过对应测试；预先取消的CLI/use/sync避免已复现的写入。verbose只向stderr写命令/平台/耗时。Darwin提示实现已交叉编译但未原生验证。
- shell-init只生成函数，不修改用户Shell配置；PowerShell实际制品→真实Python出口验证通过，Bash组件通过。zsh/fish实际运行缺失。卸载只删除二进制，不暗中删除环境。

Linux监督当前状态：
- 独立subreaper、生命周期管道、wait4至ECHILD、回执写入并Sync；转发INT/TERM/HUP/QUIT/CONT并支持Ctrl-Z、Bash fg/bg。停止/继续事件不等于回收，leader回收后不再使用其旧PID操作终端。
- 收到排队Stopped且context已取消时，继续读取最终完成确认；未知/无效协议不误报成功。
- Deadline随请求传给监督器，调用端暂停时仍回收子树；deadline使用绝对墙钟时间，系统时钟突变行为未验证。
- TTY调用端传入自身pidfd和确认管道，子树结束后监督器重复CONT直至确认EOF，覆盖CONT早于STOP的顺序。调用端在Wait监督器前确认；描述符CloseOnExec。需要允许pidfd系统调用，通常Linux>=5.3；预检失败在用户子进程启动前返回。
- 真实PTY无外部CONT诊断证明截止后调用端已退出、子树已回收：.build/stop-deadline-probe/result-with-wake-check.json。诊断main退出0不代表用户命令取消退出码。
- 最近WSL测试：九项真实Node/TTY/Bash及截止回执通过；延迟STOP恢复0.11秒；确认EOF/意外数据/错误pidfd边界通过；真实TTY fd3至8隔离及暂停恢复0.64秒。测试二进制路径与逐项记录见归档。
- 无deadline且取消尚未送达监督器便暂停的窗口尚未解决/完整验证。不要将上述截止修复宣称为全部手工取消竞态关闭。

当前Windows同SHA性能（a2a939242daa1ac65f6b3583f6e93e66abc50f64af355982b5dd5bc2c6115b75）：

| 场景 | 原始报告（.build/perf/） | 结果与范围 |
| --- | --- | --- |
| 混合无变更sync | mixed-noop-protection-diagnostics-windows.json | 50样本p95 62.7409ms，通过200ms；初始反馈热p95 35.0176ms，所有样本最大48.7038ms（管道反馈） |
| status | status-protection-diagnostics-windows.json | p95 55.7956ms，未通过50ms |
| run附加开销 | run-protection-diagnostics-windows.json | 相同真实Node直接执行为基线，p95 75.3146ms，未通过50ms；未重测进程内子项 |
| help/version启动 | startup-protection-diagnostics-windows.json | .NET直接启动，每项50样本；version/help p95 84.0593/80.9506ms，未通过30ms |
| run父进程内存 | run-memory-protection-diagnostics-windows.json | 20次最大PeakWorkingSet 12.2266MiB，通过32MiB父进程子项，不含子树 |

2026-09-09固定批次按顺序执行，没有并行性能采样：run测试12.39秒、混合项目26.18秒、启动脚本正常退出、内存测试5.41秒；五份报告SHA均独立核验匹配当前制品。测试采样成功不等于延迟门禁通过。使用保留真实Node/Python与本地wheel，未重复下载；旧d263…报告完整保留，不做跨批次因果对比。
首个观测值单列，未清除OS缓存，不能代表受控冷启动。不同启动器/制品不作因果对比。保留的最小Go宿主基线startup-minimal-go-baseline-windows.json两组p95为49.6982/51.0294ms，已重新核验其独立诊断制品SHA一致；这是旧宿主证据，不是myEnv门禁，不重复测量或直接相减p95。需要参考宿主复核尾延迟。启动脚本已拒绝异常输出和覆盖旧报告，复核制品前后SHA；其20样本脚本验证报告不替换上述50样本门禁。进程内help/version约0.043/0.026ms；inittrace单次net约10ms，诊断不是p95，也不能解释全部映像加载成本。

未完成项与下一步（逐组核对见[七组风险缺口](acceptance-gaps.md)）：
- [macOS接口调查](macos-supervision-investigation.md)已确认当前XNU拒绝NOTE_TRACK/NOTE_CHILD；通用Unix Finish现已返回ErrTreeUnconfirmed，使普通run保留租约；这只是纠正错误完成判断，并未实现完整监督。macOS完整后代监督仍未实现，新代准备受ErrTreeUnconfirmed策略限制；不能只放开保守策略。原生macOS/原生Linux机器或runner仍不可用，之前已询问连接方式，未收到答复；并非全任务无路可走。
- 继续解决Linux手工取消送达与暂停的交界，使用针对实际风险的诊断/测试；避免只添加重复实现的测试。
- 为当前Linux最终制品补与其范围匹配的证据；现有WSL组件不替代原生平台，Darwin脚本scripts/verify-prompt-darwin.sh待原生执行。
- Windows启动/status/run预算仍失败；定位加载/初始化成本，不能降低预算或重复挑选较低样本。冷热缓存、完整子树内存及原生平台矩阵缺失。
- T00–T06和架构第11节七组风险必须逐项核对完整证据，不能凭功能已接通或子集通过关闭目标。

Windows监督收尾验证（2026-09-09）：独立等待goroutine现先关闭主进程观察句柄，再发布completion，确保Finish返回时该句柄已释放。真实Node夹具增加父进程exit17、独立后代继承stdout管道的场景；父已退出后发送Ctrl-C，后代回收且保留exit17，不被覆盖成通用1。三个孤儿后代场景分别PASS0.42/0.40/0.42秒，包1.351秒，均未触发5秒兜底。此为退出码与收尾顺序验证，未重测无关性能或声明完整Windows竞态覆盖。

Windows监督包集中回归（2026-09-09）：针对新增独立句柄等待与信号处理，设置保留真实Node记录后执行go test ./internal/runner -count=1 -json -timeout=90s，整包PASS5.608秒；日志.build/runner-windows-supervision-full.jsonl。实际覆盖argv、环境隔离、取消、启动失败/未知完成保护、七个真实控制台场景、Job等待/取消/监督器崩溃及句柄ABI。唯一顶层skip为由其他测试按环境变量启动的TestCompletionErrorChild辅助入口，不是漏跑真实Node用例。无生产修改，未据此扩大为原生Linux/macOS或性能验收；下一步无需无变化重复本包。

Linux手工取消诊断（2026-09-09）：.build/stop-manual-cancel-probe使用无deadline的WithCancel，由调用端time.AfterFunc在1秒后发起取消；真实PTY子程序先SIGSTOP，驱动等待调用端确实停止后再观察1.3秒。结果调用端状态T、取消回调尚未执行、子进程仍存在；外部SIGCONT后回调执行，runner返回137且后代已回收。result.json保留输出；诊断main本身exit0不代表runner成功。此WSL组件证据区分“暂停进程无法调度取消回调”与“已送达取消丢失”，没有关闭取消已发生但lifetime EOF尚未交付的极短竞态，也不代表原生Linux产品验收。已知deadline由独立监督器执行的旧证据仍适用，本轮未改生产代码。

Linux监督协议状态修复（2026-09-09）：此前握手只检查Ready=true，完成读取也能接受Ready+Complete混合帧。现在握手要求纯Ready状态，运行阶段拒绝再次Ready，Stop事件拒绝携带Complete/Ready/Canceled/Code/Error混合结果；协议错误沿既有未知完成路径保留保护。新增异常握手及伪完成帧测试，扩展已取消Stop校验。Linux交叉编译成功，WSL定向状态/暂停转换/取消后排空及真实监督器deadline三场景通过（deadline合计0.62秒）。本项针对协议完整性，不是已解决手工取消与SIGSTOP竞态；未声明原生平台或性能验收。

新候选run性能（2026-09-09）：对console-protocol-release Windows SHA 13ca08939dd0202daafadb4d98154f3df341340e3e575226345ebf5c007588b2执行TestMeasureFullRunCLI，仅测外部制品与保留真实Node直接基线；51次轮换顺序、首个单列、50个配对差值。中位59.1937ms，p95 76.6396ms，仍未通过50ms预算。报告.build/perf/run-console-protocol-windows.json的SHA及50样本数量独立复核；采样测试PASS12.76秒不等于性能门禁通过。无缓存清除，不作冷启动或跨批次因果推断；新候选其他性能项尚未采样，不重复挑选本项较低结果。

run阶段诊断（2026-09-09）：当前源码TestMeasureRunStages使用保留真实Node，51次首个排除，报告.build/perf/run-stages-console-protocol-windows.json；选代中位14.3955/p95 16.1279ms，执行（包含Node自身）89.6042/101.7191ms，释放8.0811/9.6812ms，测试PASS5.99秒。这是进程内诊断，不是候选CLI启动开销，也不能相加p95或与另一批直接Node相减。源码复核SelectRun每次state.Open都会执行完整CREATE IF NOT EXISTS/触发器初始化；可继续测量既有库初始化成本并评估版本化初始化，但当前证据不足以断言它是主要瓶颈。租约登记/释放的FULL持久性不能为预算直接删去。

既有状态库打开诊断（2026-09-09）：新增BenchmarkOpenExistingStore，在专用真实SQLite库上分别执行100次可写Open+Close与只读OpenReadOnly+Close；Windows结果平均2.878763ms/1.344703ms，19324/7239 B/op，434/141 allocs/op，日志.build/perf/open-existing-store-windows.txt。两者包含不同初始化路径，约1.53ms差值并非纯schema成本，也不是p95。该量级不足以将重复schema初始化认定为run约26.64ms超预算的主要原因；暂不引入数据库版本迁移复杂度，不降低FULL持久性。下一步应继续拆分租约事务或执行监督成本；本轮仅诊断测试，生产制品未改变。

租约事务诊断（2026-09-09）：新增BenchmarkLeaseLifecycle，专用真实SQLite库发布合法活动代，保持生产synchronous(FULL)，100次AcquireActive/ReleaseLease循环，结束验证无残留租约。Windows平均登记6.760620ms、释放6.520096ms、完整循环13.289998ms，7821 B/op/190 allocs/op；日志.build/perf/lease-lifecycle-windows.txt，测试PASS1.802秒。该路径确有可观固定成本，但这些是均值、无完整CLI加载、不与其他批次p95相加或相减。没有通过异步登记、提前释放或降低同步级别优化，生产源码与候选SHA未变；后续应优先定位监督/进程启动部分的额外成本。

Windows监督额外耗时（2026-09-09）：TestMeasureWindowsSupervision以同一真实Node、同目录/环境/参数/管道，轮换直接exec与生产runner Execute，51对首对排除、50个配对差值。监督额外耗时中位1.2583ms、p95 10.2920ms，测试PASS8.86秒；原始数据.build/perf/runner-supervision-windows.json。该诊断不包含CLI加载与租约事务，不能将不同批次p95相加；未发现足以解释完整CLI超预算的固定监督延迟，继续保留Job与信号回收约束。本轮仅增加可选测量测试，生产候选不变。

新候选Windows延迟批次补齐（2026-09-09）：同一13ca0893…制品用Go启动器TestMeasureInformationalCLI采样help/version各51次、首个单列50次，p95 62.7965/60.9157ms，均未通过30ms预算；不与旧PowerShell启动器数据作因果对比。随后顺序执行真实保留Node/Python与本地wheel混合项目TestMixedProjectSyncRetained（26.91秒），关闭本地制品服务后的无变更sync中位58.9688/p95 63.9278ms，通过200ms；初始反馈热p95 36.8924ms、全部最大42.0560ms；status中位50.0975/p95 60.8988ms，未通过50ms。三份.build/perf/{informational,mixed-noop,status}-console-protocol-windows.json的SHA独立核验匹配，原报告保留。测试采样成功不代表全部门禁通过；未清除OS缓存，当前候选内存尚未测量。

新候选run父进程内存（2026-09-09）：TestMeasureFullRunMemory对13ca0893…实际Windows制品运行保留Node20次，最大PeakWorkingSet为12.3633MiB，通过32MiB父进程子项；测试PASS5.73秒。报告.build/perf/run-memory-console-protocol-windows.json的SHA与20样本数量独立核验。新增监督goroutine后的制品仍在此预算内；没有测完整子树RSS、准备下载内存或其他原生平台，也不由此消除当前延迟失败。

Windows创建时Job关联可行性（2026-09-09）：核验Microsoft UpdateProcThreadAttribute文档，PROC_THREAD_ATTRIBUTE_JOB_LIST可在CreateProcess中指定Job（Windows10+/Server2016+）；Microsoft Windows metadata常量131085。本地Go1.26.6 syscall.SysProcAttr没有此属性入口，现有exec.Cmd创建后Assign仍有崩溃窗口。新增TestWindowsJobAtProcessCreation通过x/sys StartupInfoEx直接创建挂起且无窗口的cmd.exe，未调用Assign或恢复线程，Job会计已显示Active=1；关闭唯一Job句柄后稳定进程句柄确认进程退出，PASS0.02秒。此为原生接口可行性证据，尚未接入生产。下一步需在Windows launcher接入创建属性，同时保留argv转义、环境、管道、取消与退出码合同，不能仅把测试通过写成竞态已修复。来源：https://learn.microsoft.com/en-us/windows/win32/api/processthreadsapi/nf-processthreadsapi-updateprocthreadattribute 和 https://microsoft.github.io/windows-docs-rs/doc/windows/Win32/System/Threading/constant.PROC_THREAD_ATTRIBUTE_JOB_LIST.html。

Windows原生创建函数（2026-09-09）：新增createProcessInJob，使用已有x/sys ComposeCommandLine编码argv、UTF16工作目录和环境（拒绝NUL、保留nil继承语义与SystemRoot），创建属性同时支持Job列表及显式继承句柄白名单；返回仍挂起的原生进程/线程句柄。创建时Job测试已改为调用此函数，验证恢复前归属及关闭Job回收，PASS0.02秒。此函数尚未接入Execute，不能宣称生产创建窗口已关闭；下一步需实测argv/环境/stdio及完成调用端管道与取消整合。旧候选制品不含此新增函数。

Windows原生创建真实I/O验证（2026-09-09）：TestCreateProcessInJobNodeIO使用保留Node22.23.2与专用目录，显式复制三个可继承文件句柄并通过HANDLE_LIST传入，恢复原生主线程后验证空参数、空格、引号、尾反斜杠、中文及--json不变；Unicode环境大小写重复键采用最后值，cwd、stdin内容、独立stdout/stderr及exit17均正确。测试PASS0.13秒，包0.235秒。未改生产函数；目前验证文件stdio，尚未验证匿名管道、任意io.Reader/Writer泵送、控制台及取消生命周期；Execute仍未切换至原生创建。

Windows匿名管道继承（2026-09-09）：新增duplicateNativeStdio，仅拥有复制后的三个可继承子端句柄，创建后可独立关闭且不关闭调用方原文件，避免父端副本阻止EOF。TestCreateProcessInJobPipes以真实Node stdin.pipe(stdout)验证匿名管道Unicode往返、输入关闭产生EOF、输出读取自然结束及exit0，PASS0.12秒；等待有5秒兜底并回收读取goroutine。生产Execute尚未接入，下一步仍需任意Reader/Writer泵送、取消与整个Job的统一收尾；本项不声明完整launcher已完成。

Windows生产原生launcher接入（2026-09-09）：executePlatform现使用createProcessInJob，在创建时指定Job并保持挂起；复制标准句柄后只允许显式白名单继承，启动前检查取消/排队Ctrl-C，再直接恢复返回的主线程句柄。新增nativeIO支持文件直通、nil对应NUL、Reader/Writer匿名管道泵送、共享输出写入串行化；先等稳定主进程句柄再等Job清空，避免继承管道阻止Ctrl-C回收。独立context观察持续至整个Job完成，完成未知仍保留ErrTreeUnconfirmed。真实Node启用的runner整包PASS5.670秒，日志.build/runner-native-job-integration.txt，覆盖七个控制台场景、退出码、Job等待/取消/崩溃；旧启动前注入测试仍验证executeDirect测试接口，不冒充新launcher创建边界验证。随后对齐Go stdin忽略已退出子进程的BROKEN_PIPE/NO_DATA写入错误，新1MiB未读取输入正常exit0、argv/取消三项PASS0.582秒。创建后再Assign的生产窗口已由JOB_LIST替换，但新路径的创建阶段强杀及错误I/O完整矩阵仍需专项验证；旧候选13ca…不含本次改动，性能不可迁移。任意自定义Reader阻塞的可取消性仍受Reader本身约束，不承诺能强制中断用户回调。

Windows原生I/O错误与顺序修复（2026-09-09）：真实Node验证Reader/Writer故障在子进程exit0后仍返回原始cause，且不误报未知子树。共用Writer的交替stdout/stderr测试先发现两管道虽然串行写却改变顺序（200字节数量正确、顺序失败）；prepareNativeIO现对可比较的相同Writer共用一个子端管道，匹配exec.Cmd行为，逐字节验证100次oe顺序。三个I/O场景及未读取stdin、argv回归均通过，未触发兜底。生产I/O源码改变，旧候选性能仍不适用；独立Writer不承诺跨流顺序。

Windows创建阶段强杀验证（2026-09-09）：TestWindowsNativeCreationOwnerCrash启动独立测试进程，通过生产createProcessInJob创建挂起cmd.exe并报告PID，确认Job会计Active=1后保持主线程未恢复；父测试取得稳定SYNCHRONIZE句柄，强杀所有者且不运行其defer，随后确认挂起进程自动退出、无context兜底。PASS0.08秒；受辅助入口调整影响的原生创建/主动关闭Job测试PASS0.02秒，包0.200秒。验证的是生产创建原语在创建后恢复前的所有者崩溃，而非全部Execute指令边界；未改变生产代码，未重测无关性能。

Windows新launcher上层集成回归（2026-09-09）：启用保留真实Node/Python记录，生产当前源码TestMixedProjectSyncRetained通过20.11秒（包20.276），覆盖本地wheel混合项目同步；随后CLI整包go test -count=1 -json -timeout=90s通过45.491秒，包含真实Node镜像、自定义CA、Python运行及verbose Node等已启用集成。日志.build/native-launcher-mixed-sync.txt与native-launcher-cli-regression.jsonl。未设置性能采样变量或其他未提供的真实后端配置，可选跳过不能计入完成；无生产修改，不把旧制品或其他平台证据迁移到新launcher。

Windows原生launcher准备恢复回归（2026-09-09）：启用保留真实uv/Python，TestPythonSyncCancellationRetained PASS2.89秒，TestSyncKilledDuringRealUVRequest PASS2.05秒，TestSyncKilledRealUVPreservesActive PASS3.50秒，包8.595秒；日志.build/native-launcher-uv-recovery.txt。针对新创建/监督路径运行实际uv本地请求中的取消、首次准备强杀及已有活动代升级强杀，复用既有持久状态和恢复断言，未伪造安装成功或重下载运行时。生产代码未变，当前Windows恢复证据已覆盖这些受影响场景；不推广到原生Linux/macOS。

Windows关闭stdio启动漏洞修复（2026-09-09）：真实cmd测试发现已关闭stdout仍成功启动并exit0；os.File.Fd的无效值-1被DuplicateHandle当作当前进程伪句柄。duplicateNativeStdio现先拒绝InvalidHandle并返回带原文件名的os.ErrClosed，空文件也拒绝，prepareNativeIO明确拒绝typed-nil文件。回归验证失败发生在子进程执行前、无执行标记且不误报未知子树；缺失可执行文件保留ErrNotExist。真实Node文件/管道与I/O错误回归同时通过。生产源码改变，旧制品不含修复；调用者并发关闭正在传入的文件仍须遵守文件生命周期合同。

Windows原生创建最低系统与跨编译（2026-09-09）：README及内置manual说明JOB_LIST要求Windows10+/Server2016+，这是接口下限而非完整OS版本验收声明。go run ./cmd/myenv help manual实际退出0并核验新文本，输出.build/native-launcher-manual.txt。新execute_other构建约束及Windows专用文件加入后，Linux amd64与Darwin arm64 runner测试二进制交叉编译成功（runner-native-integration-{linux,darwin}.test）；未原生执行，不重复无变化WSL回归。文案改变，旧候选手册仍为旧版。

Windows新launcher Python镜像/Shell补验（2026-09-09）：确认保留uv及CPython真实归档可用后，仅设置MYENV_TEST_UV_ARCHIVE/PYTHON_ARCHIVE启用此前跳过用例；TestPythonMirrorsRealCLI PASS11.82秒、TestPythonMirrorsCustomCARealCLI PASS10.50秒、TestPowerShellAppliedProfile PASS11.31秒，包33.800秒。本地镜像各提供一次真实归档，关闭服务后locked/noop验证继续通过，自定义CA仅在测试环境生效；PowerShell生成函数经当前测试CLI入口实际运行全局Python。日志.build/native-launcher-python-mirrors.txt。未重复公网下载、未修改用户CA/配置；Shell用的是当前源码测试入口，不是旧发布制品。生产源码未变，原生其他平台仍待验收。

Windows原生创建后取消验证（2026-09-09）：executePlatform提取内部executeWindowsWithCreation，生产固定传createProcessInJob，无环境/用户可覆盖入口。测试包装真实创建函数，在返回挂起进程后立即取消context；独立复制稳定观察句柄，断言launcher返回context.Canceled/1、没有ErrTreeUnconfirmed、子进程已退出、用户命令未写marker且5秒兜底未触发。PASS0.03秒；受入口提取影响的真实Node取消/argv通过0.26/0.13秒，包0.521秒。证明取消已发生在恢复检查前的边界，不冒充检查与ResumeThread之间所有极短竞态已穷尽。

Windows原生创建后真实Ctrl-C（2026-09-09）：复用创建边界夹具与独立隐藏控制台，在真实CreateProcess返回挂起进程后GenerateConsoleCtrlEvent，通过额外观察器及signal.Stop同步交付，生产注册保持有效。launcher返回取消/1、稳定句柄确认回收、marker未出现，且context完全未取消，排除兜底掩盖。TestWindowsNativeInterruptAfterCreation PASS0.12秒；参数化后的context取消场景PASS0.02秒，包0.267秒。仅测试修改，不重测无关套件；检查与ResumeThread之间的极短交界仍不由此测试穷尽。

原生Job候选首批性能（2026-09-09）：实际Windows候选62c9b7dc…顺序运行固定批次TestMeasureFullRunCLI与TestMeasureInformationalCLI，真实Node直接基线50个配对差值run中位59.9660/p95 96.8549ms，未通过50ms；help中位42.3409/p95 65.5971ms、version中位44.9871/p95 60.6407ms，均未通过30ms。每项51次首个单列、50样本，无缓存清除；报告.build/perf/{run,informational}-native-job-windows.json SHA独立匹配。采样测试分别PASS12.31/4.56秒不等于门禁通过；不同批次不能据p95变化断言launcher造成回退，不重复挑选低值。当前候选noop/status和内存尚待采样。

原生Job候选剩余固定采样（2026-09-09）：同一62c9b7dc…候选顺序完成混合项目noop/status与run父进程内存。noop中位60.5978/p95 65.8379ms，通过200ms；初始反馈热p95 37.2916ms、全样本最大40.5537ms；status中位51.9866/p95 57.3165ms，未通过50ms。混合测试26.18秒，真实Node/Python与本地wheel、关闭服务后采样。20次run最大PeakWorkingSet12.2578MiB，通过32MiB父进程子项，测试5.48秒。报告.build/perf/{mixed-noop,status,run-memory}-native-job-windows.json SHA独立匹配。至此当前候选这五份报告已齐；启动/status/run延迟仍失败，完整子树RSS和冷缓存、原生Linux/macOS矩阵仍缺，不重复采样挑选低值。

当前Linux候选协议验收（2026-09-09）：runner-native-integration-linux.test显式指向native-job-release/myenv-linux-amd64，测试日志SHA 4f9a06bf7eab9b96c3f030a36c3551299a19f02d49d83bb03ce045325b916855与独立文件哈希一致。真实制品运行中截止、已过期不启动、只读完成回执失败三场景PASS0.80秒（0.31/0.05/0.31）。此为WSL内部监督器组件，不代表产品接受WSL或原生Linux完整运行矩阵；不重复已通过且无变化的源码协议测试。本轮无生产修改。

Windows launcher静态检查（2026-09-09）：go vet ./internal/runner通过，覆盖当前原生创建/句柄/I/O代码。Get-Command gcc,clang未发现编译器，当前未运行Windows race检测，未将普通测试当作race证据。源码复核I/O回收通过结果channel汇合、共享Writer受锁或单管道保护，context观察在关闭Job前join；这只是检查结果，不替代竞争检测或原生其他平台验收。生产代码与候选SHA未变，不重复已完成性能采样。

Windows原生启动错误上下文（2026-09-09）：CreateProcess失败现在包装os.PathError（create process、可执行文件路径、原始cause），恢复原先exec.Cmd诊断中可定位文件的能力；不把参数或环境加入错误。实际不存在文件测试验证路径与errors.Is(os.ErrNotExist)同时保留，已关闭stdio的启动前失败回归也通过，包0.110秒。仅失败诊断源码变化，当前发布候选未重建；不为这项错误路径变更重复成功路径性能批次。

macOS新监督接口发现（2026-09-09）：重新核实Apple官方文档，Beta es_new_descendants_client可限制到后代且不要求root/TCC，但仍需获批ES entitlement。更新macos-supervision-investigation.md，保留SDK版本、事件完整性/排序、所有者死亡与持久完成证明待验证项，未把可观察事件当作回收证明。已异步询问现有Apple团队/entitlement与签名入口；没有原生Mac/SDK，尚未实现或运行。旧kqueue NOTE_TRACK不可用结论仍有效，但不能据此排除这一新方向。本轮是改变下一步设计依据的官方接口调查，无生产变更。

macOS ES完成证明调查（2026-09-09）：Apple官方global_seq_num可检测后续消息可见的丢事件间隙，Beta es_sync_client可同步队列，但销毁客户端或传空客户端也会调用同步回调。调查文档新增独立存活状态、消息版本检查及尾部丢事件不可仅凭既有序号连续排除的证明要求。由此排除“sync回调即完整回收”的不安全实现；需要原生压力/生命周期证据与可靠最终完整性协议。签名权限问题仍待回复，未实现或运行Mac API，无生产变更。

外部条件审计首次记录（2026-09-09）：核对acceptance-gaps与实际native-job manifest，修正其旧候选引用及Windows创建/PowerShell已完成证据，明确旧doctor制品证据不迁移。当前xcrun/xcodebuild不可用，虽有ssh客户端但没有用户提供的原生Mac/Linux连接或Apple获批签名入口；Windows参考宿主问题也未收到回复。最近ES调查为实质进展，此轮开始记录外部条件阻塞审计，尚不标记blocked。不能用更多无变化Windows子集测试代替macOS完整实现、原生平台矩阵与性能失败；目标范围保持完整。

外部条件审计第二次（2026-09-09）：重新检查工作区入口，存在原生验证脚本和交叉制品，但没有可执行的远端CI配置或已提供的连接，xcrun/xcodebuild仍不可用；对原生主机、Apple ES签名及Windows参考宿主的问题未收到新信息。本轮无实现或测试进展，也不是等待已确认运行的任务；不把再次记录阻塞算成功能推进。相同外部条件连续第二轮阻止完整验收，目标暂仍active，未重复既有测试或缩小平台范围。

证据与接续：
- 逐轮CLI/监督证据完整保留在 [本次归档](implementation-cli-supervision-history.md)；更早记录为 [恢复归档](implementation-recovery-history.md) 和 [历史归档](implementation-history.md)。归档中的“当前”“未实现”均需按时间解释。
- 保留 .build/node-real/prepared.json、node-linux-real/prepared.json、python-real/python-prepared.json、uv-real/uv-download.json 及python-mirror-real真实归档；不要重复下载或删除有效制品。
- 每轮只更新本摘要中发生变化的状态和证据；较长逐轮记录写入独立归档，避免再次把状态文件扩展为历史全文。
- 最近验证：deadline_linux_test.go新增仅测试使用的MYENV_TEST_SUPERVISOR_EXECUTABLE，记录目标制品SHA后直接执行其监督器；当前Linux制品运行中截止、已过期不启动、只读回执失败三项通过（合计0.81秒）。原始日志.build/caller-wake-release/deadline-supervisor-test.txt，SHA匹配ae1b970d…；无生产代码或平台检测变更。此为WSL内部协议测试，不代表产品绕过WSL限制运行。
最近修复（2026-09-09）：通用Unix进程组监督不再以主进程Wait完成代替后代完成，Finish返回ErrTreeUnconfirmed。executeDirect因此返回错误并由RunEnvironment.Finish保留租约。Linux生产Execute使用独立subreaper，不走此回退。Linux/Darwin runner交叉编译成功；WSL直接回退错误、启动前缓存信号及Linux专用传输回归均通过（0.00/0.00/0.33秒），Darwin原生仍未运行。现有Darwin完整后代功能测试的目标预期未降低，macOS仍不满足首版支持要求。

租约链路复核（2026-09-09）：CLI run将同一runErr传给defer中的RunEnvironment.Finish；ErrTreeUnconfirmed路径仅关闭连接而保留SQLite租约。clean的回收函数只在Windows/Linux注册，Darwin无回收器时实际回收不删租约、预览也拒绝删除有租约代。复用既有TestCleanRetainsAppliedAndLeased持久化/重新打开/实际clean证据（恢复归档第56至58行），以及Linux真实未知子树集成证据；未重复不受改动影响的测试。此为源码链路与既有证据核对，未声称Darwin原生集成已运行。

CLI提示改进（2026-09-09）：runFailure针对ErrTreeUnconfirmed明确说明环境保护仍保留，并提示检查剩余子进程及运行doctor；与取消错误并存时保留CANCELED/130分类，但不丢失保护提示。普通执行错误提示不变。gofmt后Windows定向TestRunMissingEnvironmentExit/TestRunLeaseFailureExit通过，包0.503秒；本轮未制造真实监督失败来验证终端文案，不据此声明macOS支持。CLI源码改变，旧同SHA性能仅对应旧制品。

doctor诊断补齐（2026-09-09）：新增run_protection（none/present）及protection_detail；状态库存在时用SELECT EXISTS只读检查租约，不扫描目录、不推测进程存活，也不回收记录。文本及JSON均能解释运行保护仍保留，并提示clean --dry-run；无状态库时不创建。扩展已有DoctorDeepMissingEvidence夹具验证真实持久租约与诊断前后DB字节不变，未初始化状态为none。Windows core/cli定向Doctor测试通过（0.350/0.190秒）；未测新源码性能或原生Darwin，此项不代表完整监督实现。

旧库诊断验证（2026-09-09）：doctor改用state.HasLeases单条EXISTS查询，避免为存在性判断加载身份/回执扩展表。新增真实SQLite旧库夹具（移除两张可选元数据表）验证仍报告保护、只读重开前后文件字节一致；空库返回false。首轮夹具缺运行时入口，被Publish正确拒绝，补齐合法状态字段后state测试通过0.306秒；core DoctorDeepMissingEvidence已通过0.330秒，无需重复。此为存储/诊断验证，不是运行时安装或性能门禁。

CLI保护诊断验证（2026-09-09）：新增TestDoctorRunProtectionOutput，以真实SQLite持久租约和缺失运行时的状态夹具执行文本/JSON doctor，验证present、clean --dry-run提示、exit0、changed=false、stderr为空及调用前后DB字节一致。Windows用例PASS0.20秒（包0.357秒）。此状态夹具不代表运行时安装成功；本轮仅测试变更。

全局诊断修复（2026-09-09）：发现commandHint未转换myenv clean，导致doctor --global的保护提示指向项目清理；已补全--global。复用CLI保护夹具扩展全局场景，注入专用用户配置目录并放置故意损坏的当前项目声明，验证全局诊断不读取该项目、文本/JSON提示clean --global --dry-run且DB字节不变。项目/全局测试分别PASS0.20/0.21秒，包0.569秒。未执行实际clean或修改用户环境。

近期改动集中回归（2026-09-09）：针对run错误提示、doctor保护诊断和全局commandHint改动，Windows执行go test ./internal/cli -count=1 -timeout=90s通过3.525秒；go test ./internal/core -count=1 -timeout=90s通过6.244秒。未设置保留运行时/性能测试启用变量，因此这些是常规包回归，不能声称可选真实后端、性能或原生其他平台门禁全部运行。无需无变化重复这些套件，后续按新改动范围继续。

run全局错误提示修复（2026-09-09）：runFailure现在携带global上下文，普通执行错误及未知子树完成提示均指向doctor --global；项目提示不加该标志。错误Unwrap保留取消与未知完成标识。四种上下文/错误组合测试通过，Windows包0.165秒；未重跑刚通过的全量包，也未把错误包装测试当成真实监督失败验收。

诊断文档（2026-09-09）：README与内置manual说明run_protection/protection_detail、记录不等于进程存活、项目/全局clean预览命令，以及不统计准备hold/不自动回收的范围。go run ./cmd/myenv help manual退出0并核验新字段文本存在；仅文案验证，未重测性能。已异步询问是否有现成Windows参考测试机/CI入口，当前未收到答复；此前原生macOS连接问题仍待用户信息。

新制品行为验证（2026-09-09）：CLI保护诊断夹具新增仅测试使用的MYENV_TEST_DOCTOR_EXECUTABLE，以实际Windows制品执行项目及全局文本/JSON，分别PASS0.30/0.31秒。全局只对子进程设置APPDATA为专用目录，数据库字节保持不变，制品SHA独立匹配a2a939…；不是运行时安装测试。现有Linux协议测试指向新制品7b0f1a…，三项PASS合计0.81秒（0.31/0.05/0.31秒）。本轮仅修改测试，未改生产源码或重建。

Windows真实Ctrl-C修复（2026-09-09）：新增独立CREATE_NEW_CONSOLE隐藏测试，先确认控制台只有测试自身并清除继承的Ctrl-C忽略位，再启动真实Node父子，等待子节点注册处理器后GenerateConsoleCtrlEvent。修复前调用端直接以0xc000013a退出；修复后supervise_windows注册os.Interrupt保护调用端，事件由共享控制台直接交付，不重复转发，Job完成后撤销注册；恢复主线程前若已有事件排队则取消启动。Node父子均写信号处理标记，父退出23被保留。加入父处理器重复事件检测后PASS0.49秒（包0.582秒）；真实Job等待/子树取消/父先退出后取消三项回归PASS2.26秒。未向用户控制台广播，也未改变用户配置。测试字符串修改一次PowerShell解析失败未写文件，随后apply_patch成功。创建/恢复线程边界全部竞态及Ctrl-Break仍未完整验证，不能将这一场景推广为全部Windows信号验收。

Windows Ctrl-Break验证（2026-09-09）：复用独立隐藏控制台夹具，按测试名隔离子进程入口，GenerateConsoleCtrlEvent(1,0)实际触发Ctrl-Break；真实Node父子注册SIGBREAK，均记录处理完成、父进程保留exit23，并检测重复事件。Ctrl-Break PASS0.46秒，受夹具参数化影响的Ctrl-C回归PASS0.45秒，包1.008秒。无生产修改，创建/恢复主线程边界仍未完整验证。

Windows启动前排队Ctrl-C（2026-09-09）：提取独立控制台测试夹具，新增TestWindowsConsoleInterruptBeforeStart，在生产prepareSupervision完成而exec.Start尚未调用时发送真实事件，通过额外信号观察器与Stop同步确认交付。随后实际cmd.exe以挂起方式创建，Start阶段消费排队事件返回context.Canceled；断言没有执行标记、没有ErrTreeUnconfirmed、5秒兜底期限未触发。用例PASS0.11秒，受夹具提取影响的真实Node Ctrl-C回归PASS0.45秒。没有生产代码修改；事件发生在检查与ResumeThread之间的极短窗口仍不能由此证明。

Windows默认Ctrl-C退出对照（2026-09-09）：TestWindowsConsoleDefaultInterruptExit在独立控制台分别直接执行和经生产Execute执行同一真实Node无自定义处理器程序，准备标记就绪后发真实Ctrl-C。仅直接基线的测试驱动自行保护信号，受管分支依赖生产注册。两者实际退出状态均为3221225786（0xc000013a），无兜底超时，测试PASS0.30秒（包0.395秒）。未预设或按实现猜测退出码；证明本场景原生退出状态透传，不等于完整终端/创建竞态矩阵。未改生产代码。

Windows主进程退出后Ctrl-C修复（2026-09-09）：真实Node创建detached后代，测试先取得稳定进程句柄，再允许主进程退出，确认后代存活后在独立控制台发送Ctrl-C。stdio忽略与继承stdout管道两个场景均先复现5秒context兜底失败。Finish消费排队事件、终止Job并等待ActiveProcesses为零，以内部已确认中断结果返回非零退出码，不误报ErrTreeUnconfirmed；普通Wait错误仍保留。继承管道会阻塞Cmd.Wait，因此Windows现在保留带SYNCHRONIZE权限的主进程句柄，启动后由独立goroutine等待该句柄退出再监督Job，Finish领取完成结果；主进程仍在运行时不强杀其信号处理器。真实控制台六项、Job三个等待/取消场景及监督器崩溃回归通过，包4.922秒，日志.build/windows-console-pipe-regression.txt；继承管道场景0.47秒，未触发兜底，稳定句柄确认后代死亡。生产源码已改变，既有发布制品及其性能报告不包含此修复。创建/恢复线程竞态仍未完整关闭。

外部条件审计第三次（2026-09-09）：相同原生平台/SDK、ES签名及Windows参考宿主条件仍未提供，复核前两轮记录与当前工具可用性后，目标标记blocked。没有完成首版，没有放弃macOS、降低性能预算或把WSL视为原生验收；获得所需入口后从现有实现和证据继续。当前源码、保留运行时、候选制品及报告全部保留。

2026-09-09 用户恢复任务：SSH kali@192.168.61.129 已成功，Linux 6.16.8+kali-amd64/x86_64、glibc 2.42；Python3/Bash/zsh 可用，未发现 Go/fish。测试目录 /home/kali/myenv-validation-20260909，不改用户配置。历史 macOS 缺口只作历史记录，不属于调整后的本次验收范围。

Kali 首轮原生 runner 全包已执行（保留测试二进制，未重编）：.build/kali-validation/runner-native-kali.txt。真实 Node 取消/信号、9 项 PTY/Bash 暂停恢复、独立候选截止与回执均通过。整包结果 FAIL：TestDirectCompletionErrorRetainsProtection 与 TestDirectStartFailureCompletion 的注入包装先遇到通用 Unix Finish 返回 ErrTreeUnconfirmed，未到达原 Windows 假设下的注入错误；需调整跨平台边界测试设计，不能宣称全包通过。唯一 SKIP 是子进程辅助入口。CLI --version 成功（初次误用 version 子命令返回 USAGE_ERROR，后改正确参数）。远端 CLI/Node 摘要分别匹配 4f9a06bf… 与 b294a556…；未改生产源码或系统配置。后续继续原生 Linux 完整 CLI/runtime 验证及失败测试处理。
