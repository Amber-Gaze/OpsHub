package domain

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// ConfigItem 一条配置记录；Key 采用 business/module/name 三段式逻辑键
// （例如 pay/gateway/timeout_ms），单段键视为业务级配置（模块为空）。
type ConfigItem struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	Version   int       `json:"version"`
	UpdatedAt time.Time `json:"updated_at"`
	UpdatedBy string    `json:"updated_by"`
}

// ConfigChange 一条配置变更记录（审计/历史）。Action 取值：
// create（新建）/ update（修改）/ delete（删除）。
// 后端只负责提供变更前后两份内容（Before/After），差异对比由前端公共库完成。
// Revision 为该变更发生时的全局配置版本号（供下游增量拉取判断更新）。
type ConfigChange struct {
	Key       string    `json:"key"`
	Version   int       `json:"version"`  // 变更发生时的配置版本（delete 为被删版本的下一序号）
	Revision  int64     `json:"revision"` // 全局配置版本号（每次写操作 +1，单调递增）
	Action    string    `json:"action"`
	Before    string    `json:"before"` // 变更前的值（create 为空）
	After     string    `json:"after"`  // 变更后的值（delete 为空）
	Operator  string    `json:"operator"`
	CreatedAt time.Time `json:"created_at"`
}

// ConfigHistoryResponse 配置历史对比响应：当前值 + 全部历史变更。
// Current 为 null 表示该配置已被删除。
type ConfigHistoryResponse struct {
	Key     string         `json:"key"`
	Current *ConfigItem    `json:"current"`
	History []ConfigChange `json:"history"`
}

// PullResponse 下游服务拉取配置的响应（机器消费友好）。
// Revision 为本次拉取时的全局配置版本号；增量拉取（?since=）时 Items 只含变更项、
// Removed 为区间内被删除的 key 列表，下游据此增量更新即可。
type PullResponse struct {
	Revision    int64        `json:"revision"`
	Items       []ConfigItem `json:"items"`
	Removed     []string     `json:"removed,omitempty"`
	GeneratedAt time.Time    `json:"generated_at"`
}

// BusinessNode 业务节点：业务 → 模块 → 具体项，对应控制台左侧「业务」层。
type BusinessNode struct {
	Business string       `json:"business"`
	Modules  []ModuleNode `json:"modules"`
}

// ModuleNode 模块节点：业务下的一个模块及其配置项列表，对应「模块」层。
type ModuleNode struct {
	Module string       `json:"module"`
	Items  []ConfigItem `json:"items"`
}

// BusinessView 单个业务子树的响应视图。
type BusinessView struct {
	Business string       `json:"business"`
	Modules  []ModuleNode `json:"modules"`
}

// ModuleView 单个模块下配置项列表的响应视图。
type ModuleView struct {
	Business string       `json:"business"`
	Module   string       `json:"module"`
	Items    []ConfigItem `json:"items"`
}

// SplitKey 将逻辑 key 拆分为 (业务, 模块)。
//   - 三段及以上 business/module/name → (business, module)
//   - 两段 business/module           → (business, module)
//   - 单段 business                  → (business, "")
func SplitKey(key string) (business, module string) {
	parts := strings.Split(key, "/")
	if len(parts) < 2 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}

// ItemName 返回配置项展示名（key 的末段），供控制台列表展示。
func ItemName(key string) string {
	parts := strings.Split(strings.Trim(key, "/"), "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

// ValidateKey 校验逻辑 key：非空、各段非空（禁止 pay//name、/pay 等写法）。
func ValidateKey(key string) error {
	if strings.TrimSpace(key) == "" {
		return errors.New("key is required")
	}
	for _, seg := range strings.Split(key, "/") {
		if strings.TrimSpace(seg) == "" {
			return fmt.Errorf("invalid key %q: empty segment", key)
		}
	}
	return nil
}

// BuildTree 由配置项列表构建「业务 → 模块 → 项」的完整树，各级按 key 字典序排序。
// 单段 key（业务级配置）归入模块名为空串的节点。
func BuildTree(items []ConfigItem) []BusinessNode {
	grouped := map[string]map[string][]ConfigItem{}
	var businessOrder []string
	for _, it := range items {
		b, m := SplitKey(it.Key)
		if _, ok := grouped[b]; !ok {
			grouped[b] = map[string][]ConfigItem{}
			businessOrder = append(businessOrder, b)
		}
		grouped[b][m] = append(grouped[b][m], it)
	}
	sort.Strings(businessOrder)

	tree := make([]BusinessNode, 0, len(businessOrder))
	for _, b := range businessOrder {
		node := BusinessNode{Business: b}
		mods := grouped[b]
		moduleOrder := make([]string, 0, len(mods))
		for m := range mods {
			moduleOrder = append(moduleOrder, m)
		}
		sort.Strings(moduleOrder)
		for _, m := range moduleOrder {
			items := mods[m]
			sort.Slice(items, func(i, j int) bool { return items[i].Key < items[j].Key })
			node.Modules = append(node.Modules, ModuleNode{Module: m, Items: items})
		}
		tree = append(tree, node)
	}
	return tree
}

// GroupByBusiness 将配置项按业务分组（用于业务子树/模块列表查询）。
// 返回的 map 仅包含本业务下的模块分组；模块不存在时返回 false。
func GroupByBusiness(items []ConfigItem, business string) ([]ModuleNode, bool) {
	grouped := map[string][]ConfigItem{}
	for _, it := range items {
		b, m := SplitKey(it.Key)
		if b == business {
			grouped[m] = append(grouped[m], it)
		}
	}
	if len(grouped) == 0 {
		return nil, false
	}
	moduleOrder := make([]string, 0, len(grouped))
	for m := range grouped {
		moduleOrder = append(moduleOrder, m)
	}
	sort.Strings(moduleOrder)

	nodes := make([]ModuleNode, 0, len(moduleOrder))
	for _, m := range moduleOrder {
		items := grouped[m]
		sort.Slice(items, func(i, j int) bool { return items[i].Key < items[j].Key })
		nodes = append(nodes, ModuleNode{Module: m, Items: items})
	}
	return nodes, true
}
