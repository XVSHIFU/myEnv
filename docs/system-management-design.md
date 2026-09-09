# 系统与项目环境管理扩展

2026-09-09，依据用户本轮要求调整原架构范围；这是实施合同，不是完成清单。

- 不安装 Anaconda。已有 Conda/Anaconda 安装允许发现和诊断，位置可能属于当前用户，也可能属于整机。uv 是管理工具，CPython/Anaconda 是不同的发行或环境体系，列表分别记录工具、发行方、所有者与作用域。
- 保留项目 init/sync/use/run 的声明与租约合同。新增 Java JDK、Go、Rust；C/C++ 独立安装暂缓，Rust 所需链接器作为前置依赖诊断。
- 系统管理入口使用 `myenv system`；现有 `--global` 仍表示当前用户默认环境，不偷换为管理员安装。系统清单覆盖已知注册信息、PATH、标准位置、管理器登记及显式路径；报告探测范围和错误，不能声称发现任意磁盘位置的一切安装。
- 外部安装必须记录来源与管理能力。只发现路径不能取得删除权限；升级/修复/删除须通过确认身份的原管理器适配器。没有可靠适配器时明确只读、给出原生处理方式，不能提供假成功或直接递归删除。
- 版本查询使用官方完整索引或分页，不固定几条推荐版本。区分发布目录、当前平台制品、稳定/预览状态与实际安装支持。某版本已发布但无本机包、缺组件或来源证据不足时显示不可安装及原因。
- Java 首先采用 Eclipse Adoptium/Temurin 官方 API 与其指向的发行归档，明确不是 Oracle JDK。保留完整版本及 build 标识；Java 四段版本/8u 不能无提示截成三段。
- Go 使用 go.dev 完整下载 JSON；Rust 使用官方发布目录及固定版本 manifest，稳定版与 beta/nightly 区分。Python 分别显示 python.org 与 Astral 构建的来源和可用性，不把一个发行目录当作全部 Python 版本。
- 新后端必须经过来源、重定向、摘要校验、解包边界和真实编译/运行验证；现有租约、失败保留旧代、删除保护及性能预算不变。

计划入口：`myenv versions <tool>` 查询/搜索版本；`myenv system list` 清单；`myenv system doctor` 深度诊断；`myenv system install/upgrade/repair/remove` 管理明确拥有或具有原管理器适配器的安装。实现前不能将这些命令写进已支持指南。

参考：[Adoptium API](https://api.adoptium.net/q/swagger-ui/)、[Go 下载](https://go.dev/dl/)、[Rust 渠道](https://rust-lang.github.io/rustup/concepts/channels.html)、[Conda 信息查询](https://docs.conda.io/projects/conda/en/stable/commands/info.html)。
