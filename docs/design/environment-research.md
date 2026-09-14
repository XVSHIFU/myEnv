# myEnv 开发环境小调研与首版范围建议

日期：2026-09-09。性质：供用户审阅的调研建议，未实施新后端，也未改变发行验收要求。平台限定 Windows amd64、原生 Linux amd64；macOS 按用户要求跳过。

## 结论

建议首版围绕 **Python、Node.js、Java JDK、C/C++ 基础开发**，配套 Git、CMake 和 Maven 使用指南。Go、.NET SDK、Rust 进入下一批；数据库服务、容器、移动开发 SDK、GPU 驱动与大型 IDE 不作为本次统一安装的必备项。

这里的优先级是结合课程用途、跨领域复用和安装复杂度作出的产品判断，不是语言市场份额排名。若要求 C/C++ 也必须完全由 myEnv 隔离安装，需要先验证发行来源及系统依赖；不能把“检测到系统 gcc”算成安装能力完成。

## 调研依据与局限

开发者侧参考 2025 Stack Overflow 和 JetBrains 官方调查。前者指出 Python 使用增长，Visual Studio/VS Code 仍居 IDE 前列；后者指出 Web 是主要部署平台，TypeScript 增长明显，Go/Rust/Kotlin 持续增长。这支持先覆盖 Web、后端和 Python 生态，但不等于所有人都需要安装全部语言。[Stack Overflow](https://survey.stackoverflow.co/2025/technology)、[JetBrains 技术趋势](https://devecosystem-2025.jetbrains.com/tools-and-trends)

JetBrains 样本为清洗后的 24,534 人，并说明即使加权仍可能存在产品用户偏差。大学生部分采用公开课程和教学资料作为用途样本，不能代表中国所有高校的使用比例，也不能把调查里的“学习编程者”直接当作大学生。动态图表提取不完整，因此本报告不编造语言使用率或覆盖率。[调查方法说明](https://devecosystem-2025.jetbrains.com/)

## 开发者与学生常见环境

下表为典型组合，同行工具通常是备选，绝非需要全部安装。课程直接证据见表后；未逐项取样的细分领域作为选型清单，不作为普及率结论。

| 方向 | 开发者常见组合 | 学生课程/项目常用组合 | myEnv 的切入点 |
| --- | --- | --- | --- |
| 编程入门、算法、软件工程 | Python、Java、C/C++、Git | Python/C/C++、JDK；VS Code、IntelliJ IDEA 等编辑器 | 解释器、编译器和版本切换 |
| Web 前端、全栈 | Node.js、npm/pnpm、TypeScript、浏览器 | Node.js、npm、HTML/CSS/JavaScript、课程框架 | Node 提供基础；框架与 TypeScript 属项目依赖 |
| 后端、企业应用 | JDK + Maven/Gradle；Python；Go；.NET；PHP + Composer | Java Web、Python Web、SQL 数据库、接口测试工具 | 先 JDK/Python/Node；构建工具按项目需要 |
| 人工智能、数据科学 | Python、NumPy/Pandas、Jupyter、PyTorch；部分用 R | Python、Notebook、数据处理和机器学习库 | 隔离 Python 依赖；GPU 环境另处理 |
| 操作系统、体系结构、网络 | Linux、C/C++、GCC/Clang、GDB、Make/CMake | Linux、GCC、GDB、Git、模拟器与课程代码 | 基础编译调试；模拟器按课程另配 |
| 网络安全 | Linux、Python、C/C++、调试及抓包工具 | Kali/Linux、Python、GDB、Wireshark 等 | 支持语言环境；安全工具不一揽子安装 |
| 嵌入式、物联网 | C/C++、交叉编译器、厂商 SDK、PlatformIO | 单片机/开发板工具链、烧录及串口工具 | 后续按目标板适配，不能只装宿主 GCC |
| Android、跨平台移动 | JDK、Android SDK、Gradle/Kotlin；Flutter/Dart | Android Studio、模拟器、课程指定 SDK | JDK 可先复用，SDK/模拟器分期 |
| 游戏、图形、桌面 | C++、C#/.NET、Unity/Unreal；图形库 | C/C++ 或 C#、引擎/课程图形框架 | 通用工具链先行；引擎及驱动另配 |
| 云原生、运维 | Go、Python、Shell、Git、Docker、kubectl | Linux、脚本、容器实验 | 语言和 CLI 工具先行；容器服务另配 |
| 大数据、科研计算 | Java/Scala、Python、Spark；R/Julia/MATLAB | 数据处理框架、统计或专业计算软件 | 按需求扩展，不把集群/商业软件纳入首批 |

课程证据：Harvard CS50 明列 C、Python、SQL、HTML/CSS/JavaScript；Berkeley CS61B 的项目使用 Java；南京大学 ICS 实验列出 build-essential、GDB、Git 等。这说明语言运行时之外，编译、调试与版本控制也是学生的实际需求。[CS50](https://cs50.harvard.edu/x/2025/syllabus/)、[CS61B](https://sp26.datastructur.es/proj5/proj5b/)、[南京大学 ICS](https://nju-projectn.github.io/ics-pa-gitbook/ics2024/0.3.html)

AI 教学资料《动手学深度学习》使用 Python、Jupyter 和深度学习框架；这里只引用环境组成，不照搬其较旧的版本固定值。Android 文档说明 JDK 与 Gradle 的兼容要求；PlatformIO 文档展示嵌入式环境涉及多种目标平台。[D2L](https://d2l.ai/chapter_installation/index.html)、[Android JDK](https://developer.android.com/build/jdks)、[PlatformIO](https://docs.platformio.org/en/latest/what-is-platformio.html)

## 官方来源安装的可行性与优先级

| 项目 | 来源及实现判断 | 建议 |
| --- | --- | --- |
| Python | 当前通过 uv 获取 Astral 的 python-build-standalone；属于第三方 CPython 发行构建，不是 python.org 安装包 | 保留已验证后端，明确发行方；严格只允许 python.org 时须另立后端评估 |
| Node.js + npm | Node 官方预编译包及校验信息；已有后端 | 首版保留，完善来源展示 |
| Java JDK | 建议 Eclipse Adoptium 的 Temurin JDK 归档；它是 OpenJDK 发行版，不能标成 Oracle JDK | 首个新增后端；包含 javac，处理 JAVA_HOME 与项目隔离 |
| C/C++ | Windows 可评估 MSYS2 UCRT64 GCC 完整工具链；Linux 通常使用发行版包。两者都不是单个 GCC 官网通用二进制 | 首版重点验证；明确隔离管理和系统前置依赖的界线 |
| Git | Git 官网指向 Git for Windows，提供便携发行；Linux 需要单独选定获取方式 | 首版配套；不接管已有 Git 配置和凭据 |
| CMake | 官方二进制归档便于管理，但不包含 C/C++ 编译器 | 随 C/C++ 支持；不能单装 CMake 就宣称环境完整 |
| Maven / Gradle | 优先使用项目已有 Wrapper；Maven Wrapper 能下载项目指定 Maven | 首版验证 JDK 下实际构建；不要强制全局安装两套构建工具 |
| Go | Go 官方下载，工具链组合较完整；涉及 cgo 时另需 C 编译器 | 第二批优先，安装成本相对可控 |
| .NET SDK | 微软官方 SDK；只装 Runtime 不能满足开发编译需要 | 第二批；Linux 系统依赖另验证 |
| Rust | 官方 rustup 管理 rustc/cargo；Windows 默认 MSVC 路径还需 C++ 构建工具 | 第二批，在完整编译验证后宣称支持 |
| 数据库 | PostgreSQL/MySQL/Redis 等涉及数据目录、端口、进程及升级；SQLite CLI 可独立考虑 | 服务管理另立范围，不塞进环境代清理 |
| Android SDK、CUDA、Docker、IDE | 涉及平台组件、驱动、服务或独立应用生命周期 | 先提供官方链接和前置条件说明，后续专项支持 |

来源：[uv Python 发行说明](https://github.com/astral-sh/uv/blob/main/docs/guides/install-python.md)、[Node 下载](https://nodejs.org/en/download)、[Temurin](https://adoptium.net/installation)、[MSYS2](https://www.msys2.org/)、[Git Windows](https://git-scm.com/install/windows)、[CMake](https://cmake.org/download/)、[Maven Wrapper](https://maven.apache.org/tools/wrapper/)、[Go](https://go.dev/doc/install)、[.NET Windows](https://learn.microsoft.com/en-us/dotnet/core/install/windows)、[Rust Windows 前置条件](https://learn.microsoft.com/en-us/windows/dev-environment/rust/setup)、[PostgreSQL](https://www.postgresql.org/download/)。以上是文档可行性评估，未实际试装新环境。

## “干净安装”的产品约定建议

1. 明示软件、发行方、版本、平台和下载来源。区分上游官方包、发行方官方包、Linux 发行版仓库；不能统称为语言官网原版。
2. 使用固定版本与可核验摘要；来源提供签名时验证签名。下载缓存可复用，来源与校验结果可追溯，不使用来历不明的重新打包安装器。
3. 尽量安装到 myEnv 管理目录，允许多版本并存。默认只在项目运行上下文配置 PATH/JAVA_HOME；系统级前置安装需单独说明影响，不能承诺与普通归档一样回滚。
4. 已有环境先检测并标记为外部安装，不覆盖、不自动卸载，也不把外部目录交给 clean。选择使用外部环境不等于 myEnv 拥有其生命周期。
5. 框架和库遵循各自项目清单。装好 Python 不代表已经装好 PyTorch；装好 JDK 不代表 Android SDK 已可用。初学者文档提供最短成功路径，工程师保留显式版本与机器可读状态。

## 实施顺序与上线条件建议

第一步补 Java JDK，并让 Python/Node/Java 的安装来源、已有环境处理和最小运行示例一致。第二步验证 C/C++ 完整工具链及 Git/CMake 配套；Windows 与 Kali 分别说明托管能力。如果只能给出系统安装指南，支持表必须写“外部依赖”，不能算作托管安装完成。第三步再决定是否将安装相对独立的 Go 纳入首版，避免为凑语言数扩张范围。

每个新增后端至少实际验证：无预装情况下安装成功、版本探测、最小程序编译/运行、两个版本切换、已有外部版本不受影响、失败保留旧环境、清理只删除自身拥有且未使用的内容。只对修改范围做定向回归；涉及共享运行/清理路径时再补相关可靠性场景，复用既有端到端统计方法。

当前源码仅接受 node/python，其他均未实现。正式上线仍需处理原有 Windows/Linux 性能开放项，不能仅因新增安装项完成而视为旧预算达标。公开发行与文档部署也尚未完成。本轮只形成建议，未安装环境、修改 CLI 或发布制品。
