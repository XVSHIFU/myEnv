# Windows / Kali 验证记录（2026-09-09）

用户授权跳过 macOS，并提供 `kali@192.168.61.129` VMware 虚拟机。本次只交付 Windows amd64 与 Linux amd64/glibc 候选；macOS 历史缺口不再阻塞本次范围。

## 环境与隔离

Kali：Linux 6.16.8+kali-amd64、glibc 2.42、2 vCPU，CPU 型号 Intel Core i7-10850H。虚拟磁盘报告 ROTA=1，不能据此认定底层为受控 SSD 参考宿主。原生 Bash/zsh/Python 可用；fish 4.7.1 与 libpcre2-32 仅下载、解包到测试目录，没有安装系统软件包或修改用户 Shell 配置。

远端目录为 `/home/kali/myenv-validation-20260909`；本地日志在 `.build/kali-validation`。用户配置、数据与缓存均使用临时测试命名空间。真实 Node 22.23.2 复用已有官方压缩包；Linux uv 0.11.26 经固定摘要验证后准备，受管理 Python 为 3.12.13。系统 Python 仅运行测试驱动。

## 修复与功能证据

- `runner/completion_error_test.go` 原先调用平台监督器再注入错误；Unix 的保守完成状态使注入没有发生。改为独立收尾边界夹具，保留真实叶子进程、原有错误/退出/关闭断言；真实进程树另由原生集成覆盖。Windows 与 Kali 修复测试通过。
- `core/download_failure_test.go` 锁文件写死 Windows；改用实际平台，确认测试真正进入下载失败与操作登记路径。Windows/Kali 通过。
- Node 镜像、缓存和 Python 镜像测试补齐 Linux 制品布局。真实镜像安装、关闭服务器后复用、独立 CA 均通过；没有放宽摘要校验。
- Linux run 将租约和随机恢复令牌在同一 SQLite FULL 事务中提交，省去独立令牌绑定事务。注入令牌 INSERT 失败时租约/身份/令牌三张表均无部分记录；另一个连接可观察完整提交。旧显式绑定路径仍保留。
- 原生配置与状态测试通过；真实 Node/Python 混合项目、本地 wheel 安装、取消、准备所有者强杀恢复通过。最终更新后的 CLI 全包通过，未启用的下载/性能门控项另行记录。
- Linux runner 的真实 Node、9 项 PTY/Bash 信号/暂停/fg/bg，以及最终监督器截止与回执协议通过。首次整包的两个边界测试失败及修复日志都保留，不能删除初次失败记录。
- 最新候选的 `RunFinishProtectsUnconfirmedTree` 与 `CleanCompletedLinuxOrphan` 通过：完整确认可清理，监督器被杀后未知子树仍保留租约与环境。
- 最终 Linux 候选直接执行 `init → sync → 离线 sync → run → doctor`；随后离线修改配置、同步失败保留活动环境、`run --current`、回滚和 clean 预览均通过。
- Bash/zsh/fish 的 shell-init 与全局 Python run 显式使用最终 Linux 候选，输出 3.12.13、退出 23 均正确。Windows 镜像回归及最终候选 PowerShell 实际全局 Python 接入通过。

主要日志：`runner-boundary-fixed.txt`、`runner-native-kali.txt`、`config-native.txt`、`state-native.txt`、`backend-native.txt`、`core-native.txt`、`cli-native.txt`、`core-followup.txt`、`cli-followup.txt`、`python-mirror.txt`、`node-cache.txt`、`atomic-state.txt`、`atomic-core.txt`、`atomic-cli.txt`、`atomic-shell.txt`、`artifact-recovery.txt`、`windows-mirror-regression.txt`。

`backend-native.txt` 有一次门控参数错误：设置 UV_PREPARED 同时误触发需要独立目录和版本的 Python 安装测试。该安装此前已在 `python-prepare.txt` 以完整参数通过；不把错误调用记成产品故障或整包通过。`core-native.txt` 的 Windows 锁夹具失败以 `core-followup.txt` 的修复验证补齐。

## 候选与性能

`.build/kali-validated-release` 使用 Go 1.26.6、CGO0、trimpath、buildvcs=false、去符号，未对外发布：

| 制品 | 字节 | SHA256 |
| --- | ---: | --- |
| myenv-windows-amd64.exe | 12835840 | 416d5f4699ec39a8eaf9dfc6e240a0cb8f02542651914728ff22e8502a99caab |
| myenv-linux-amd64 | 12476578 | 75ede63a747902e5d9919863f1be92f9be64c8b49b97d5a60c43b9343d013591 |

