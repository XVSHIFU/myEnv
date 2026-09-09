# SDK 与系统环境管理：候选实现记录

2026-09-09。新增功能在候选源码与 `.build/myenv-sdk-candidate.exe`，尚未进入 dev.2 试用包或正式发行。用户的试用记录、安装演示图片、PATH 和现有安装均保留；没有安装 Anaconda，也没有安装 C/C++ 工具链。

## 已接入的命令

```powershell
myenv versions java --search 21
myenv versions go --search 1.26
myenv versions rust --search 1.98
myenv versions python --search 3.12
myenv system list
myenv system doctor --path "D:\SDKs\Python"
myenv system install java@21
myenv system install go@1.26
myenv system install rust@1.98
myenv system upgrade java
myenv system repair java
myenv system remove java
myenv system clean --dry-run
```

以上为新候选命令，不适用于原 dev.2。`system install` 等同于管理 myenv 当前用户默认环境，不是管理员级整机安装。升级遵守已声明的版本范围；换大版本用 `system install java@25` 等显式选择。`repair` 重新准备整个默认环境，不只重建单个工具。`remove` 从活动默认环境移除工具；历史代与正在使用的目录仍受原有回退、租约和 clean 规则保护，不保证立即释放全部磁盘空间。

项目仍使用 `myenv use java@21`、`myenv use go@1.26`、`myenv use rust@1.98`，随后 `myenv run java -version` 等。Java 注入 JAVA_HOME；Go 注入 GOROOT 并使用 GOTOOLCHAIN=local，避免执行时隐式下载另一套 Go；run 不安装环境、不改写用户命令参数。

## 来源与版本范围

| 工具 | 本轮来源与安装 | 版本查询限制 |
| --- | --- | --- |
| Java | Eclipse Adoptium API 指向的 Temurin JDK 官方归档 | 分页读取各 Java major 的 GA/EA；不是 Oracle JDK 或所有 JDK 厂商集合；安装只接受稳定版 |
| Go | go.dev 完整下载 JSON 与官方归档 | include=all，筛选当前平台归档；预览版可查询，安装只接受稳定版 |
| Rust | Rust 官方稳定版归档及对应 SHA256 | 稳定版索引；beta/nightly 尚未实现 |
| Node | 原官方 Node 后端；新增官方版本目录查询 | 当前平台归档 |
| Python | 原 uv/Astral 安装后端未替换；新增 python.org 发布目录查询 | python.org 发布记录不代表存在当前平台安装包，也不代表 uv 能安装全部这些版本 |

版本不固定为几个推荐值，用户可搜索并复制完整版本安装。Java 保留四段版本、8u 与 build，例如 `jdk-21.0.12.1+1`。数字前缀表示版本族；必要时 `go@=1.2` 精确选择历史 1.2，避免匹配 1.2.x。没有实现交互式版本选择器。

支持平台仍为 Windows amd64、原生 Linux amd64 glibc；没有 macOS 验收。归档经 HTTPS 官方来源/重定向限制、SHA256、路径边界与解包限制校验，固定进入新环境代，再走原有发布流程。SHA256 由官方 HTTPS 元数据取得，不能表述为另做了独立签名验证。SDK 大包使用有界分段下载并验证完整最终摘要，不运行归档的安装脚本。

## 外部环境管理边界

清单合并 PATH、运行时 HOME、常见安装位置、Windows Python 注册信息、Conda 环境登记、uv/rustup 等已知目录及显式 `--path`。不做全盘扫描，不保证发现其他用户或任意自定义路径。

区分 available、broken、alias、shim 等状态；WindowsApps 别名不当作真实 Python，rustup 调度入口不执行，避免触发自动安装。`available` 只表示入口能报告预期版本，不能证明所有项目依赖和编译链都完好。当前 `system doctor` 与 list 共用入口诊断，尚无完整编译链健康检查。

