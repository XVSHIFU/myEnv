目标范围：根据[初始架构方案](../design/initial-architecture.md)，按 [architecture.md](../architecture.md) 实现 CLI 首版 T00–T06。
当前状态：in_progress；T03–T05 的剩余完整性与平台验证、T06 性能/发布门禁均未关闭。日期：2026-09-08。

本文件是当前接续摘要，不是验收通过声明。完整逐轮证据保留在 [implementation-history.md](implementation-history.md)，其中旧“未实现”、旧制品指标和运行中的进程句柄均是历史状态，不能直接作为当前事实。

当前代码与范围：
- T00/T01：Go+Cobra CLI、严格配置、init、离线帮助/手册与补全已接通；工作区无Git仓库，不虚构提交或远端。完整选项/平台合同仍需最终核对。
- T02/T03：Node、固定uv与受管Python、新代venv、原生锁和输入摘要、sync/use/run已接通。Windows真实Python、Node及混合项目已有隔离测试证据，包括本地纯wheel；不据这些子集宣称全部依赖来源、构建许可或原生平台已验收。
- T04：用户profile已实现（core/context.go、cli/profile_test.go），历史中有真实profile测试；status、doctor/--deep、rollback、run --current已接通。“profile未实现”的旧摘要已作废。
- T05：工作区修改锁、代发布、运行租约、clean和缓存清理已实现。准备操作有持久hold和显式子树完成标记；clean仅恢复有完成证据的preparing操作。Linux运行完成回执及Windows Job身份用于保守恢复。未知完成的准备操作仍受保护，不因PID消失便自动清理。
- T06：README、docs/build.md、固定参数发布构建脚本及测量入口已存在。性能仍失败；无完整原生三平台验收或正式签名发布证据。

最近已测Windows状态资源证据（早于准备恢复改动，不能代表当前构建）：
- 制品：../.build/perf/dangling-path-release/myenv-windows-amd64.exe，12720640字节。
- SHA256：63d30b02743f0455938b2d2214fe6901b48f7b3b9541a4521d044a96b527845b。
- 状态：../.build/perf/mixed-project-status-dangling-path-windows.json；真实Python+Node+本地wheel项目，51次完整CLI，首样本单列、后50次中位51.5402ms/p95 57.4922ms。ready/同一代、数据库字节不变；50ms门禁失败。
- 状态内存：../.build/perf/mixed-project-status-memory-dangling-path-windows.json；同SHA的20次完整CLI，父进程峰值11.6719MiB，低于32MiB。不是进程树/冷缓存证据。
- 帮助/版本：前一SHA 3ce779149538ee419fd7e898e67bc422f32cd42e02f3810d9564004b1f382c4b，../.build/perf/startup-handle-path-go-windows.json，p95 61.6851/57.9605ms，均失败30ms。不能替代当前SHA完整门禁。
- run、小项目无变更sync及其他RSS的历史报告见归档，制品不同不能混作同一版本完整矩阵。不要为重现旧数字重复无变化测试。

最近生产修复与验证：
- Windows config.Within使用最终句柄路径跟随junction；未知命名空间/API错误直接返回。其他平台保留EvalSymlinks。
- 解析not-exist后先Lstat：现有但无法解析的条目拒绝，避免悬空连接点/符号链接被祖先回退放过；普通缺失路径仍允许。
- Windows配置全集通过1.224秒；真实TestSyncRetainedPython通过4.79秒；WSL中的Linux组件TestWithinRootAndSymlinkBoundary真实symlink/悬空场景通过。Windows原生symlink创建仍受权限限制，不能称通过。
- 真实TestMixedProjectSyncRetained最后通过26.18秒（状态计时与内存顺序测量）。旧session63107已exit0，不应恢复或重启。
- 本机Go1.26.6 net.init调用poll.InitWSA，在CLI参数处理前启动Winsock；延后http.Client创建不会移除这部分初始化。没有修改工具链或采用非标准规避方案。

明确剩余工作（保留完整目标，不以此代替逐项验收）：
- T03/T04：按方案核对依赖来源、groups、原生workspace输入和构建许可的完整矩阵；已存在的profile代码应继续验证，不重做。
- T05：准备期间崩溃的子树完成证明尚未完整接入；历史未知操作保守保留。共享运行时跨项目引用/卸载及早期保护仍需完整证明。
- T05：macOS后代进程生命周期；Linux后台/暂停继续及完整信号矩阵；真实Windows控制台信号。WSL组件结果不能替代受支持原生Linux产品验收，产品仍拒绝WSL。
- 路径：当前静态/悬空链接检查不证明并发替换的原子隔离；共享盘、Windows ACL及原生符号链接权限验证仍开放。
- T06：启动30ms、状态/run 50ms尚未通过；完整同制品延迟/RSS、冷缓存、进程树峰值、初次下载阶段反馈和三平台运行证据仍缺。
- T06：完整安装/PATH与卸载合同、可审查发布候选和签名/发布流程；发布或远端操作按已有授权边界处理。

下一项实现入口：审查core/sync.go的BeginGuardedOperation/FailGuardedOperation/ConfirmOperationTreesDone与backend启动边界，补准备期崩溃完成证明；必须保持未知子树不回收，不能用“进程已消失”替代完成证据。若先处理其他明确缺口，记录理由并保留全部范围。

工作与验证约束：
- 优先读取AGENTS.md和受影响代码。既有依赖/运行时保留在.build，复用，不重复下载或改动真实用户环境。
- Go命令使用工作区.build/gopath、.build/gocache、.build/tmp；PowerShell含点/正则的原生命令参数加引号。
- 真实Windows路径/进程测试如受沙箱限制，按工具批准范围执行，不改校验预期。WSL组件需明确标注。
- 状态文件只保留当前摘要；新证据更新相应段落。长历史归档，不再无限追加到本文件。

runner完成检查修复证据：
- 审查sync失败defer与Windows Job完成检查，发现executeDirect直接返回Finish错误，缺少ErrTreeUnconfirmed，可能使core释放准备hold或运行租约。现将两者errors.Join，保留原始原因并明确要求保留保护；未改变正常退出码。
- 新增TestDirectCompletionErrorRetainsProtection：启动真实测试子进程、使用真实监督与清理，在完成边界注入错误，验证退出1、原始错误与未知完成标记同时可检测、Close已调用。并非声称制造了真实内核Job查询故障。通过0.172秒。
- 复用保留Node，获准 go test ./internal/runner -run '^(TestWindowsJobRetainedNode|TestCancelRetainedNode)$' -v，通过2.604秒，覆盖后代等待、取消及父进程先退出；无下载或用户环境修改。
- executeDirectWithPreparation仅为局部依赖注入，无全局可变测试钩子。准备期崩溃回执整体仍未完成；macOS完成保证仍需继续审查；Start失败处理见下条。未重建发布制品，旧SHA不代表当前runner内容。

runner启动失败清理修复证据：
- executeDirect的supervisor.Start失败分支原本忽略Kill/Wait错误且跳过Finish。现终止/等待直接子进程后调用Finish；预期的已退出与被终止退出状态单独处理，其他清理错误和Finish失败与原始启动错误一起返回，并携带ErrTreeUnconfirmed。清理证据完整时仅返回原始启动错误。
- 新增TestDirectStartFailureCompletion：真实测试子进程与平台监督，注入启动错误，验证完成可确认/不可确认两个分支，原始错误不丢失，Finish/Close均被调用；与上一条完成错误测试一起通过0.172秒。再新增before_start子项并定向运行通过0.121秒，覆盖Windows尚未分配Job/恢复线程的子进程终止与空Job完成检查。
- 实际gofmt；go test ./internal/runner -run '^TestDirect(StartFailureCompletion|CompletionErrorRetainsProtection)$' -v；go test ./internal/runner -run '^TestDirectStartFailureCompletion/before_start$' -v。没有重复正常Node路径的既有验证，无下载或用户环境修改。未制造真实内核Kill失败或测试全部平台，亦未完成准备期崩溃回执集成。

