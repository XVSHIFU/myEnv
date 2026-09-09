目标：按 [architecture.md](architecture.md) 实现 myEnv CLI 首版 T00–T06。状态：in_progress，2026-09-09。没有完成首版验收，不缩减目标。

最新制品提示：最近三目标构建为 .build/caller-wake-release，新增Linux截止取消、调用端恢复与严格确认；Windows/Darwin与prompt-errors-release字节一致，该Windows SHA已测混合无变更同步p95 78.2770ms通过200ms预算，status p95 57.0460ms及run开销p95 82.3604ms未通过50ms预算；20次run父进程最大峰值工作集12.2422MiB通过32MiB子项。启动通过PowerShell直接进程测得version/help p95为76.6930/82.1271ms，未通过30ms预算；冷缓存与完整内存矩阵仍未完成，Darwin原生验证未运行。较早 .build/active-read-release 的Windows状态查询p95 57.5821ms未达预算；更早 .build/mirrors-shell-release 的启动/run/状态延迟未达预算，混合无变更同步延迟通过。具体测量范围与原始报告见文末逐轮记录，不将旧制品或父进程内存子集拼为完整验收。

本文件只保留当前接续摘要。逐轮证据见 [近期实现归档](implementation-recovery-history.md) 和更早的 [历史归档](implementation-history.md)。归档中的“未实现”、制品指标和进程句柄必须结合时间判断，不作为当前状态直接复用。

当前实现与约束：
- Go 1.26.6 + Cobra，模块为cli/config/core/backend/state/runner；无GUI/服务/插件平台。工作目录没有Git仓库，不虚构提交。Windows本地、WSL Linux组件可验证；原生Linux/macOS产品验收没有证据，产品仍拒绝WSL。
- init/sync/use/run、profile、status/doctor/--deep、rollback、clean、缓存清理、离线manual及completion已接通。Windows真实Node/Python及本地wheel混合项目已有证据；不能据子集声称完整依赖来源/平台合同已验收。
- CLI安装命令通过runtimeService使用用户Data/Cache。内部Service.Storage=nil使用工作区兼容布局，不代表CLI共享存储未实现。uv、Python安装有版本/平台锁；Node传输缓存按摘要锁下载、复制、清理。项目代与共享缓存没有硬链接复用。卸载文档只删除二进制，不暗中删除环境；没有待补的自动运行时卸载功能。
- Windows路径边界用文件句柄解析；拒绝越界与悬空junction，真实符号链接受宿主权限限制。Linux回执最终叶使用openat的NOFOLLOW/NONBLOCK，已验证symlink/FIFO/目录拒绝。不能退回os.Root.OpenFile加NOFOLLOW的旧方案。

持久化与恢复当前状态：
- 操作hold、进程出生身份、子树登记与完成令牌持久化；Sync登记已前移到managedUV/ResolvePython之前、dry-run和构建权限之后。
- BeginTrackedOperation原子写入operation_tracked，承诺所有后续后端启动先登记。历史BeginGuardedOperation无此标记，升级不回填；旧库只读预览不建表、不改字节。
- Windows用所有者出生身份/session和命名Job消失确认，Linux用所有者退出及精确令牌回执。完整登记且零子树允许在所有者退出后回收，即使操作目录未创建；历史未知零子树保留。
- 恢复按128条游标分页；最终事务再次比较目录、身份、hold、子树状态，排除生成代ID/目录引用。257候选与257历史混排、边恢复边翻页通过。hold删除故障会回滚status。
- runner未知完成错误保留ErrTreeUnconfirmed；RunEnvironment.Finish保留租约并关闭连接。ReleaseLease要求恰好删除本人一条记录，错配/重复返回明确错误。CLI释放失败写stderr，保留已完成子进程退出码。
- Windows实际Sync首次启动前、阻塞版本探测中、真实uv本地依赖请求中强杀均有集成证据。真实uv升级同步强杀后，清理只回收失败准备，原活动代ID不变，run --current选择的真实Python仍能运行。

Linux运行器当前状态：
- 独立subreaper监督器、父生命管道、wait4至ECHILD才确认完成，回执写入并Sync。INT/TERM/HUP/QUIT/CONT沿监督链转发。
- 真实Node单进程及进程组QUIT、PTY Ctrl-C/Ctrl-反斜线、前台恢复/再次输入通过；Node实际SIGSTOP后通过调用端CONT恢复通过；父子均停止时取消能回收并写精确回执。
- Ctrl-Z停止通知、调用者暂停、恢复时前台组交还已实现；WSL真实PTY与Bash fg/bg通过。暂停取消竞态、其他Shell及原生Linux验证仍待完成。
- macOS没有完整后代监督保证，当前新代准备受ErrTreeUnconfirmed策略限制。不能只放开保守策略来宣称支持。

最近CLI变化：
- 无参数use已在真实TTY接通工具/版本选择；CI/管道/JSON/no-input缺参仍拒绝。Windows init/use/构建提示实际Ctrl-C返回130，Linux use的INT/TERM/HUP和init的INT经WSL独立PTY通过；Linux/Mac原生验收不能据此关闭，macOS提示可取消读取已实现并交叉编译，原生PTY验证未运行。
- Windows提示通过专用线程CancelSynchronousIo并等待读取完成；Linux使用独立非阻塞终端句柄，不改变调用者文件标志或termios。预先取消的CLI、核心use/sync现在提前返回，避免已复现的配置/状态写入。
- --verbose实际输出命令名、平台和命令内耗时至stderr，状态JSON及真实Node argv/stdout/退出码回归通过。HTTP请求及响应体错误复用安全文本，保留cause；真实截断/中途取消无部分归档残留。
- run帮助已描述Python、应用PATH、相对/绝对路径及argv；manual/README解释Linux信号128+编号、Windows原生状态及作业控制当前验证边界。
- runFailure包装环境构造和执行器错误，使内部失败为RUN_FAILED/1；PathError/取消分类仍优先。损坏相对运行时路径曾复现USAGE_ERROR/2，修复后通过；真实Node/npm/npx、argv和退出码、释放失败0/7回归合计2.523秒通过。该测试未制造真实内核监督失败。

最近构建与性能（严格按SHA区分）：
- 历史制品在.build/supervision-recovery-release，包含近期Linux作业控制、回调失败清理、取消错误保留及CLI修复。三目标manifest与SHA256SUMS一致，Windows版本/manual冒烟通过；其完整同SHA性能矩阵尚未采样，下面旧制品指标只作为历史证据。
- .build/tracked-recovery-release三目标构建成功，manifest三项SHA已核对，Windowshelp/version冒烟通过。Windows12755968字节 SHA d510bb82a7095514c5518e69df063642b872f51b6eb9e6f9ec6d9eeda1450b2f；Linux12419234字节 SHA c5e5ef5657ddc07796184437fadce9a098e30be3c5a4f74726c49b48c2d8cb9e；Darwin11915394字节 SHA a43396b3b2164d17d0ac2fc569d878d5e281c20c0b6d6178c77c047b35946d11。
- 这些制品早于QUIT/CONT、手册及runFailure改动，不能代表当前源码全部内容。均为本地开发制品，无签名发布。
- 同Windows SHA报告 .build/perf/run-tracked-recovery-windows.json：run配对附加中位58.6703ms/p95 71.9819ms，失败50ms；startup-tracked-recovery-windows.json：help39.3904/64.8993ms、version39.6195/62.7649ms，失败30ms。51组首组剔除，50组保留；测试PASS只表示采样完成。
- 最小独立Go诊断程序同宿主p95约49.70/51.03ms（startup-minimal-go-baseline-windows.json），说明宿主启动尾延迟影响明显。不是myEnv制品、不是配对差值，不能据此降低预算；下一步需要参考宿主复核，避免无证据反复微调。
- 更早SHA faf26a3c... 的真实混合noop sync p95 67.1421ms、CLI峰值12.0898MiB通过200ms/64MiB子集预算；run峰值12.2422MiB。更早状态p95 57.4922ms失败50ms。均不得拼作当前同一制品完整矩阵。原始路径/完整SHA在近期归档。

下一步与完成审计：
1. 补齐Linux暂停取消竞态及其他Shell/原生验证；补充Windows实际控制台信号。Windows真实uv请求期间Sync强杀及旧活动代保护已验证，首次版本探测瞬间的真实uv中断仍未覆盖。
2. 实现并原生验证macOS后代监督。原生Linux产品/运行时、macOS产品验证仍待可用宿主；WSL组件与交叉编译不能代替。
3. 对T00–T06逐项对照当前代码与已有证据，核对全部命令/选项、Python组与依赖来源/构建许可、安装/PATH/manual及失败合同；已实现共享存储/profile不要按旧摘要重建。
4. 固定候选后完成同SHA热/冷延迟、CLI与整树RSS、首次反馈、真实平台矩阵；当前预算失败，不宣称已达标。发布制品/校验与安装材料须完整，公开发布仍按用户授权范围。

本轮记录：上一轮为progress（RUN_FAILED修复及真实回归）。完整复制原implementation.md为implementation-recovery-history.md，60617字节，SHA256 aae7ac98ba560d8c3ba016282ea0d1a9b591b30fe3cb2939174fcef7d0ed7ff6，复制前后摘要相同。更早归档未改动。重写当前摘要消除过时待办与制品混用；没有重复测试、下载、生产代码修改或发布。

最新进度：Linux等待事件接入（完整作业控制仍未完成）。上一轮为progress（接续摘要整理）。依据 https://www.man7.org/linux/man-pages/man2/waitpid.2.html 核对WUNTRACED/WCONTINUED，reaper现在订阅停止/继续事件并显式跳过非终止状态，不把它们记为leaderReaped或覆盖最终退出码。尚未向调用者发送停止通知，因此Ctrl-Z/fg缺口仍开放。
验证：gofmt、Linux amd64 CGO0编译.build/runner-wait-events-linux.test；获准WSL真实ContinueRetainedNode0.38秒、CancelStoppedProcessTree0.02秒、PTY Interrupt0.57秒全部PASS/exit0，无下载或发布重建。下一步接入监督协议的停止通知与终端组交还；新事件处理只是该实现的第一步，不是完成声明。

最新进度：Linux停止通知及前台继续初步接通。reaper在前台leader停止时调用监督端回调，交还调用者终端组并发送Stopped事件；调用端收到事件后SIGSTOP自身，CONT继续沿既有信号链传递。监督端只在终端当前属于调用者时将前台交还子进程，避免bg抢占Shell终端。停止/继续事件不释放租约，不替代最终ECHILD。解码错误会关闭父生命管道再Wait，避免协议错误时无限等待仍存活树；暂停前检查取消上下文。
验证：gofmt、Linux amd64 CGO0编译.build/runner-job-control-linux.test；WSL真实PTY InputAndInterrupt0.61秒、InputAndQuit0.59秒、StopAndContinue0.68秒全部PASS/exit0。新场景写Ctrl-Z，Python waitpid WUNTRACED确认调用者停止，再CONT并完成原终端退出/前台恢复断言。之后补暂停前ctx检查并gofmt，未重复测试。
仍需完成：实际Shell fg/bg、暂停/取消竞态、多次暂停协议（目前结果读取保留原总64KiB限制，加入事件后需改成单帧有界）、恢复后再次输入验证。README/manual的未完整实现声明继续有效。无下载或发布构建；本轮为代码与PTY证据progress，不是全目标完成。

最新进度：监督回复改为单帧有界。上一轮为progress（停止通知/前台恢复初步实现）。调用端用固定64KiB+1 bufio缓冲读取json.Encoder的换行帧，每帧最大64KiB；移除全会话总64KiB限制，避免长期暂停事件耗尽总预算。拒绝未换行截断、超长、无效JSON和单帧多个对象；每次解码清零回复，避免Stopped残留污染最终结果。
验证：Linux CGO0编译.build/runner-reply-frames-linux.test；获准WSL SupervisorReplyFraming通过0.01秒，含10000个停止事件后最终Code23、以及坏帧；真实PTY StopAndContinue通过0.71秒，exit0。10000事件是内存协议流测试，不冒充10000次真实进程暂停。暂停前ctx检查也包含在本次PTY编译验证中。未下载/发布重建，实际Shell fg/bg及暂停取消竞态继续开放。

最新进度：重复暂停后的终端输入验证。上一轮为progress（有界回复协议）。加强TestLinuxTerminalStopAndContinue：同一真实Node两次Ctrl-Z，逐次waitpid确认调用者停止；CONT后等待Node独立编号通知，再发送resume-N并等待对应输入确认。移除原先固定100ms继续等待，用明确事件同步验证前台读取恢复与第二次暂停协议。
验证：gofmt、Linux amd64 CGO0编译.build/runner-repeat-stop-linux.test；获准WSL保留Node/隔离PTY定向PASS0.61秒、exit0。此前最终Ctrl-C、调用者前台恢复及再次读取断言保留。无生产改动或下载；真实交互Shell fg/bg及停止取消竞态仍待覆盖，不以两次PTY循环代替全部作业控制验收。

最新进度：真实Bash fg往返验证。上一轮为progress（两次PTY暂停/恢复输入）。扩展隔离PTY测试启动/bin/bash --noprofile --norc -i，HISTFILE=/dev/null，独立prompt；真实Node两次Ctrl-Z后观察Shell Stopped与prompt，再由fg恢复，逐次CONT通知和终端输入通过，最终fg状态23。首次测试把printf放在初始命令后，Bash在任务停止时即执行导致最终取码超时；按观察改为从fg取码后通过，不是产品缺陷或降低断言。
验证：gofmt、Linux amd64 CGO0编译.build/runner-shell-fg-linux.test；获准WSL LC_ALL=C复用Node，TestLinuxShellForegroundResume PASS0.62秒/exit0。无下载、用户Shell配置/历史修改或生产代码变化。bg、停止取消竞态及其他平台验收仍开放。

