# myEnv Windows 安装与入门

这是 CLI 开发版。Windows amd64 功能已有验证，性能验收尚未完成。安装向导需要 Windows 自带的 .NET Framework 4.x；CLI 便携版不依赖 .NET，也无需安装 Go。

## 安装向导（推荐）

1. 双击 `myenv-<版本>-windows-amd64-setup.exe`。
2. 默认安装到 `%LOCALAPPDATA%\Programs\myEnv`，保持“加入当前用户 PATH”勾选。
3. 点击“安装 / 升级”。不需要管理员权限，不修改系统级 PATH。
4. 退出终端应用后重新打开 PowerShell，执行：

```powershell
myenv --version
myenv --help
myenv help manual
```

中文系统默认中文；可用 `myenv --lang en --help` 或 `myenv --lang zh-CN --help` 切换。本次是未签名的开发制品，校验和用于核对下载完整性，不代表发行者签名；受组织应用策略限制的机器需按本机规则处理。

安装向导会检查 PATH 中的同名程序。发现冲突时显示路径，不直接覆盖；先处理冲突，或取消加入 PATH，通过绝对路径调用。PowerShell 中的同名函数/别名还需用 `Get-Command myenv -All` 检查。

升级时运行新下载的安装向导，沿用原目录。先退出正在运行的 myEnv 命令；不覆盖缺少安装记录或内容已被修改的已有文件。更换目录前先卸载旧安装。

从 Windows“已安装的应用”中卸载 myEnv CLI，或重新打开安装向导点击“卸载”。只删除安装器登记的程序文件，以及由本安装器加入的 PATH 项；其他 PATH 项、用户放入的文件、项目和运行时数据保留。卸载助手临时复制到系统临时目录以避免删除正在运行的 exe，该副本可由系统临时文件清理移除。

## 便携包（工程师）

解压 `myenv-<版本>-windows-amd64-portable.zip`，目录中有 `myenv.exe`。可按绝对路径执行，或把它放入固定目录，例如 `C:\Tools\myEnv`。

手动加入 PATH：搜索“编辑账户的环境变量” → 用户变量 `Path` → 新建 `C:\Tools\myEnv` → 保存并重新打开终端。添加的是目录，不是 exe 文件路径。便携包不会自动登记卸载项。

## 第一个项目

在自己的项目目录里创建 `.node-version`，内容为 `22.23.2`，随后执行：

```powershell
myenv init
myenv sync
myenv status
myenv run node --version
```

Python 可改为 `.python-version`，内容 `3.12.13`，最后执行 `myenv run python --version`。首次同步需联网下载。已有 `myenv.yaml` 时直接从 `sync` 开始。

把 myEnv 加入 PATH 不会替换系统 Node/Python。默认使用 `myenv run ...` 选择项目环境；有需要时再按离线手册设置当前用户默认工具和 Shell 包装函数。

## 构建安装包

源码目录中执行：

```powershell
./scripts/build-release.ps1 -Targets windows-amd64 -Version 0.1.0-dev.2
./scripts/package-windows.ps1
```

输出包含安装向导、便携 ZIP 和 `SHA256SUMS`；打包前核对 CLI 与构建清单摘要一致。不会运行安装器、修改当前用户 PATH 或发布到外部。

安装器保留 PATH 中未展开的变量引用及注册表字符串类型，仅修改自己的条目。实现参考微软的 [RegistryKey.GetValue](https://learn.microsoft.com/en-us/dotnet/api/microsoft.win32.registrykey.getvalue) 和 [环境变更广播](https://learn.microsoft.com/en-us/windows/win32/winmsg/wm-settingchange) 文档。