运行租约到clean的集成证据：
- 扩展已有TestCleanRetainsAppliedAndLeased，不创建重复fixture：四代中第一代已不属当前/前代，通过leasedRunEnvironment.Finish接收errors.Join包装的ErrTreeUnconfirmed；关闭并重新打开SQLite后核验原租约ID仍存在。
- 随后的dry-run与实际clean均仅清理无租约旧代，受保护代的文件仍在；再以无子进程的测试fixture模拟已确认完成，Finish(nil)释放租约后才清理该代，并核验再次clean无变化。使用的是状态/清理集成fixture，不声称本测试制造了真实Job故障；真实runner故障边界与后代测试证据见上文。
- 实际gofmt；工作区GOPATH/GOCACHE/TMP下 go test ./internal/core -run '^TestCleanRetainsAppliedAndLeased$' -v，通过0.408秒（用例0.25秒）。无新下载、发布重建或用户环境修改。准备期崩溃证明与其他平台完整保证仍开放。

准备失败平台策略修复证据：
- sync错误清理原仅检查ErrTreeUnconfirmed，macOS runner没有后代完成保证却可能返回普通错误，导致失败准备hold被解除。提取并接通finishFailedPreparation：仅Windows/Linux既有完整监督路径且无未知完成标记时FailGuardedOperation；其他平台保留hold并将未知完成标记与原始错误一起返回。nil/既有未知错误保持原样。
- TestPreparationFailureCompletionPolicy在真实SQLite/工作区锁下覆盖Windows和Linux策略的可清理失败、Windows未知失败、macOS尚无证明失败；恢复扫描后只前两者进入候选。平台字符串是策略注入，不是原生Linux/macOS进程验证。初版策略测试通过0.690秒；随后补上恢复函数要求的实际工作区锁前置条件，再定向验证。
- 获准 go test ./internal/core -run '^(TestPreparationFailureCompletionPolicy|TestSyncRetainedPython)$' -v，通过5.551秒；策略用例0.56秒、保留Python/uv真实同步4.84秒。无新下载或用户环境修改，未重建发布制品。
- macOS准备失败可能保守保留更多目录（包括尚未实际启动子进程的早期失败）；这是当前缺少完成证明时的保护，不是最终macOS实现。成功发布的后代完成保证、准备期崩溃回执与跨项目共享运行时保护仍需继续完成。

准备失败操作匹配证据：
- FailGuardedOperation原先即使UPDATE未匹配操作也返回成功。现要求恰好更新一个属于当前PID的preparing操作；错误ID、其他所有者或非preparing状态返回明确错误，事务回滚，避免把未完成清理记录成成功。
- TestFailGuardedOperationOwnership验证missing/foreign-owner拒绝、原hold与preparing状态保留；恢复当前所有者并确认子树完成后可失败收尾；重复失败调用拒绝。既有TestGuardedOperationRecovery覆盖持hold的正常失败路径，core平台策略测试覆盖上层调用。
- 实际gofmt；go test ./internal/state ./internal/core -run '^(TestFailGuardedOperationOwnership|TestGuardedOperationRecovery|TestPreparationFailureCompletionPolicy)$' -v，state通过0.449秒、core通过0.706秒。无新依赖、下载或用户环境修改。这里核验的是所有者PID和状态匹配，不声称解决PID复用或完整准备期崩溃回执。

准备操作进程身份实现证据：
- 新增operation_identity表（operations外键删除级联）。BeginGuardedOperation复用supervisorIdentity，在同一事务中写操作、hold和进程身份。ConfirmOperationTreesDone/FailGuardedOperation在其事务内同时检查PID、preparing状态与进程身份；不为历史操作回填伪造身份。
- 扩展所有者测试模拟PID相同但身份陈旧，以及身份缺失：两种收尾接口均拒绝，hold仍在；恢复实际身份后既有正确收尾继续通过。此为身份匹配故障注入，不声称实际制造OS PID重用。
- go test ./internal/state通过3.572秒；增加故障场景后定向state/core验证通过0.308/1.017秒，包含所有者保护、已确认准备清理和平台失败策略。获准 go test ./internal/core -run '^TestSyncRetainedPython$' -v，真实Python同步通过4.86秒。没有新的依赖或外部下载。
- 身份表用于新受保护准备的两个收尾入口；不是完整崩溃回执或历史未知对象自动恢复方案。旧缺身份preparing操作继续保守保留，已存在的显式完成记录仍按原恢复规则处理。状态schema已变，未重建发布制品。

准备身份旧库升级验证：
- 新增TestOperationIdentityUpgrade，在自有SQLite fixture中保留unknown/settled/failed操作，删除新身份表模拟紧邻上一版schema。全程持工作区锁，未操作真实用户数据库。
- OpenReadOnly可预览既有failed与有完成证据的settled，身份表仍不存在，关闭后数据库SHA256不变。Open升级后身份表为空，不回填历史身份；未知历史操作的确认/失败接口均拒绝，hold仍在；显式完成记录恢复恰好1个操作，新建操作登记身份成功。
- 实际gofmt；工作区GOPATH/GOCACHE/TMP下 go test ./internal/state -run '^TestOperationIdentityUpgrade$' -v，通过0.321秒（用例0.20秒）。此测试覆盖紧邻旧schema，不代表所有历史版本或其他原生平台均已验收。无生产代码变化、下载或发布重建。

Linux准备身份组件验证：
- 读取Linux processIdentity实现，确认取/proc进程启动时刻与boot_id；新operation_identity调用既有平台机制，不新增独立身份算法。
- GOOS=linux GOARCH=amd64 CGO_ENABLED=0交叉编译state-linux.test成功；获准现有WSL Ubuntu --exec运行 '-test.run=^(TestProcessIdentityLifecycle|TestRecoveryKilledOwner|TestFailGuardedOperationOwnership|TestOperationIdentityUpgrade)$' '-test.v' '-test.timeout=60s'，四项全部PASS（分别0.01、0.20、0.01、0.01秒）。真实杀死的是测试自行创建的所有者进程；未知准备记录仍保留，只恢复已明确完成项。
- GOOS=darwin GOARCH=arm64 CGO_ENABLED=0编译state-darwin.test成功，仅编译证据，没有原生macOS运行。WSL是Linux组件环境，产品仍拒绝WSL，不能替代原生Linux端到端验收。无下载、用户环境修改或生产代码变化。

准备子树登记实现：
- 新增operation_children（随机子树ID、操作外键级联、完成位和操作索引）。BeginOperationChild在操作所有者身份/hold检查的同一事务中登记，先提交再允许launch；CompleteOperationChild只接受当前所有者的未完成记录。关闭启动阶段后拒绝新登记；ConfirmOperationTreesDone/FailGuardedOperation拒绝仍有未确认子树的操作。
- backend.WithProcessObserver以context局部传递登记/收尾回调，无全局钩子；executeBackend启动前登记，runner返回后调用收尾。core.Sync在BeginGuardedOperation之后接通，随机子树ID同时用于Windows命名Job；只有具备后代完成保证且无ErrTreeUnconfirmed时持久标记完成，写入失败也保留未知保护。
- 状态TestOperationChildCompletion验证未完成时两种收尾拒绝、错误/重复子树完成拒绝、正确完成后阶段关闭且禁止后续启动，通过0.292秒。前置定向state/core旧升级及失败策略也通过（backend该筛选无匹配测试，未称backend测试通过）。获准TestSyncRetainedPython通过5.29秒，复用保留Python/uv；真实查找、校验、venv及项目同步沿executeBackend执行。
- 当前仅覆盖新代BeginGuardedOperation之后的后端启动；此前共享uv/解释器解析准备仍属未完成的早期保护范围。记录尚未关联Linux独立完成回执，所有者崩溃后的pending子树不会自动恢复。下一步接独立回执及相应崩溃证明，不能把已登记等同已完成。
- macOS runner尚无后代完成保证，受观察的新代后端调用会返回ErrTreeUnconfirmed并保留准备保护，不能发布该准备结果；这是明确未完成的平台能力限制，不代表macOS同步已可用。未重建制品或性能重测；额外SQLite写入影响实际同步路径，需后续按新SHA验证。