外部安装目前仅提供 inspect；不能通过裸路径升级、修复或删除。原管理器的 mutation 适配器、外部安装切换/接管以及 Python 发行方选择尚未完成。因此还不能称为“所有 Python 来源统一管理完成”，也不能称为“官方网站所有版本均可安装”。Conda/Anaconda 只读发现没有安装或调用其安装器。

## 真实验证

测试使用隔离目录 `.build/sdk-real` 与 Kali `/home/kali/myenv-validation-20260909/sdk-real`，内部 namespace 注入；没有向用户真实默认环境安装 SDK。

| 环境 | Windows | Kali 原生 Linux |
| --- | --- | --- |
| Go 1.26.6 | 官方安装、locked noop、deep、编译运行通过，249.15s | 同流程通过，194.20s |
| Temurin jdk-21.0.12.1+1 | 同流程通过，284.97s | 同流程通过，370.00s |
| Rust 1.98.0 | 安装、noop、deep、cargo 版本通过；默认 MSVC 编译后的退出等待失败 | 全流程含编译运行通过，660.41s |

上述时间是整次集成测试时间，包含网络与安装，不是 run 性能成绩。命令为编译后的 CLI 测试二进制加 `-test.run=^TestSDKRealTrial$ -test.v -test.timeout=40m`，以 MYENV_SDK_TEST_ROOT / MYENV_SDK_TEST_SELECTION 选择隔离根和版本。

Windows Rust 的失败已定位：Job 查询显示主编译过程结束后 vctip.exe 仍在 Job，超时栈停在既有后代退出等待。不能删除后代检查来通过验收。VSCMD_SKIP_SENDTELEMETRY=1 的 30 秒诊断仍失败；显式指定该 Rust 归档自带的 rust-lld.exe 后编译 3.18 秒通过，生成程序通过 myenv run 输出 myenv-sdk-ok。没有修改系统 VS、注册表或安装新链接器。这是显式链接器方案的证据，不代表默认 MSVC 路径已修复；Rust/Cargo 原生依赖仍可能需要 MSVC/Windows SDK。

可供有经验用户验证的显式参数形式：

```text
myenv run rustc main.rs -C linker=<当前托管Rust目录>/lib/rustlib/x86_64-pc-windows-msvc/bin/rust-lld.exe -o hello.exe
```

绝对路径随环境代变化；尚未产品化为稳定项目配置入口。不得把这条诊断命令当作普通用户最终安装体验。依据 [rustc 官方 linker 选项](https://doc.rust-lang.org/rustc/codegen-options/index.html#linker)。

定向 config/backend/core/state/cli 测试及 go vet 通过，覆盖版本完整性和排序、SDK 发布入口、范围下载边界/摘要失败、官方重定向拒绝、外部发现、最后一个默认工具移除、空项目拒绝及 JSON/help。最后新增 SDK 约束合并再次单测通过。真实 system list 输出保留在 `.build/sdk-system-inventory.json`。没有做所有历史版本安装矩阵，没有新增完整 SDK 版本切换/故障/并发清理矩阵；原有可靠性证据不等同于这些组合已经实测。

早先依据目录属性推断下载零字节、依据单次直接执行推断内存输出捕获导致等待，均已被后续证据否定：打开文件可读到增长长度，Job 成员与超时栈证明等待 vctip。不能沿用这两个初步结论。

## 发行前仍需完成

1. Windows Rust 默认链接流程兼容与缺失编译依赖诊断。
2. 外部安装的原管理器适配器与明确的作用域/目标选择。
3. Python 官方安装来源选择、目录条目的可安装性说明，以及 Rust 预览渠道等版本覆盖缺口。
4. 新命令中英文本与正式站点指南集成、受影响生命周期组合回归及新试用包。
5. 原先未达标的性能指标仍开放，不降低预算、不改变租约或 Windows 监督架构。

本轮没有公开发布；当前目录仍非 Git 仓库，不存在本轮提交或远端推送。
