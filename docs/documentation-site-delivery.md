# 使用文档站与用户安装实测归档

2026-09-09。用户偏好导航式使用文档而非另写白皮书。本轮没有修改 Go CLI 或安装器，也没有安装 Java。

## 已完成

- 创建 `website/` 文档站源码：快速开始、安装演示、支持范围/Java、Node/Python 项目、版本与回滚、默认工具、排障、命令、配置、版本限制，共 10 篇指南及首页。
- 使用 Sites 指定脚手架及已安装 UI 组件，输出静态文件；用户目标为 GitHub Pages，没有擅自改为其他托管服务。
- `.github/workflows/docs-pages.yml` 提供手动 GitHub Pages 发布；尚未收到仓库地址，未创建远端仓库、提交或公开发布。
- 用户的五张原图逐一查看后原样复制至 `website/public/images/installation`，来源、日期、版本、SHA 保存在 `website/screenshot-manifest.json`。再次逐一校验，两处文件均匹配，未修改原测试 README 或原图。
- 用户截图和文字记录提供了实际安装成功、重开终端后 `myenv` 可用、中英文切换的实测证据。这是用户提供的验证，不冒充本轮 Agent 自动执行的安装。
- 图片揭示中文命令列表 `help` 简介漏翻译，已如实记入站点“已知限制”；原图不修饰。该 CLI 文案修复尚未实施。

## 验证与构建限制

`npm run build` 根路径静态导出成功。框架在 trailingSlash=true 时动态路由导出返回 308，basePath 前缀导出返回 404；保留静态导出，改为 assetPrefix 配合显式文档链接，不修改框架依赖源码。`NEXT_PUBLIC_BASE_PATH=/myenv` 的 Vinext 导出成功；随后修正 postbuild 的 Pages 目录和资源整理，`node scripts/finalize-static.mjs` 成功检查 11 页、263 个内部资源/页面引用。脚本不依赖服务器重写 URL。`npx tsc --noEmit` 通过。

记录位于 `.build/docs-build.txt`、`.build/docs-build-github-base.txt`；后者保留后处理修复前的错误日志，最终后处理通过结果见 `.build/docs-static-final.txt`。没有将失败日志当成功结果，也没有运行无关 CLI 全量回归。

本地 dev 服务曾响应 HTTP 200；环境没有 Sites 技能指定的 open_in_codex 工具，未进行自动浏览器截图/交互 QA。用户原图查看不属于文档站浏览器 QA。交付时停止本轮 dev 服务。

首次受限 npm 请求失败，随后获准联网完成官方脚手架安装。保留 lockfile；没有放行被 npm 策略阻止的安装脚本。审计详情 `.build/docs-npm-audit.json`：11 项依赖问题（8 high、2 moderate、1 low），涉及构建/开发服务和 RSC 依赖；未做未经评估的强制升级。Pages 仅部署静态产物，不运行 RSC/Worker 服务器。工具链安全审计未关闭，不宣称公共服务器可直接上线。

## Java 事实核对

`internal/config/config.go` 当前仅接受 python/node；没有 Java 下载、导入、锁定、版本切换或 JAVA_HOME 管理。已安装 Java 且位于传入 PATH 时，已准备项目中的 `myenv run java -version` 属于外部命令执行，不构成受管 Java。没有 JDK 的电脑目前不能用 myEnv 安装 Java。

建议后续独立实现 JDK 后端：受管下载与摘要校验、显式只读登记已有 JDK、项目级版本选择、对子进程注入 JAVA_HOME/PATH、区分外部与受管路径并保留运行删除保护。这只是范围建议，本轮未实现。
