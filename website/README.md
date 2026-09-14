# myEnv 中文文档站

手册对应 `0.1.0` 首版，在线地址为 [GitHub Pages 文档站](https://xvshifu.github.io/myEnv/)，软件制品与校验清单位于 [GitHub Release v0.1.0](https://github.com/XVSHIFU/myEnv/releases/tag/v0.1.0)。支持范围包括 Python、Node、Temurin JDK、Go、Rust；各来源、版本选择与验证边界见站点的「支持范围与工具链」。

手册按实际使用顺序组织：快速开始与安装、GUI 工作台、TUI 终端界面、包与工具管理，以及项目、版本、默认工具、排障和命令参考。本机检查、语言下工具入口、包管理抽屉、官方目录、单项/批量操作都有独立说明。GUI 仍需 WebView2，setup 仅安装 CLI/TUI，制品未签名；性能与平台限制保留在相应指南中。

## 维护与构建

需要 Node >=22.13.0。在 `website/` 中执行：

```sh
npm ci --ignore-scripts
npm run dev
# GitHub 项目页面的本地构建
NEXT_PUBLIC_BASE_PATH=/myEnv npm run build
```

PowerShell 使用 `$env:NEXT_PUBLIC_BASE_PATH='/myEnv'; npm run build`。产物是 `dist/client`，需经 HTTP 服务预览。构建末尾会检查内部页面、图片和资源链接；这不代替 TypeScript 检查。

正文维护在 `app/articles.tsx`，导航在 `app/navigation.ts`。GUI、包和 TUI 指南分别位于 `/guide/gui/`、`/guide/packages/`、`/guide/tui/`；正文与 README 共用 `public/images/rc11/` 下的实际界面截图，拍摄版本为 0.1.0-rc.11，沿用原图和路径。图注标明版本、示例场景和是否仅预览，不将这些原图标成 0.1.0 新截图，也不用截图暗示未执行的操作已经成功。

`public/images/installation` 的五张用户原图保持不变，图注与 `screenshot-manifest.json` 标明 dev.2 来源；它们只用于历史安装演示。TUI 目前用操作说明与命令示例，不用 GUI 图或模拟终端图替代原生截图。

## 发布

仓库已配置 `.github/workflows/docs-pages.yml`，在 Actions 手动运行 **Publish myEnv documentation**；成功后上传静态产物到 GitHub Pages。普通代码推送不触发公开部署。本地 `.openai/hosting.json` 不是 Pages 构建前置条件。

现有 Vinext/React 导出链包含开发服务器和 RSC 构建依赖，Pages 不运行这些服务器。历史依赖审计尚有开放项，需在更新构建依赖时复核；不要将历史漏洞数量当作当前审计结果。后续先清理未引用组件，再评估构建链升级，不自动强制升级全部依赖。