最新进度：真实Bash bg边界通过。上一轮为progress（fg恢复）。扩展PTY场景，在首次Ctrl-Z后执行bg并等待Shell返回，TIOCGPGRP确认前台仍是Bash；jobs确认后台终端读取使任务重新Stopped，再fg恢复。Node续次计数及每次输入、第二次Ctrl-Z/fg、最终fg退出23均断言。CONT时替换fixture旧data监听器，防止后台未完成读取监听在再次继续后干扰输入验证。
验证：gofmt、Linux amd64 CGO0编译.build/runner-shell-bg-linux.test；获准WSL LC_ALL=C保留Node定向TestLinuxShellBackgroundResume PASS0.60秒/exit0。没有生产变化、下载或用户Shell配置修改。当前证据覆盖后台终端读取后重新暂停，不覆盖不读取终端且持续后台运行的任务再fg；该路径与停止取消竞态继续开放。

最新进度：持续后台任务fg验证。上一轮为progress（后台读取停止后fg）。新增TestLinuxShellRunningBackground，Node第一次CONT后暂停stdin读取并报告background-running；Bash jobs明确必须Running且不得Stopped，随后fg，轮询TIOCGPGRP必须等于Node实际PID，再完成Ctrl-C/调用者恢复及fg退出23。验证避免把再次停止的任务误当持续后台运行。
执行：gofmt及Linux CGO0编译.build/runner-running-bg-linux.test；首次场景通过，随后增加jobs Running反假阳性断言并重编译，获准WSL LC_ALL=C定向再次PASS0.60秒/exit0。未复现最初怀疑的前台切换缺口，因此未添加生产轮询/额外逻辑；不能泛化为所有Shell版本。无下载/发布，暂停取消竞态及原生平台完整验收继续开放。

最新进度：主进程回收后的作业控制边界。上一轮为progress（持续后台fg）。reaper在leaderReaped后不再使用该PID触发停止/前台继续回调，也不再由同数值PID的新事件覆盖已保存的主进程退出状态；SIGCONT仍转发给当前自有后代。防止主进程先退出时继续使用已失效身份操作前台组。
验证：gofmt、Linux CGO0编译.build/runner-leader-reaped-linux.test；获准WSL DetachedDescendant0.32秒、Cancellation0.31秒、真实Bash ForegroundResume0.60秒均PASS/exit0。前两项覆盖主进程exit17后等待/取消独立后代，后项确认存活主进程fg不受影响；没有制造真实PID复用，也不声称全部终端PID/组复用竞态已完成。无下载或发布构建。

最新进度：Linux runner集中回归通过。上一轮为progress（主进程回收边界）。复用最新.build/runner-leader-reaped-linux.test，在WSL LC_ALL=C及保留Node环境跑完整runner模块，约8.79秒exit0；协议、真实信号、取消/父死亡/回执、独立后代等待、全部PTY及Bash fg/bg/持续后台fg均PASS。唯一SKIP为只能作为子进程进入的CompletionErrorChild辅助入口，不是缺失正常用例；不把WSL测试视为原生产品验收。
README和离线manual将过时“Ctrl-Z未实现”改为“已接入且通过WSL Bash/PTY；停止取消竞态、其他Shell、原生Linux/macOS仍待验收”。gofmt并实际go run help manual核对新文本成功。未重建发布制品或下载。
当前摘要中的旧“完整Ctrl-Z尚未实现”以本条替代：基本协议已实现并集中回归，尚未完成全部平台/竞态验收，目标继续进行。

最新进度：前台调用端SIGTSTP转发。上一轮为progress（runner集中回归与文档）。终端会话的调用端和监督端现在额外Notify SIGTSTP，沿自有子树传递；等子进程真实停止后再通过既有Stopped消息暂停调用者。非终端路径不新增该订阅，避免暗中改变后台无终端调用行为。
验证：新TestLinuxTerminalStopBySignal只向调用端发TSTP；父测试确认调用端停止，读取自有Node/proc状态必须T后才CONT。原Ctrl-Z场景也加强相同子进程停止断言，避免只暂停调用端的假阳性。Linux CGO0编译.build/runner-directed-stop-linux.test；获准WSL两项PASS0.64/0.58秒、exit0，无下载/发布。暂停取消竞态与完整原生平台验收仍开放。

最新进度：SIGTSTP握手窗口修复。上一轮为progress（调用端TSTP转发）。监督器原在发Ready后、解析终端请求时才Notify TSTP，调用端此时已可能转发。现将私有监督器TSTP订阅与其他信号一起前移至Ready之前；公开非终端调用端仍不新增TSTP订阅。
验证：新TestSupervisorReadyCatchesStop启动真实独立进程组监督器，收到Ready后读取该自有PID的SigCgt位确认TSTP已捕获，再发送TSTP及无效相对路径请求，监督器正常回复错误并退出，未启动用户子进程。gofmt、Linux CGO0编译.build/runner-stop-handshake-linux.test；获准WSL定向PASS0.01秒/exit0。无下载/发布；其余暂停取消竞态与原生平台验收继续开放。

最新进度：暂停调用者被强杀时的独立清理。新增TestLinuxTerminalStopOwnerKilled，隔离PTY写Ctrl-Z，waitpid确认调用者已停止，并确认真实Node的/proc状态为T；只向该测试调用者发SIGKILL并核对终止信号。保持PTY打开，等待Node的/proc条目消失，排除终端关闭导致清理的假阳性。未设置完成回执，因此本项只证明暂停中的子进程被回收，不扩张为回执恢复集成证据。
验证：Linux CGO0测试制品.build/runner-stopped-owner-kill-linux.test；获准WSL Ubuntu、LC_ALL=C、保留Node定向测试PASS0.37秒/exit0。无生产代码改动、下载或发布。同步修正本文件顶部已过时的Ctrl-Z待办；暂停取消竞态及原生平台验收仍开放。

最新进度：暂停期间待处理TERM的恢复验证。上一轮为progress（暂停调用者强杀清理测试）。新增TestLinuxTerminalStopPendingTerm：真实PTY Ctrl-Z后确认调用者和Node均停止，仅向调用者发TERM；读取自有调用者/proc/status，合并SigPnd/ShdPnd明确断言TERM处于待处理状态，然后CONT。真实Node的TERM处理器以23退出（重复接收会返回其他码），测试继续验证调用者前台恢复、再次读取与最终23，不额外发送终止信号。
验证：gofmt、Linux amd64 CGO0编译.build/runner-pending-term-linux.test；获准WSL Ubuntu LC_ALL=C及保留Node定向测试PASS0.63秒/exit0。未修改生产逻辑，未重跑不受影响的全套测试。此证据只覆盖恢复后处理已排队信号；调用者暂停期间自身Go定时器不能运行，暂停取消竞态仍未解决。工作目录git status仍明确非Git仓库，无下载或发布。

最新进度：作业控制回调失败后的子树清理。上一轮为progress（暂停TERM恢复验证）。审查发现reaper停止/继续回调出错会直接返回未确认，可能遗留已启动子树。现在保存controlErr并进入既有SIGKILL/回收循环，停止进一步作业控制回调，只有观察leader终止及ECHILD才返回complete=true，同时保留原始错误；清理自身出错仍保守返回未确认。没有用错误本身推断完成，也没有放宽租约证据。
验证：新增独立进程TestJobControlFailureReapsTree，真实shell启动sleep后自行SIGSTOP，分别注入停止和继续回调错误，断言原始错误、leader SIGKILL、complete及额外Wait4 ECHILD。Linux CGO0制品.build/runner-control-error-linux.test，获准WSL两项共0.02秒PASS；受影响取消0.31秒及真实Bash ForegroundResume0.61秒PASS/exit0。随后仅补解释注释，无需重复执行。此项为回调故障注入，不冒充真实终端设备故障；暂停取消竞态、原生平台与完整性能门禁仍开放。无下载或发布。

最新进度：取消不再掩盖监督错误。上一轮为progress（终端回调失败清理）。审查发现errors.Is(Canceled)会把同时发生的终端/回执错误当作普通取消，调用端可能丢弃错误。reaper在纯取消时返回原ctx错误，回调错误与取消并存时保留二者；监督器只对纯context.Canceled设置Canceled，终端恢复或回执写入失败均清除此标志，因此执行器仍报告故障。
验证：新增真实只读回执失败TestCanceledReceiptFailureIsReported，等待shell输出就绪后取消，通过公开Execute断言实际bad file descriptor未被取消掩盖且树完成已确认；故障回调测试增加stop-cancel同时错误并断言两项原因。Linux CGO0编译.build/runner-control-cancel-linux.test；获准WSL定向回执失败0.02秒、完成/监督器死亡回执0.03秒、三项回调故障0.03秒、暂停树取消0.02秒、真实PTY Interrupt及其取消恢复0.59秒全部PASS/exit0。未模拟真实终端设备故障，未声称全部竞态完成；无下载、发布或预算变更。

最新进度：Windows实际Sync运行中探测进程强杀恢复。上一轮为progress（取消不掩盖监督错误）。复用早期Sync崩溃fixture新增DuringBackendProbe：实际Service.Sync通过生产UV.VerifyVersion/observer/Windows Job启动测试可执行文件的阻塞--version入口，收到子进程PID后确认已有一条子树登记、持锁期间clean超时；强杀Sync所有者后，保留的SYNCHRONIZE进程句柄由WAIT_TIMEOUT变为WAIT_OBJECT_0，直接证明探测进程退出；只读预览及实际回收1项、字节一致、未生成声明锁均通过。
边界：该后端是不会返回成功版本的阻塞测试进程，不是真实uv；补齐实际Sync→后端启动→Job清理→状态恢复链，真实uv运行中强杀仍待验证，不替代原生完整平台门禁。首版新增测试0.40秒通过；加入进程句柄断言时WAIT_TIMEOUT类型不匹配编译失败，显式uint32修正后BeforeBackendLaunch0.34秒、DuringBackendProbe0.41秒均PASS，包0.907秒。无生产改动、下载、用户环境修改或发布。

最新进度：真实Windows uv运行中Sync强杀验证。上一轮为progress（阻塞探测fixture集成）。新增TestSyncKilledDuringRealUVRequest，读取保留Python/固定uv记录，使用独立项目和预选Python版本锁，经实际Service.Sync建立准备操作、创建环境并由真实uv请求本地直接wheel依赖。httptest服务器收到指定URL后保持响应阻塞；此时只读数据库确认一个Tracked所有者及恰好一条未完成子树。强杀自行启动的Sync所有者并Wait，服务器观察uv连接取消，随后Clean预览1项、实际恢复准备操作1项并删除1项，字节与预览一致。恢复生产路径仍要求所有者出生身份退出及所有已登记命名Job消失，不仅凭网络断开判断。
验证：gofmt，Windows go test ./internal/core -run=^TestSyncKilledDuringRealUVRequest$ -count=1 -v -timeout=45s，MYENV_TEST_PYTHON_RECORD指向.build/python-real/python-prepared.json；测试PASS2.27秒，包2.423秒。没有运行时下载、外部依赖下载或构建脚本执行，服务器不返回wheel。此证据补齐真实uv依赖请求阶段运行中强杀；首次版本探测执行瞬间、完整原生平台矩阵及性能门禁仍非本项覆盖。未修改生产逻辑或发布。

最新进度：真实uv强杀后保留旧活动代。上一轮为progress（首次Sync真实uv请求中断）。扩展同一fixture新增TestSyncKilledRealUVPreservesActive：先以空依赖项目实际Sync成功建立活动代，再向pyproject加入本地阻塞wheel依赖；第二次Sync真实uv发请求后强杀所有者，完成既有子树登记/连接断开/预览与恢复1项断言。清理后SelectRun(current=true)必须返回基线活动ID，并使用该代真实Python、正式TreeID租约与runner执行版本命令，退出0且版本匹配，Finish释放成功。
验证：gofmt，保留MYENV_TEST_PYTHON_RECORD，Windows go test ./internal/core -run=^TestSyncKilledRealUVPreservesActive$ -count=1 -v -timeout=45s，PASS3.75秒，包3.916秒。旧首次同步场景保留独立入口；未重复已通过且未改变行为的首次场景。该项证明崩溃清理不替换/删除原活动代且原解释器可运行，不扩张为项目业务数据回滚。无生产改动、外部下载或发布；原生其他平台与完整性能门禁仍开放。

最新进度：更新本地三目标制品。上一轮为progress（真实uv崩溃保留活动代）。执行scripts/build-release.ps1 -OutputDirectory .build/supervision-recovery-release，Go固定现有配置CGO0、trimpath、buildvcs=false、-s -w、0.1.0-dev，三目标成功。Windows12758016字节 SHA 3b6a9ccbac035e2d69d7e4d799dd75f1f8acec06a934bb03d243fc25f34a13d2；Linux12427426字节 SHA f3756fb44958f974a397f445fedf97542078fec7d3556d2873204486ce14a3cb；Darwin11915474字节 SHA ed208cb4c7d05f117ec733a462b5f2577dc6b3267f6dafe9b12f7f00e215762f。
随后逐项重新计算文件SHA及长度匹配manifest，核对SHA256SUMS文本与manifest全部条目一致；直接运行新Windows制品--version成功、help manual成功且包含Ctrl-Z。本次未运行Linux/macOS制品，不以交叉编译证明原生支持；新SHA完整性能未测，旧预算失败结果不转移为新SHA实测。旧制品全部保留，未签名或公开发布；同步修正当前摘要中的过时Windows uv集成待办。

最新进度：新Windows制品启动/run采样。上一轮为progress（三目标构建与校验）。使用当前Go测试测量入口、.build/supervision-recovery-release/myenv-windows-amd64.exe及保留Node，串行运行TestMeasureFullRunCLI与TestMeasureInformationalCLI；两项51组、首组剔除、50组nearest-rank p95，run仅外部CLI与直接Node配对。run附加中位63.0996ms/p95 84.9460ms，失败50ms；help43.0650/63.2840ms、version42.1019/68.3035ms，失败30ms。测试12.73/4.66秒PASS、包17.540秒仅表示采样完成。
报告.build/perf/run-supervision-recovery-windows.json和startup-supervision-recovery-windows.json，分别cli_sha256/sha256与新Windows制品现场重算SHA 3b6a9ccbac035e2d69d7e4d799dd75f1f8acec06a934bb03d243fc25f34a13d2一致。没有清OS缓存，不是冷启动验收；同SHA完整sync/status/RSS/首次反馈矩阵仍缺。旧宿主最小Go基线已提示启动尾延迟，不再次重复基线或据此降低预算。下一步需要进一步定位可改开销及参考宿主证据，未发布或修改生产逻辑。

