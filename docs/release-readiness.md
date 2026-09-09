# 0.1.0 发布检查（2026-09-09）

结论：可整理本地 0.1.0-rc.1 候选，不能按原约定宣称 0.1.0 正式验收通过。用户 dev.5 反馈确认列表与重复安装提示正常；重复安装没有进入准备阶段，因此不能据此证明下载动画验收通过。

## 正式版待完成

1. 性能：Windows 启动/status/run、Kali run 与聚合 myenv RSS 的原门槛仍开放。历史证据见 kali-validation.md、run-stage-evaluation.md、localization-delivery.md。此轮未改运行路径，不重跑挑样，不降低门槛。
2. 文档站：已同步 rc.1 SDK、系统管理、Python 来源和终端行为，保留 dev.2 安装原图及版本标注。本地构建、TypeScript 与 262 个内部引用校验通过；站点已有依赖审计开放项仍保留。
3. 发布身份：用户已创建并授权推送 https://github.com/XVSHIFU/myEnv ，已完成 main 首次推送并启动 Pages 部署。未创建正式版标签或上传发行包；项目尚未选择开源许可证，不能默认授予开源许可。
4. 最新显示层：Windows 彩色列表及单元/CLI 回归通过；Linux 原生动画与下载/Ctrl-C 视觉验收待补。既有平台 SDK 与可靠性证据按未修改范围复用。

## 候选整理范围

Windows/Linux amd64 CLI、Windows 安装器和便携 ZIP、构建清单及 SHA256SUMS、中文离线手册、更新日志和入门说明。未签名，不冒充正式渠道认证。安装器只安装 myenv 与可选当前用户 PATH，不自动安装 SDK。

功能边界以 sdk-dev3-delivery.md 为准，颜色/阶段行为以 terminal-output-design.md 为准。Windows 安装逻辑未变，复用已有隔离验证及用户安装记录；本轮不写真实用户 PATH，不重新安装 SDK，不修改旧 trial README 或截图。
