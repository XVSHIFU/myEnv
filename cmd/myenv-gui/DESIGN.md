---
name: myEnv Windows GUI
description: 清晰、紧凑、保留 Windows 原生操作习惯的开发环境工作台。
colors:
  bg: "#ebece8"
  surface: "#fbfcfa"
  paper: "#fff"
  subtle: "#f1f3ef"
  chrome: "#f0f3ee"
  text: "#22352e"
  muted: "#5c6a62"
  line: "#dce3da"
  green: "#22674d"
  green-hover: "#175039"
  green-soft: "#e8f2eb"
  amber: "#8a591c"
  amber-soft: "#fbf2df"
  red: "#b03f3a"
  red-soft: "#fff0ec"
  dark-bg: "#131d19"
  dark-surface: "#1d2923"
  dark-paper: "#223028"
  dark-subtle: "#1c2821"
  dark-chrome: "#1b2821"
  dark-text: "#e3ebe3"
  dark-muted: "#a0b1a3"
  dark-line: "#384b3e"
  dark-green: "#96d3ac"
  dark-green-hover: "#b0e3c1"
  dark-green-soft: "#2c4935"
  dark-amber: "#e2bf85"
  dark-amber-soft: "#443b26"
  dark-red: "#f1aaa0"
  dark-red-soft: "#4b302c"
  dark-primary-text: "#17261c"
typography:
  body:
    fontFamily: '"Segoe UI","Microsoft YaHei UI","Microsoft YaHei",sans-serif'
    fontSize: "13px"
    fontWeight: 400
    lineHeight: 1.5
  headline:
    fontSize: "22px"
    fontWeight: 650
    lineHeight: 1.3
    letterSpacing: "-.4px"
  title:
    fontSize: "14px"
    fontWeight: 650
  label:
    fontSize: "11px"
    fontWeight: 500
    lineHeight: 1.3
  version:
    fontSize: "25px"
    fontWeight: 650
    lineHeight: 1.15
    letterSpacing: "-.7px"
  mono:
    fontFamily: 'Consolas,"Cascadia Mono",monospace'
    fontSize: ".96em"
rounded:
  tag: "4px"
  control: "6px"
  dialog: "9px"
spacing:
  control-gap: "7px"
  action-gap: "8px"
  section-gap: "12px"
  content-inline: "28px"
  compact-content-inline: "22px"
components:
  button-primary:
    backgroundColor: "{colors.green}"
    textColor: "{colors.paper}"
    rounded: "{rounded.control}"
    padding: "6px 12px"
  button-primary-hover:
    backgroundColor: "{colors.green-hover}"
  button-primary-dark:
    backgroundColor: "{colors.dark-green}"
    textColor: "{colors.dark-primary-text}"
  button-secondary:
    backgroundColor: "{colors.paper}"
    textColor: "{colors.text}"
    rounded: "{rounded.control}"
    padding: "6px 12px"
  button-quiet:
    backgroundColor: "transparent"
    textColor: "{colors.text}"
    rounded: "{rounded.control}"
    padding: "6px 12px"
  input:
    backgroundColor: "{colors.paper}"
    textColor: "{colors.text}"
    rounded: "{rounded.control}"
    padding: "8px 10px"
  language-selected:
    backgroundColor: "{colors.paper}"
    textColor: "{colors.text}"
    rounded: "{rounded.control}"
    padding: "11px 10px"
  tag-green:
    backgroundColor: "{colors.green-soft}"
    textColor: "{colors.green}"
    typography: "{typography.label}"
    rounded: "{rounded.tag}"
    padding: "2px 6px"
  notice:
    backgroundColor: "{colors.subtle}"
    textColor: "{colors.text}"
    rounded: "{rounded.control}"
    padding: "13px"
---

# Design System: myEnv Windows GUI

## Overview

**Creative North Star: "清晰的环境工作台"**

温和的森林绿、暖中性色与细分隔线支持长时间的环境操作。信息密度服从当前任务：明确区分当前生效环境、目标草稿与执行结果，主动作保持易找，技术详情按需展开。

系统沿用 Windows 的窗口框架和 Segoe UI 阅读习惯。普通控件紧凑、平实，颜色和小幅层次用于表达选择、状态与覆盖关系；不依赖装饰图片。

**Key Characteristics:**

- 森林绿表达主要动作、选择和成功，暖中性色承载工作内容。
- 系统字体负责阅读，等宽字体负责版本、命令与技术信息。
- 固定操作区域与局部滚动保持任务入口可达。

本规范覆盖 `cmd/myenv-gui`。数值取自 [style.css](frontend/src/style.css) 的最终级联及 [appearance_windows.go](appearance_windows.go)；`dark-` 为同名 CSS 变量在深色主题中的记录前缀，`dark-primary-text` 对应深色主按钮的文字覆盖。产品行为以 [architecture.md](../../docs/architecture.md) 为准，工作台方向见 [.impeccable/surfaces/workbench.md](.impeccable/surfaces/workbench.md)。