最新进度：合并run选代读取。上一轮为progress（当前SHA性能采样，预算失败）。检查选代事务发现AcquireActive先读取活动代再调用loadPythonEntry做第二条查询。现当generation_python存在时用LEFT JOIN与COALESCE在同一条查询读取完整代，Node-only保持空Python入口；旧表结构分支保持原查询。没有修改出生身份、随机租约、两条登记写入、事务提交、释放CAS或synchronous(FULL)。每次正常选代少一条SQL，但尚未测得端到端改善，不把结构性减少当预算通过。
验证：gofmt；Windows state模块全部普通测试PASS6.866秒；保留Node的TestRunRetainedNode真实Node/npm/npx/argv/退出码回归PASS1.40秒、包1.550秒。状态测试覆盖Python/混合代与租约选择、清理/恢复保护；计时类门控未启用。此源码变化晚于supervision-recovery-release，不改写该制品及其既有SHA测量事实；下一次候选构建需纳入此变更。无下载或发布。

最新进度：选代合并查询的Linux组件验证。上一轮为progress（减少选代SQL且Windows状态/真实Node回归通过）。重新读取AcquireActive和对应测试，Linux amd64 CGO0编译.build/state-joined-selection-linux.test；获准WSL临时数据库定向执行CleanReservationProtectsReferences0.04秒、LeaseRecoveryObservationAndPaging0.01秒、LeaseRetainsSelectedGeneration0.01秒、PythonGenerationPublication0.01秒，全部PASS/exit0。混合代完整入口比较、无活动代错误、出生身份错配拒绝释放、活动切换后旧租约保护、清理预约及恢复分页均包含在所选用例中。
未重复Windows测试或性能采样；这是Linux状态组件证据，不是原生Linux产品或真实Linux Python运行时验收。未修改生产源码、下载或发布；最新选代源码仍晚于现有三目标制品。

最新进度：安装/网络手册对照。上一轮为progress（Linux状态组件回归）。核对方案第270行要求四个手册专题，发现离线manual缺少代理/镜像/凭据专题，且仍写full recovery pending。现补充源码可确认的当前边界：后端继承普通环境（包含HTTPS_PROXY），uv过滤继承UV_*；内部URL注入不等于CLI镜像配置，用户镜像/自定义CA接口仍待实现；凭据不写入声明，运行时传输错误去除含凭据URL。profile段明确直接Shell查找仍缺PATH集成，已实现入口为run --global，移除笼统恢复未实现说法。
README将共享解释器卸载未完成改为准确的缓存清理不卸载解释器，避免暗示首版需要额外自动卸载功能。gofmt并实际go run ./cmd/myenv help manual验证新专题渲染成功。仅帮助/说明改动，未重复进程测试或宣称网络设置验收；镜像/CA明确配置、用户默认工具显式Shell集成仍是方案开放项，不因文档补齐而标记功能完成。

最新进度：显式Node镜像入口。上一轮为progress（手册对照暴露配置缺口）。安装服务读取MYENV_NODE_MIRROR，验证为HTTP(S)绝对基址、拒绝凭据/query/fragment且错误不回显输入；复用原Node客户端、解析、锁和摘要校验链，仅设置基址。未设置时仍由core创建官方Node后端；已有锁定URL不改写，不以设置镜像触发隐式更新。只在runtimeService读取，帮助/版本/run不新增读取或网络行为。uv镜像与自定义CA仍待实现。
验证：gofmt；本地httptest从/dist/index.json与版本SHASUMS解析并验证制品URL及SHA；非法/含秘密URL拒绝且不泄漏、空值默认及共享存储构造回归全部PASS，包0.174秒。该项为真实HTTP元数据fixture，不是真实Node归档下载验证；未下载外部数据。README/manual说明布局、锁行为与来源信任边界。无新依赖或发布。

最新进度：Node镜像真实CLI同步闭环。上一轮为progress（MYENV_NODE_MIRROR入口及元数据校验）。新增TestNodeMirrorRealSync，用保留Windows Node22.23.2真实ZIP由本地HTTP镜像按官方布局提供；通过execute的真实sync --no-input路径和隔离用户存储完成下载、原SHA校验、解压、验证与发布。读取实际myenv.lock断言镜像制品URL及原归档SHA，关闭镜像后调用CLI run node --version必须输出v22.23.2。没有手工创建状态代或通过Service.Node注入绕过新配置入口。
验证：gofmt，保留MYENV_TEST_PREPARED_RECORD，Windows go test ./internal/cli -run=^TestNodeMirrorRealSync$ -count=1 -v -timeout=60s，PASS18.41秒、包18.558秒。仅本机传输已有归档，无外部下载/发布。uv镜像、CA、显式Shell集成及完整平台/性能门禁仍待完成。

最新进度：固定uv引擎镜像。上一轮为progress（Node镜像真实CLI闭环）。MYENV_UV_MIRROR通过runtimeService传入Service.UVMirror，managedUV无已安装入口/内部归档时调用DownloadUVFromMirror；基址下拼固定版本/官方归档名，固定发布SHA与PrepareUV再次校验不变。原DownloadUV保持默认包装，Node与uv共用不回显秘密的MirrorBaseURL验证。缓存复用不因镜像改变而重装；Python运行时源独立，尚未提供镜像入口。
验证：Node显式镜像/非法值/存储回归PASS0.170秒；新增uv配置绑定与凭据拒绝PASS，真实本地HTTP返回错误字节时下载报错、版本/SHA等于内置发布、请求路径正确、临时目录无残留归档；固定三平台发布信息测试通过。backend包0.164秒、cli包0.141秒。README/manual已同步。尚未在本轮用真实uv归档完成镜像安装，后续需补该集成；无外部下载、新依赖或发布。

最新进度：真实uv镜像安装与离线复用。上一轮为progress（固定uv镜像入口和错误字节拒绝）。新增TestManagedUVMirrorRetained，以保留uv0.11.26真实官方归档经本地HTTP版本路径提供，Service.UVMirror驱动managedUV实际下载、固定摘要校验、解压、版本验证和共享入口发布。断言只下载一次，关闭服务器后另一个项目上下文复用同一可执行入口；两个项目目录均未被共享引擎安装创建。
验证：gofmt；读取.build/uv-real/uv-download.json的Archive到MYENV_TEST_UV_ARCHIVE，Windows go test ./internal/core -run=^TestManagedUVMirrorRetained$ -count=1 -v -timeout=45s，PASS2.03秒、包2.191秒。此为核心实际安装链，CLI环境变量绑定已有上一轮独立证据；没有冒充完整Python项目镜像同步。无外部下载或发布；Python运行时镜像、自定义CA及Shell集成继续开放。

最新进度：CPython镜像入口。上一轮为progress（真实uv引擎镜像安装）。检查保留uv0.11.26 python install --help确认--mirror，并核对官方 https://docs.astral.sh/uv/reference/cli/ 的基址替换约定。新增MYENV_PYTHON_MIRROR，经共享URL验证进入Service.PythonMirror，再由InstallPythonFromMirror显式追加--mirror；原InstallPython保留默认包装。不继承任意UV_*设置，不修改版本选择、共享锁或Python版本证据。
验证：真实uv请求测试用临时空运行时目录、本地HTTP镜像；收到/python/下包含目标cpython版本的请求后取消，安装返回错误、监督完成。Windows PASS0.18秒、包0.293秒。CLI绑定测试加入Python基址断言并通过。README/manual说明目录布局、已安装解释器复用和版本锁边界。没有提供归档响应或完成本次Python安装，不冒充完整镜像安装证据；自定义CA与Shell集成继续开放。无外部归档下载或发布。

最新进度：真实CPython镜像安装。上一轮为progress（镜像请求/取消与CLI绑定）。检查原python-real缓存仅有解释器元数据等、无原始归档；固定uv离线python list返回官方20260623 CPython3.12.13 Windows归档URL。获准下载该官方文件到.build/python-mirror-real/cpython-3.12.13-20260623-windows.tar.gz，经.partial完成后改名，21931932字节，现场SHA256 de3e362376859b060fa8b856c434efa81fcf6d4ede3d6e177c7e2169670cac50（本地留存标识，不宣称独立来源认证）。
新增TestPythonMirrorRealInstall，真实uv由本地/python基址下载该归档到独立空运行时目录，实际安装与版本验证通过，断言一次下载；关闭镜像后同路径复用成功，再创建真实venv并通过现有验证。gofmt，MYENV_TEST_PYTHON_RECORD及MYENV_TEST_PYTHON_ARCHIVE明确指向保留记录/归档，Windows go test ./internal/backend -run=^TestPythonMirrorRealInstall$ -count=1 -v -timeout=45s，PASS8.43秒、包8.558秒。归档保留供复用，未修改系统Python或发布；这是后端安装链，不替代全部CLI多镜像组合、CA和平台门禁。

最新进度：uv/Python双镜像CLI闭环。上一轮为progress（真实Python后端镜像安装）。新增TestPythonMirrorsRealCLI，隔离空用户存储与项目，仅由MYENV_UV_MIRROR和MYENV_PYTHON_MIRROR驱动CLI sync --no-input --json，真实官方保留归档各下载一次；断言成功/changed及活动代ID。关闭本地服务器后sync --locked --no-input --json必须成功、changed=false、同代；run python --version输出Python3.12.13。没有通过Service注入运行时或预置代跳过安装路径。
验证：gofmt后初次go test构建临时cli.test.exe遭文件占用并明确exit1，未进入测试；改用go test -c -o .build/cli-python-mirrors.test.exe构建独立制品成功，再运行定向用例PASS10.86秒/exit0。复用MYENV_TEST_UV_ARCHIVE与MYENV_TEST_PYTHON_ARCHIVE，无外部下载。已覆盖双镜像空存储安装/离线无变更/运行；项目依赖源、自定义CA、Shell集成和完整平台预算仍待验收。

最新进度：显式PEM信任源。上一轮为progress（双镜像CLI）。核对官方uv证书文档 https://docs.astral.sh/uv/concepts/authentication/certificates/，SSL_CERT_FILE为替换默认信任源的标准入口。新增backend.DownloadClient：绝对路径普通PEM文件、4MiB有界读取、有效证书校验，新CertPool与独立克隆Transport；不修改DefaultTransport、不关闭TLS验证。runtimeService将其用于Node（与镜像共存），uv引擎下载也使用；uv子进程既有环境继承该变量。SSL_CERT_DIR未接入Go下载器，文档明确边界。
验证：自签名HTTPS指定PEM成功、默认客户端仍拒绝（测试服务端bad certificate日志为预期）、无效PEM及相对路径拒绝；uv错误归档与镜像配置回归通过，backend0.363秒/cli0.167秒。README/manual更新。尚未完成真实uv携带该CA下载及HTTPS多镜像CLI集成；不能把客户端单测视为全部TLS门禁。无系统信任库修改、外部下载、新依赖或发布。

最新进度：真实uv与Python自定义CA CLI验证。上一轮为progress（SSL_CERT_FILE下载器）。扩展双镜像测试为独立HTTPS入口：用本次生成根CA签发127.0.0.1服务器叶证书，只把根PEM通过SSL_CERT_FILE提供，清空测试进程SSL_CERT_DIR；真实Go uv引擎下载及真实uv Python归档下载均由同一CA验证。空存储安装、各下载一次、镜像关闭后locked同代无变更及Python运行断言保留。
初次使用httptest默认CA兼服务器证书，Go接受但uv准确拒绝CaUsedAsEndEntity，测试FAIL9.47秒；改为有效独立根/叶链后，Windows TestPythonMirrorsCustomCARealCLI PASS11.58秒、包11.739秒。仅修正测试证书，未关闭验证/修改系统信任库。gofmt，复用两份已保留归档，无外部下载；Node HTTPS镜像组合及完整原生平台门禁尚未由本项覆盖。

最新进度：Node HTTPS镜像与CA组合。上一轮为progress（真实uv/Python HTTPS双镜像）。将现有真实Node镜像fixture保留HTTP入口并增加TestNodeMirrorCustomCARealSync；独立根CA签发服务器叶证书，SSL_CERT_FILE与MYENV_NODE_MIRROR同时设置。实际CLI从空隔离存储解析HTTPS元数据、下载保留Node归档、校验及发布，锁中URL/SHA保持准确，关闭镜像后运行Node版本成功，证明镜像设置未覆盖自定义TLS客户端。
验证：gofmt，MYENV_TEST_PREPARED_RECORD复用保留Node；Windows go test ./internal/cli -run=^TestNodeMirrorCustomCARealSync$ -count=1 -v -timeout=60s，PASS18.28秒、包18.439秒。未重复HTTP或uv已通过组合；无外部下载、系统证书修改或发布。完整原生平台与性能门禁及显式Shell集成仍开放。

最新进度：显式用户工具Shell集成。上一轮为progress（Node HTTPS组合）。新增shell-init <bash|zsh|fish|powershell>，只读已存在用户profile，为声明工具生成函数（Node含npm/npx），通过引用的绝对myEnv路径调用run --global并转发参数。保持现有选代/租约/监督路径，不导出易失效代目录PATH、不安装或编辑Shell文件；拒绝--json，文档解释同名覆盖、移除、重新加载及子程序PATH边界。这是显式Shell函数集成，未宣称系统级shims或项目自动切换。
验证：四Shell代码输出按profile选择工具且不创建状态；真实PowerShell子进程执行生成包装，含空参数、空格、字面$和分号、含单引号/空格/$的可执行脚本路径均保留，PASS0.74秒，配置只读0.01秒，包0.927秒。尚未实际运行Bash/Zsh/Fish包装或用真实已应用profile执行该新入口；需继续补对应集成。README/manual同步，无用户Shell或PATH修改、下载或发布。