Linux准备回执绑定实现：
- 新增operation_child_completion外键表与BindOperationChildCompletion：规范32字节十六进制令牌，验证操作所有者身份、未完成子树及一次性绑定。Linux在操作临时目录通过os.Root独占创建0600回执文件，绑定提交后才传给runner.TreeCompletion。
- observePreparationChildren接通平台回执设置。设置失败尚未launch，标记该登记子树结束；runner完成且数据库登记成功后关闭/删除回执；未知完成或登记失败则关闭并保留。新代操作目录创建后再安装观察器。非Linux不创建虚假回执，仍沿用既有监督/未知策略。
- Linux TestPreparationChildReceipt通过真实/bin/sh退出7，验证监督器写出准确令牌回执；正常登记完成后删除，模拟上层尚未登记完成时保留，且pending记录继续阻止操作完成。WSL组件运行两项PASS共0.08秒；并非杀死上层进程的崩溃测试，不代表原生Linux产品验收。
- 状态测试补充非法令牌、错误子树、重复绑定拒绝；工作区Go环境执行TestOperationChildCompletion通过。未下载运行时，未重建发布制品。
- 下一步必须接入所有者死亡身份确认、回执安全读取、逐条CAS恢复和整操作完成条件；目前clean尚不消费这些准备回执，所以崩溃pending仍保守保留。早期共享准备及macOS完整监督缺口不变。

Linux准备回执实际恢复实现：
- Linux实际clean持工作区锁后扫描带身份/hold/至少一条子树记录的preparing操作；按128条分页读取操作和子树。确认Linux原所有者身份已退出后，os.Root内以NOFOLLOW|NONBLOCK打开回执，限制读取到固定协议长度并核验普通文件/规范令牌/精确内容。
- 子树恢复UPDATE比较观察到的操作ID、PID、进程身份、子树ID和token；仅全部子树完成、仍持hold且身份匹配时事务转failed并解除hold。没有子树记录的历史操作不适用此协议，缺失/损坏回执及身份查询错误继续保留。部分子树CAS成功也保留Changed标记。
- 新增TestPreparationRecoveryExitedOwner：真实Linux测试所有者创建两个准备操作与真实子进程回执后退出，不登记DB完成；一个回执有效、一个改为错误令牌。实际clean恢复/删除恰好一个，另一个payload保留，第二次clean无变化。WSL初次0.07秒PASS；补Changed传播后重编译定向复测0.09秒PASS。Windows既有CleanConfirmedPreparation定向测试亦通过。
- 当前dry-run尚不预演这一新增回执恢复路径，只读取既有候选；需要补同口径预览。测试是所有者正常退出时遗漏收尾，不是准备仍运行时强杀场景；后续需覆盖活所有者、部分子树、回执替换/分页及强杀恢复。新schema/路径的原生Linux产品与完整性能验收仍开放，未发布制品。

部分子树恢复验证：
- 扩展同一个真实Linux所有者退出fixture，增加第三个准备操作：第一子树真实执行并产生有效回执；第二子树仅完成启动前登记/回执绑定，保持空回执，模拟登记后尚未启动便退出的保守窗口。
- clean仍只删除全部完成的第一操作，错误令牌与部分完成操作的payload均保留。只读重开数据库，第三操作两条子树记录恰好一条completed；第二次clean无变化。证明部分证据不会解除整操作hold，不声称空回执能自动判定为从未启动。
- 实际gofmt、GOOS=linux GOARCH=amd64 CGO_ENABLED=0重编译core-linux.test；获准现有WSL --exec运行 '-test.run=^TestPreparationRecoveryExitedOwner$' '-test.v' '-test.timeout=60s'，0.09秒PASS。无生产变化、下载或用户环境修改；dry-run同口径、活所有者/强杀及回执替换仍待补。

准备所有者强杀恢复验证：
- 新增TestPreparationRecoveryKilledOwner：真实Linux测试所有者持修改锁、登记准备/子树并绑定回执；通过runner启动/bin/sh，先启动后台sleep后写started标记并等待。父测试先调用80ms超时clean，明确要求DeadlineExceeded且Changed=false，再只Kill测试自己创建的所有者进程并Wait。
- 独立Linux监督器检测所有者管道关闭，结束并回收后代后写回执；测试在5秒有界窗口内调用实际clean，完成证据到达前要求Removed=0，到达后RecoveredPreparations=1且Removed=1，准备目录最终消失。没有向枚举所得PID发送信号。
- 初版通过0.15秒；收紧为先创建后代再发ready、精确锁超时类型后，重编译core-linux.test并获准WSL '--exec'运行 '-test.run=^TestPreparationRecoveryKilledOwner$' '-test.v' '-test.timeout=30s'，再次0.15秒PASS。运行仅测试临时目录，无下载/用户环境修改。
- 这是WSL中真实Linux监督/回执/状态/clean强杀链路证据，不是原生Linux产品验收或所有崩溃时点证明。dry-run一致性、回执替换/分页、早期共享准备与Windows/macOS准备恢复仍开放。

准备回执dry-run一致性实现：
- clean dry-run补充Linux准备子树回执预览。与实际恢复共用recoverPreparationReceipts校验逻辑；nil结果表示只读，不写子树完成位。仅所有者已退出且全部登记子树已有DB完成位或有效回执时预览候选，并用既有failedFiles统计两目录字节数。
- PreparingChildOperations兼容缺表旧库（只读检测，返回空结果不迁移），并排除已发布同ID/目录；操作记录额外返回目录用于既有generationName边界验证。普通平台仍不伪造回执恢复能力。预览仍是建议，实际clean重新持锁核实。
- 扩展TestPreparationRecoveryExitedOwner：预览仅1候选、Changed/Removed/RecoveredPreparations为零，数据库前后SHA256相同；实际恢复候选数/字节数完全一致，错误令牌/部分完成仍保留。旧身份表升级测试补旧库准备子树预览空结果。
- Windows定向state/core测试通过0.346/0.488秒；重编译Linux组件并获准WSL运行ExitedOwner与KilledOwner，分别0.06/0.16秒PASS。无下载或用户环境修改；未重建发布制品。此前“dry-run不预演回执”的缺口已由本轮关闭，其他平台/早期共享准备与回执替换边界仍需推进。

