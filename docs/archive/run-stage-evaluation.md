# Linux run 阶段计时与监督合并评估

日期：2026-09-09。本轮按用户明确限定的任务执行：补齐缺失阶段，验证监督进程合并是否保持 SIGKILL/SIGSTOP 行为；Windows 架构、租约格式、事务及持久化设置不变；仅做一次修改前后对照和定向回归。不是对全部首版性能预算重新验收。

## 结论与交付状态

保留独立监督器。测试专用的单进程 subreaper + PDEATHSIG 候选不能保持现有行为：调用端 SIGKILL 后仍有孙进程存活且无回执；调用端 SIGSTOP 时超过 deadline 后仍未回收。现有双进程路径在相同评估中通过。该单进程候选只存在于测试代码，没有启用到产品。

发现并实现了一个独立的重复工作优化候选：Linux `run` 打开已初始化状态库时，检查所需 schema 对象，齐全则不重复执行整套幂等 DDL 和 active 行初始化；缺对象/active 行时回退原初始化流程。删除保护触发器、FULL、foreign_keys 和租约登记/释放完全保留。Windows 的 run 仍调用原 `state.Open`。

该候选通过定向可靠性检查，但本轮端到端延迟没有改善。因此源码保留候选供审阅，`.build/run-stage-evaluation/myenv-after` 是评估制品，不替换 `.build/kali-validated-release`，不作为性能优化成功或 50 ms 门禁通过的证明。没有继续调参数、重跑挑选较好样本或改造租约。

## 合并评估的依据

Kali 当前 SSH 会话在 `/user.slice/user-1000.slice/session-368.scope`。cgroup v2 挂载可用，根目录不可由 kali 写入，用户服务子树的 cgroup.procs 可写。只读检查没有创建或迁移 cgroup；这不等于证明完整委派和跨分支迁移可行。