最新进度：Bash包装真实执行。上一轮为progress（shell-init与PowerShell参数验证）。新增Linux TestBashProfileWrapperArgumentsAndExit，在临时目录创建含空格/单引号/$的探测入口，生成正式Bash包装；用/bin/bash --noprofile --norc执行，NUL分隔检查run/--global/node前缀及空参数、空格、字面变量/分号、通配符不展开，并断言原退出23。不是仅比较生成文本。
验证：gofmt、Linux amd64 CGO0构建.build/cli-shell-init-linux.test；获准WSL定向PASS0.01秒/exit0。未修改用户Shell配置、下载或重跑无关测试；Zsh/Fish与实际已应用profile包装仍待集成，WSL不替代原生产品验收。

最新进度：shell-init失败合同。上一轮为progress（真实Bash argv/退出码）。缺失profile从泛化IO错误改为ENV_NOT_READY/1并明确提示myenv use --global <tool>@<version>；保留原始路径错误链。新增五项CLI验证：缺profile、未知Shell、缺参数、多参数、--json；普通失败stdout无部分Shell代码，JSON失败仅一个ok=false/changed=false/USAGE_ERROR对象且stderr空，各场景不创建profile/state。
验证：gofmt，Windows TestShellInitFailures五项PASS0.01秒、包0.190秒。未重复成功Shell执行测试或下载；Zsh/Fish和真实已应用profile包装集成仍待补，完整平台与性能验收继续开放。

最新进度：PowerShell真实已应用profile集成。上一轮为progress（shell-init失败合同）。扩展双镜像fixture增加TestPowerShellAppliedProfile：隔离用户profile从空存储实际sync --global安装uv/Python，关闭镜像后locked同代无变化与run --global版本通过。调用正式shell-init powershell生成脚本，独立pwsh -NoProfile执行python函数；测试可执行文件仅在专用子进程环境且argv首项run时转入实际CLI并注入隔离namespace，其余选代/租约/监督保持生产路径。真实Python输出3.12.13并exit23，PowerShell最终状态23。
验证：gofmt，保留UV/Python归档，Windows go test ./internal/cli -run=^TestPowerShellAppliedProfile$ -count=1 -v -timeout=60s，PASS12.06秒、包12.220秒。未修改真实profile或Shell、无外部下载。此为真实运行时及CLI集成，入口宿主为测试二进制，不冒充已发布制品验收；Zsh/Fish及其他原生平台仍待验证。

2026-09-09接续：其他Shell执行入口。上一轮为progress（PowerShell真实profile）。只读WSL检查确认当前只有Bash，没有zsh/fish。将实际argv/退出码fixture共享到Zsh与Fish独立测试；按官方调用说明使用zsh -f、fish --no-config --private，Fish使用$status，缺失程序明确SKIP。参考 https://zsh.sourceforge.io/Doc/Release/Options.html 与 https://fishshell.com/docs/current/cmds/fish.html；zsh -f仍可能读取系统/etc/zshenv，不宣称完全不读系统配置。
验证：gofmt、Linux CGO0编译.build/cli-shell-matrix-linux.test；获准WSL Bash在调整后的共同fixturePASS0.01秒，Zsh/Fish明确SKIP（未安装）。这是准备可执行验证入口并确认宿主缺口，不是两种Shell兼容通过；未安装Shell、修改配置或发布。全目标继续，尚有其他可推进的生产/验收工作，不属于阻塞终止。

最新进度：自定义TLS连接池收尾。上一轮为progress（其他Shell入口与缺失宿主证据）。SSL_CERT_FILE创建独立Transport后，CLI同步/use返回时此前未关闭空闲连接，嵌入宿主重复调用可暂留连接。新增closeRuntimeService，仅关闭该调用自定义Node Transport的空闲连接，不关闭进程共享DefaultTransport；sync/use/缓存clean构造成功后defer执行。uv独立下载客户端已有对应收尾。
验证：真实独立根/叶HTTPS连接完整读取响应后仍保持存活；显式服务收尾后服务端ConnState确认关闭，重复收尾安全，TestRuntimeServiceClosesCustomTLSConnections PASS0.02秒、包0.204秒。gofmt，无下载或发布，未重新跑不受影响的完整归档安装。该修复减少调用结束后的连接残留，不声称解决所有资源/性能预算。

最新进度：CA客户端类型断言失败改为错误。上一轮为progress（自有TLS连接收尾）。DownloadClient此前直接断言进程DefaultTransport为*http.Transport，嵌入宿主替换为其他RoundTripper或nil时会panic。现使用受检断言并拒绝不能配置CA的transport，不绕过CA或替换宿主全局对象。
验证：新增opaque RoundTripper、typed-nil *Transport和nil三项，均返回错误及nil客户端；测试恢复全局变量，再跑实际HTTPS显式CA/默认拒绝回归。Windows PASS0.01/0.20秒，包0.349秒，gofmt完成。预期bad certificate服务端日志来自默认信任拒绝。无下载或发布；本项不改变普通CLI默认Transport行为或重跑归档安装。

最新进度：近期配置/后端/CLI集中回归。上一轮为progress（CA Transport错误边界）。镜像、CA、shell-init跨模块变更后执行go test ./internal/config ./internal/backend ./internal/cli -count=1 -timeout=90s，backend0.638秒、cli2.179秒通过；config两项在测试前置filepath.EvalSymlinks读取自有TempDir时Access is denied，集中命令exit1，不能记为整体通过。
查看失败位置后仅对TestWindowsWorkspacePath和TestWithinRootAndSymlinkBoundary申请授权重跑，普通路径/大小写/缺失路径、junction越界/项目来源/workspace成员、长路径均PASS，包0.815秒；真实symlink子项仍因缺少系统权限明确SKIP。无修改测试预期或生产逻辑，不重复已通过模块。真实归档/性能等门控本次未启用，沿用各场景独立证据；无下载或发布。

最新进度：镜像/Shell新三目标制品。上一轮为progress（集中回归及权限定向复核）。scripts/build-release.ps1 -OutputDirectory .build/mirrors-shell-release成功，纳入选代合并、三个镜像入口、SSL_CERT_FILE、TLS收尾和shell-init。Windows12794880字节 SHA256 36d7b3ba23525f17110fb61e8e379b053d654be5db81187f2b4ad914343edfbc；Linux12452002字节 dc9e12b82465d44e217f78d8cf6d583f0275382d77d7cc9d15e9e6f69695f731；Darwin11948850字节 20ff5fe9ead29211b638178ff546ecbfa6e719fb2c42b05871918c6f00369923。
重新计算三项哈希/大小匹配manifest，SHA256SUMS逐条一致；直接运行新Windows制品--version、help manual、help shell-init成功，手册包含所有镜像变量/CA/Shell主题。Linux/macOS仅交叉编译，不宣称原生支持；本批SHA尚无性能报告，上一批预算失败仍是历史证据。旧制品保留，未签名或公开发布。

最新进度：新制品真实Shell/profile运行。上一轮为progress（镜像/Shell三目标制品）。扩展PowerShell已应用profile测试的可选MYENV_TEST_SHELL_CLI_EXECUTABLE，实际启动该制品shell-init生成包装，再由PowerShell包装调用同一制品run --global，测试入口分发不参与这两步。只对子进程APPDATA指向隔离namespace，清空测试入口变量；调用设10秒超时。安装fixture仍由当前测试CLI和保留归档建立真实profile。
验证：gofmt，指定.build/mirrors-shell-release/myenv-windows-amd64.exe（已验证SHA36d7b3ba23525f17110fb61e8e379b053d654be5db81187f2b4ad914343edfbc），TestPowerShellAppliedProfile PASS11.97秒、包12.127秒，真实Python输出3.12.13且PowerShell退出23。没有修改用户APPDATA或系统配置，无外部下载。此补齐制品shell-init/run边界，不冒充安装阶段也由制品执行；其完整性能与平台门禁仍待完成。

最新进度：CA输入限制验证。上一轮为progress（发布制品Shell/profile运行）。核对DownloadClient文档与源码，新增TestCustomCABundleInputBounds，实际空文件、目录、4MiB+1文件、缺失文件均拒绝并返回nil客户端；超限文件用Truncate创建避免测试自身大缓冲，缺失文件保留os.IsNotExist原因。配合既有真实TLS与无效PEM测试补充输入边界证据，不声称覆盖文件被并发替换竞态。
验证：gofmt，Windows定向四项PASS0.01秒、包0.145秒。没有创建网络请求或修改生产代码，不重复归档安装/性能测量。新制品SHA仍有效；全目标平台与性能门禁未完成。

最新进度：当前镜像/Shell制品延迟。上一轮为progress（CA输入边界）。当前Windows制品SHA36d7b3ba23525f17110fb61e8e379b053d654be5db81187f2b4ad914343edfbc按既有Go os/exec测量，51组首组剔除，余50组nearest-rank p95：run配对直接Node附加中位61.9731ms/p95 82.0752ms（失败50ms）；help45.1518/65.3240ms、version45.0860/69.2461ms（失败30ms）。两测试12.60/4.73秒PASS、包17.483秒只表示采样完成。
报告.build/perf/run-mirrors-shell-windows.json与startup-mirrors-shell-windows.json，现场重算制品SHA与两报告字段均匹配。上一批与本批不是交替A/B，不能从p95差值归因选代SQL优化；没有稳定达标证据，不降低预算。无缓存清除，不是冷启动矩阵；同SHA其他命令/RSS/整树资源仍待测。无生产改动、下载或发布。

最新进度：当前制品run父进程内存。上一轮为progress（当前SHA延迟采样失败）。使用scripts/measure-memory-windows.ps1及保留Node22.23.2 fixture，TestMeasureFullRunMemory对当前mirrors-shell-release Windows制品执行20次run node -e空脚本，CLI最大峰值工作集12.2227MiB，满足32MiB父进程子集预算。测试PASS5.76秒、包5.910秒。
报告.build/perf/run-memory-mirrors-shell-windows.json，现场重算制品SHA与报告一致：36d7b3ba23525f17110fb61e8e379b053d654be5db81187f2b4ad914343edfbc。采样保持进程句柄读取PeakWorkingSetSize，不含Node/子树或测量宿主，不是整树RSS门禁，也不抵消同SHA延迟失败。无生产改动、外部下载或发布；完整命令/平台资源矩阵继续开放。

最新进度：Shell包装生命周期说明。上一轮为progress（当前制品run内存证据）。审查profileShellScript确认只输出当前工具定义，不清除之前加载的包装或保存用户同名函数。README/manual明确删除profile工具/卸载myEnv后开启新Shell，重新加载不是完整重置；不声称会自动恢复被覆盖函数。保留用户显式启用模型，未加入无所有权判定的函数删除。
验证：实际go run ./cmd/myenv help manual成功并检查新生命周期说明渲染。仅文档/离线手册更新，未重复进程测试或重新构建发布制品；因此当前源码手册晚于mirrors-shell-release，功能制品指标仍按原SHA使用。完整目标继续，自动函数清理不是本项实现结果。

最新进度：镜像/CA配置可操作诊断。上一轮为progress（Shell生命周期说明）。审查发现sync已将镜像语法错误分类为INVALID_CONFIG，但next_action仍指项目文件；缓存clean也只建议重试清理。新增保留cause的environmentConfigError标记三个镜像变量和SSL_CERT_FILE；两条错误输出路径给出修正或取消对应调用环境变量的具体动作，文件IO原因仍可由errors.As识别，未改变既有退出类别。
验证：四变量分别在sync JSON与clean --cache uv JSON下输入无效值，八场景均失败、changed=false、下一步含对应变量、不回显测试秘密值、无用户命名空间状态创建。gofmt，Windows定向PASS0.02秒、包0.177秒。无下载或发布；当前源码诊断已晚于mirrors-shell-release，原制品测量不冒充本次更新后的结果。

最新进度：同一制品混合项目无变更同步延迟。使用保留 Node/Python 记录及 MYENV_TEST_CLI_EXECUTABLE、MYENV_TEST_MIXED_NOOP_TIMINGS，运行 go test ./internal/core -run=^TestMixedProjectSyncRetained$ -count=1 -v -timeout=90s，PASS24.42秒、包24.585秒。实际 Node+Python 项目含 pyproject.toml、uv.lock 和已安装纯 Python wheel；测量前关闭本地制品服务。外部 CLI 执行 sync --locked --no-input --json 共51次，每次检查成功、changed=false及同一活动代；首个82.3509ms单列，余50次中位58.1790ms、nearest-rank p95 63.6340ms，满足200ms无变更同步延迟预算。
报告 .build/perf/mixed-noop-mirrors-shell-windows.json 的 SHA256 与现场制品重算一致：36d7b3ba23525f17110fb61e8e379b053d654be5db81187f2b4ad914343edfbc。首条完整 stderr 反馈热 p95 35.2726ms、全样本最大54.3052ms，包含管道传递而非终端渲染。未清缓存，不是冷启动或内存测量；此前同 SHA 启动/run 超预算仍未解决。当前源码后续诊断改动不在该制品内。无生产代码修改、下载或发布；下一步继续性能缺口及原生平台验收，完整目标未完成。

最新进度：当前镜像/Shell制品状态查询延迟。上一轮为progress（无变更同步性能证据）。审查现有Status与测量入口，使用MYENV_TEST_MIXED_STATUS_TIMINGS及保留Node/Python记录，运行go test ./internal/core -run=^TestMixedProjectSyncRetained$ -count=1 -v -timeout=90s。真实混合依赖项目准备后关闭制品服务，外部CLI裸命令--json连续51次均返回ready、changed=false及相同活动代；前后state.db摘要相同。测试PASS22.80秒、包22.969秒只表示采样与行为验证成功。
报告.build/perf/status-mirrors-shell-windows.json：首个60.3172ms单列，剩余50次中位52.8043ms、nearest-rank p95 59.6860ms，失败50ms状态预算。报告与现场制品重算SHA一致：36d7b3ba23525f17110fb61e8e379b053d654be5db81187f2b4ad914343edfbc。未清缓存、不含准备时间，不是冷启动/内存验证；数据库字节不变证据不等于所有文件均未写入的审计。更新页首提示以避免旧摘要误导。无生产代码修改、下载或发布；后续需要定位/解决延迟缺口及原生平台验收，完整目标继续。