Linux回执最终文件链接修复：
- 扩展真实退出所有者fixture，新增回执替换为指向精确有效令牌的symlink、FIFO和目录。初次测试失败：dry-run错误接受第4项symlink，证明os.Root.OpenFile的内部链接解析不能仅靠传入O_NOFOLLOW阻止。
- 阅读本机Go root_unix.go后，新增openCompletionReceipt：先用os.Root.OpenRoot固定相对父目录并取得目录文件句柄，再unix.Openat单一basename，使用O_NOFOLLOW|O_NONBLOCK|O_CLOEXEC。最终叶链接由内核拒绝；FIFO可非阻塞打开后由既有普通文件检查拒绝。准备恢复及运行租约回执统一采用该读取器。
- 修复后Linux组件TestPreparationRecoveryExitedOwner通过0.08秒：仅真实普通回执可预览/清理，symlink/FIFO/目录及错误/不完整证据都保留，DB只读与预览/实际字节一致性仍通过。KilledOwner通过0.11秒，CleanCompletedLinuxOrphan通过0.07秒。
- 实际gofmt、Linux交叉编译core-linux.test；获准WSL --exec运行上述三个定向测试，60秒测试上限，exit0。无下载或用户环境修改。此前仅靠Root.OpenFile的NOFOLLOW保证已由失败证据推翻并替换，不沿用旧保证；原生Linux产品与其他平台验收仍开放。

Windows准备Job恢复实现：
- 新增Windows准备恢复/预览，复用windowsLeaseReclaimable对进程身份格式、原PID所有者退出及命名Job消失的检查。每条子树均须通过；Linux令牌记录不被Windows Job路径接受。实际恢复比较操作ID/PID/身份与子树ID，无完成令牌记录才可标记；全部完成后沿共用事务解除hold。部分成功保留Changed。
- 非Linux默认实现的build tag改为同时排除Windows。只读预览复用同样Job条件，统计既有安全目录字节；其他平台仍不声称实现。定向WindowsCleanConfirmedPreparation/PreparationFailureCompletionPolicy通过1.154秒。
- 新增TestPreparationRecoveryKilledWindowsOwner：真实Windows测试所有者持锁登记准备和子树，runner启动命名Job内测试子进程；子进程写ready后保持运行。80ms clean明确超时且不变；父测试只Kill自己创建的所有者并Wait，随后实际clean确认Job消失，恢复/删除恰好1个准备操作。获准测试通过0.555秒（用例0.39秒）。
- 无下载或用户环境修改，未重建发布制品。当前真实强杀证据为单子树；Windows多子树、异常Job查询和预览/实际一致性仍需补。macOS/早期共享准备及完整平台性能验收继续开放。

Windows准备恢复预览一致性验证：
- 扩展TestPreparationRecoveryKilledWindowsOwner：所有者和Job子进程存活时dry-run候选为0、Changed=false，实际clean仍明确锁超时。强杀测试所有者后，在5秒有界窗口内只读预览等待Job消失；预览Removed/RecoveredPreparations为0，数据库SHA256不变。
- 后续实际clean恢复并删除恰好1项，候选数量和两目录字节数与预览一致。该场景的Windows预览/实际一致性缺口已验证；仍不代替多子树和Job访问错误矩阵。
- 实际gofmt；获准工作区Go环境 go test ./internal/core -run '^TestPreparationRecoveryKilledWindowsOwner$' -v，通过0.576秒（用例0.41秒）。仅自有测试进程/目录，无下载、用户环境修改或生产变化，未重建制品。

Windows部分Job恢复验证：
- 扩展Windows真实强杀fixture为两条子树登记：一条启动真实命名Job worker，一条登记后未启动。父测试用OpenJobObjectW的QUERY权限保留自有Job句柄，原所有者被Kill后Job仍存在且worker仍受该Job管理。
- 保留Job期间dry-run零候选且不变；实际clean仅记录无Job登记项的完成，Changed=true但RecoveredPreparations/Removed为零，准备payload仍在。关闭最后Job句柄触发既有KILL_ON_JOB_CLOSE后，预览与实际清理才完成整操作回收，数据库只读摘要和字节一致性断言仍通过。
- 实际gofmt；获准工作区Go环境运行TestPreparationRecoveryKilledWindowsOwner，通过0.594秒（用例0.44秒）。所有打开/关闭的句柄均指向测试随机ID的自有Job，未枚举或控制其他进程；无下载或生产变化。此场景验证多登记项中一个仍活动Job的保护，不声称全部并行子树/Job访问拒绝矩阵已完成。

准备恢复CAS与回滚验证：
- 新增TestPreparationRecoveryObservedIdentity，在实际SQLite/工作区锁下独立验证恢复CAS：旧操作ID、PID、进程身份、子树ID、令牌全部不生效；未完成子树阻止整操作恢复，匹配更新一次生效且重复无变化，子树完成后错误所有者仍不能解除hold。
- 在自有测试数据库设置BEFORE DELETE触发器模拟hold删除失败，确认同事务前面的status更新回滚，仍为preparing；移除测试触发器后匹配恢复成功。这是数据库故障注入，不冒充实际进程死亡或磁盘故障。
- 实际gofmt；工作区Go环境 go test ./internal/state -run '^TestPreparationRecoveryObservedIdentity$' -v，通过0.306秒（用例0.19秒）。无生产代码变化、下载或用户环境修改，未重建制品。

准备恢复集中回归与三目标构建：
- 准备身份、子树登记、后端观察器、Windows Job/Linux回执恢复及预览累计改动后，获准一次 go test ./internal/state ./internal/backend ./internal/core ./internal/runner，全部通过：4.854/0.403/4.884/0.204秒。使用工作区Go缓存/临时目录；未启用外部运行时集成开关，相关门控测试跳过，不冒充完整真实后端矩阵通过。
- 固定脚本 ./scripts/build-release.ps1 -OutputDirectory .build/recovery-release 三目标成功，构建session95993已exit0。Windows12753920字节，SHA faf26a3cfaa3d905690d96675fcaa455e09e5cf93618ac5d27a205e3b6f4c8cd；Linux12411042字节，SHA 05615cfe0bbe754655d5775e1e8810e0164a0108893b93c3afab7d17eea74cb9；Darwin11915378字节，SHA 647107fc783237fcc041f63acb24e7ee4e793ef8c0cbe53bf3ff2ad62fcdff5d。build-manifest.json/SHA256SUMS在同目录，已逐项核对制品SHA匹配。
- 新Windows制品--version及--help冒烟成功。Linux/macOS本轮只有交叉构建，没有原生运行；macOS新代后端准备仍受缺失后代完成保证限制。未对这些新SHA测延迟/RSS，旧数据仅为历史定位，不拼接为当前完整门禁。
- 全部为本地开发制品，无发布、签名、下载或用户环境修改。准备恢复主要链路已有集中回归证据，但早期共享准备、macOS、完整依赖/平台/性能与发布验收仍开放。

恢复制品真实无变更同步性能：
- 使用.build/recovery-release/myenv-windows-amd64.exe（SHA faf26a3cfaa3d905690d96675fcaa455e09e5cf93618ac5d27a205e3b6f4c8cd）重编译core测试，获准复用保留Node/Python与自有本地wheel，在同一混合项目顺序测量51次完整无变更sync及20次CLI峰值内存。
- TestMixedProjectSyncRetained通过28.32秒；.build/perf/mixed-project-noop-recovery-windows.json中位62.7746ms/p95 67.1421ms（首样本单列），首行反馈热p95 38.2486ms、全部最大51.5939ms。所有输出核验同代、changed/lock_changed/native_lock_changed为false。
- .build/perf/mixed-project-noop-memory-recovery-windows.json最大CLI峰值12.0898MiB；两报告SHA已核对相同。本小型真实混合项目符合200ms/64MiB预算；不是子进程树、冷缓存、首次下载反馈或全部依赖规模门禁。
- 实际core测试 '-test.run=^TestMixedProjectSyncRetained$' '-test.v' '-test.timeout=4m'，session29736已exit0，没有重复启动或下载。当前SHA的状态/run/帮助版本及全平台完整资源门禁仍需验证，首版目标继续进行。