cgroup 的 populated 通知和 cgroup.kill 提供机制，但仍需要运行中的执行者处理事件、发起回收和写完成回执。SIGKILL/SIGSTOP 不可被应用捕获、屏蔽或忽略；单进程被杀后不能完成用户态回执，被暂停时不能执行自己的 deadline 回调。PDEATHSIG 也不是对任意孙进程的完整替代。本轮没有证明可用且等价的外部执行机制，不能因存在 cgroup 就删除监督器。[Linux cgroup v2](https://docs.kernel.org/admin-guide/cgroup-v2.html)、[Linux 信号语义](https://man7.org/linux/man-pages/man7/signal.7.html)、[PDEATHSIG](https://man7.org/linux/man-pages/man2/PR_SET_PDEATHSIG.2const.html)。

`TestSupervisorMergeAssessment` 使用真实 PID 句柄观察后代，单进程候选还给直接子进程设置了 PDEATHSIG，并锁定启动 OS 线程；孙进程通过 setsid 脱离，未继承父死亡设置。SIGSTOP 场景先确认调用端确实停止，再等到截止时间之后检查后代与回执；检查前没有发送 CONT。测试末尾使用已持有的 pidfd 清理自己的测试进程。

| 场景 | 双进程生产路径 | 单进程测试候选 |
| --- | --- | --- |
| 调用端 SIGKILL | 后代退出，完整回执存在 | 孙进程仍活着，回执为空，拒绝候选 |
| 调用端 SIGSTOP 超过 deadline | 后代退出，完整回执存在 | 孙进程仍活着，回执为空，拒绝候选 |

两种架构四个场景共通过评估 4.72 秒；“通过评估”包括按预期拒绝单进程候选，不是单进程满足可靠性。

初始化记录还证明 `runner.init` 先于 SQLite、配置和 Cobra 初始化；监督器在该入口执行并退出，因此现有实现已经跳过后续 CLI 包初始化。未据“第二个进程”推断整个 CLI 初始化重复执行。GODEBUG=inittrace 只用于确认顺序，其打印开销不计入发布性能。

## 缺失阶段计时

增加 `internal/runtrace`，只在 Linux `-tags runtrace` 诊断构建中收集事件。默认构建为空操作；无用户命令、环境值记录。每进程至多 256 个事件，在退出时一次写出，避免逐事件文件写入干扰阶段。诊断产物和未插桩端到端产物分开。

每个源码版本各 10 次真实 Node 空命令归因，表中为中位 ms。跨进程使用墙钟时间戳；这些区间可能重叠，不应相加，也不是验收 p95。收尾和分派区间由同一批原始事件补算，没有再次执行采样。

| 阶段 | 修改前 | 候选 |
| --- | ---: | ---: |
| 外部启动请求到业务入口 | 13.4621 | 10.7106 |
| 业务入口到选择环境 | 0.1861 | 0.1598 |
| 定位、读取声明 | 0.6550 | 0.5587 |
| 打开状态库（本次修改范围） | 8.1111 | 4.1349 |
| 取得租约并提交 | 12.2798 | 7.7399 |
| 校验已应用环境 | 1.4734 | 1.3406 |
| 准备回执文件 | 0.3161 | 0.2302 |
| 监督启动前准备 | 3.0246 | 3.1388 |
| 启动监督器到收到就绪 | 26.2391 | 15.3425 |
| 其中：启动到监督器入口 | 16.2808 | 10.6063 |
| 其中：入口到写就绪 | 6.6537 | 4.8166 |
| 用户进程创建 | 2.7085 | 3.6838 |
| 主用户进程被回收到全部后代完成 | 0.7642 | 0.5592 |
| 完成回执写入与 Sync | 2.7362 | 1.8958 |
| 回执完成到调用端等到监督器退出 | 9.0678 | 6.0804 |
| 释放租约并关闭状态库 | 9.3496 | 8.1739 |
| 其后回执清理与 CLI 收尾 | 0.2108 | 0.1464 |

多个未修改阶段也发生变化，不能将差值全部归因于 schema 快路径，亦不能把阶段中位数差当成端到端收益。

## 端到端一次对照

直接复用 `TestMeasureFullRunCLI`，未改统计流程：51 次、首次单列，50 次直接 Node/包装执行轮换顺序，每对计算差值，再求差值分布的 nearest-rank p95。没有用两个独立 p95 相减。两个制品各一个固定批次，按顺序执行，无并行性能任务。

| run 附加开销 | 修改前 | 候选 |
| --- | ---: | ---: |
| 中位 ms | 60.8994 | 70.4663 |
| p95 ms | 118.0288 | 170.2873 |

本轮观测变差；既不能宣称提高，也不能从两个批次精确分离宿主变化与修改影响。原 50 ms 门槛未通过，不重复取样挑选结果。

修改前 SHA：`75ede63a747902e5d9919863f1be92f9be64c8b49b97d5a60c43b9343d013591`。

候选 SHA：`ccc10a7d6a91e1258e9f93aedae39a5b86f6909206285ac2d15b195dcc5a218e`，12476578 bytes；Go 1.26.6、CGO0、trimpath、buildvcs=false、`-s -w -X main.version=0.1.0-dev`。候选二进制与端到端/内存报告 SHA 已独立一致。

## RSS/PSS 与私有页一次对照

各 20 次，以交替制品顺序运行相同等待 200 ms 的 Node。通过 smaps_rollup 记录每个 myEnv 进程，再在每个采样轮内求和；不是把不同时间的每进程峰值相加。两边每次都捕获了调用端和监督器。Node 不混入 myEnv 成本。

| 观测峰值 KiB | 修改前 | 候选 |
| --- | ---: | ---: |
| 调用端 RSS | 16824 | 18620 |
| 监督器 RSS | 16436 | 11880 |
| 每轮聚合 RSS 最大值 | 32872 | 30484 |
| 每轮聚合 PSS 最大值 | 32856 | 25092 |
| 每轮聚合私有页最大值 | 32856 | 20688 |

每进程 RSS/PSS/私有页及每次聚合值均在原始 JSON 中。采样间隔约 2 ms，但读取不是原子快照；最慢一轮前/后为 43.33/29.20 ms，因此这些不是严格同时峰值或精确峰值保证。不同指标最大值也不一定发生在同一轮。PSS 解释共享页分摊，不能替换原 RSS 门槛；本轮观测不证明内存泄漏或因果改善。

## 定向验证与原始文件

全部在授权 Kali 隔离目录中运行，没有系统包安装、cgroup 修改或 Windows 性能重测。

- Schema 完整性清单、跳过重复初始化、缺失删除保护触发器恢复、缺失数据库不创建：通过。
- `TestOpenForRunCleanOverlap`：真实命令等待输入期间，代已变成非当前/非前代，clean 仍不能标记删除；命令完成且租约释放后才可标记。通过 0.08 秒。
- `TestCleanCompletedLinuxOrphan`、`TestRunFinishProtectsUnconfirmedTree`：完成回执清理与未知子树保护通过 0.15/0.68 秒。
- 原有后台后代等待 0.36 秒、所有者 SIGKILL 0.07 秒、独立 deadline/只读回执 0.66 秒：通过。
- Windows 下 state 新接口定向测试通过；未启用 Windows 快路径或改变 Windows 监督方式。Linux 默认构建 vet 通过。

原始目录 `.build/run-stage-evaluation`：`stages*.json`、`trace-*/*.json`、`init-version.txt`、`schema-tests.txt`、`reliability.txt`、`run-clean.txt`、`core-regression.txt`、`before-run.json`、`after-run.json`、`memory-comparison.json`、`artifacts.json`。远端同名目录位于 `/home/kali/myenv-validation-20260909` 下。旧批次报告完整保留。

最终静态检查：Windows 默认构建和 Linux runtrace 构建的 state/core/runner/runtrace/cmd/myenv go vet 均通过。候选仍未作为性能改善通过，不追加重复基准。
