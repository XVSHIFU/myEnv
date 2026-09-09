# 更新日志

## 0.1.0-rc.1（2026-09-09，本地候选，未公开发布）

- 项目与当前用户默认环境管理：Node、Python、Temurin JDK、Go、Rust，官方来源查询与校验。
- Python Windows 官网/Astral 来源选择；明确选择受支持的预览版本。
- Windows 托管 Rust 默认使用自带 LLD，规避已定位的 MSVC vctip 等待；显式自定义链接配置保留。
- 本机环境发现、详细诊断与受限的原管理器外部操作；不安装 Anaconda。
- 中文帮助与离线手册，Windows 当前用户安装向导与 PATH 配置。
- myenv list 分组摘要、--details、完整运行示例、TTY 颜色与动态阶段提示。

已知限制：原性能验收未通过；macOS 跳过，WSL 不支持；Linux python.org 源码构建未实现；C/C++ 仅检测和官方指引；Conda 未做真实安装验收；未知外部来源不接管；并非全部历史版本、厂商和变体都已验收。制品未签名。