本轮run资源验证（上一轮为progress：真实无变更同步性能）：
- 复用同一恢复制品SHA faf26a3cfaa3d905690d96675fcaa455e09e5cf93618ac5d27a205e3b6f4c8cd，重新编译cli测试；获准TestMeasureFullRunCLI，仅MYENV_TEST_RUN_CLI_ONLY=1，固定配对直接保留Node与完整myenv run，剔除首组、保留50组差值，不启用进程内CLI测量。
- .build/perf/run-full-cli-recovery-windows.json：配对附加开销中位60.7489ms、p95 72.2225ms，仍失败50ms。用例12.33秒PASS、session62102已exit0；PASS表示基准正确执行，不是延迟门禁通过。
- 随后获准TestMeasureFullRunMemory通过5.41秒，.build/perf/run-memory-recovery-windows.json：同SHA、Node22.23.2，20次CLI父进程峰值最高12.2422MiB，低于32MiB；不含Node子进程树。
- 全部是自有隔离已发布代fixture，不下载或修改用户环境。未调低目标，当前run启动/检查/租约成本仍需进一步定位；其他同制品命令及原生平台完整门禁未关闭。

已初始化状态库打开成本定位：
- 新增门控 TestMeasureInitializedStoreOpen，MYENV_TEST_STORE_OPEN_TIMINGS 启用。专用临时数据库交替测量51次可写/只读打开与关闭，首样本单列，其余50个原始样本及最近秩p95写入独占创建报告；median_ms采用排序第25项（下中位数）。默认跳过，无生产路径变化。
- 已执行 gofmt -w internal/state/open_timing_test.go 及 go test ./internal/state -run '^TestMeasureInitializedStoreOpen$' -v。已核实 .build/perf/initialized-store-open-recovery-windows.json：两组各50个保留样本，可写中位2.9998ms、p95 4.035ms，只读1.1843ms、2.0346ms。原始命令终态输出在上下文截断中不可见，仅记录已核实报告，不另行宣称测试终态PASS。
- 可写打开重复执行幂等schema语句，但本诊断包含整个连接打开/关闭，不能将两组差值全部归因于schema。该量级不足以解释完整run附加p95 72.2225ms，暂不引入schema版本迁移。进程内测量不是CLI门禁；继续定位启动与运行检查，预算保持不变。git status确认当前目录仍非Git仓库。

run分阶段定位与重复文件检查削减：
- 上一轮分类为progress：状态库测量产生了排除schema优先优化的证据。本轮阅读SelectRun、CLI run、租约实现和架构性能/租约约束；复用既有cli测试制品及保留Node，在隔离临时项目获准运行 TestMeasureRunStages，明确PASS（5.84秒，进程exit0）。报告 .build/perf/run-stages-recovery-windows.json：51次首样本单列，50次选代中位13.7058ms/p95 15.4676ms、执行含Node 87.9606/101.3136ms、释放8.4627/10.1550ms。此为改动前进程内诊断，p95不可相加，执行数字不是附加开销。
- SelectRun的Node检查先Stat再立即Lstat覆盖结果；改成按工具二选一，Node只Lstat，Python仍Stat。保留原有链接与普通文件判定，减少一次无效文件系统调用，不声称这能解决完整启动预算。
- 实际gofmt及 go test ./internal/core -run 'Test.*Run' -count=1 成功0.170秒；检查测试名称后确认该筛选主要覆盖RuntimeSnapshotIdentity，真实运行时用例未启用，不将其称作真实SelectRun集成验证。本次局部等价调整未重建发布制品，前述测量仅适用于改动前代码。
- 释放包含带身份条件的持久化租约删除及连接关闭。后续需进一步区分事务与平台监督成本，不能通过省略租约或降低持久化保证达标。首版全部平台与性能目标仍开放。

run租约释放失败合同验证：
- 上一轮为progress（阶段测量及删除重复Stat）。本轮核对architecture第268行：run启动后透传子进程退出码。因此保留现有释放失败写stderr、不覆盖子进程退出码的行为；不将退出0等同于租约清理成功。
- 新增TestRunLeaseReleaseFailure，复用保留真实Node，在各自临时项目SQLite安装BEFORE DELETE触发器模拟释放失败。分别执行真实子进程退出0和7，检查stdout确实执行、原退出码保留、stderr同时包含IO_ERROR及底层故障，数据库租约及进程身份关联仍存在。
- 实际gofmt、go test -c -o .build/cli-release-failure.test.exe ./internal/cli；获准运行该定向用例，0/7两个子用例均PASS，总0.62秒，进程exit0。包含上一轮SelectRun修改后的真实Node路径验证，无下载或真实用户环境修改。故障来自测试数据库触发器，不冒充磁盘断电验证。
- 未改变生产退出码或持久化策略，未重建发布制品；全部性能与平台缺口继续开放。

租约释放零行结果修复：
- 上一轮为progress：真实Node故障注入确认退出码合同。本轮发现ReleaseLease对身份不匹配的DELETE零行仍返回nil，旧测试只验证保留记录却接受成功返回。调整为检查RowsAffected恰好1，否则返回ErrLeaseNotOwned；不改变DELETE身份条件或增加查询/事务，不删除非本人保护记录。重复释放也明确报错，调用方原有Finish诊断能报告失败。
- 更新TestLeaseRetainsSelectedGeneration，身份错配须返回可errors.Is识别错误且原记录保留；恢复正确身份可成功释放及级联删除；再次释放须返回错误。该变更修正内部完成报告，不覆盖已启动子进程退出码。
- 实际gofmt及 go test ./internal/state ./internal/core ./internal/cli，分别通过4.977/4.985/1.589秒，进程exit0。未启用保留运行时门控测试，不能称作本次真实后端矩阵；上一轮真实故障测试为此前代码证据。未重建发布制品，性能与全平台目标仍开放。

早期后端进程登记前移：
- 上一轮为progress（释放零行错误修复）。核查Sync发现managedUV的VerifyVersion/PrepareUV以及ResolvePython位于BeginGuardedOperation和观察器之前。将原有随机操作ID、准备hold、失败defer、操作目录和观察器整段前移至构建权限检查之后、managedUV之前，沿用同一个最终代操作及完成确认，不另建旁路记录。
- dry-run及无变更快捷返回仍在登记之前；早期uv/Python后端调用现在经过既有子树登记/平台完成证据路径。早期解析或锁写入失败也进入准备失败处理。共享后端目录的跨项目引用、卸载保护及完整共享运行时回收仍未完成，本改动只覆盖该Sync路径的早期进程登记，不声称解决全部共享存储生命周期。
- 实际gofmt及go test ./internal/core通过4.642秒。编译.build/core-early-preparation.test.exe，获准复用保留Node/Python，运行TestMixedProjectSyncRetained，通过20.70秒；session23419已exit0。自有本地wheel/隔离项目，未启用性能测量开关，无下载或真实用户环境修改。真实混合新代准备及无变更路径通过，但早期强杀恢复专用故障场景仍待补。
- 未重建发布制品。macOS监督实现、共享目录引用及全平台资源预算仍开放。

早期uv启动失败专项验证：
- 上一轮为progress（前移准备登记及真实混合同步）。新增TestEarlyUVPreparationFailure，在隔离项目走实际Service.Sync，两种失败分别为不存在的uv路径、普通文件但无效可执行映像；后者经过backend观察器及真实OS启动拒绝，不使用假后端成功结果。
- 检查两种路径均返回失败且没有ErrTreeUnconfirmed、结果不报告环境或声明锁变化、没有写myenv.lock；SQLite恰好一个failed操作、无hold、无生成代、无待完成子树。缺失路径未尝试启动故子树0条，无效映像已登记启动故子树1条且完成。实际Clean能删除恰好一个失败准备。
- 实际gofmt及 go test ./internal/core -run '^TestEarlyUVPreparationFailure$' -v，Windows两个子用例PASS，总0.591秒（用例0.43秒），exit0。测试按runner.Platform限制可确认后代的平台，本轮没有Linux实际运行或macOS证据。没有下载、生产代码变化或发布构建。
- 此为启动前失败验证，不能代替早期uv运行中强杀/跨项目共享目录生命周期验证；后者与完整平台及性能验收继续开放。

