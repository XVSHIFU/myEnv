# v0.1.0-dev.3 SDK 与系统管理试用交付

2026-09-09。试用目录：`trial/v0.1.0-dev.3`。这是未签名开发版，不是原性能门槛全部达标的正式发布。没有安装 Anaconda、C/C++ 或修改用户现有默认 SDK、PATH/注册表；新增 SDK 安装验证使用工作区及已授权 Kali 的隔离目录。没有公开发布。旧 dev.2 README 和五张重要截图未改写。

## 已交付

- Windows 托管 Rust：未配置链接器时，直接 rustc 增加同包 rust-lld 选项，Cargo 使用目标链接器变量；原参数保留，链接选项放在 `--` 之前，跨目标不套用 Windows 默认。显式配置优先，`--rust-linker=system` 可关闭调整。默认 MSVC 曾启动长驻 vctip，导致既有 Job 等待；本次解决方式是选择官方 Rust 包内已有的链接器，没有删除进程确认、修改 Windows Job 或租约。已有自定义 Cargo 编译配置采用保守策略，必要时显式选择 bundled；使用 system/自定义 MSVC 仍可能保留原 vctip 行为。
- Python 来源：`use python@3.14 --provider python.org|astral`，`system install` 同样支持。来源写入 tools.python（例如 python.org/3.14），参与声明摘要与锁有效性，失败保留旧活动代。
- 官网 Python：分页读取 Python install manager 的官方 Windows 索引，下载其中完整 x64 runtime ZIP，经 SHA256 校验后在新代解包；排除嵌入式、测试、free-threaded 变体，不运行 EXE/MSI，不写系统注册表。随后用固定 uv 创建独立 venv。索引如强制要求本后端尚不能验证的目录签名则拒绝，不忽略要求；目前实际官方索引没有该强制字段。当前验证基于官方 HTTPS+SHA256，不能描述为完整独立签名验证。
- 预览版：Python a/b/rc、Go beta/rc、Temurin EA（含按日期命名的 beta）、Rust beta/nightly 与日期选择、Node nightly/rc/v8-canary。预览必须显式选择，稳定范围不自动升级到预览版。Java 发行名与运行时版本分开记录，Rust 渠道锁定实际日期/版本/制品摘要。`versions java --major 21` 减少不必要分页；Rust `--channel nightly --date YYYY-MM-DD` 精确查询日期，不伪造官网未提供的全历史 nightly 索引。
- 系统检测：运行时发现保留别名/调度入口区别；doctor 增加 C/C++/CMake 入口、MSVC 登记、Windows SDK x64 库的存在性检查与官方安装指引。存在性不等于完整编译测试。
- 外部环境：`system external ID --manager ABSOLUTE_EXE --action upgrade|repair|remove` 默认仅计划，`--apply` 才执行。查询原管理器身份与成员记录，核验解释器/目录文件身份以处理 uv 的 junction 别名，拒绝 myenv 自有代/共享运行时从外部入口绕过保护。使用已有 junction-aware 路径解析。外部操作由原管理器执行，使用其渠道/镜像配置，不享有 myenv 新代事务回退保证。
- 帮助补正：Cobra 延迟注册的 help 子命令在中文主帮助里也翻译；系统清单的人类状态标签中文化，机器枚举保留。离线手册加入本轮命令和边界。

## 外部适配器的真实能力

| 来源 | 能力 | 验证与限制 |
| --- | --- | --- |
| uv | 原安装修复、移除；原生预览补丁升级 | 隔离 Python 3.12.13 的真实 repair/remove 通过；升级核对固定 uv CLI 帮助，未执行升级矩阵。可能影响同 minor 关联 venv，计划中说明 |
| rustup | 更新、移除原工具链 | 最终制品对本机既有 stable 工具链生成只读更新计划通过，没有操作用户真实工具链；不假称普通 update 是强制修复，repair 明确拒绝 |
| Conda | 非 base Python 更新、按已安装版本/build 强制重装、移除整个环境 | 成员核验、base 拒绝和修复版本/build 锁定的协议夹具通过；未安装 Conda/Anaconda，因此不称实际 Conda 安装验证通过 |
| 其他安装器/未知来源 | 发现与诊断 | 未实现其可靠原管理器适配器时拒绝修改，不能称全来源无条件接管 |

Conda 移除的是整个环境及包，不只是 Python。Conda base 受保护。外部方案是按调用显式选择目标及原管理器，没有创建统一的后台接管服务。

## 实际验证记录

Windows 隔离根 `.build/sdk-real`、`.build/sdk-preview`、`.build/external-validation`；Kali 根 `/home/kali/myenv-validation-20260909/sdk-preview`。测试原始输出在任务工具记录，Python/SDK 项目保留 stdout/stderr 文件；元数据查询保留 `.build/*-preview-catalog.json`、`.build/rust-nightly-catalog.json`、`.build/rustup-external-plan.json`。