最新进度：活动代只读查询合并。上一轮为progress（状态查询超预算的同制品证据）。Store.Active此前先读取活动代，再单独查询Python入口；现对具备generation_python的库使用LEFT JOIN与COALESCE一次读取全部入口，避免第二次SQL执行，并在同一查询快照中取得代与入口。旧库无Python表时保留原四列查询，未引入写事务或租约。AcquireActive已有同类查询，本轮不改其事务逻辑。
验证：gofmt；go test ./internal/state ./internal/core -run=^(TestPythonGenerationPublication|TestReadOnlyLegacyNodeDatabase|TestReadOnlyStore|TestPublishPreservesActiveOnConflict|TestStatusReadOnly)$ -count=1 -v -timeout=60s。五项通过，state0.847秒、core0.374秒，涵盖混合代Python入口持久化、旧库无表兼容、空活动引用、只读拒写、发布冲突与状态not_ready/incomplete/ready/drifted及数据库字节不变。未重新构建或测量完整CLI，不声称达到50ms预算；现有制品报告不包含本次改动。无下载、用户环境修改或发布。完整目标继续。

最新进度：活动代合并后Windows制品与状态延迟。上一轮为progress（Store.Active生产查询合并）。运行scripts/build-release.ps1 -Targets windows-amd64 -OutputDirectory .build/active-read-release，Windows12796928字节，SHA256 fdf48f50f074f639466430bf63370406279ef073260b8acbb700baa729f7e8f6；重算大小/哈希与manifest、SHA256SUMS一致，实际--version成功。本批仅Windows，纳入此前手册/环境诊断与Active变更，不冒充三目标构建。
设置MYENV_TEST_CLI_EXECUTABLE指向新制品、MYENV_TEST_MIXED_STATUS_TIMINGS指向.build/perf/status-active-read-windows.json，复用保留Node/Python记录，运行go test ./internal/core -run=^TestMixedProjectSyncRetained$ -count=1 -v -timeout=90s，PASS23.42秒、包23.553秒。51次真实混合项目外部CLI状态均ready、同代、changed=false，数据库字节不变；首个70.0665ms单列，余50次中位51.8635ms/p95 57.5821ms，仍失败50ms预算。报告SHA与新制品核对一致。不是与旧制品交替配对测量，不能据差值归因SQL合并。没有下载或发布；后续仍需定位整体启动/状态延迟与原生平台缺口，完整目标继续。

最新进度：只读打开去掉重复连接探测。上一轮为progress（活动代合并后制品性能仍超预算）。openStore只读分支在必执行sqlite_master查询之前另做PingContext；删除重复探测，保留带context的结构查询、错误时Close、mode=ro和全部原有pragma。查询自身负责建立连接及验证数据库，未放宽旧库兼容或只读行为。
新增TestReadOnlyOpenFailures实际损坏文件和已取消context，分别拒绝打开并保留context.Canceled错误、原文件字节不变。gofmt，go test ./internal/state -run=^(TestReadOnlyOpenFailures|TestReadOnlyStore|TestReadOnlyLegacyNodeDatabase|TestPythonGenerationPublication)$ -count=1 -v -timeout=60s，四项通过，包0.630秒；既有测试同时覆盖缺失数据库不创建、只读拒写、旧库无Python表和混合入口持久化。未构建/测量新制品，不声称延迟达标；active-read-release早于本次改动。无下载或发布，完整性能/原生平台目标仍未完成。

最新进度：补齐无参数use交互入口。上一轮为progress（只读连接重复探测删除）。对照architecture第12节发现use仍固定ExactArgs(1)，未实现TTY选择。现在仅真实输入/输出终端且无CI、--json、--no-input时允许零参数，复用versionPrompt取得tool@version后调用原UseWithRequest；提示在runtimeService及声明编辑前执行。其余场景保留用法退出2，帮助描述可选参数与非交互限制。
验证：gofmt；go test ./internal/cli -run=Test(Use|VersionPrompt|Input|NonInteractive) -count=1 -v -timeout=60s实际运行VersionPrompt和UseFailureReportsDeclarationChange两项通过，包0.344秒。新增TestUseMissingSelectionNonInteractive，普通非TTY/--no-input/--json三场景均退出2、不消费预置输入、不创建隔离目录内容，定向通过0.01秒、包0.166秒。真实TTY完整入口尚未执行，不能将输入函数测试视为PTY验证，下一步补该场景。无下载或用户环境修改；完整目标未完成。

最新进度：Windows真实终端无参数use验证。上一轮为progress（交互入口实现及非TTY约束）。go build -o .build/myenv-use-interactive.exe ./cmd/myenv成功，调试制品SHA256 062862c0eede337b0c9cc83472df0da6e09023dd429d060086f66cd0c084ea8f；专用项目路径保留在.build/use-terminal-path.txt，声明初始node22、锁为预设损坏{}。exec_command tty=true启动实际CLI -C <fixture> use，子进程CI清空、APPDATA仅指专用目录；真实控制台显示Select the runtime to change和tool@version提示，向活跃session23775发送node@24回车。
进程退出2，输出INVALID_CONFIG与declaration_changed=true、lock_changed=false、native_lock_changed=false和活动环境未变提示。现场读取声明为node24、锁仍{}，.myenv只有modify.lock/state.db，未创建代目录。这验证终端检测、提示、输入到真实声明编辑与失败报告路径，不是成功安装验证。另一次真实TTY执行use --no-input无参数立即USAGE_ERROR/2，无交互提示；既有非TTY和JSON单元证据继续有效。没有下载或修改真实用户环境，调试构建不是发布/性能制品。完整目标继续，原生平台与性能等缺口未关闭。

最新进度：交互use取消路径调查。上一轮为progress（Windows真实TTY选择）。实际tty启动myenv-use-interactive.exe use，session13757提示后write_stdin发送U+0003，进程退出3、NEEDS_INPUT/input ended，没有进入同步。新增提示返回后cmd.Context().Err优先检查，已收到取消时不继续处理输入或调用runtimeService；构建.build/myenv-use-cancel.exe后同路径session51616复测仍退出3。故此改动不能记为Windows Ctrl-C修复；该PTY工具发送控制字符是否交付Console控制事件没有独立证据，仍需真实信号观察与可取消读取设计，不能把EOF等同取消。
现场声明仍node24，.myenv仍仅modify.lock/state.db。go test ./internal/cli -run=^(TestVersionPrompt|TestUseMissingSelectionNonInteractive)$ -count=1 -v通过；保留EOF为NEEDS_INPUT及非交互不消费输入合同。构建与两次进程均已终止，无后台遗留；无下载、安装或用户环境修改。下一步应解决/验证提示读取与取消，而非重复发送相同控制字符；完整目标继续。

最新进度：独立观测终端输入与信号。上一轮为progress（use输入返回后取消检查及未解决证据）。确认operationSignals注册os.Interrupt；检查本机Go internal/poll/fd_windows.go，控制台读取走ReadConsole。参考Microsoft ReadConsole文档 https://learn.microsoft.com/en-us/windows/console/readconsole ，没有据EOF推断控制事件已交付。
创建保留诊断源.build/console-signal-probe/main.go与对应.exe：注册容量1的os.Interrupt通道，直接os.Stdin.Read，读返回后最多1秒等信号。go build成功；真实tty session25770显示ready后write_stdin发送U+0003，输出read n=0 err=EOF bytes=""，随后no signal observed within 1s after read，进程exit0。这是限定1秒观察窗口的无信号证据，不是永久不存在事件的证明；说明上一轮控制字符测试不能作为myEnv取消失败/成功验收，不应将普通EOF改为CANCELED。
无产品改动或重复测试，诊断进程已终止。下一步需显式GenerateConsoleCtrlEvent/可观察控制事件的隔离测试，另外处理外部context取消时阻塞读取；完整目标继续，无下载或用户配置修改。

最新进度：显式Windows控制事件隔离探针。上一轮为progress（PTY控制字符仅EOF证据）。依据Microsoft GenerateConsoleCtrlEvent说明 https://learn.microsoft.com/en-us/windows/console/generateconsolectrlevent ，CTRL_C_EVENT不能用非零组ID限定；新.build/console-event-probe/main.go由父进程以CREATE_NEW_CONSOLE及HideWindow创建子探针，子进程GetConsoleProcessList必须恰好只有自身PID才允许生成事件，未向共享用户控制台广播。使用CONIN$读取与Go os.Interrupt通道观察，父进程独立超时负责终止。
首次5秒超时exit1，输出已验证私有控制台。增加阶段日志并将诊断超时改10秒后，session65572输出handler registered、console input opened、generating event、event generated，确认API返回成功；后续观察仍未完成，父超时终止，最终exit1。不能记为信号已被Go消费或取消通过，也不能据此改产品EOF语义。两次子进程均由父Wait结束，诊断源码/可执行文件保留；下一步定位事件后停滞，需要区分控制台与Go信号处理，不重复无诊断发送。无产品改动、下载或用户配置修改，完整目标仍在进行。

最新进度：控制事件探针停滞根因分离。上一轮为progress（隔离控制台事件生成成功但观察超时）。读探针发现missing signal超时panic会先执行defer input.Close，等待仍阻塞的CONIN$读取，导致panic信息未输出；移除诊断子进程的该Close，超时明确打印后os.Exit(2)，父进程及时取得exit2。这仅用于一次性诊断子进程，由进程退出回收句柄，不是生产输入管理方案。
进一步依据上一轮Microsoft文档中可继承忽略Ctrl-C属性，在确认私有控制台只有自身后调用SetConsoleCtrlHandler(NULL,FALSE)，保留Go已注册handler。重建并运行.build/console-event-probe.exe，输出event generated、observed signal=interrupt、read remains blocked after observed interrupt，最终父exit0。相较前次新增明确属性设置，真实信号送达已证实；信号后等待1秒读取仍未结束。这证明原use输入返回后的context检查不足以处理等待中的显式事件，后续需可取消读取设计，并将隔离/忽略属性设置留在测试边界，不更改用户终端。
本轮只修改保留诊断源码，无产品改动或下载；所有诊断进程终止。前次“事件后停滞”记录由本轮更强证据澄清，不作为产品死锁结论。完整目标继续。

最新进度：Windows提示读取支持context取消。上一轮为progress（信号送达而ReadConsole仍阻塞）。新增prompt_windows.go：仅实际终端文件包装Read，专用锁定OS线程执行读取；取得THREAD_TERMINATE权限线程句柄，取消时CancelSynchronousIo并以10ms间隔处理读取尚未开始的竞态，等待读取实际结束才返回，保护调用者缓冲区且不关闭stdin。工作线程在调用者停止取消前保持专用，避免误取消复用线程其他I/O。init版本、use版本与构建确认接入；非Windows仍保留现有输入，未宣称跨平台提示取消完成。参考 https://learn.microsoft.com/en-us/windows/win32/api/ioapiset/nf-ioapiset-cancelsynchronousio ，取消请求不等于I/O完成；如系统始终不完成读取，当前实现仍等待，未声称任意驱动有界退出。
验证：gofmt，VersionPrompt/BuildPrompt/UseMissingSelectionNonInteractive定向通过，包0.157秒。新增TestWindowsPromptReadCancellation，在CREATE_NEW_CONSOLE+HideWindow子进程中确认GetConsoleProcessList仅自己，实际CONIN$无输入读取由100ms context截止取消，返回DeadlineExceeded且n=0，GetConsoleMode确认调用者句柄仍可用，Close正常完成。真实子项0.10秒、父项0.19秒、包0.341秒PASS。无模拟读取或假取消；尚需显式Ctrl-C到完整CLI与正常输入回归，不能将context读取测试冒充全部交互验收。无下载或用户终端修改，完整目标继续。

补记上一轮：TestWindowsUsePromptInterrupt接通完整测试CLI execute，私有控制台GetConsoleProcessList仅自身，测试边界清除Ctrl-C忽略属性；等版本提示后100ms生成控制事件。首次辅助函数重构漏bool返回导致编译失败，修复后测试未达提示；进一步确认CONOUT$只写句柄不能通过终端检测，改为读写句柄后成功。诊断writer同时实现Write/WriteString避免绕过提示屏障。最终真实子项0.11秒、父项0.20秒、包0.351秒PASS，CLI返回130/CANCELED、隔离目录无任何配置或状态创建。是完整CLI函数路径与真实控制事件集成，宿主为测试二进制，未声称发布制品测试。

最新进度：可取消读取正常输入回归。上一轮为progress（完整CLI Ctrl-C集成）。go build -o .build/myenv-prompt-reader.exe ./cmd/myenv，SHA256 8892d830521feafa657cf1dede466d9aa81f989ca788cd57190f3d833c7bd857。真实tty session73784启动无参数use，输入node@22回车，实际声明从24变22，随后预设损坏锁{}导致INVALID_CONFIG/2，报告declaration_changed=true且锁/活动环境未变；无卡住或读取丢失。这是新读取实现的正常输入回归，不是成功安装场景。APPDATA仅该子进程指专用目录。
因上一轮抽取privatePromptConsole改变了既有测试宿主入口，定向运行go test ./internal/cli -run=^TestWindowsPromptReadCancellation$ -count=1 -v -timeout=20s，真实子项0.10秒、父项0.22秒、包0.553秒通过，确认读取取消及输入句柄保留。未重跑已通过的Ctrl-C同内容用例。无外部下载或用户配置修改；Linux/macOS提示取消与全目标验收仍未完成。

最新进度：构建授权提示真实控制事件。上一轮为progress（读取正常输入回归）。扩展Windows私有控制台集成辅助函数，按测试名称隔离子进程；新增TestWindowsBuildPromptInterrupt，创建Python项目及需要构建的backend-path声明，正式CLI sync到[y/N]提示后生成真实Ctrl-C。返回130/CANCELED，myenv.lock和generations目录均未创建。测试不是简单返回false的授权回调；版本选择与构建确认共享实际promptInput取消路径。前置同步可创建修改锁和state.db，未宣称整个项目无写入。
验证：gofmt，go test ./internal/cli -run=^TestWindowsBuildPromptInterrupt$ -count=1 -v -timeout=20s，子项0.28秒、父项0.37秒、包0.530秒PASS。因共享fixture重构，定向复核TestWindowsUsePromptInterrupt通过，保留版本提示取消时完全无状态创建约束。无外部下载或用户环境修改；本轮新增集成测试，没有生产逻辑改动，其他平台和全性能验收继续开放。