完整子进程登记协议标记（恢复接入待续）：
- 上一轮为progress。确认当前恢复查询有意排除零子树记录，历史guarded记录无法证明从未启动。新增operation_tracked外键标记表及BeginTrackedOperation，与操作/hold/进程身份在同事务插入；既有BeginGuardedOperation不标记，旧数据不回填。Sync改用新API，其全部后端启动已由前移观察器登记。此标记目前尚未接入零子树恢复，不宣称该崩溃窗口已关闭。
- 新TestTrackedOperationAtomicRegistration用标记INSERT触发器拒绝验证整个登记回滚，再验证成功的新记录有标记、历史记录无标记。初次fixture复用了受唯一约束的directory导致测试失败，改为独立目录后定向测试PASS0.343秒。首次go test ./internal/state ./internal/core中core通过5.518秒，state仅新增测试fixture失败；修正后未重复无关回归。
- 实际gofmt及上述测试，无下载/发布。下一步必须让恢复查询、只读预览及平台所有者死亡验证识别完整登记的零子树操作，并验证历史未知记录仍受保护；共享目录、macOS和资源验收仍开放。

Windows零子进程准备恢复接入：
- 上一轮为progress（原子完整登记标记）。PreparingChildOperations现在有界返回有标记的零子树操作并携带Tracked；只读旧库缺少标记表时使用常量false，不迁移。RecoverPreparationOwner的事务条件允许完整登记标记替代至少一个子树，仍要求本人观测身份、preparing、hold及没有未完成子树。
- Windows恢复与预览在零子树且Tracked时复用严格Windows身份/session/所有者退出检查；意外同名Job存在仍保护。历史无标记零子树不进入候选。Linux零子树恢复尚未接入（现有目录/回执检查仍保留），不声称跨平台完成。
- 扩展state测试验证新零子树候选、历史排除、错误身份CAS拒绝与匹配CAS成功。go test ./internal/state ./internal/core通过5.203/5.421秒。
- 新Windows TestEmptyTrackedPreparationRecovery：真实独立所有者进程建立tracked及历史guarded记录，尚未创建操作目录或启动子树；存活时预览0，所有者退出后预览恰好1且DB摘要不变，实际恢复/删除1、字节匹配，重复clean无变化保留历史记录。定向测试PASS0.441秒（用例0.27秒）。随后给测试总上下文补充10秒超时并gofmt，未重复执行。此为真实正常退出模拟未完成准备，不冒充强杀测试。
- 无下载或发布；后续完成Linux零子树分支及实际验证，原目标其他共享存储/macOS/性能缺口继续开放。

Linux零子进程准备恢复接入：
- 上一轮为progress（Windows恢复/预览与真实退出测试）。新增linuxPreparationComplete供实际恢复和预览共用：已验证所有者退出后，有Tracked标记且查询无子树则确认完成，无需操作目录已创建；非空记录继续使用安全回执读取及子树完成检查。实际恢复现在也显式检查该完成布尔值，再执行既有CAS。
- 将Windows零子树测试移为windows||linux共用测试，保留10秒上下文上限。实际gofmt、CGO0 Linux amd64编译.build/core-empty-linux.test；获准WSL Ubuntu --exec运行TestEmptyTrackedPreparationRecovery（0.03秒）、PreparationRecoveryExitedOwner（0.09秒）、PreparationRecoveryKilledOwner（0.16秒），全部PASS且exit0。
- 共用测试证明真实所有者存活时不预览，退出后仅完整登记操作可预览/实际回收，DB只读摘要、字节一致及历史未知保护有效；既有回执错误/链接/FIFO/不完整证据与强杀测试仍通过。此为WSL真实Linux组件，不是原生Linux产品或性能门禁。
- 未重建发布制品或下载。完整早期Sync进程强杀集成、共享目录生命周期、macOS监督与全平台资源目标继续开放。

Linux准备所有者身份校验修复：
- 上一轮为progress（Linux零子树恢复）。发现准备恢复/预览仅检查linux:前缀，损坏身份与当前PID真实身份不同可能被误判为所有者退出。将运行租约既有boot UUID十六进制/连字符及规范无符号启动时间解析提取为validLinuxOwnerIdentity，准备恢复、预览和租约共用，不再只检查前缀。
- 新TestMalformedPreparationOwnerProtected在自有SQLite中替换仍存活所有者的身份，覆盖缺字段、非规范前导零及负启动时间；dry/实际clean均无候选无变更，hold保留。属于持久化记录损坏注入，不声称防御任意恶意数据库伪造。
- 实际gofmt、CGO0 Linux amd64编译.build/core-owner-identity-linux.test；获准WSL执行MalformedPreparationOwnerProtected0.02秒、EmptyTrackedPreparationRecovery0.03秒、PreparationRecoveryExitedOwner0.08秒、CleanCompletedLinuxOrphan0.07秒，全部PASS、exit0。验证严格拒绝同时未破坏正常准备与运行租约恢复。仅Linux组件证据，不是原生产品平台验收。
- 未下载或重建发布制品；早期Sync强杀、共享目录、macOS及完整性能仍开放。

完整登记标记旧库升级验证：
- 上一轮为progress（Linux身份校验修复）。新增TestTrackedOperationUpgrade，在自有SQLite保留有子树和零子树两类guarded操作后移除operation_tracked，模拟紧邻旧schema。只读打开仍返回有子树操作且Tracked=false，零子树不入候选；确认没有建标记表，关闭后DB字节SHA一致。
- 可写打开创建空标记表但不回填历史记录；查询结果保持，直接使用匹配的历史零子树所有者调用恢复CAS也不能解除保护。该测试验证数据库升级与候选边界，不模拟真实进程死亡。
- 实际gofmt及 go test ./internal/state -run '^TestTrackedOperationUpgrade$' -v，PASS0.352秒（用例0.20秒），exit0。无生产变化、下载或发布构建；全目标其他缺口继续开放。

准备恢复分页删除验证：
- 上一轮为progress（旧库升级验证）。新增TestPreparationRecoveryPaging，以单事务建立514条交错的完整登记/历史零子树fixture；读取257个可恢复操作的分页应依次为128/128/1/0。每读取一项即通过恢复CAS将其移出候选集合，验证基于ID游标的后续页不跳项、不重复，历史257条preparing记录全部保留。
- 实际gofmt及 go test ./internal/state -run '^TestPreparationRecoveryPaging$' -v，通过1.954秒（用例1.83秒）。随后给fixture补充工作区锁以遵守恢复API调用前置条件并gofmt，未重复运行。此为真实SQLite查询/CAS分页测试，所有者字段是明确测试数据，不声称进程退出或大量子树监督证据。
- 未改生产实现、下载或构建发布制品。全平台/共享存储/资源目标仍开放。

恢复事务引用复核：
- 上一轮为progress（跨页恢复验证）。RecoverPreparationOwner现在在最终UPDATE中比较观测directory，并再次排除同ID或同目录的generations引用；此前这些引用只在候选查询排除。身份/hold/子树完成条件不变。
- 扩展ObservedIdentity测试，确认目录观测不一致拒绝；观察候选后插入同目录不同ID、同ID不同目录的生成代记录，恢复均不生效，移除测试引用后原有hold删除回滚及最终成功路径仍通过。这是明确的数据库状态变化模拟，不冒充并发发布故障。
- go test ./internal/state -run 'TestPreparationRecovery|TestTrackedOperation' -v通过2.614秒，含补充锁后的分页测试。随后升级测试补齐观测目录字段，避免因缺目录而掩盖历史协议保护判断；定向TestTrackedOperationUpgrade再通过0.320秒。已gofmt，无下载/发布制品。全目标其他平台、共享存储和性能工作继续开放。

