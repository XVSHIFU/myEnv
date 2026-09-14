# 当前状态与下一步

更新：2026-09-14。`0.1.0-rc.11` 源码已通过 [PR #1](https://github.com/XVSHIFU/myEnv/pull/1) 合入 `main`（`651fd28`），Windows/Linux 核心 CI 与 Windows GUI 构建通过；[文档站](https://xvshifu.github.io/myEnv/)已部署。GUI 包管理、语言下工具入口、README、真实截图与目录/Git 整理已完成，合并前另修复 Python 路径别名核验。未创建软件标签或 Release；保留原 rc.11 及更早本地制品。现有本地候选未重建，不包含路径修复 `d9e1470`，源码快照仍记录其实际构建内容。

## 当前行为

- 选中 Python / Node.js 后，仅在当前语言下显示已检测的 uv/pip 或 npm/pnpm；点击直接管理，“更多…”进入完整包与工具页。小窗口入口移至页签下方。pip绑定具体解释器，项目pip不回退到本机pip；uv仍是Astral独立工具，同名多位置先选择位置。关闭抽屉恢复当前可见入口的焦点，保持键盘与读屏分组语义。
- 管理工具全部展示，每行右侧“管理”打开侧边抽屉。包按项目、全局位置或解释器分组，可搜索、选择、全选筛选结果、批量管理；默认10项，其余可展开。全选包含隐藏的匹配项，同次最多100包，超限明确提示分批，不静默截断。
- 抽屉提供安装、更新、切换版本、卸载；期望在抽屉编辑，更新默认最新版，切换默认已安装版。显示具体安装位置、原版本与目标、依赖类型、计划、真实阶段、取消和逐项结果。关闭完成的操作后重新检查列表，历史保留部分失败/取消结果，不承诺外部包自动回滚。普通/最小窗口主动作与任务底栏固定。
- 接入 npm 官方关键词搜索及版本、PyPI 完整包名查询。官方404、网络失败、预览/撤回版本分别表达；列表最多1000版本并标明截断，精确版本查询独立。最新版在计划阶段解析为具体版本；Node 项目保存原生 latest 期望与精确安装锁，外部解释器/全局 latest 只针对本次操作，不后台自动升级。
- Go 控制器绑定原成功计划ID和完整请求，深复制包条目、单次消费；执行前重验位置、管理器及状态。npm/pnpm项目包、npm明确全局位置、pip明确解释器支持写操作；单次批量只作用于同一位置。npm/pip等正在执行的管理器自身必须单独处理。
- Python计划只读取隔离元数据，不执行 .pth/sitecustomize。myEnv 活动Python代继续通过声明与同步更新，不能原地改包；外部Python操作不自动改写pyproject/requirements。安装脚本禁用、Python仅wheel，界面计划明确说明；复杂workspace/未知管理器所有权交回原管理器。
- npm/pnpm/pip工具按已验证安装位置处理；uv独立更新要求官方安装收据匹配，独立安装没有卸载API，删除交回原安装方式。实际工具升级仍受原目录权限限制，不自动提权。不能把只读计划核验称为真实升级成功。
- rc.10本机PATH语义、首次检查、Windows图标、Java常见名称/稳定目录、单滚动区和TUI显式运行交接保护继续保留。GUI不修改终端默认；CLI合同、run不隐式安装、受管环境失败保留旧代等不变。包管理本轮接入GUI，CLI/TUI既有操作界面未新增包抽屉或包命令。

## 本轮有效验证

Windows使用 `.build/gopath` / `.build/gocache`、`GOPROXY=off`。实际运行：

```text
go test ./internal/backend -run '^TestPackageRegistry' -count=1
go test ./internal/core -run '^TestPackageManage(PythonPreview|Binds|Reports|Protects|Reject|Command|Latest)' -count=1 -v
MYENV_TEST_PACKAGE_REAL=1 go test ./internal/core -run '^TestPackageManageReal(NodeLifecycle|GlobalLifecycle|ToolPlans)$' -count=1 -v
go test ./internal/cli -run '^TestUI(Package|ExternalPlanBinding|TaskScopeAndBoundedHistory|Summary)' -count=1
go vet ./internal/backend
go vet ./internal/core
go vet ./internal/cli
go vet -tags gui,desktop,production ./cmd/myenv-gui
npm run build --prefix cmd/myenv-gui/frontend
node .build/rc11/check-history.cjs
node .build/rc11/check-shortcuts.cjs
./scripts/build-release.ps1 -Version 0.1.0-rc.11 -GUI -OutputDirectory .build/rc11/shortcuts-release
./scripts/package-windows.ps1 -ReleaseDirectory .build/rc11/shortcuts-release -OutputDirectory .build/rc11/shortcuts-packages
```

- 官方目录12项定向测试通过。真实只读 npm scoped包/PyPI/搜索证据在 `.build/rc11-registry-check/`，大型npm包详情本机测得11.634秒，不声称查询即时完成。
- 核心8个顶层定向测试及锁子用例通过。Windows隔离npm/pnpm项目：dev安装6→prod切换7→latest→删除；npm独立prefix安装/切换/删除通过。独立venv的idna3.9→3.10→删除通过（最初合并命令中Node失败，修复后只重跑受影响Node部分）。npm/pnpm/uv真实工具仅只读计划通过，未更新或删除真实用户工具。命令、失败修复与验证边界在 `.build/rc11/package-core/verification.json`。
- 独立审查发现并修复：dev→prod缺少显式保存类型、npm shrinkwrap生效锁优先级、结果遗漏声明/类型核验、Python预览触发启动钩子；含执行管理器自身的批量操作改为要求单项。Windows和Kali相关回归通过。
- 原生GUI两批：`.build/rc11/gui-rc11.ps1 -ShotsOnly` 的 `screen-runs/20260914-152346-089-69729c19` 为9张普通/最小窗口截图和真实官方查询→安装6→更新7→删除；`gui-rc11-confirm.ps1 -ShotsOnly` 的 `screen-runs/20260914-152810-037-8adbed09` 为5张确认图，官方404、已声明但上游缺失包删除、两个包批量安装/删除、实际查询取消通过。只操作 `.build/rc11/native-package-project`；本机包仅检查。
- 独立设计审核为 ship / faithful。流式事件不抢焦点，主工作区/顶栏/侧栏在管理抽屉打开时inert。最后历史取消结果修补通过React实际渲染检查 `canceled-history-check.json`，未重复无变化截图。原生QA使用Wails入口和程序化DOM输入，不等于真实IME/键鼠/高DPI验收。
- Kali原生新包核心定向测试通过，日志 `.build/rc11/package-core/kali-targeted.log`。远端仅隔离/tmp目录、禁用联网；未执行Linux真实包安装。rc.10 Windows/Kali TUI交接与信号证据继续复用，WSL TUI原有门禁仍开放。
- 工具入口最终原生确认：`.build/rc11/gui-shortcuts.ps1 -ShotsOnly` 的 `screen-runs/20260914-160424-679-f1404921` 共8张浅深色/普通最小窗口截图，实际本机检查和pip/uv/npm路径、更多导航、关闭与跨尺寸焦点恢复通过；未执行用户包写操作。前一轮最后截图采集失去原生窗口，不能算整轮通过；最终确认完整结束。独立设计审核 ship / faithful。`shortcut-identity-check.json` 使用标明的fixture验证项目pip隔离、同名不同路径保留、未知/损坏工具排除；多位置选择未声称本机实际存在多个uv。
- Windows GUI/CLI、Linux CLI最终发布构建通过；两种CLI哈希与原rc.11一致，复用既有运行时证据。`.build/rc11/check-shortcuts-assets.py` 核对4个当前前端资源在生产GUI中、未进入CLI。真正生产GUI隔离APPDATA启动与正常退出0在 `shortcuts-production-smoke/result.json`。
- 发布准备：`shortcuts-package-check.json` 核对8项SHA256、5个zip/portable文件、源码快照和setup内嵌许可。完整第三方许可由缓存离线生成，47个Go生产模块、83份上游原文，两次生成哈希一致；保留图标声明。两份CI YAML结构检查通过。GUI仍需要WebView2，setup仅CLI/TUI，未签名。
- 公开文档：README重新组织介绍、下载、GUI/TUI/CLI入门和支持边界；网站新增GUI、包管理、TUI三篇指南。3张原生rc.11截图仅使用隔离示例项目与真实官方查询/安装预览，无安装执行；来源摘要在 `.build/rc11/docs-captures.json`。`NEXT_PUBLIC_BASE_PATH=/myEnv npm run build`（website）通过14页/403内部引用；1280px/390px无横向溢出，图片加载正常。仓库Markdown相对链接与原文许可Git字节核对通过，本次未重测运行时/性能。
- 目录与Git：21个既有文件按用途迁移并逐项核对迁移前后摘要；设计/原型在 `docs/design`，历史在 `docs/archive`，旧patch仅本地归档。`.build`、dist和有效缓存保留；安装在仓库内的Impeccable工具副本通过本地Git排除保留原位，项目DESIGN和surface契约随源码提交。仓库换行规则明确，第三方原文与摘要文件保留字节。
- 合并检查：Windows CI 首次暴露 Python 短路径与规范路径误判。修复仅规范化探测的根目录/包路径，保留解释器入口与 `-I -S`；本机 `go test ./internal/core -run '^TestPackageManagePythonPreview' -count=1 -v` 通过 junction、真实8.3短路径、不同环境拒绝和启动钩子隔离，启用 `MYENV_TEST_INVENTORY_PYTHON=1` 的 `go test ./internal/core -run '^TestInventory(Python|Metadata|Probe)' -count=1 -v` 通过7项相关测试。最终 `d9e1470` 的 [Windows/Ubuntu Go interface checks](https://github.com/XVSHIFU/myEnv/actions/runs/34825282146) 与 [Windows GUI build](https://github.com/XVSHIFU/myEnv/actions/runs/34825282136) 全部通过；Windows CI Python为3.12.10。本轮未执行真实用户包写操作。
- 文档部署：合并提交 `651fd28` 的 [GitHub Pages 工作流](https://github.com/XVSHIFU/myEnv/actions/runs/34825512732)通过 `npm ci --ignore-scripts`、`npm run build` 和部署；远端构建核对14页/403内部引用。`.build/rc11/verify-public-pages.ps1` 在线核对6个主要页面HTTP 200及正文，3张rc.11截图SHA256与源码一致；结果在 `public-pages-check.json`。此后的部署状态记录仅更新仓库文档，不改变网站产物。

## 仍开放与下一步

- Linux GUI按用户允许暂缓；本轮没有承诺Linux包写操作原生闭环。WSL TUI门禁仍未改。
- 性能预算不降低：GUI可交互≤2s、空闲单核CPU≤1%；上次2.703s、Windows TUI空闲CPU3.53%仍未达标，本轮未重测或宣称修复。其余预算和CLI缺口见 architecture.md / performance-gap.md。
- 高DPI、真实IME、无WebView2干净机器、真实外部升级/修复仍未验收。管理器自身更新/版本切换/卸载的真实写验收，以及安装脚本必需包，仍需独立受控样本。
- Node解压运行时共享池、更多生态/私有包仓库、Python受管声明编辑器不在本轮。源码合并与文档部署已完成；实际公开Release前从选定提交重新构建并核对源码/制品，再关联标签，不能直接将旧本地候选标为包含合并修复。项目自身LICENSE尚未由作者选择，本次只补齐第三方许可。性能缺口继续按原预算处理，不因准备预发布而降标。