最新进度：init取消与CLI集中回归。上一轮为progress（构建授权取消）。新增TestWindowsInitPromptInterrupt复用私有控制台，空目录正式init在tool@version提示后生成Ctrl-C，返回130/CANCELED、目录仍为空，证明取消不会创建myenv.yaml。第一次补丁因gofmt对齐空格未匹配且未应用，修正上下文后成功；gofmt与定向测试PASS，子项0.11秒、父项0.19秒、包0.325秒。
近期root交互与promptInput跨入口变更完成后运行一次go test ./internal/cli -count=1 -timeout=90s，Windows包3.117秒通过。此集中回归包含无门控CLI单元与Windows私有控制台测试；真实归档安装/发布制品/性能门控没有启用，不替代各自证据。三个交互入口的Ctrl-C已有Windows集成，但非Windows promptInput仍为原读取，跨平台取消未完成。无生产改动、下载或用户环境修改，完整目标继续。

最新进度：Linux提示读取可取消。上一轮为progress（Windows init取消及CLI集中回归）。新增prompt_linux.go，仅终端文件通过/proc/self/fd重新打开独立O_NONBLOCK/O_CLOEXEC/O_NOCTTY句柄，50ms poll检查context，非阻塞read处理EAGAIN/EINTR及EOF，返回前关闭自有句柄，无后台读取线程。不使用dup后修改共享文件标志，不改变termios；重新打开权限不足会返回错误，不退回不可取消读取。参考Linux man-pages https://www.man7.org/linux/man-pages/man5/proc_pid_fd.5.html 与 https://www.man7.org/linux/man-pages/man2/poll.2.html 。prompt_other改为排除Linux，macOS仍原实现，未宣称已完成。
验证：gofmt、CGO0 Linux amd64 go test -c -o .build/cli-prompt-linux.test ./internal/cli成功。新增TestLinuxPromptCancellationAndInput创建真实ptmx/pts，75ms截止取消返回DeadlineExceeded，随后同一输入正常读取node@22行，F_GETFL与termios前后完全一致。首次WSL调用参数未引用被PowerShell拆分为-test而exit1，未运行测试；修正引用后同一制品获准WSL定向PASS0.10秒。没有反复重跑业务场景、下载或修改用户终端；WSL组件证据不替代原生产品验证，Linux CLI实际信号与macOS仍待推进。

最新进度：Linux提示实际SIGINT集成。上一轮为progress（Linux可取消读取及PTY正常输入）。新增TestLinuxUsePromptSignal，创建独立ptmx/pts与setsid/控制终端子进程，测试二进制转入正式execute use；仅子进程清空CI并使用专用namespace。在stderr完整版本提示屏障后向该子进程PID发送SIGINT，验证最终130/CANCELED、项目目录无状态创建、PTY termios保持。诊断读取有大小边界，父context5秒限制子进程；不向用户进程组发送信号。
验证：gofmt，Linux CGO0编译.build/cli-prompt-signal-linux.test成功；获准WSL执行-test.run=^TestLinuxUsePromptSignal$ -test.v -test.timeout=10s，PASS0.06秒。这覆盖真实信号到CLI取消，但不是原生Linux发布制品验收，不放开WSL平台限制；尚未将Linux其他提示/信号和macOS取消标为完成。无下载、用户终端修改或生产代码变化，完整目标继续。

最新进度：Linux提示信号合同补充。上一轮为progress（use实际SIGINT）。将PTY子进程验证共享为linuxPromptSignal，按t.Name精确执行对应子入口，提示屏障改为最多4096字节的共同版本提示后缀。新增use SIGTERM/SIGHUP与init SIGINT；仍只向独立子进程PID发信号，检查130/CANCELED、空目录及termios不变。使用已有操作取消退出约定，不将run原生信号退出规则混入操作提示。
验证：gofmt与CGO0 Linux编译.build/cli-prompt-signals-linux.test成功。首次wsl --将正则括号交给shell解析导致语法失败，未运行测试；修正为wsl --exec直接执行同一制品后，use SIGINT/SIGTERM/SIGHUP及init SIGINT各PASS0.06秒。SIGINT复核因公共fixture改变而必要，未重复安装或性能测量。WSL组件验证不作为原生Linux产品验收；macOS和完整性能缺口继续。无生产改动、下载或用户终端修改。

最新进度：交互读取改动三目标制品。上一轮为progress（Linux提示信号合同）。scripts/build-release.ps1 -OutputDirectory .build/interactive-release成功，纳入活动代查询合并、只读连接探测删除、无参数use、Windows/Linux可取消提示读取。Windows12807680字节 SHA256 cf4d9852326e61cc9ede326e952fee9f415eecc7ae290218fb3be2bd8b6d1410；Linux12460194字节 c5b088c16a668ec8120e4cf5b6cdf805276f03f56efa80f539faea1c2d6df8d8；Darwin11948978字节 7bd8ef8e0e7c5424add94c8f3f7ea84c3314daeeea56a2ebb555d451fcd817b2。
现场重算三文件大小/哈希与manifest及SHA256SUMS逐项一致；实际Windows--version、help use及help manual成功，检查可选use参数/TTY说明、SSL_CERT_FILE与shell-init内容。新SHA尚无性能报告，不沿用旧制品指标冒充新门禁；Linux/macOS仅交叉编译，macOS提示取消与子树监督缺口没有关闭。旧制品保留，未签名或公开发布。下一步在新制品上继续必要验收与原生平台缺口，完整目标继续。

最新进度：新交互制品启动预算。上一轮为progress（三目标制品与清单）。对.build/interactive-release/myenv-windows-amd64.exe设置MYENV_TEST_INFORMATIONAL_TIMINGS=.build/perf/startup-interactive-windows.json，运行go test ./internal/cli -run=^TestMeasureInformationalCLI$ -count=1 -v -timeout=30s，采样PASS4.54秒、包4.695秒。51组交替help/version，首组单列，剩余50组nearest-rank p95：help中位40.6717ms/p95 64.9848ms，version中位40.0904ms/p95 64.4814ms，均失败30ms预算。采样确认正常帮助输出、stderr为空和专用项目目录无状态创建，不是全部用户存储写入审计。
现场重算制品SHA与报告一致：cf4d9852326e61cc9ede326e952fee9f415eecc7ae290218fb3be2bd8b6d1410。未清缓存，不是冷启动矩阵；不同旧SHA批次不是交替A/B，不能归因近期提示读取改动。新制品其他命令与内存尚未测量，历史指标不拼接为同SHA验收。无生产改动、下载或发布，完整目标仍未完成。

最新进度：use已取消调用不创建状态。上一轮为progress（新制品启动超预算证据）。检查UseWithRequest发现入口没有ctx检查，预先取消仍走到MkdirAll/LockWorkspace，创建.myenv。新增TestUseAlreadyCanceledPreservesProject先实际复现FAIL（多出.myenv），声明本身未变；不将该复现夸大为已改声明。
生产修复在函数入口、MkdirAll之前、取得工作区锁之后检查ctx.Err，最后一项释放锁后返回。不能原子排除检查与文件系统写入之间的任意时刻取消，但已取消调用不再有该副作用，并在修改前重新观察取消。gofmt；go test ./internal/core -run=^TestUse(AlreadyCanceledPreservesProject|RetainsEditedDeclarationOnFailure|ExpectedInputRejectedBeforePreparation)$ -count=1 -v -timeout=30s三项通过，包0.365秒；已有本地HTTP失败集成确认编辑后同步失败仍保留请求声明与原活动代。无外部下载/用户环境修改。interactive-release早于本次源码修复，原SHA性能证据不改写；完整目标继续。

最新进度：sync预先取消不创建状态。上一轮为progress（use已取消入口修复）。新增TestSyncAlreadyCanceledPreservesProject覆盖apply/preview；先运行实际复现apply多出.myenv失败，preview已通过。Sync现在入口、准备写入前及取得修改锁后检查ctx.Err；已取消调用不再继续发现/解析或创建状态，不宣称检查与写入间取消具备文件系统原子性。
验证：gofmt，go test ./internal/core -run=^(TestSyncAlreadyCanceledPreservesProject|TestUseRetainsEditedDeclarationOnFailure|TestUseExpectedInputRejectedBeforePreparation)$ -count=1 -v -timeout=30s通过，包0.375秒。apply/preview均返回context.Canceled，无结果代/计划/锁变化，目录仅原声明；相关本地失败同步与输入变化保护仍通过。无外部下载或发布；源码晚于interactive-release，完整目标继续。

最新进度：CLI执行前统一观察取消。上一轮为progress（sync预先取消无状态）。新增TestInitAlreadyCanceledJSON，已有.node-version可直接推断工具的空项目，传预先取消context调用init --json，先实际复现exit0/ok=true/changed=true并创建myenv.yaml。将root.PersistentPreRun改为PersistentPreRunE，在保留现有操作信号注册和run监督分工后返回cmd.Context().Err，阻止已取消的合法命令进入RunE；不承诺配置发现与最终文件提交间任意时刻取消完全原子。
验证：gofmt，go test ./internal/cli -run=^(TestInitAlreadyCanceledJSON|TestDoctorCanceledJSON|TestCleanCanceledPartialJSON|TestWindowsInitPromptInterrupt)$ -count=1 -v -timeout=30s四项通过，包0.383秒。init取消返回130/CANCELED，目录仅原.node-version；已有诊断取消、清理部分进度及真实终端运行中取消仍通过。无下载或发布；公共入口源码晚于interactive-release，完整目标继续。

最新进度：公共前置取消改动后的真实run回归。上一轮为progress（init预先取消修复）。审查run/help无覆盖PersistentPreRunE的子钩子，run仍由runner处理子进程信号，公共入口只拒绝预先取消。设置MYENV_TEST_PREPARED_RECORD为保留Node记录，运行go test ./internal/cli -run=^TestRunRetainedNode$ -count=1 -v -timeout=30s，Windows真实运行时PASS1.52秒、包1.676秒。
该场景实际验证子argv字面--json和退出17、npm/npx版本、doctor路径、损坏锁拒绝及run --current运行旧环境；覆盖公共钩子改动后的成功run路径，不是信号/性能全集。复用已有运行时，无下载/安装或用户环境修改。本轮无生产变更，完整目标继续。

最新进度：近期核心入口变更集中回归。上一轮为progress（真实Node run公共钩子回归）。UseWithRequest与Sync新增取消检查后，按受影响模块运行go test ./internal/core -count=1 -timeout=120s，Windows PASS6.283秒。没有启用真实归档/性能环境门控，结果只覆盖该配置下执行的核心测试；已有真实Node/Python与崩溃场景依各自原证据，不将本次包PASS写成全部T00–T06或原生平台验收。
本轮无生产代码变更、下载或发布；有效运行时缓存保留。下一步继续方案功能/平台缺口，启动性能仍未达预算，完整目标保持进行中。

最新进度：落实--verbose诊断输出。上一轮为progress（core模块回归）。对照方案发现root的verbose变量仅绑定参数，没有使用。现显式--verbose在命令前置处理输出命令名与GOOS/GOARCH，命令结束输出该入口后耗时，全部stderr；不记录用户argv、路径或环境值。耗时包含该命令RunE及输出，但不包含进程启动/参数解析，不用作性能门禁。普通模式无新增输出，--help/--version快捷路径不强制进入前置诊断。
验证：gofmt，新增TestVerboseStatusKeepsJSON比较普通/verbose裸状态，stdout均可解析为单个成功changed=false JSON，普通stderr空、verbose包含命令/平台/耗时且不回显-C路径；预取消init回归同批通过，包0.174秒。无下载、用户环境修改或发布；源码晚于当前制品。该功能补齐已承诺选项，不声称完整平台/性能目标达成。

最新进度：verbose与真实run输出合同。上一轮为progress（--verbose实际诊断实现）。新增TestVerboseRunRetainedNode复用已安装Node，--verbose run node实际输出字面参数并exit19；检查stdout精确等于该参数、退出码19，stderr含command=run/platform/elapsed且不含该参数或-C项目路径。验证myEnv自身新增诊断不回显这些值，不承诺清洗子进程自行输出。
运行go test ./internal/cli -run=^TestVerboseRunRetainedNode$ -count=1 -v -timeout=30s，Windows PASS0.33秒、包0.496秒；gofmt。无生产代码修改、下载或用户环境修改。与此前状态JSON测试共同覆盖verbose输出边界，不替代其他平台或性能门禁，完整目标继续。

最新进度：启动初始化定位。上一轮为progress（verbose真实run输出）。对子进程设置GODEBUG=inittrace=1，执行interactive-release Windows制品--version，exit0，原始stderr保留.build/perf/inittrace-interactive-windows.txt；制品SHA重算cf4d9852326e61cc9ede326e952fee9f415eecc7ae290218fb3be2bd8b6d1410。解析本次跟踪显示net初始化11ms、config0.54ms、regexp/syntax和modernc/libc各0.52ms、modernc/sqlite显示0ms（跟踪分辨率，不等于绝对零成本）。本机Go src/net/fd_windows.go init调用poll.InitWSA；据此将网络栈初始化列为后续启动调查对象，没有修改Go工具链或弱化网络功能。
单次初始化跟踪有输出扰动，不能拿总时间或单项替代无跟踪p95，也不能从SQLite本次显示0推出其所有启动成本为零。该证据不支持继续无依据微调SQLite初始化，不将net耗时自动等同myEnv可消除开销。无生产修改、下载或发布，原30ms预算仍未通过，完整目标继续。

最新进度：HTTP响应读取错误统一脱敏。上一轮为progress（启动初始化定位）。审查代理/凭证合同发现Node.metadata的ReadAll错误直接返回，归档CopyBuffer错误直接%w格式化，绕过已有requestFailure安全文本；两路径现复用该包装，仍Unwrap保留原cause供取消/超时/IO分类，不输出底层可能含凭证的URL文本。
新增TestResponseBodyFailureDoesNotExposeCredentials用明确故障Transport响应体注入含测试凭证的读取错误，元数据/归档均DOWNLOAD_FAILED且不含测试秘密、errors.Is仍可识别cause、Body关闭、归档临时目录为空。gofmt与该测试、既有请求建立失败脱敏回归同批PASS，包0.136秒。该fixture专门验证错误边界，不冒充真实下载成功；无外部网络或用户环境修改。完整目标继续，源码晚于当前构建制品。