完整登记恢复制品集成回归与构建：
- 上一轮为progress（恢复CAS目录/引用复核）。Windows core定向EmptyTrackedPreparationRecovery、PreparationRecoveryKilledWindowsOwner、EarlyUVPreparationFailure全部通过1.311秒。重新编译Linux测试后，获准WSL验证MalformedOwner0.05、KilledOwner0.12、EmptyTracked0.03、ExitedOwner0.09秒，全部PASS/exit0；目录条件与真实平台恢复记录兼容。
- 固定build-release.ps1输出.build/tracked-recovery-release三目标成功exit0。Windows12755968字节 SHA d510bb82a7095514c5518e69df063642b872f51b6eb9e6f9ec6d9eeda1450b2f；Linux12419234字节 SHA c5e5ef5657ddc07796184437fadce9a098e30be3c5a4f74726c49b48c2d8cb9e；Darwin11915394字节 SHA a43396b3b2164d17d0ac2fc569d878d5e281c20c0b6d6178c77c047b35946d11。三项manifest与文件SHA已逐项核对，Windows版本0.1.0-dev及help冒烟通过。
- 初次help验证脚本对PowerShell行数组使用-notmatch导致误报；修正为先join文本后验证通过，没有产品代码变化。新制品尚未测性能/RSS，不能沿用旧SHA报告为当前门禁；Linux/macOS制品仅交叉构建，WSL组件测试不是原生产品验收。
- 全部本地开发制品，无签名发布或下载。早期Sync运行中强杀、共享目录、macOS监督及资源目标仍开放。

新完整登记制品启动性能：
- 上一轮为progress（集成恢复与三目标构建）。复用既有Go测量宿主，仅测外部新CLI，MYENV_TEST_RUN_CLI_ONLY=1，保留Node配对基线；TestMeasureFullRunCLI通过12.35秒，InformationalCLI通过4.47秒，session56602已exit0。PASS只表示采样正确执行。
- .build/perf/run-tracked-recovery-windows.json：50个首组剔除后的配对附加开销，中位58.6703ms/p95 71.9819ms，失败50ms门槛。.build/perf/startup-tracked-recovery-windows.json：help中位39.3904/p95 64.8993ms，version39.6195/62.7649ms，均失败30ms。完整制品SHA d510bb82a7095514c5518e69df063642b872f51b6eb9e6f9ec6d9eeda1450b2f，不与旧制品数据拼接为当前门禁。
- 同制品单次GODEBUG=inittrace=1 --version成功，日志.build/perf/inittrace-tracked-recovery-windows.txt显示包初始化推进至约18ms；这是额外诊断样本，不是时延基准或冷缓存测量，也不能将包初始化时间解释为全部启动成本。未改持久化/校验或降低预算。
- 无下载、代码变更或外部发布；启动性能失败、共享目录及原生平台完整验收仍开放。

Windows最小进程启动基线定位：
- 上一轮为progress（新SHA启动测量）。当前包初始化日志不足以解释p95尾部，因此在.build/startup-baseline创建独立诊断Go程序，仅os.Stdout.WriteString固定文本；不接入生产、不冒充CLI功能。按相同Go版本/CGO0/trimpath/去符号参数构建baseline.exe。
- 使用同一既有Go测量宿主TestMeasureInformationalCLI，51次交替两组参数（诊断程序实际忽略参数），首样本单列。获准测试PASS2.66秒、exit0；.build/perf/startup-minimal-go-baseline-windows.json：两组中位20.9216/21.384ms，p95 49.6982/51.0294ms。报告路径和SHA指向baseline，不是myEnv。没有缓存驱逐。
- 最小程序本身在当前宿主的p95已超过30ms，这改变下一步归因：需要参考硬件及宿主启动波动复核，不能把myEnv尾部全部归因于命令实现。该实验与前一轮myEnv测量未逐次配对，不能相减p95声称净开销，也不能据此调整产品预算或标记通过。生产帮助/版本/run门槛仍失败。
- 无生产代码变化、下载或发布。后续优先推进剩余生命周期实现，避免在缺少参考宿主证据时反复同机微调启动；共享目录/macOS/完整验收仍开放。

共享存储清理范围核查与同路径验证：
- 上一轮为progress（宿主启动基线）。核查当前CLI runtimeService已解析用户Data/Cache；Service.Storage为空的工作区布局是内部兼容/测试回退，不能据其推断产品没有共享运行时。managedUV使用版本/平台锁，installSharedPython使用版本/平台锁，独立venv创建在共享安装锁之外。卸载手册要求只移除二进制，不隐式删环境。
- 项目Clean处理项目目录；另有显式CleanNodeCache遍历共享node-archives，以摘要锁保护传输副本删除，不清理runtimes/backends。修正初步“无共享清理路径”的宽泛说法：存在Node传输缓存清理，受管Python/uv安装不在其范围内。不为不存在的自动卸载路径增建引用系统。
- 新增TestNodeCacheCleanPreservesColocatedRuntimeStorage，模拟Windows用户Data/Cache相同应用路径，runtimes/backends/locks/uv内放置同名摘要文件；dry/实际缓存清理均只识别node-archives对象，其他资源逐项内容不变。gofmt及定向go test通过0.184秒（用例0.03秒）。这是目录边界测试，不是真实解释器运行或跨项目进程证明。
- 无生产代码变化或下载。后续以实际剩余需求推进早期Sync强杀和macOS监督；不沿用泛化的“共享存储未实现”历史描述，完整平台资源与发布验收仍开放。

实际Sync早期强杀恢复验证：
- 上一轮为progress（共享缓存边界验证）。新增Windows TestSyncKilledBeforeBackendLaunch，独立真实测试进程调用Service.Sync，在已有PhaseBackend回调设置确定性测试屏障。该位置在BeginTrackedOperation/操作目录/观察器之后、managedUV调用之前；只注入未使用uv路径，未伪造后端成功。
- 父测试在屏障处只读核验恰好一条Tracked操作和零子树；实际clean在80ms工作区锁等待后明确超时无变化。随后仅Kill自身创建的Sync所有者并Wait，dry候选1不变更，实际恢复/删除1且预览字节一致，myenv.lock未写入。测试总上下文10秒、失败清理会终止并等待自有进程。
- 实际gofmt及 go test ./internal/core -run '^TestSyncKilledBeforeBackendLaunch$' -v，Windows PASS0.507秒（用例0.36秒），exit0。没有下载、生产代码变化或发布制品。此证据覆盖真正Sync登记到首次启动之间的崩溃窗口；不代表uv运行中后代强杀场景已验证，也不代替Linux/macOS产品验收。

Linux SIGQUIT转发补齐：
- 上一轮为progress（实际Sync启动前强杀验证）。审查Linux暂停/继续时发现独立监督器与调用端仅Notify INT/TERM/HUP，SIGQUIT落入Go运行时退出处理。两端现在均订阅并沿既有自有子树转发链交付SIGQUIT；不改变后代等待、租约保留或取消清理。
- 扩展既有真实Node信号fixture支持显式信号参数，新增Linux TestForwardQuitRetainedNode，Node安装SIGQUIT处理器后ready，父测试只向自有监督调用进程发信号，确认用户程序退出23且无Go崩溃诊断。SIGTERM原测试路径复用保持。
- 实际gofmt及Linux amd64 CGO0测试编译.build/runner-quit-linux.test；获准WSL复用保留Linux Node，Quit用例0.58秒、Term用例0.32秒均PASS/exit0，无下载。组件测试不代表终端产生Ctrl-反斜线的PTY场景已验证。
- 本轮解决明确终止信号缺口，未宣称暂停/继续的前台进程组协议完成；macOS监督、早期后端运行中强杀及原生平台/资源验收仍开放。现有发布制品尚不包含本次Linux生产改动。

