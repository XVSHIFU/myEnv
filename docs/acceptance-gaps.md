# 首版验收缺口（2026-09-09）

> 本文保留首版 CLI 验收基线。当前 rc.11 的 GUI/TUI、包管理和开放项见 [implementation.md](implementation.md)；下文“最新”均指本记录当时。

本次范围为 Windows amd64 与 Linux amd64/glibc。用户明确跳过 macOS；历史 macOS 实现、SDK、签名和原生机器要求不再阻塞本次交付。最新候选是 `.build/kali-validated-release`，完整证据见 [Kali 与 Windows 验证记录](archive/kali-validation.md)。

| 验收组 | 最新有效证据 | 仍需明确的边界 |
| --- | --- | --- |
| 配置发现、冲突、非交互 | 原生 Kali 配置/CLI 测试、Windows 既有提示与配置验证通过 | 不将无穷输入/权限组合称为穷尽验证 |
| 首次同步、紧接无变更同步 | Windows/Kali 真实 Node/Python、混合项目、本地 wheel、镜像、独立 CA、离线复用通过；最终 Linux CLI 从 init 起直接验证 | 当前已测依赖来源不代表所有第三方构建脚本 |
| 失败保留旧环境与回退 | Kali 最终 CLI 同步失败后 run --current 可用、回滚通过；真实 Windows uv 强杀恢复既有证据保留 | 业务数据不属于环境回滚承诺 |
| run 参数、stdio、退出码、取消 | Windows 原生 Job、Linux 独立 subreaper；Kali 9 项真实 PTY/Bash；最终制品截止/回执验证通过 | 已暂停进程不能运行自己的 Go 手工取消回调；未宣称任意 context/SIGSTOP 微时序全部关闭 |
| 全局/项目隔离与 Shell | 最终 Windows PowerShell、最终 Linux Bash/zsh/fish 实际全局 Python 通过；CLI 隔离测试通过 | 不修改用户真实 Shell 配置 |
| 并发、崩溃、清理保护 | 原生状态测试、所有者强杀恢复、未知子树保留、完成回执清理通过；新原子租约与令牌登记提交/回滚验证通过 | 无完整证明时仍保留保护，不能为了清理成功删除未知对象 |
| 制品与性能 | 两平台最终制品 manifest/大小/SHA 独立一致；功能、Shell 与固定性能批次都有对应日志 | Windows 启动/status/run、Kali run 延迟不达原预算；Kali 聚合 myEnv RSS 超 32 MiB；Windows 受控冷缓存不足 |

当前主要未闭合门槛是性能。最新 Windows help/version/status/run p95 分别为 60.1930/94.8281/139.8283/98.5564 ms；Kali 为 19.2329/20.3291/34.3731/88.1465 ms。两平台无变更 sync 均通过 200 ms。原预算为启动 30 ms、status/run 50 ms，没有调整或选择性删除样本。

Kali 父进程 VmHWM 观察峰值 23.7461 MiB；调用端与监督器聚合 RSS 采样最大 39.2422 MiB；含 Node 的树为 72.1992 MiB。聚合 RSS 重复计算共享页，且采样可能漏掉瞬时峰值，不拿父进程达标替代聚合结果。Windows 父进程峰值 12.3359 MiB，有限 myEnv+Node 树各自峰值之和上界 43.4297 MiB。

是否按实测性能交付开发版已询问用户；未收到变更验收目标的答复前，不标记整体目标完成。已有可运行候选不等于所有原预算通过，也没有对外发布。
补充：用户后续限定的 run 阶段/监督合并评估已完成，见 [本轮报告](archive/run-stage-evaluation.md)。该限定任务没有要求或授权放宽原性能预算；源码新增初始化快路径候选尚未证明端到端收益，不替换上述已验证发行候选。