最新进度：真实HTTP截断读取验证。上一轮为progress（响应体错误脱敏）。新增TestTruncatedHTTPResponseCleanup，本地httptest服务声明Content-Length=100但仅发送partial后结束；元数据与归档均通过真实http.Client读取，确认DOWNLOAD_FAILED包装保留io.ErrUnexpectedEOF，专用下载目录无残留部分归档。区别于前一轮故障RoundTripper，这覆盖实际HTTP响应截断路径，不冒充真实运行时归档安装。
验证：gofmt，go test ./internal/backend -run=^TestTruncatedHTTPResponseCleanup$ -count=1 -v，Windows PASS0.01秒、包0.151秒。无生产变更或外部下载、用户环境修改；完整目标继续。

最新进度：响应体中途取消清理。上一轮为progress（真实HTTP截断）。新增TestHTTPBodyCancellationCleanup，本地服务发送头与partial并Flush后等待请求结束；测试包装真实Transport的Body，只在首批真实读取字节后调用cancel，不替换读取结果。正式downloadArtifact随后返回保留context.Canceled的requestFailure，文本request canceled，专用目录无残留临时归档。此时读取已进入文件复制阶段，区别于建立请求前取消。
验证：gofmt，go test ./internal/backend -run=^TestHTTPBodyCancellationCleanup$ -count=1 -v -timeout=15s，Windows PASS0.01秒、包0.138秒。服务与请求正常收尾，无外部下载、生产修改或用户环境修改。完整目标继续，平台/性能门禁仍未完成。

最新进度：HTTP错误变更后backend集中回归。上一轮为progress（真实响应体取消）。运行go test ./internal/backend -count=1 -timeout=90s，Windows包0.581秒通过；本次未启用真实运行时归档门控，沿用先前按场景记录的安装证据，不将模块PASS扩大为全部平台/后端验收。页首补充当前交互、取消、verbose和HTTP错误处理摘要，避免接续只读旧概述而遗漏近期实现。
无生产代码修改、下载或发布。核心6.283秒回归、近期CLI定向结果及该backend回归分属各自代码/配置范围，不拼作一个发布候选完整验证。完整目标继续，原生平台与性能缺口仍开放。

最新进度：macOS提示取消路径核实。上一轮为progress（backend集中回归）。核对本地x/sys Darwin绑定存在F_GETPATH/TIOCPTYGNAME常量，但没有直接可用的ttyname字符串包装；不能从常量存在推断任意stdin句柄支持该ioctl。读取Apple官方Libc gen/FreeBSD/ttyname.c https://raw.githubusercontent.com/apple-oss-distributions/Libc/main/gen/FreeBSD/ttyname.c ，ttyname_r先fstat/isatty检查，再以设备号经devname_r取得/dev下名称，并非Linux/proc路径方案。尝试同目录devname.c返回404；核对xnu tty_ptmx.c/tty_pty.c也未找到所假定的TIOCPTYGNAME分支，未据此断言内核不支持。
本轮排除了直接复制Linux路径或仅凭常量调用ioctl的无依据实现。下一步需确定Darwin终端路径取得/重新打开的实际合同及设备身份检查，再实现并用原生PTY验证；当前prompt_other仍原读取，macOS取消未完成。无生产代码改动、外部可执行下载或用户环境修改；此为源代码调查证据，不冒充平台验证，完整目标继续。

最新进度：Darwin提示取消实现。上一轮为progress（macOS路径调查）。读取Apple XNU kern_descrip.c F_GETPATH分支 https://raw.githubusercontent.com/apple-oss-distributions/xnu/main/bsd/kern/kern_descrip.c ，确认vnode路径写入MAXPATHLEN缓冲区；本地x/sys FcntlInt调用libc fcntl。新增prompt_darwin.go，以runtime.Pinner固定PathMax堆缓冲区后传递指针整数，获取绝对路径并独立非阻塞打开，比较原/新字符设备的Dev/Ino/Rdev，拒绝路径替换。没有修改调用者flags/termios或引入CGO。
将已验证Linux轮询循环抽为prompt_posix.go（linux||darwin），Linux文件仅保留/proc/self/fd打开；其他平台fallback排除Darwin。gofmt与CGO0 Darwin arm64编译.build/cli-prompt-darwin.test成功；Linux编译.build/cli-prompt-posix-linux.test成功。获准WSL定向PromptCancellationAndInput PASS0.10秒、UsePromptSignal PASS0.07秒，确认共享循环重构后行为。Darwin只有交叉编译，尚无原生PTY/设备路径/取消证据，当前状态为已实现待验证，不关闭平台缺口。完整目标继续，无外部制品下载/发布或用户终端修改。

最新进度：Darwin原生PTY验收入口。上一轮为progress（Darwin取消实现与Linux共享循环回归）。新增prompt_darwin_test.go的TestDarwinPromptCancellationAndInput，由宿主python3创建隔离pty并启动专用测试子进程；子进程先75ms截止取消，再通过ready-next屏障让父进程写入node@22行，比较TIOCGETA与F_GETFL前后不变。子测试5秒超时、Python finally关闭PTY和回收子进程，父进程15秒限制；无python3明确SKIP，不使用用户终端。
验证仅gofmt及CGO0 Darwin arm64 go test -c -o .build/cli-prompt-native-darwin.test ./internal/cli成功。当前Windows/WSL不能运行Darwin制品，未执行该测试，不能标记取消或设备路径在macOS已通过。测试入口使后续原生验收可直接运行，不替代实际证据；完整目标继续，无下载/发布或用户环境修改。


最新进度：macOS原生验证可执行脚本。上一轮为progress（Darwin原生测试编译入口）。当前无.github目录或已知远端仓库，未假设存在CI。新增scripts/verify-prompt-darwin.sh，仅接受原生Darwin arm64并要求Go/python3，在专用.build目录记录host.txt与test.log，运行Darwin PTY定向测试并拒绝SKIP作为通过；本地Go缓存/临时目录显式隔离。脚本不代表完整运行时/发布门禁，不自动安装依赖或修改宿主设置。
已异步询问用户是否有现有macOS测试机/CI连接；未收到的答复不当作授权或证据。脚本待原生执行，下一步先校验shell语法；不把缺少一个宿主扩大为整个目标阻塞，继续可进行的工作。
脚本补充验证：WSL bash -n语法检查exit0；文件统一为UTF-8无BOM/LF。没有在Darwin执行，原生结果仍待运行。

最新进度：原生验证证据防空通过。上一轮为progress（Darwin验证脚本）。检查脚本发现仅检查go test退出与SKIP，若测试名将来失配会错误报告通过；现要求顶层精确TestDarwinPromptCancellationAndInput的PASS行。改为先go test -c保留cli.test，保存build.log及该二进制SHA256SUMS，再直接运行该制品，test.log可关联实际二进制，而非仅临时Go测试进程。
验证：获准WSL bash -n语法检查exit0。未执行Darwin编译/测试脚本、未伪造原生宿主；现有交叉编译证据继续单列。无产品逻辑修改、下载或发布，macOS原生宿主信息仍待用户答复但其他工作可继续，完整目标未完成。

最新进度：近期错误/提示改动汇总构建。上一轮为progress（Darwin验证脚本证据要求）。Windows go test ./internal/cli -count=1 -timeout=90s PASS3.144秒；运行scripts/build-release.ps1 -OutputDirectory .build/prompt-errors-release成功。Windows12809728字节 SHA256 d263763560864a99f3dfe143cbc2b62cd949a796a91b97c617de05f876caf0b6；Linux12460194字节 a0dbb5e60cb3ef0e1eac15a0144862fd1959cc0e9f9ff77f20670aa77136a16c；Darwin11965666字节 b8dfea8437549d378f807410c15930f260449ceaa802ad27ab229d60cddb7ac1。
现场逐项重算大小/摘要匹配manifest及SHA256SUMS，直接Windows--version与--verbose help use成功，stderr包含help命令/平台/耗时，诊断保留在制品目录help-use-diagnostic.txt。新构建纳入预先取消、verbose、HTTP读取错误及Darwin提示实现；新SHA无性能报告，旧门禁失败仍是历史证据。macOS交叉编译不能作原生执行证明。CLI常规测试未启用真实归档门控；旧制品/缓存均保留，无外部下载/发布，完整目标继续。


最新进度：新制品真实PowerShell/profile运行。上一轮为progress（当前三目标汇总构建）。设置MYENV_TEST_SHELL_CLI_EXECUTABLE为prompt-errors-release Windows制品，UV/Python归档复用已保留文件，运行go test ./internal/cli -run=^TestPowerShellAppliedProfile$ -count=1 -v -timeout=60s，PASS12.07秒、包12.234秒。fixture由当前测试CLI准备真实用户profile并完成关闭本地镜像后的同代locked同步；实际新制品生成shell-init，再由PowerShell包装启动同制品run --global，真实Python3.12.13与退出23通过。
制品SHA重算d263763560864a99f3dfe143cbc2b62cd949a796a91b97c617de05f876caf0b6，与清单一致。安装准备仍由测试CLI执行，不冒充全部步骤来自发行二进制；这是新SHA的Shell/run边界证据，不是原生跨平台或性能全集。APPDATA仅测试子进程隔离，无外部下载/用户配置修改/发布，完整目标继续。

最新进度：当前SHA混合同步/状态基准。上一轮为progress（新制品真实profile运行）。保留Node/Python fixture，指定prompt-errors-release Windows制品及MYENV_TEST_MIXED_NOOP_TIMINGS、MYENV_TEST_MIXED_STATUS_TIMINGS，运行go test ./internal/core -run=^TestMixedProjectSyncRetained$ -count=1 -v -timeout=90s，PASS26.54秒、包26.732秒。真实混合依赖项目准备后关闭本地制品服务，外部CLI每项51次首个单列、50热样本：noop sync中位60.6600ms/p95 78.2770ms满足200ms；status中位51.6073ms/p95 57.0460ms仍失败50ms。sync首行反馈热p95 44.4466ms/全样本最大55.6063ms，仅管道接收时间。
报告.build/perf/mixed-noop-prompt-errors-windows.json及status-prompt-errors-windows.json均核对制品SHA d263763560864a99f3dfe143cbc2b62cd949a796a91b97c617de05f876caf0b6。同步逐次同活动代且无变化、状态ready且数据库字节前后相同；未清缓存，不是冷启动或整树内存。不能由跨批差值归因某个源码改动，无生产修改/外部下载/发布，完整目标继续。

最新进度：同SHA真实Node run附加延迟。上一轮为progress（混合noop/status测量）。指定prompt-errors-release Windows制品、保留Node记录与MYENV_TEST_RUN_CLI_TIMINGS=.build/perf/run-prompt-errors-windows.json，运行go test ./internal/cli -run=^TestMeasureFullRunCLI$ -count=1 -v -timeout=45s，采样PASS18.09秒、包18.238秒。51组首组单列、50组nearest-rank p95的配对附加开销：完整CLI中位60.2808ms/p95 82.3604ms，失败50ms预算；进程内CLI中位23.2443ms/p95 32.4714ms，仅诊断子集，不能替代完整进程门禁。
现场重算制品SHA匹配报告cli_sha256：d263763560864a99f3dfe143cbc2b62cd949a796a91b97c617de05f876caf0b6。真实Node直接运行作配对基线，无缓存清除，不是冷启动/整树内存测量；不同旧SHA不是本次A/B，不据差值归因近期改动。无生产改动、下载或发布，完整目标继续。