Kali 最新候选固定批次：每项 51 次，首次单列，后 50 次 nearest-rank p95；run 使用轮换顺序的相同 Node 直接运行基线。采样没有并发测试，没有择优选样。

| 场景 | p95 ms | 原预算 ms | 结论 |
| --- | ---: | ---: | --- |
| help | 19.2329 | 30 | 通过 |
| version | 20.3291 | 30 | 通过 |
| status | 34.3731 | 50 | 通过 |
| 混合无变更 sync | 40.8718 | 200 | 通过 |
| run 附加开销 | 88.1465 | 50 | 未通过 |

初始反馈热 p95 为 20.2619 ms，所有样本最大 23.8718 ms。报告为 `perf/atomic-{informational,run,noop,status}.json`，都绑定 Linux SHA。前一候选的 run p95 77.7811 ms 也失败；不同批次不能证明合并事务提高或降低端到端性能。减少了一次提交是实现事实，不等于性能门禁通过。

20 次新候选 run 内存采样：父进程观察到的 VmHWM 最大 24316 KiB；myEnv 调用端加监督器的聚合 RSS 采样最大 40184 KiB；包含 Node 的完整子树采样最大 73932 KiB。父进程子项低于 32 MiB，聚合 myEnv RSS 超过 32 MiB，不能用父进程数字替代；聚合 RSS 会重复计算共享页。约 2 ms 的采样可能遗漏瞬时峰值。

5 次指定制品文件冷缓存 help：每次 posix_fadvise 后用 mincore 确认 3047 页均未驻留，耗时 66.2624、53.5067、52.1359、51.0053、53.6515 ms。仅制品文件冷缓存，不是整个系统/共享库/目录缓存冷启动。报告 `artifact-probe-atomic.json` 同时记录 CLI 操作、内存和冷缓存原始数据。

Windows 同一最终 SHA 的固定批次：help p95 60.1930 ms、version 94.8281 ms、status 139.8283 ms、run 附加开销 98.5564 ms，均未通过相应原预算；混合无变更 sync p95 95.9234 ms，通过 200 ms。初始反馈热 p95 60.8659 ms、所有样本最大 77.3251 ms。报告 `perf/windows-{informational,run,noop,status}.json`；所有样本保留，没有因结果较差重跑取低值。

Windows 有限树内存报告 `perf/windows-memory-tree.json`：20 次、myEnv 与一个等待 500 ms 的真实 Node，跨退出持有两个原始进程句柄读取 PeakWorkingSet。父进程最大 12935168 bytes（12.3359 MiB）；两进程各自峰值之和最大 45539328 bytes（43.4297 MiB），是同时峰值上界，不是同时聚合峰值，也不覆盖任意用户子树。测试日志只汇报父进程子项，完整树数据在 JSON。

Windows 镜像与最终 PowerShell 回归通过 76.420 秒；最终内存测试通过 16.466 秒。`go vet ./internal/state ./internal/core ./internal/cli ./internal/runner` 通过。两份 manifest 中的大小/SHA 已独立比对，Windows `--version` 与 `help manual` 实际运行成功。首次误用 `manual` 子命令返回 USAGE_ERROR；正确手册输出在 `manual-verified.txt`，不把错误调用算通过。

## 暂停与取消的边界

Linux 的 OS 暂停会冻结调用端所有 Go goroutine。已暂停进程中的手工 context 取消回调要等 fg/SIGCONT 后才能执行；尚未写入生命周期管道的取消不能宣称已经送达监督器。独立监督器仍处理已传入的绝对 deadline，并在完成后唤醒调用端。真实 CLI 的信号、暂停恢复和 deadline 路径已有测试；没有把任意 Go 手工取消与 SIGSTOP 交界的时序证明为全部关闭。README 和手册已明确这一限制。

## 未完成的验收项

Windows 与 Kali 的部分延迟仍超原预算；Kali myEnv 聚合 RSS 也不应标成 32 MiB 门禁通过。Windows 受控制品冷缓存证据仍不足，有限 Node 树的内存上界不能代表任意依赖构建/用户程序。没有更改预算、关闭安全检查或降低 SQLite 持久化设置来换取通过。是否按实测性能交付开发版已向用户询问，在用户答复前仍保留原验收目标。

最终补充核验：Linux 目标的 state/core/cli/runner go vet 通过；四份 atomic 性能报告 SHA 均独立核验匹配最终 Linux 制品。最终监督器三项截止/回执测试通过 0.69 秒，日志 atomic-supervisor.txt。
