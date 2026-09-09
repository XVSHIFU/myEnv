# myEnv 中文文档站

面向使用者的导航式文档，包含快速开始、真实安装演示、Node/Python 项目、版本与回滚、默认工具、排障、命令和配置参考。对应 0.1.0-dev.2；Java 尚未支持，在支持范围页面明确区分了已有外部命令执行与受管 JDK。

使用 Sites 脚手架的 Vinext/React 静态导出；不需要数据库、服务端身份认证或常驻服务器。GitHub Pages 仅上传 `dist/client`，不上传研发记录、安装器、试用目录或服务端产物。

## 本地预览与构建

需要 Node >=22.13.0。在 website 目录中：

```sh
npm ci --ignore-scripts
npm run dev
```

按终端给出的本地地址打开。生成可部署文件：

```sh
npm run build
```

输出 `dist/client`，包含 11 个阅读入口（首页与 10 篇指南）。末尾检查内部页面和图片链接。不能仅双击 HTML 来验证应用资源加载，请通过 HTTP 服务预览。

## GitHub Pages

1. 将 `website/` 和 `.github/workflows/docs-pages.yml` 提交到自己的 GitHub 仓库（当前工作区尚未关联仓库）。不要把 `.build/`、完整 `trial/`、本机日志和运行时一并上传。
2. 仓库 Settings → Pages → Source 选择 GitHub Actions。
3. 在 Actions 中手动运行 Publish myEnv documentation。
4. 等待 deploy 成功，使用工作流返回的网址。

仓库根目录 `.github/workflows/docs-pages.yml` 已配置；仅手动触发，不因普通推送自动公开。工作流从 configure-pages 获取 base_path，用于内部链接和 assetPrefix，兼容 `https://<用户>.github.io/<仓库>/`。后处理将导出的页面和资源整理为 Pages 目录布局，并检查内部链接。部署所需的仓库权限和 Pages 可用性以实际账户为准。

参见 [GitHub 官方 Pages 工作流说明](https://docs.github.com/en/pages/getting-started-with-github-pages/using-custom-workflows-with-github-pages)。本轮没有凭空创建仓库或公开发布，也没有发布到另一托管服务替代用户所选 GitHub。

## 截图与版本维护

五张原始截图位于 `public/images/installation`，来自用户 `trial/v0.1.0-dev.2/assets`。`screenshot-manifest.json` 记录原始路径、版本、日期和 SHA-256；复制件与原件逐个一致，没有重绘或修改。

前三张用于安装演示，后两张用于语言切换实测。原件始终保留。正式发行时核对画面与发行版本并更新图注；截图中的用户目录是示例，不是安装参数。当前截图保留 help 简介漏翻译的真实状态，该问题列在已知限制中。

主要内容在 `app/articles.tsx`；导航在 `app/navigation.ts`。新增页面需同时更新导航，构建检查会验证生成链接。

Sites 技能已用于建站结构和验证；按用户选定的 GitHub 部署方向准备工作流，待仓库明确后才能进行真实发布。

构建工具链审计（2026-09-09）仍报告 11 项依赖问题（8 high、2 moderate、1 low），涉及 RSC 服务端、开发服务器及其依赖。本交付是静态 HTML/JS/CSS，不在 Pages 上运行这些服务端；未据此宣称依赖已无漏洞。升级前应逐项复核，未执行自动强制修复。请勿将开发服务器公开暴露为正式站点。