## Colors

### Primary

`green` 是低饱和森林绿，承担主按钮、选中项、焦点和成功状态；`green-hover` 加深悬停反馈，`green-soft` 承载选中底色。深色主题以明亮的灰绿维持同一语义。

### Neutral

`chrome` 连接原生标题栏与项目工具栏；`paper` 是主工作面，`surface` 用于固定操作区和任务底栏，`subtle` 用于语言导航、日志和辅助容器。`text`、`muted` 和 `line` 分别表达主要信息、辅助信息和细边界。

`amber` / `amber-soft` 表示待处理和本次构建许可，`red` / `red-soft` 表示失败。取消使用中性色和明确文字。状态必须同时具有文字或图标，颜色不单独承载含义。

**The 状态配色 Rule.** 颜色跟随实际状态；选择、成功、等待许可、失败和取消不得互相借用含义。

## Typography

系统字体栈由 `body` 定义，所有界面标题继承它。`headline` 用于页面主标题，`title` 用于内容分组；抽屉标题在普通窗口使用 15px，窄窗使用 14px。辅助说明和状态文字主要为 11–12px。

`version` 强调当前值与目标值，窄窗最终缩为 20px；它是数值角色，不是通用大标题。版本号与命令采用 `mono`；含中文的日志和参数编辑采用 Consolas、Microsoft YaHei UI、monospace。长路径和技术正文可换行；紧凑摘要省略时保留完整值的 title 或详情入口。

## Layout

客户端填满可用高度，取消网页模拟外窗的圆角、边框与阴影。顶栏、当前页操作栏和底部任务栏不收缩；工作区与抽屉内容各自滚动，避免整页滚动挤走保存、确认或取消入口。

每个阅读区域只保留一个垂直滚动所有者：来源弹窗由正文滚动，任务详情与原始日志由抽屉正文滚动，内部原始证据不再嵌套滚动。滚动条细而淡，透明轨道、随浅深主题变化的滑块；长路径与日志换行。首次检查弹窗的标题、跳过入口与底部“开始检查”固定，正文再长也不能挤走主动作。

普通布局使用 192px 语言侧栏。容器宽度不超过 880px 时侧栏为 174px，工具栏由 56px 缩到 49px；不超过 780px 时切换为 55px 高的横向语言栏。内容水平留白由 `content-inline` 转为 `compact-content-inline`。高度紧张时压缩重复说明、行高和间距，保留主要信息和操作。

原生默认外窗为 1100×780，最小外窗为 760×540；本轮 100% 缩放截图对应客户端 1084×741、744×501。CSS 容器断点以客户端为准；更小的 580px 防御样式不代表新增受支持外窗尺寸。完整断点条件见 sidecar。

## Elevation & Depth

工作面主要靠色阶和细线分层。任务抽屉以向上的柔和阴影表达覆盖，弹窗与查询选项使用柔和阴影，作用范围的当前选项仅有微弱抬升。实际阴影值收录在 sidecar，原生外窗不使用 CSS 阴影。

**The 原生边界 Rule.** 保留 Windows 标题栏、DWM 边界与系统窗口按钮；网页只负责客户端，不再绘制第二套外窗。

## Shapes

控件和通知采用 `control` 圆角，紧凑标签采用 `tag` 圆角，弹窗采用 `dialog` 圆角。细线边界比阴影更常见。圆形用于状态节点、单选标识与任务图标底座；操作图标采用本地 Lucide 静态 SVG，24×24 viewBox，常规显示 18px、线宽 2、圆头圆角；myEnv 叶片保留原图形及 1.65 线宽。五种语言采用 Devicon 原彩 SVG，Go 使用紧凑 GO 字标。

## Components