2026-09-09：同一 prompt-errors-release Windows 制品 run 父进程内存采样
- TestMeasureFullRunMemory PASS 5.49秒（包5.641秒），实际Node 22.23.2执行20次；报告 .build/perf/run-memory-prompt-errors-windows.json。
- 最大PeakWorkingSet为12,836,864 bytes，即12.2422MiB，低于32MiB父进程子项预算。测量不包含Node或完整子进程树，不能据此关闭全平台RSS验收。
- 独立Get-FileHash确认报告SHA为d263763560864a99f3dfe143cbc2b62cd949a796a91b97c617de05f876caf0b6，与本轮实际制品一致。未修改生产代码，未复用旧SHA内存结果。
- 修正首页“新SHA尚无性能报告”的过期描述。相同SHA的status/run延迟仍超预算；启动、冷热矩阵、原生Linux/macOS和macOS完整后代监督仍待完成，T00–T06目标保持进行中。
2026-09-09：当前SHA启动门禁采样
- 执行 scripts/measure-startup.ps1，Executable为.build/prompt-errors-release/myenv-windows-amd64.exe，输出.build/perf/startup-prompt-errors-windows.json。每项51次，首个观测单列，余下50次nearest-rank p95；PowerShell/.NET Process.Start无Shell、并行排空stdout/stderr、等待退出，无OS缓存清除。
- --version：首个66.5120ms，median 53.4976ms，p95 76.6930ms；--help：首个135.0966ms，median 53.4005ms，p95 82.1271ms。两项均未满足30ms；命令实际退出成功。首个样本不是受控冷启动。
- 报告二进制SHA为d263763560864a99f3dfe143cbc2b62cd949a796a91b97c617de05f876caf0b6。该批次启动器是.NET，不与旧Go os/exec批次作因果性能对比；不为获得较低结果重复采样。
- 已复核main入口及startup_windows.go：Cobra Explorer进程枚举已经禁用；现有BenchmarkInformationalCLI仅覆盖进程内命令构建/格式化，不能代替启动门禁。下一步定位加载/初始化成本，保持原预算。当前目录git status确认仍无Git仓库。
2026-09-09：当前源码启动成本定位
- 当前Windows制品以GODEBUG=inittrace=1执行--version成功，stderr保存.build/perf/inittrace-prompt-errors-windows.txt；只在测量子进程环境设置。net初始化单次约10ms，modernc/libc约0.54ms、sqlite/lib约0.61ms，cli初始化标记约18ms。跟踪输出会扰动执行，不据此推算总启动p95或认为其余加载成本为零。
- 本机Go源码src/net/fd_windows.go的init调用poll.InitWSA；现有Windows Cobra Explorer枚举禁用仍在。没有修改工具链或绕过Winsock，也没有将数据库初始化替换为不可靠的延后注册。
- 查阅历史确认上次进程内微基准早于近期CLI变化；本轮对当前源码执行go test ./internal/cli -run '^$' -bench '^BenchmarkInformationalCLI$' -benchmem -benchtime=500x -count=1 -timeout=30s，PASS，包0.187秒。version 26080ns/op、24549B/op、201allocs/op；help 43303ns/op、34730B/op、337allocs/op。原始输出.build/perf/informational-prompt-errors-bench.txt。
- 结论：当前进程内命令构建/格式化远小于端到端超预算量，优先调查进程创建、映像加载及初始化；微基准不是发布制品p95，不能直接相减得精确阶段耗时。本轮未改生产代码，原门禁失败仍有效；原生平台与macOS监督缺口仍未关闭。
2026-09-09：启动测量报告完整性修复
- scripts/measure-startup.ps1原先丢弃stdout，只检查退出码，且Set-Content会覆盖既有报告。现在拒绝已存在输出路径，并在最终写入使用CreateNew防止检查后覆盖；版本须匹配myenv version前缀，帮助须包含Usage，拒绝空输出和任何stderr；测量结束再次核对制品SHA，变化时不发布报告。
- 实际负向验证：指定已有startup-prompt-errors-windows.json立即拒绝，前后文件摘要相同；用真实pwsh --version（退出0但输出属于其他程序）立即拒绝，未创建startup-invalid-output-check.json。此为测量器错误输入测试，不冒充运行时验收。
- 仅因脚本修改，执行20样本实际制品正向验证，生成独立startup-script-validation-windows.json，成功。版本/help p95为63.8556/70.5385ms，仍超30ms预算；此批次用于验证改动后的测量器，不替换此前50样本门禁、不选较低结果宣称改善。生产制品SHA未变，旧报告保留。
- 尚未验证运行中替换制品分支或磁盘写故障；未改变生产CLI及预算。首版平台/监督与性能缺口继续保留。
2026-09-09：Linux取消时保留排队暂停之后的完成确认
- launcher_linux.go原先收到Stopped后若ctx已取消会退出协议读取，返回complete=false，丢失监督器随后提供的完成确认。提取awaitSupervisorCompletion到reply_linux.go；已取消时跳过暂停、继续读取最终确认，未知/损坏协议仍报错。现有生命周期管道取消与监督器回收机制保持使用。
- 新增TestCanceledStopDrainsCompletion：两个已排队暂停帧后，保留Complete/Canceled/137；无最终确认的EOF仍失败，且不调用暂停动作。这是协议故障/竞态边界测试，不代替真实调度竞态验收。
- CGO0 Linux amd64交叉编译.build/runner-cancel-drain-linux.test成功；WSL运行新测试、TestSupervisorReadyCatchesStop、TestCancelStoppedProcessTree全部PASS（0.00/0.01/0.02秒）。首条WSL命令未引号包裹-test.v被PowerShell改写，参数解析失败、未运行测试；修正后执行同一二进制通过。
- 真实停止子树取消与完成回执回归通过；ctx检查后至SIGSTOP之间的取消竞态仍未解决，不能称完整作业控制验收。WSL组件结果不是原生Linux产品验收。生产Linux源码已变化，prompt-errors-release属于修改前制品，其测量不能归到本次源码。
2026-09-09：暂停协议边界与真实PTY回归
- 新增TestSupervisorStopTransitions覆盖连续两次正常暂停、第一次恢复时取消后继续读取完成、暂停回调错误保留；TestCanceledStopRejectsInvalidEvent覆盖无终端及Stopped同时Complete/Ready，确认取消不能绕过协议校验。
- CGO0 Linux amd64编译.build/runner-stop-transitions-linux.test成功；WSL新增测试全部通过。首轮真实PTY测试因未设MYENV_TEST_PREPARED_RECORD而SKIP；随后以保留的node-linux-real/prepared.json启用TestLinuxTerminalStopAndContinue，实际Node/PTY回归PASS 0.87秒。
- 本轮只改测试，生产代码仍为上一轮完成确认修复。回调测试不证明ctx检查与SIGSTOP之间的内核调度竞态已解决；真实PTY通过也不替代原生Linux产品验收。该竞态及macOS监督、性能预算继续作为未完成项。
2026-09-09：完成确认读取修改后的Bash/PTY受影响回归
- 使用同一.build/runner-stop-transitions-linux.test，WSL Ubuntu与保留Node制品，执行TestLinuxTerminalStopOwnerKilled、TestLinuxTerminalStopPendingTerm、TestLinuxShellForegroundResume、TestLinuxShellBackgroundResume、TestLinuxShellRunningBackground，全部PASS，耗时分别0.40/0.58/0.59/0.64/0.60秒。
- 夹具启动独立PTY；Bash以--noprofile --norc运行，HISTFILE=/dev/null。验证停止所有者被终止、停止时挂起TERM、fg恢复、bg后恢复前台以及后台实际继续运行；正常恢复路径另检查前台归属和再次输入。未修改真实Shell配置，未下载运行时。
- 本轮完成上一轮协议读取改动对应的剩余作业控制回归，未重复基本暂停测试。结果限于WSL组件，不能证明ctx检查至SIGSTOP间的取消竞态或原生Linux/macOS完整验收。生产代码未变化，目标继续进行。
2026-09-09：实际复现调用端暂停时截止取消无法执行
- 审阅TestLinuxTerminalStopPendingTerm确认其由PTY驱动发送CONT之后才处理挂起TERM；TestCancelStoppedProcessTree仅停止子树，调用端没有被SIGSTOP。两者不证明调用端暂停时context截止仍有效。
- 新建隔离诊断.build/stop-deadline-probe/main.go与probe.py：实际runner在独立PTY执行/bin/sh自停，调用端context设1秒期限；驱动确认调用端WIFSTOPPED后再等待1.3秒。WSL实测result.json：caller_state_after_1s_deadline=T，child_alive_after_deadline=true。外部CONT后调用端退出，child_reaped_after_resume=true。诊断main未透传runner退出码，因此报告exit_after_external_resume=0只表示诊断正常结束，不是用户命令取消退出码。
- 初次go build参数误写为两个目录而失败；修正为./.build/stop-deadline-probe后编译成功，再实际运行诊断，退出0。没有依赖安装或生产代码变更。
- 这将未完成项从推测收敛为已复现：调用端SIGSTOP也暂停Go计时器/取消转发协程，生命周期管道不会仅因context时间已到而自动关闭。下一步需要监督器侧可独立执行的取消/截止协议，并处理调用端暂停与恢复的确认顺序；单加一次ctx.Err检查不足。保留现有前后台合同，不以跳过暂停消除问题。目标保持进行中。
2026-09-09：Linux监督器独立执行调用端截止时间
- supervisorRequest新增Deadline，调用端从ctx.Deadline传递绝对时间，监督器基于生命周期context建立WithDeadline；子树回收使用该context。仅纯Canceled/DeadlineExceeded进入取消结果，终端/回执混合错误仍不屏蔽。不强制CONT调用端，不改变Shell前后台规则。
- 同一独立PTY诊断重新编译为probe-with-deadline运行：result-with-deadline.json显示1秒截止后调用端仍T，但child_alive_after_deadline=false；外部恢复后正常结束。对比修改前true，证明子树回收现在不依赖暂停调用端的计时器运行。
- 新增TestSupervisorDeadlineWithoutLifetimeEOF：保持生命周期写管道打开，真实停止shell在期限到达后Complete/Canceled非零退出；已过期请求不创建started标记。CGO0 Linux编译runner-deadline-linux.test成功；WSL上述两子项通过0.31/0.01秒，生命周期握手前关闭回归0.01秒、实际传输/脱离会话子树取消0.33秒、排队暂停完成确认回归0.00秒均PASS。
- 限制：跨进程传递的是绝对墙钟时间（Go单调时钟部分不能序列化）；本轮不证明系统时钟突变行为。调用端仍需Shell/外部恢复才能处理最终结果；无deadline的手工取消竞态尚未解决。Linux生产源码已变，旧发布制品性能不能归给本轮源码，首版目标继续进行。
2026-09-09：监督器截止取消的回执与失败语义
- 扩展TestSupervisorDeadlineWithoutLifetimeEOF，继承真实回执fd及精确令牌；运行中截止和请求已过期均要求完整回执文本且Canceled=true。增加只读回执fd模式，真实WriteAt失败必须报告bad file descriptor、Canceled=false，回执保持为空。
- CGO0 Linux amd64交叉编译runner-deadline-receipt-linux.test成功；WSL该测试全部PASS 0.62秒（三项0.31/0.01/0.31秒），生命周期写管道保持打开，确保验证的是监督器自身期限，而非调用端取消。
- 本轮只修改测试；证明截止分支没有吞掉实际回执写入失败，不证明磁盘Sync故障、调用端自动恢复或无deadline手工取消竞态。首版缺口继续保留。
2026-09-09：子树结束后安全恢复暂停调用端
- Linux TTY调用端打开自身pidfd并以信号0预检，连同独立确认管道传给监督器（fd7/8，fd6仍为可选完成回执槽位），监督器设置CloseOnExec避免用户子进程继承。已有暂停通知时，最终回复后通过pidfd每10ms重试CONT，直到调用端离开暂停/读取循环并关闭确认管道；调用端在Wait监督器前确认，避免相互等待。进程死亡ESRCH退出，不使用可复用数值PID。
- resume_linux.go实现恢复确认，TestResumeCallerDelayedStop使用真实shell延迟100ms再STOP，验证恢复循环未因先发CONT而丢失后续暂停，WSL PASS0.11秒。它是调度顺序回归，非所有手工取消竞态的穷尽证明。
- 独立PTY probe-no-resume.py不发送外部CONT；最终源码probe-with-wake-check实测caller_state=Z、child_alive=false、external_resume_sent=false，正常wait回收。Z表示诊断驱动尚未wait的已退出调用端；诊断退出0不代表用户命令退出码。原始result-with-wake-check.json保留。
- runner-caller-wake-linux.test的真实Node九项TTY/Bash回归均PASS（0.35至0.63秒），覆盖INT/QUIT、停止恢复、信号停止、停止所有者死亡、挂起TERM及fg/bg；截止回执三项PASS0.62秒，取消回执失败PASS0.02秒。随后新增信号0预检，仅对最终源码重编译并运行上述无外部CONT诊断通过，未重复全套。
- 新平台条件：TTY恢复依赖pidfd_open/pidfd_send_signal，通常Linux>=5.3，并受系统调用策略约束；失败在用户子进程启动前返回，README已明确。依据Linux man-pages https://www.man7.org/linux/man-pages/man2/pidfd_open.2.html 与 https://www.man7.org/linux/man-pages/man2/pidfd_send_signal.2.html；本地x/sys v0.47.0提供对应调用，无新依赖。信号0预检不能证明所有信号特定策略都允许CONT。
- 截止取消导致子树回收和调用端自行返回已在WSL真实PTY复现修复；无deadline且取消尚未送达就暂停的窗口仍待验证/处理。原生Linux/macOS、完整性能门禁仍未完成。本轮生产源码已改变，不将旧发布制品指标归到本轮源码。
2026-09-09：恢复确认通道严格要求EOF
- resumeCallerUntilAcknowledged此前将POLLIN视为确认，但协议只允许调用端关闭管道；存在数据的管道不应停止恢复。现在先检查poll错误，在可读/HUP时读取一个字节，仅n=0的EOF成功；有数据明确失败，EINTR重试，读取/发信号错误保留。
- 新增TestResumeCallerAcknowledgment实际管道覆盖EOF、仍打开的数据、数据后关闭、无效pidfd；确认数据加HUP也不被当成EOF。CGO0 Linux编译runner-resume-ack-linux.test成功，WSL四子项全部通过；真实延迟STOP恢复TestResumeCallerDelayedStop回归PASS0.11秒。
- 本轮修复的是内部确认协议校验，没有证明手工取消竞态全部消失。生产Linux代码已变，原生平台与性能门禁缺口仍在，首版目标保持进行中。
2026-09-09：真实TTY用户程序描述符隔离
- terminal_linux_test.go在实际Node暂停恢复后，新增执行/bin/sh内建test循环，检查用户程序中/proc/self/fd/3至8均不存在；覆盖请求、回复、生命周期、空回执槽、pidfd与确认管道的exec隔离，并再次验证终端前台归属。
- CGO0 Linux编译runner-terminal-fds-linux.test成功，使用保留真实Node和独立PTY执行TestLinuxTerminalStopAndContinue，WSL PASS0.64秒。该检查刻意在shell执行早期进行，避免Node自身重用描述符编号造成误判；不声称全宿主所有描述符均被审计。
- 本轮仅修改测试，生产源码未变；证实新增fd7/8的CloseOnExec实际生效。手工取消竞态、原生平台及性能门禁仍未完成。
2026-09-09：当前源码三平台制品与证据复用
- scripts/build-release.ps1 -OutputDirectory .build/caller-wake-release成功；Go1.26.6、CGO0、trimpath、buildvcs=false、-s -w、0.1.0-dev。独立复核manifest每项字节数、实际SHA和SHA256SUMS全部一致。
- Windows12809728 bytes，SHA d263763560864a99f3dfe143cbc2b62cd949a796a91b97c617de05f876caf0b6；Darwin11965666 bytes，SHA b8dfea8437549d378f807410c15930f260449ceaa802ad27ab229d60cddb7ac1，均与prompt-errors-release完全相同。因此复用该Windows制品已有延迟/内存/PowerShell结果，不重新采样；Darwin仍只有交叉编译证据。
- Linux12468386 bytes，SHA ae1b970d247a71aa679bd36a04279872b65b8ea63079c48735e4fc5f1c210ef6，包含最新恢复确认修复。WSL执行该最终制品--version退出0，输出myenv version 0.1.0-dev。该冒烟只证明制品可加载，不将测试二进制的PTY结果视为此发布制品完整运行时验收。
- 未发布外网或安装到用户PATH。Windows原有启动/status/run超预算结果保持有效；Linux新制品全套性能与原生平台仍未测，首版目标继续进行。