Linux SIGQUIT组广播验证：
- 上一轮为progress（SIGQUIT双端转发实现与真实Node验证）。新增TestForwardGroupQuitRetainedNode，复用既有组信号fixture：调用进程创建独立进程组，父测试只向该组发送SIGQUIT，Node计数处理器等待100ms后仅在恰好一次交付时退出23，重复交付导致不同退出码。
- 实际gofmt、Linux amd64 CGO0编译.build/runner-group-quit-linux.test；获准WSL复用保留Node执行定向测试，PASS0.47秒、exit0。确认专属监督进程组与调用进程组隔离对新增QUIT仍有效，不等同于PTY实际键盘信号或暂停/继续测试。
- 无生产变更、下载或发布构建；完整交互终端生命周期、macOS监督和原生性能验收仍开放。

Linux PTY实际QUIT字符验证：
- 上一轮为progress（QUIT组广播验证）。将既有TerminalInputAndInterrupt fixture参数化信号名及控制字符，新增TerminalInputAndQuit；独立Python pty.fork提供真实控制终端，向PTY写0x1c而非直接kill信号，Node收到一次SIGQUIT后退出23。保留输入hello验证、正常/启动失败/取消后的前台进程组恢复及调用者再次读取again的断言。
- 实际gofmt及Linux amd64 CGO0测试编译.build/runner-pty-quit-linux.test；获准WSL复用保留Node，PTY Interrupt0.61秒、Quit0.57秒均PASS/exit0。没有下载或接管用户终端；测试进程与PTY均隔离。
- 此证据补上终端产生QUIT的交互路径，仍不是暂停/继续或macOS支持证明。无生产代码变化，现有发布制品未包含此前QUIT生产改动；原生平台与性能门禁继续开放。

Linux继续信号传递：
- 上一轮为progress（QUIT真实PTY验证）。检查暂停协议发现双端Notify未订阅SIGCONT。调用端与独立监督器现在把SIGCONT沿已有自有子树转发路径交付；不将该改动声称为完整Shell作业控制。
- 新TestForwardContinueRetainedNode，复用真实Node信号fixture，Node安装SIGCONT处理器并ready后，父测试只向自己的调用进程发送CONT；监督链交付后Node退出23且无诊断。实际gofmt、Linux amd64 CGO0编译.build/runner-continue-linux.test，获准WSL复用保留Node，定向PASS0.34秒/exit0。
- 本测试仅证明CONT交付，Node未先进入停止态；真实停止/继续、Ctrl-Z调用者暂停以及恢复终端前台仍待实现/验证。没有下载或发布构建，旧制品不包含本次Linux改动。

Linux实际停止后继续验证：
- 上一轮为progress（CONT转发实现）。加强既有ContinueRetainedNode场景：Node注册CONT处理器并输出固定宽度自有PID后，对自身SIGSTOP；父测试读取该PID的/proc/stat直到确认为T停止态，再只给调用端发送SIGCONT。消除ready之后尚未停止就提前CONT的竞态。
- 既有10秒上下文及失败时终止/等待调用进程清理保留；无无界输出读取，PID输出固定10字节，停止态检查5ms间隔且受上下文限制。停止后沿双端监督链继续，Node处理CONT退出23，诊断为空。
- 实际gofmt、Linux amd64 CGO0编译.build/runner-stopped-continue-linux.test；获准WSL复用保留Node，定向测试PASS0.36秒/exit0，无下载或生产变化。该证据是实际SIGSTOP/CONT，不等于Ctrl-Z后Shell接管与fg恢复的完整作业控制；后者、macOS及原生资源门禁仍开放。

停止态进程树取消与回执验证：
- 上一轮为progress（实际SIGSTOP/CONT）。新增Linux TestCancelStoppedProcessTree，自有/bin/sh父子分别写PID并SIGSTOP自身；测试读取两者/proc/stat确认均为T，再取消Execute上下文。断言非零退出、无未知完成错误、两个自有PID均已回收，独占完成文件具有精确绑定令牌回执。
- runner goroutine在失败清理中也取消并join后才关闭回执，5秒上下文；外层测试30秒上限。实际gofmt、Linux amd64 CGO0编译.build/runner-stopped-cancel-linux.test，获准WSL定向PASS0.02秒/exit0。无Node下载或用户终端控制。
- 证明既有取消清理不依赖停止的用户程序先自行继续，不代表Ctrl-Z/Shell fg协议已实现。本轮无生产修改；macOS及完整平台/性能门禁仍开放。

运行信号与退出码手册补齐：
- 上一轮为progress（停止态取消验证）。架构第268行要求手册解释信号退出码，现有手册仅有myEnv 0/1/2/3分类，未解释run。补充离线manual及README：启动后原生退出码、Linux信号128+编号及130/131/143例子、自行处理信号后正常退出的区别、Windows原生状态、Linux转发信号和未实现Ctrl-Z/fg限制；解释未知子树完成保留保护及租约收尾stderr不替换子进程状态。
- 已gofmt。首次go test按Help/Manual/Completion名字筛选报告no tests to run，不计为验证通过；随后实际go run ./cmd/myenv help manual成功，核对渲染包含SIGQUIT退出码与作业控制限制。未运行安装/网络操作，未重建发布制品。
- 文档只描述当前源码能力，保留原生Linux/macOS待验收声明；完整实现与性能门禁仍开放。

run命令查找文档纠正：
- 上一轮为progress（信号退出码手册）。核查runner.Lookup和cli/run后，发现run帮助仍称仅node/npm/npx和绝对路径，遗漏已经实现的Python、相对路径与子进程PATH查找。帮助及离线manual现反映实际查找范围；manual同时说明argv直接传递，管道/重定向需要显式Shell。
- 实际gofmt及go run ./cmd/myenv help run成功，渲染含managed node/python/npm/npx和applied PATH；此为文本文档调整，不重复真实运行时测试，也不声称新增命令执行能力。未下载/发布重建。
- 另在审查中注意到runner返回非PathError的监督失败可能沿CLI默认路径被分类为USAGE_ERROR/2；尚未改动，下一步需用具体失败场景验证并修正，保持子进程原生退出码和未知完成租约保护。

run内部执行失败分类修复：
- 上一轮为progress（查找文档纠正）。新增隔离状态损坏fixture，把受管Node条目设为仍可Stat的相对路径。初测明确失败：环境构造拒绝相对bin，却被报告USAGE_ERROR/2；错误发生在runner.Environment而非预想Execute边界，按实际证据调整修复范围。
- 引入私有可Unwrap的runFailure，包装子进程环境构造和runner.Execute错误；CLI识别为RUN_FAILED/1并建议doctor。原有PathError/取消分类仍有优先覆盖，子进程childExit不变；defer仍使用原runErr决定保留租约，未丢失ErrTreeUnconfirmed。
- 实际gofmt，复用保留Node运行ExecutorFailureExit、RunRetainedNode、LeaseReleaseFailure，全部PASS2.523秒（0.22/1.49/0.65秒），验证损坏状态错误分类、启动前租约释放、真实Node/npm/npx/argv退出合同和释放故障时0/7原码保留。无下载或发布构建。
- 本fixture不制造真实内核监督失败；后者由相同类型包装覆盖，独立runner已有未知完成测试证据。完整目标其他平台与资源门禁继续开放。
