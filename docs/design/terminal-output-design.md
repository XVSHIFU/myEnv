# 终端输出改进（2026-09-09）

## 调研依据

- GitHub CLI：https://cli.github.com/manual/gh_help_formatting 。提供对齐表格和 JSON 格式化，适合摘要与机器数据分离。
- uv：https://docs.astral.sh/uv/getting-started/help/ ，通过 verbose 展开诊断；https://docs.astral.sh/uv/reference/cli/ ，颜色按终端能力自动启用。
- CLI Guidelines：https://clig.dev/ ，强调可读输出、错误修正建议、终端与脚本使用的区别。

## 本次落地

原输出直接打印制表符，长路径和完整 JVM 输出占据主体，安装完成示例缺少命令参数。改为无需中文字宽计算的分组行布局；默认显示工具、紧凑版本、状态、管理来源，异常原因仍展示。完整路径、ID、原始版本通过 system list --details 查看；system doctor 默认详细。版本摘要只用于显示，原始证据和 JSON 不变。

新增 myenv list 为 system list 快捷入口（同一处理逻辑与扫描范围）；安装/升级/修复成功提供工具对应的完整版本验证命令；run 无参数报错提供完整示例，中英文均支持。用户程序输出不做翻译或装饰。

本次采用纯文本，无 ANSI 控制字符、动画或新依赖，重定向也可读。颜色与动态进度尚未实现；后续若增加，须检测 TTY、尊重 NO_COLOR，并保持非终端日志逐行输出。现有阶段进度保留，不伪造百分比。通用错误框架仍保留错误码与原始原因，本次只修正漏写 run 参数的具体提示。

## 验证与交付

通过 go test ./internal/cli -run 'Test(System|Run|Locale|Language|Help)' -count=1；通过新增 InventoryOutput / RunMissing 定向测试，覆盖摘要隐藏路径与 ID、详细保留中文路径、异常不隐藏、完整修正命令。go vet ./internal/cli 通过。

通过 scripts/build-release.ps1 生成 Windows dev.4，实际执行 list 成功识别用户刚安装的 JDK 17 和其他入口。未重新安装 SDK、未修改系统 PATH，未重跑无关性能和 Linux 原生验收。交付 trial/v0.1.0-dev.4/myenv.exe 和测试说明，dev.3 记录未修改。本次无安装器与 Linux 制品。


## dev.5 已实现颜色和动画

自动检测输出流 TTY 与 Windows VT 能力，颜色仅用于分组、状态和系统操作完成提示。NO_COLOR 非空或 TERM=dumb 时禁用颜色与动画。阶段动画每 120ms 刷新，同一时刻一个 goroutine，普通输出前清行、返回时停止并等待退出；不隐藏光标、不伪造百分比。仅交互、非 JSON、非 verbose/no-input 操作启用；run 不包装输出。不改变核心阶段通知、下载、租约或监督机制。

定向 CLI 回归与 vet 通过，Progress 测试覆盖输出交接、停止和非终端无控制字符；Windows PTY 实际彩色列表通过（测试进程临时清除 NO_COLOR 设置）。下载过程和用户 Ctrl-C 的终端视觉效果尚未进行实机验收，提供 dev.5 README 供试用。旧版试用记录完整保留。
