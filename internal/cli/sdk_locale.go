package cli

func (l locale) sdkLabel(value string) string {
	if !l {
		return value
	}
	labels := map[string]string{"available": "可用", "broken": "异常", "alias": "执行别名", "shim": "管理器入口", "unverified": "未验证", "external": "外部安装", "unknown": "未知", "not_found": "未找到", "present": "已找到", "registered": "已登记", "stable": "稳定版", "preview": "预览版"}
	if translated, ok := labels[value]; ok {
		return translated
	}
	return value
}
