# 距离性能目标还有多远

截至 2026-09-09：以下是已有实测记录，不是 0.1.0-rc.1 的新测成绩。启动采用最近的 dev.2 批次；status/run 采用此前固定批次，不能拼成同一次验收。

| 平台 / 指标 | 实测 | 预算 | 超出预算 | 达标需要减少当前值 |
| --- | ---: | ---: | ---: | ---: |
| Windows help p95（dev.2） | 62.82 ms | 30 ms | 32.82 ms | 52.2% |
| Windows version p95（dev.2） | 61.59 ms | 30 ms | 31.59 ms | 51.3% |
| Windows status p95（此前批次） | 139.83 ms | 50 ms | 89.83 ms | 64.2% |
| Windows run 附加 p95（此前批次） | 98.56 ms | 50 ms | 48.56 ms | 49.3% |
| Kali help p95（dev.2） | 65.99 ms | 30 ms | 35.99 ms | 54.5% |
| Kali version p95（dev.2） | 102.23 ms | 30 ms | 72.23 ms | 70.7% |
| Kali run 附加 p95（此前批次） | 88.15 ms | 50 ms | 38.15 ms | 43.3% |
| Kali 聚合 myenv RSS（此前采样） | 39.24 MiB | 32 MiB | 7.24 MiB | 18.5% |

后续 Linux OpenForRun 实验 run p95 为 118.03 → 170.29 ms（目标 50 ms），并未证明延迟收益；该实验采样聚合 RSS 为 32.10 → 29.77 MiB，但采样非原子、可能漏峰，不能拿它替代之前的超预算记录宣布通过。记录见 run-stage-evaluation.md。Kali 早期启动曾达标，后续批次不再达标，反映批次波动，不能只选早期较好成绩。

Windows 43.43 MiB 的历史数字包含 Node 且是各自峰值之和，不是 myenv 自身的同时聚合 RSS；不与 32 MiB 直接作同口径比较。尚缺当前 RC 的统一有效验收证据。

实践含义：单次 CLI 多等几十到一百多毫秒，手动使用通常可以接受；频繁调用会累积。内存差约 7 MiB，但延迟需要约减半或更多，不能承诺一次参数调优即可完成。下一轮应保持统计方法与可靠性约束，以当前 RC 固定一次基线后只优化实际热点，避免反复重采样挑结果。

来源：kali-validation.md、localization-delivery.md、run-stage-evaluation.md。此轮只汇总，没有新增性能采样。