| 场景 | 结果 |
| --- | --- |
| Windows Rust 1.98.0 rustc/Cargo 默认链接、运行、退出 | PASS 3.98s，复用前轮已安装归档；最终发布二进制另通过 rustc 编译和产物运行 |
| Windows python.org 3.14.7 安装/noop/deep/venv/SSL/SQLite | PASS 32.02s；首次因缺少新代父目录失败，修复后复用有效缓存重试 |
| Python Astral→失败版本→旧环境 run --current→官网来源切回 | PASS 76.07s；锁记录与来源一致。固定 uv 的最新 3.14 补丁为 3.14.6，官网当时为 3.14.7 |
| Windows Python 3.15.0rc2 | 安装/noop/deep/运行 PASS 144.76s |
| Windows Temurin jdk26u-2026-09-05-13-43-beta | 安装/noop/deep/javac/java PASS 241.19s |
| Windows Node 27.0.0-nightly202609082987a5965f | 安装/noop/deep/运行 PASS 61.39s |
| Windows Go 1.27rc3 | 安装/noop/deep/编译运行 PASS 233.87s |
| Kali Rust nightly-2026-09-09（rustc 1.100.0-nightly） | 安装/noop/deep/rustc/Cargo/产物运行 PASS 646.58s |
| 外部 uv 隔离运行时真实修复与移除 | PASS 56.42s；相邻 keep.txt 保留；最初被 uv minor junction 与 Windows EvalSymlinks 失败影响，已改用文件/目录身份核验 |

这些时间包含安装、网络及验证，不是 run 开销的性能成绩。不重测前轮已经通过、且运行路径没有实质变化的 Java/Go 稳定版矩阵。没有声称逐个历史版本或全部发行变体都已安装验收。

执行了 config/backend/core/state/cli 的定向测试：Provider、OfficialPython、RustChannel、RustLinker、RustCargo、External、SDK、Catalog、Inventory、System、RemoveDefault、Lock、Profile、Use、Publish、Run、Language/Locale、Manual/Help。测试通过；旧 Run 错误断言因系统中文使用全角冒号而失败，给这些英文断言显式设置 MYENV_LANG=en 后重跑通过，没有改生产默认语言。go vet 对相关包通过。最后的外部路径保护与 Conda 修复 pin 补测通过；本机最终 rustup 计划验证经过新的路径保护。

最终 Windows 二进制版本/中文帮助、官网 Python SSL/SQLite、Rust 编译与运行通过；Kali 最终二进制版本、Rust 产物运行及 SHA256 一致。安装器内嵌 myenv.exe 摘要与最终 CLI 一致。安装器逻辑未变，复用 dev.2 已有验证，未执行本机真实安装/升级向导。

## 构建与试用

实际构建：`scripts/build-release.ps1 -Version 0.1.0-dev.3 -Targets windows-amd64,linux-amd64 -OutputDirectory .build/sdk-release`。
实际打包：`scripts/package-windows.ps1 -ReleaseDirectory .build/sdk-release -OutputDirectory .build/sdk-packages`。
制品与摘要：`trial/v0.1.0-dev.3/build-manifest.json`、`SHA256SUMS`。安装器只负责 myenv 程序与可选当前用户 PATH，不会安装 Anaconda 或整套 SDK。

Windows CLI SHA256：`eb6880e111bfebd712356e84e5c4b73f1de8e6efd5b78af65a79d9d59b18e072`。
Linux CLI SHA256：`4aac0b75ead2b378ad3fe4ba269a15e08f3d7916880fd9b9151f490eee541278`。

测试 README 给出默认只读 inspect，再按需选 Python、Rust、Java、Go、Node 和 catalog；脚本会记录日志，未预先同步 trial 示例，避免替用户把所有 SDK 都装一遍。提供安装器、便携 ZIP、独立二进制、中文离线手册。试用脚本 inspect 实际执行通过，随后补正并验证帮助翻译和状态标签。

## 保留的边界

- Linux 官网 Python 源码构建未实现；该平台选择 python.org 会明确报错，Astral 后端仍可用。不会伪称上游有不存在的通用 Linux 官方二进制。
- Windows 全历史 ZIP、Python 特殊变体、所有 JDK 厂商、全部预览版本安装矩阵都不是本轮已验证范围；官方缺本机包、来源/摘要缺失时拒绝。
- 自定义 MSVC/系统链接行为可能仍等待 vctip；新默认 LLD 不等于所有 C/C++ 原生依赖不再需要微软工具链。
- C/C++ 仅检测及官方指引，未自动安装。外部适配器限制见上表，不宣称所有环境全部接管。
- 没有修改原 Windows Job、Linux 监督器、租约或性能预算；原性能开放项仍保留。没有发布网站或改写旧用户截图；目录仍非 Git 仓库。

参考：[Python Windows 官方文档](https://docs.python.org/3/using/windows.html)、[Python 官方索引](https://www.python.org/ftp/python/index-windows.json)、[rustc linker](https://doc.rust-lang.org/rustc/codegen-options/index.html#linker)、[Cargo 配置](https://doc.rust-lang.org/cargo/reference/config.html)、[Rust 工具链渠道](https://rust-lang.github.io/rustup/concepts/toolchains.html)、[Adoptium API](https://api.adoptium.net/q/swagger-ui/)、[Go 官方下载](https://go.dev/dl/)、[Node nightly](https://nodejs.org/download/nightly/index.json)、[uv CLI](https://docs.astral.sh/uv/reference/cli/)、[Conda install](https://docs.conda.io/projects/conda/en/stable/commands/install.html)。
