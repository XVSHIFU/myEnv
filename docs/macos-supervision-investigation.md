# macOS 后代监督接口调查（2026-09-09）

目标仍是架构要求的完整后代完成确认，不能以进程组为空或主进程退出替代。本记录仅排除错误实现方向，不证明macOS支持完成。

Apple当前XNU的[事件头文件](https://raw.githubusercontent.com/apple-oss-distributions/xnu/main/bsd/sys/event.h)标明NOTE_TRACK、NOTE_TRACKERR、NOTE_CHILD自10.5起不再支持；[filt_procattach实现](https://raw.githubusercontent.com/apple-oss-distributions/xnu/main/bsd/kern/kern_event.c)在收到这些位时返回ENOTSUP。因此不能照搬旧kqueue手册或FreeBSD自动追踪fork的示例。NOTE_FORK通知本身也不等于已原子登记所有后代；注册前发生的事件不由这个接口补发。

当前代码核对：internal/runner/supervise_unix.go仅建立进程组/转发信号，Finish直接返回nil；internal/core/preparation_children.go已对非Windows/Linux准备保持ErrTreeUnconfirmed保护。还需审计普通run：internal/core/run.go只在收到ErrTreeUnconfirmed时保留租约，而通用Unix监督器未产生该错误。现有准备保护不能自动证明run租约同样安全。

后续必须先处理实际完成证据与租约保护的一致性，再寻找当前macOS可用的完整监督机制。不能通过去掉准备保护、使用轮询进程列表或依赖未经验证的NOTE_TRACK来宣称支持。原生macOS不可用，本轮没有运行kqueue调用或macOS测试；源码调查不替代原生验证。
后续修复：通用Unix Finish已改为ErrTreeUnconfirmed，普通run不再错误释放租约；Linux专用subreaper路径未变。此保护不是macOS监督实现，完整功能及原生验证仍缺失。

2026-09-09新增方向：Apple当前文档提供Beta [es_new_descendants_client](https://developer.apple.com/documentation/endpointsecurity/es_new_descendants_client%28_%3A_%3A%29)，专门观察调用者的后代，明确不要求root或TCC审批，但仍要求`com.apple.developer.endpoint-security.client`。[entitlement说明](https://developer.apple.com/documentation/bundleresources/entitlements/com.apple.developer.endpoint-security.client)要求向Apple申请；没有该权限时创建客户端失败。不能把旧全系统ES客户端的权限限制直接套用到这个新接口。

当前文档不同变更视图对创建前既存后代及process muting语义有差异，因此实现应在任何用户进程创建前建立客户端，不依赖历史事件补发。仍须核实正式SDK可用版本、事件丢失检测、退出与fork事件排序、监督者死亡后的子树处置，以及完成证明与SQLite租约的持久化关系。可观察后代不自动等于可证明完整回收。

已询问用户是否有获批entitlement、签名/CI入口；尚未收到。当前没有原生Mac/SDK或获批签名材料，未编译或运行这个API，也未申请外部权限、加入系统扩展或改变首版平台范围。此发现改变后续调查方向，不能据此宣布macOS支持已实现。

完成证明进一步核实：Apple的[global_seq_num文档](https://developer.apple.com/documentation/endpointsecurity/es_message_t/global_seq_num)说明该计数器按客户端递增，间隙可检测内核丢事件；因此发生间隙后必须保持未知完成，不能仅靠当前跟踪集合为空解除租约。需要按SDK消息版本检查字段可用性，不能直接读取不存在的字段。

Beta [es_sync_client文档](https://developer.apple.com/documentation/endpointsecurity/es_sync_client%28_%3A_%3A%29)提供队列同步标记：回调发生在标记前消息处理后，且相关系统调用返回前事件已入队。但文档还明确：客户端销毁会调用全部同步回调，空客户端也会立即调用回调。因此“sync回调触发”单独不是完成证明；必须维持独立的客户端存活/错误状态，并在handler外调用同步接口。

待原生验证的关键边界：序号间隙只能在后续消息到达时被观察到，队列同步标记本身是否能揭示最后一批被丢弃事件，当前文档没有给出证明。不能把“已观察序号连续 + sync回调 + 已知集合为空”直接实现成无条件完成。需要可靠的最终消息/完整性协议，并验证fork与exit交错、双重fork、detached后代、事件压力和客户端异常销毁。此处提出的是需要证明的条件，不是已运行测试或已证实接口缺陷。