- **按钮与字段：** 普通按钮最小高度 34px，主按钮使用实色强调，次按钮保留边界，quiet 按钮为透明底。禁用透明度为 .48；按钮和字段的键盘焦点使用绿色 2px 外描边、3px 偏移。按钮背景与边框过渡为 150ms。字段保留标签和可读占位文字。
- **环境范围：** “本机环境”中的“本机已安装”显示应用启动时 PATH 解析出的命令、实际版本与位置，并标明检查时间和不会自动刷新的记录属性；它与独立的“myEnv 工具链”页签分开。项目环境显示当前项目的声明与生效版本。检查记录不表示接管安装，不改变终端默认版本。
- **首次检查：** 首次可检查本机，并可另选一个项目目录，也可暂时跳过。检查完成后由用户保存记录，后续可从本机环境或维护重新检查；跳过后不反复弹出。保存的是检查快照，不是环境配置或安装操作。
- **包与工具：** 仅 Node.js / Python 提供包页签，按当前项目、全局工具安装位置或具体解释器分组，显示包名、期望 / 已安装版本与类型；npm、pnpm、pip、uv 等管理工具单独说明归属。缺失、未核验、截断与检查范围保持明确，每行末尾提供管理入口；管理工具全部展示，包组默认10项，可搜索、展开和勾选同一位置的包。分组右侧提供选择和官方包目录；全选包含筛选匹配的隐藏行，超100项明确提示分批。右侧抽屉依次展示目标与操作、官方目录、变更预览和逐项结果，期望在抽屉设置；更新默认最新版，切换默认当前版本。确认只执行保留原计划，底部操作固定，外部包不承诺回滚。npm支持关键词，PyPI要求完整包名；404与网络错误分开。其他语言的项目检查保留版本工作区。
- **导航与版本选择：** 当前语言使用白色工作面、边界和原彩品牌标志；当前页签使用绿色文字与底部细线。维护、设置与外部管理提供明确返回入口，Alt+Left 返回；检查失败或取消后的返回应回对应检查工作区。查询选项点击外部、按 Esc、切换页面或开始查询时收起，Esc 恢复到选项入口焦点。Java 8 同时显示 1.8 别名，主版本与可读发布版本并列，稳定版优先，预发布需明确开启。选中版本使用浅绿整行、单选标识与“已选择”标签，点击仅更新目标草稿。
- **工具快捷入口：** 语言保留一级导航，只有选中语言展示已检测的管理工具。Python 的 uv / pip、Node.js 的 npm / pnpm 使用轻量文字按钮，不新增一级卡片；“更多…”进入包与工具。普通窗口入口缩进于选中语言下，客户端不超过780px时移至当前页签下的一行，横向语言栏与底部任务区保持固定。pip绑定明确解释器，项目入口只使用项目检查结果，不回退本机pip；uv保留独立本机工具身份。单一安装直接打开管理抽屉，多位置先选择，再关闭位置弹窗并进入抽屉；关闭时恢复到当前窗口尺寸下可见的对应入口。工具按钮使用有名称的语义分组，路径及作用范围在抽屉中明确展示。
- **当前任务与抽屉：** 底栏持续显示当前任务和阶段；详情按概览、日志、历史组织。打开抽屉使主工作区 inert，收起后恢复入口焦点。页签支持方向键、Home/End，Esc 收起；流式事件不移动输入焦点。日志在底部时尾随，否则保留阅读位置。历史记录只读。
- **进度与结果：** 仅 `progress.kind=download` 的真实事件显示当前文件字节数；有效 total 才显示当前文件百分比。查询和其他阶段使用不定进度，不生成总体安装百分比。阶段节点是已观察阶段的去重概览，不能解释为严格时间线或工作量比例。
- **构建许可：** 只在当前任务概览显示独立固定决策栏，按钮提交后等待任务响应。详情正文再长也不能挤走许可入口；历史结果不能携带当前任务的执行动作。
- **品牌图标：** 仅用于辅助识别，名称和环境状态文字始终保留，不用品牌色编码安装状态。原始 SVG 通过静态导入随 GUI 离线分发，不依赖图标 CDN；独立 img 避免渐变 ID 冲突。普通标志约 25px，Go/Java/Rust 按视觉重量微调；窄窗约 22–26px。深色中的 Java、Rust 使用固定 `#f8faf7` 浅色承托，保持原图形与颜色。来源与固定版本在 `frontend/src/assets/icons/assets-manifest.json`，完整许可在 `THIRD_PARTY_NOTICES.txt` 并随便携包分发。
- **原生应用图标：** Windows 应用资源沿用 myEnv 叶片，置于森林绿底上；源图为 `build/windows/icon.svg`，`generate-icon.ps1` 生成多尺寸 `icon.ico`。构建时嵌入 GUI 的原生图标资源，窗口与可执行文件使用同一标识，不另绘网页标题栏。
- **主题与动效：** 偏好页是 system/light/dark 的唯一设置入口，同步更新客户端与原生标题栏。旋转指示为 .8s，非定量进度移动为 1.5s；减少动态效果偏好关闭过渡和动画，并保留静态进行中提示。

## Do's and Don'ts

### Do:

- **Do** 从最终生产 CSS 和真实任务快照扩展界面，保持浅色与深色语义对应。
- **Do** 为错误、等待和取消保留明确文字，并让恢复动作靠近原因。
- **Do** 在最小客户端检查主动作、固定底栏与局部滚动的可达性。

### Don't:

- **Don't** 把草稿选择表达为已安装或已应用。
- **Don't** 为未知总量生成百分比，或把当前文件进度表达为整个任务进度。
- **Don't** 把只读历史、模拟原型或布局样本当作可执行计划或后端验收证据。
