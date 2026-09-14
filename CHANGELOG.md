# 更新日志

## 0.1.0（2026-09-14，首版正式发布）

[下载 0.1.0](https://github.com/XVSHIFU/myEnv/releases/tag/v0.1.0)

- Windows GUI 整合森林绿工作台、固定任务区与原生图标；区分本机 PATH 检查、myEnv 默认工具链和项目环境，首次检查可跳过并保存记录。
- Node.js / Python 语言下增加已检查管理工具的紧凑快捷入口，直接打开右侧管理抽屉；多个安装位置先明确归属，窄窗口保留入口。
- 包与工具支持按明确位置安装、更新、切换版本和卸载。包列表默认10项，可搜索、全选匹配项和批量管理；同次最多100包，执行管理器自身要求单项操作。
- 接入 npm 官方搜索和版本目录、PyPI 完整包名查询；期望支持最新版和精确版本，计划绑定具体版本、安装位置和原请求，确认后单次执行。
- Node 项目保留原生声明与精确锁；Python 预览不执行启动钩子，活动 myEnv Python 代不能原地改包。失败和取消展示逐项结果，不承诺外部包自动回退。
- Java 支持常见名称、1.8 / Java 8 别名与稳定版本优先；Windows/Kali TUI 只有明确确认运行后才交接终端，信号退出不会误运行命令。
- 修复 Windows Python 环境核验中的路径别名误判：目录联接和 8.3 短路径正确对应同一环境，同时保留不同环境拒绝与预览启动钩子隔离。
- 项目采用 [MIT 许可证](LICENSE)，第三方组件保留各自许可声明。

分发范围：Windows amd64 CLI/TUI/GUI、原生 Linux amd64/glibc CLI/TUI。GUI 仍需 WebView2；便携包包含 GUI，setup 仅安装 CLI/TUI，制品未签名。发行包与校验清单见 [0.1.0 下载页](https://github.com/XVSHIFU/myEnv/releases/tag/v0.1.0)。

已知限制：既有性能预算未全部通过；Linux GUI 暂缓、WSL TUI 门禁仍在；高 DPI、真实 IME、无 WebView2 干净机器、管理工具自身真实写操作等验收仍开放。包管理禁用安装脚本、Python 仅 wheel；复杂 workspace、未知归属和独立 uv 卸载交回原管理器。更多边界见 [当前记录](docs/implementation.md)。

## 0.1.0-rc.1（2026-09-09，本地候选，未公开发布）

- 项目与当前用户默认环境管理：Node、Python、Temurin JDK、Go、Rust，官方来源查询与校验。
- Python Windows 官网/Astral 来源选择；明确选择受支持的预览版本。
- Windows 托管 Rust 默认使用自带 LLD，规避已定位的 MSVC vctip 等待；显式自定义链接配置保留。
- 本机环境发现、详细诊断与受限的原管理器外部操作；不安装 Anaconda。
- 中文帮助与离线手册，Windows 当前用户安装向导与 PATH 配置。
- myenv list 分组摘要、--details、完整运行示例、TTY 颜色与动态阶段提示。

已知限制：原性能验收未通过；macOS 跳过，WSL 不支持；Linux python.org 源码构建未实现；C/C++ 仅检测和官方指引；Conda 未做真实安装验收；未知外部来源不接管；并非全部历史版本、厂商和变体都已验收。制品未签名。
