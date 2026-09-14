# Node 平台运行验证

必须在原生 Windows x86_64、Linux x86_64/glibc、macOS arm64 分别执行。交叉编译不代替运行结果。当前仅 Windows 有实际运行证据，Unix 两目标待运行。

从项目目录执行：

```powershell
# Windows；需要 Go（版本见 go.mod）
powershell -NoProfile -File scripts/verify-node.ps1
```

```bash
# Linux / macOS；需要 Go（版本见 go.mod）和 bash
bash scripts/verify-node.sh
```

首次运行会从 Node 官方源解析 22 系列版本、下载并校验归档、准备真实运行时并验证 Node/npm 版本。制品保留在 `.build/node-real`；再次运行复用 `prepared.json`，不重新下载。可以通过脚本的第一个参数指定独立制品目录，禁止跨宿主复用该目录。已有记录损坏或平台不匹配时验证失败，不自动覆盖。

检查包括解包边界、平台识别、真实版本验证、同步与离线无变更、缺失快照重建、准备失败保留旧代、声明漂移、CLI 参数及退出码、npm/npx 入口和直接子进程取消。Windows 不假定符号链接创建权限，因此有效 tar 符号链接测试会跳过；必须由 Unix 运行补验。

这些脚本是 Node 纵向验证入口，不是完整首版验收。进程树监督、真实 kill 恢复、共享缓存、Python、用户操作、清理与性能发布门禁仍需各自证据。应保存命令输出、宿主版本和使用的源码版本，报告时区分通过、跳过、未运行。
