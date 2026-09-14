# 文档导航

使用软件从[项目 README](../README.md)或[中文文档站](https://xvshifu.github.io/myEnv/)开始。这里保存开发约定、当前验证状态和设计依据。

## 当前文档

| 文档 | 用途 |
| --- | --- |
| [architecture.md](architecture.md) | 产品行为、环境边界、可靠性约束与性能预算 |
| [implementation.md](implementation.md) | 当前候选、实际验证、开放项与下一步 |
| [build.md](build.md) | CLI/GUI 构建、打包、许可与源码证据 |
| [windows-install.md](windows-install.md) | Windows 安装、GUI/TUI 入门，随便携包分发 |
| [performance-gap.md](performance-gap.md) | 尚未达标的性能预算与历史测量证据 |
| [acceptance-gaps.md](acceptance-gaps.md) | 首版 CLI 验收基线；当前界面状态以 implementation.md 为准 |

## 设计依据与原型

当前 GUI 的视觉和交互约定在 [DESIGN.md](../cmd/myenv-gui/DESIGN.md) 与[工作台规则](../cmd/myenv-gui/.impeccable/surfaces/workbench.md)。

[design/](design/) 保留初始架构、GUI/TUI 选型、环境调研、系统管理与终端输出方案。它们记录决策背景，实际行为以当前架构和实现记录为准。

[GUI 原型](design/prototypes/myenv-gui.html)与[TUI 原型](design/prototypes/myenv-tui.html)保留原始文件，可下载后在浏览器中打开。原型使用演示数据，不属于可运行产品或验收证据。

## 历史记录

[archive/](archive/) 保存既有阶段交付、平台验证与开发历史。归档内容保留当时的版本、结论和未完成项；其中“当前”“最新”等表述均指记录当时，不能替代 rc.11 的状态。

原生截图和当前用户指南维护在 [website/](../website/README.md)。本机原始日志、旧补丁、构建缓存与发行包继续保留于忽略目录，不随源码提交。
