# 中文界面与 Windows 安装交付

2026-09-09，按用户授权执行此前记录的改进方案。交付版本 `0.1.0-dev.2`，目录 `trial/v0.1.0-dev.2`。这是开发版，不表示原 T00–T06 性能门槛全部完成。旧试用制品和用户在 trial/README.md 中的记录未改写。

## 实现范围

- CLI 按每次调用选择语言，无进程全局可变语言状态。支持 `--lang auto|zh-CN|en` 和 `MYENV_LANG`；优先级为显式参数、专用变量、系统语言、英文回退。Windows 调用本地用户 UI 语言 API；POSIX 按 LC_ALL/LC_MESSAGES/LANG 识别。
- 中文命令帮助、完整内嵌离线手册、参数解释、正常流程阶段、状态、doctor/clean、交互问题及常见失败摘要/建议。底层未分类原因保留原文；技术诊断不强行翻译。
- `status` 与裸命令共用同一处理函数；中文状态以工具列表显示，不打印 Go map。JSON 保留原数据、英文消息和错误码。
- 本地化只在 CLI 呈现边界进行，没有包裹或替换用户输出流；run 的命令参数边界保持不变。补全生成时恢复原英文元数据，四种补全脚本按字节对照一致。shell-init 输出逻辑未修改。
- Windows 安装器是独立 C# WinForms 向导，通过系统 .NET Framework 编译器打包，CLI 本身仍是 Go 命令行。默认当前用户 `%LOCALAPPDATA%\Programs\myEnv`；可选加入用户 PATH，检测同名程序，不要求管理员权限。
- 安装器检查登记文件和摘要、拒绝链接路径和不明文件；同目录升级、可回退写入、用户卸载登记、PATH 所有权与单项移除。卸载不递归删目录，保留用户添加文件和项目/运行时数据。卸载助手会临时复制到系统临时目录，以便删除安装目录中的自身副本；临时助手留给系统临时文件清理。
- 提供可复现的 Windows 安装器/便携 ZIP 打包脚本，先核验 CLI 构建清单摘要。尚未签名或公开发布，不提供包管理器渠道或系统级安装。

Linux 新制品沿用当前源码，包括上一轮已保留的 OpenForRun 候选；本轮没有新增租约/监督/状态架构修改，也没有宣称 run 性能改善。Windows Job 架构未修改。

## 有效验证

Windows 和原生 Kali 各通过 35 项顶层定向测试，无选中项跳过。包括新本地化测试、中文目录初始化、状态入口一致性、现有 JSON/错误合同、真实保留 Node/Python 运行与退出码、doctor/clean 及提示输入。记录：

- `.build/localization-windows-tests.txt`（Windows 包用时 6.630 秒）。
- `.build/localization-linux-tests.txt`（Kali 原生测试）。
- Windows/Linux `go vet ./internal/cli ./cmd/myenv` 通过。

最终发布参数制品另经 `.build/check-localized-artifact.py` 验证：中文 help/manual、原始 stdout 重定向、已有项目状态、无变更 locked sync、doctor --deep、clean --dry-run、Node stdout/stderr 中文、stdin 管道和子程序 `--lang` 透传、exit 7。Windows 还验证了真实 Python 中文 stdout/stderr；Kali Python 由上述真实保留运行时定向测试覆盖。脚本没有翻译第三方输出，中文编码验证使用显式 UTF-8 字节写入，不能据此承诺任意第三方工具管道编码相同。

- `.build/localized-release/windows-artifact-check.json`，SHA b7e2540a…。
- `.build/localized-release/artifact-check.json`，Kali SHA f26d8ca3…。
- `.build/localized-release/help-zh.txt`、`manual-zh.txt` 是真实制品输出。

安装器的 `--self-test <新目录>` 使用真实文件操作、内嵌 CLI 与模拟 ISettings，不写实际注册表/PATH。覆盖安装、重复升级、保留变量引用的 PATH、预先已有 PATH 不归本工具所有、仅移除一个自有项、卸载保留用户文件、同名冲突、登记失败回退、运行中/被修改文件保护。最终测试目录为 `.build/installer-final-test-*`，其中 result.txt 记录 PASS。通过 `--preview` 在屏幕外渲染真实表单并检查中文布局，截图 `.build/localized-packages/setup-preview.png`。没有替用户实际安装，也未将模拟设置测试冒充真实注册表集成验收。

安装器内嵌 myenv.exe 通过反射读取资源并计算 SHA-256，与发布制品独立匹配；同目录提供校验和。

## 一次受影响启动路径对照

复用原 TestMeasureInformationalCLI：51 次、首个样本单列、剩余 50 次 nearest-rank p95，help/version 交替顺序。仅将输出有效性判断扩展为接受中文“用法”，没有重写统计。设置 MYENV_LANG=zh-CN，旧版忽略该变量，新版显示中文。未清空系统缓存。

| 平台 | help p95：旧→新（ms） | version p95：旧→新（ms） |
| --- | --- | --- |
| Windows | 59.3708 → 62.8238 | 62.7692 → 61.5940 |
| Kali VMware | 101.9256 → 65.9922 | 107.3290 → 102.2262 |

原始记录：`.build/localized-release/windows-startup-{before,after}.json` 和 `startup-{before,after}.json`。旧 SHA 为 Windows 416d5f46…、Linux 75ede63a…；新 SHA 如下。仅一次前后采样，不能将差值视为翻译代码的因果开销或性能改善；两平台仍未通过 30 ms 启动门槛，没有重采样挑结果。run 延迟和内存未重测，旧开放项仍保留。

## 制品

| 文件 | 字节数 | SHA-256 |
| --- | ---: | --- |
| myenv.exe | 12889088 | b7e2540adcc43771f9bd8783c43bdaa872452db918484c07f288e52c6d8a94fb |
| myenv-linux-amd64 | 12529826 | f26d8ca37e618c2facba9a687f90504460743f66b7f1007a2080c023c5743036 |
| myenv-0.1.0-dev.2-windows-amd64-setup.exe | 12910592 | b497ca8f93b43d3f1c6eb1283030ae2c389e8cd9ae9180037b36cb2da43eadb4 |
| myenv-0.1.0-dev.2-windows-amd64-portable.zip | 5567755 | 6034b5d34814d6b8e3c6bf067f3825f728cb7aec137509c4464046a89a57fdd7 |

后续从用户实际安装和中文试用反馈推进；不据本轮交付关闭原性能目标。安装器注册表适配的真实安装/卸载以及其他 Windows 版本/DPI 矩阵尚未实际验收。
