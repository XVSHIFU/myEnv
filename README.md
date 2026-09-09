# myEnv

[中文使用文档](https://xvshifu.github.io/myEnv/) · [发行包](https://github.com/XVSHIFU/myEnv/releases) · [性能差距](docs/performance-gap.md)

myEnv 用声明文件管理项目与当前用户默认的 Node/Python/Java JDK/Go/Rust 开发环境。日常流程是 `init → sync → run`；`sync` 准备新环境代，成功后切换活动引用，失败时保留原环境。`run` 使用已应用环境，不隐式下载或安装。

当前为开发版本，性能验收仍有未达标项。Windows amd64 与原生 Linux amd64/glibc（Kali VMware）已完成真实 Node/Python、混合项目、镜像和失败恢复验证；PowerShell、Bash、zsh、fish 接入已有实际运行证据。按用户要求，本次交付跳过 macOS；产品仍拒绝 WSL。Windows 与 Kali 的部分延迟超过原预算，不能将功能验证通过等同于性能达标。详细证据和开放项见 [实现记录](docs/implementation.md)，目标行为见 [架构方案](docs/architecture.md)。

## 从源码开始

Windows 试用也可直接使用安装向导或便携包，见 [安装与入门](docs/windows-install.md)。安装向导默认加入当前用户 PATH，重新打开终端后即可输入 `myenv`；不会修改系统 Node/Python。当前发布候选为 `0.1.0-rc.1`，尚未公开发布；发布条件见 [发布检查单](docs/release-readiness.md)。

中文系统默认显示简体中文帮助和日常提示；用 `myenv --lang en --help` 切换英文，或设置 `MYENV_LANG=zh-CN`。优先级为命令行参数、专用环境变量、系统语言、英文回退。JSON、补全脚本和用户命令输出保持既有合同。直接运行 `myenv` 或 `myenv status` 均可查看当前项目状态。

需要 Go 1.26.6。Windows 执行路径要求 Windows 10 或更新版本（Server 2016 或更新版本），以便在创建子进程时直接关联 Job；该版本下限不代表这些系统版本均已完成验收。Windows PowerShell 中，在源码目录运行：

```powershell
./scripts/build-release.ps1 -Targets windows-amd64
$myenv = (Resolve-Path ./dist/release/myenv-windows-amd64.exe).Path
& $myenv --version
& $myenv help manual
```

构建脚本输出制品、`build-manifest.json` 和 `SHA256SUMS`，不会安装或修改 PATH。摘要用于核对文件，不是签名。其他构建目标及参数见 [构建说明](docs/build.md)。

可以一直通过绝对路径调用。如果需要安装到自己的工具目录，先用 `Get-Command myenv -All` 检查同名命令，再把所选制品复制为该目录下的 `myenv.exe`，显式将目录加入 PATH。myEnv 不自动修改 shell 配置。

## 安装当前用户默认工具

```powershell
myenv system install java@17
myenv run --global java -version
myenv list
myenv system list --details
```

这里的 system 管理当前用户的 myenv 默认环境，不会把 JDK 自动加入系统 PATH。Python 可显式使用 `--provider python.org`（Windows）或 `--provider astral`。Java 来源为 Eclipse Temurin。外部安装须核验原管理器，不支持任意来源无条件接管。

终端自动启用状态颜色与动态阶段提示；`NO_COLOR=1` 关闭装饰，`--no-input` 或 `--verbose` 使用逐行阶段日志，`--json` 保持结构化输出。

## 第一个项目

在自己的项目目录中建立 `myenv.yaml`，版本值使用字符串：

```yaml
schema: 1
tools:
  node: "22"
  python: "3.12"
```

如果项目已有 `.node-version`、`.nvmrc`、`.python-version` 或支持的项目清单，也可以运行 `myenv init` 从声明生成配置；它不会覆盖已有 `myenv.yaml`。缺失或冲突的输入在非交互模式下返回错误。

可选的 `env` 映射用于普通字符串环境变量；凭据从调用进程环境传入。Windows 环境变量名不区分大小写，配置中不能同时声明 `Path` 和 `PATH` 等冲突键；Linux/macOS 保留大小写区别。

下面用安装后的 `myenv` 命令示例；使用源码制品时，在 PowerShell 中将其替换为 `& $myenv`：

```text
myenv sync --dry-run
myenv sync
myenv run node --version
myenv run python --version
myenv
```

裸命令显示当前项目状态。可用 `myenv -C <项目目录> ...` 指定上下文。Python 项目依赖继续写在 `pyproject.toml` 中，由原生 `uv.lock` 锁定；myEnv 不接管业务代码或项目数据。Python 构建需要本次调用的终端确认，或显式 `--allow-build`；此参数允许构建代码以当前用户权限执行，不提供沙箱。

`sync` 和 `use` 将检查、等待工作区锁、准备工具或依赖、验证与应用等阶段提示写到 stderr。使用 `--json` 时，stdout 仍只输出结构化结果；无变更同步不会显示工具准备阶段。

`run` 启动子进程后透传其退出码。Linux 子进程被信号终止时返回 `128 + 信号编号`，例如 SIGINT 为 130、SIGQUIT 为 131、SIGTERM 为 143；若子进程自行处理信号并正常退出，则保留它选择的退出码。Windows 使用原生进程退出状态。启动前失败使用 myEnv 错误类别。租约收尾失败会写入 stderr，不覆盖已完成子进程的退出码。

Linux 监督器转发 INT、TERM、HUP、QUIT 和 CONT；当前开发版已接入 Ctrl-Z 暂停及 `fg`/`bg` 前台恢复，并通过隔离 WSL Bash/PTY 测试。TTY 运行需要内核提供并允许 pidfd_open/pidfd_send_signal（通常为 Linux 5.3 及以上）；无法取得调用端进程句柄时会在启动子进程前报错。暂停期间的截止取消已接入监督器独立回收与调用端恢复；手工取消竞态、其他 Shell 及原生 Linux 产品验收仍待完成。更详细的退出与清理说明见离线命令 `myenv help manual`。

## 锁定、更新和恢复

```text
myenv sync --locked --no-input --json
myenv use node@22
myenv doctor
myenv doctor --deep
myenv rollback
myenv run --current node --version
```

`--locked` 要求已有且匹配的锁文件，不更新锁。普通 `sync` 失败后，期望声明或锁文件可能已经改变，但原活动环境仍保留；`run --current` 明确使用已应用配置。`rollback` 切换到保留的前一代，不回退声明、锁文件、代码或数据。

快速检查不会逐文件检测手工篡改。`doctor --deep` 比较受管代内容与记录的基线，不执行修复；外部共享解释器内容、Python 字节码缓存和 ACL 不在该检查范围内。需要重建时运行 `myenv sync --rebuild --locked`，这不会重装共享基础解释器或清空共享缓存。

`doctor` 也会只读检查运行保护记录。JSON 的 `data.run_protection` 为 `present` 或 `none`；`present` 表示记录仍保留，不证明对应进程仍存活，也不表示可以强制删除环境。`data.protection_detail` 和文本输出会给出清理预览提示：项目使用 `myenv clean --dry-run`，全局环境使用 `myenv clean --global --dry-run`。此检查不统计准备中的操作保护，也不自动回收记录。

`run` 后的参数属于子命令，例如 `myenv run node script.js --json` 会把 `--json` 传给 Node 脚本。myEnv 自身的 `--json` 不适用于 `run`；其他支持它的命令输出一个包含 `schema/ok/changed/data/error` 的对象。非交互模式和 `--json` 不等待输入。myEnv 退出码为成功 0、执行或环境错误 1、用法或配置错误 2、需要输入或授权 3；子进程已运行时透传其退出状态。租约收尾失败会另写 stderr 诊断。

## 用户默认工具和补全

```text
myenv use --global python@3.12
myenv run --global python --version
myenv doctor --global
myenv completion powershell
```

用户 profile 与项目声明独立，项目不隐式继承 profile 工具。`--global` 不是管理员或系统范围安装；它使用当前用户配置目录。补全命令只输出脚本，不修改 shell profile。各 shell 的启用方式见 `myenv help manual`。

已有 profile 后，可在 PowerShell 显式运行 `myenv shell-init powershell | Out-String | Invoke-Expression`，让当前 Shell 中的 `node`、`npm`、`npx` 或 `python`（按 profile 声明）调用 `run --global`。这些函数优先于 PATH 中的同名命令；不会修改 PATH 或 Shell 文件，关闭 Shell 即可取消。添加工具或移动 myEnv 后重新执行。Bash、Zsh、Fish 的启用方式见离线手册。子程序直接查找可执行文件时仍使用其自身 PATH。

重新加载只定义当前声明的包装函数，不删除旧函数。删除 profile 工具或卸载 myEnv 后，应开启新 Shell；`shell-init` 不保存或恢复被覆盖的用户同名函数。

Node 镜像可通过调用环境中的 `MYENV_NODE_MIRROR` 显式指定，例如 `https://mirror.example/node/dist`。镜像需提供官方目录布局的 `index.json`、版本目录和 `SHASUMS256.txt`。新解析的地址与摘要会写入锁文件；已有锁定地址保持不变。选择可信来源，镜像提供的摘要不是签名。基址不接受凭据、查询参数或片段；取消此变量后恢复默认官方源。

`MYENV_UV_MIRROR` 指定固定 uv 引擎的下载基址，其下需提供 `<版本>/<官方归档文件名>`；内置版本和 SHA-256 校验保持不变。已安装的引擎会继续复用。

`MYENV_PYTHON_MIRROR` 通过 uv 的 `python install --mirror` 指定 CPython 下载基址，保留 python-build-standalone 的发布目录与归档布局。已有共享解释器继续复用；该设置不把 Python 的版本锁提升为制品摘要锁。

自定义 CA 使用 `SSL_CERT_FILE` 指向绝对路径的 PEM 证书文件。myEnv 的 Node/uv 引擎下载接受不超过 4 MiB 的普通文件，以其中证书替换默认信任根；uv 子进程继承同一变量。无效或空证书文件会报错，不关闭 TLS 验证，也不修改系统证书库。myEnv 下载器尚不支持 `SSL_CERT_DIR`。

## 清理和卸载

先预览再清理项目中不再使用的代：

```text
myenv clean --dry-run
myenv clean
myenv clean --cache node --dry-run
myenv clean --cache uv --dry-run
```

项目清理保留活动代、前一代和仍受租约或准备操作保护的对象。准备阶段已有明确的子进程完成记录、但未发布的操作，也可在获得工作区锁后恢复并清理；预览是当前候选快照，实际清理会重新检查。无法确认进程已退出的历史记录会继续受保护，不应通过手工删除租约来强制回收。`clean` 不接受 `--global`。共享 Node 下载缓存和 uv 缓存使用独立的 `--cache` 入口；这些命令不卸载共享解释器。

卸载命令本身时，删除自己安装的 `myenv.exe`，并撤销自己添加的 PATH 或补全配置。这个操作保留项目的 `myenv.yaml`、锁文件、`.myenv` 环境和用户共享数据。项目 venv 可能引用共享 Python 解释器；删除共享运行时会破坏依赖它的项目，当前没有完整的自动数据卸载流程。

完整离线命令参考可运行 `myenv help manual`，单命令帮助为 `myenv help sync` 等。开发和验证规则见 [AGENTS.md](AGENTS.md)。

范围补充：macOS 不属于本次交付；历史交叉构建不代表已支持。Linux 的独立监督器在调用端暂停时仍执行绝对截止时间；调用端自身的 Go 手工取消回调需要该进程恢复后才能运行，这遵循操作系统暂停语义。

Linux 组件中的 clean --dry-run 会预览所有者已退出且全部准备子树完成证据有效的操作，统计候选及字节但不写状态；实际 clean 会持锁重新校验。未知或不完整回执继续保留。

Windows 准备恢复使用已登记的命名 Job：原所有者退出且所有对应 Job 均确认消失后，clean 才解除准备保护。无法确认的记录继续保留。
