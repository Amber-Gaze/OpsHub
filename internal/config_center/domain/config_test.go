package domain

import (
	"testing"
)

func TestSplitKey(t *testing.T) {
	cases := []struct {
		key      string
		business string
		module   string
	}{
		{"pay", "pay", ""},
		{"pay/gateway", "pay", "gateway"},
		{"pay/gateway/timeout", "pay", "gateway"},
		{"a/b/c/d", "a", "b"},
	}
	for _, c := range cases {
		b, m := SplitKey(c.key)
		if b != c.business || m != c.module {
			t.Errorf("SplitKey(%q) = (%q, %q), want (%q, %q)", c.key, b, m, c.business, c.module)
		}
	}
}

func TestItemName(t *testing.T) {
	cases := map[string]string{
		"pay":                 "pay",
		"pay/gateway":         "gateway",
		"pay/gateway/timeout": "timeout",
	}
	for key, want := range cases {
		if got := ItemName(key); got != want {
			t.Errorf("ItemName(%q) = %q, want %q", key, got, want)
		}
	}
}

func TestValidateKey(t *testing.T) {
	valid := []string{"pay", "pay/gateway", "pay/gateway/timeout", "a/b/c/d"}
	for _, key := range valid {
		if err := ValidateKey(key); err != nil {
			t.Errorf("ValidateKey(%q) unexpected error: %v", key, err)
		}
	}
	invalid := []string{"", "  ", "pay//timeout", "/pay", "pay/", "pay/ /x"}
	for _, key := range invalid {
		if err := ValidateKey(key); err == nil {
			t.Errorf("ValidateKey(%q) should fail", key)
		}
	}
}

func TestBuildTree(t *testing.T) {
	items := []ConfigItem{
		{Key: "pay/gateway/retry", Value: "3"},
		{Key: "pay/gateway/timeout_ms", Value: "100"},
		{Key: "pay/order/limit", Value: "5"},
		{Key: "cdn/vod/bitrate", Value: "2000"},
		{Key: "debug", Value: "false"},
	}
	tree := BuildTree(items)

	if len(tree) != 3 {
		t.Fatalf("BuildTree len = %d, want 3 (cdn, debug, pay)", len(tree))
	}
	// 业务按字典序：cdn < debug < pay
	if tree[0].Business != "cdn" || tree[1].Business != "debug" || tree[2].Business != "pay" {
		t.Fatalf("business order = %q, %q, %q; want cdn, debug, pay",
			tree[0].Business, tree[1].Business, tree[2].Business)
	}

	// cdn/vod/bitrate：business=cdn, module=vod
	if len(tree[0].Modules) != 1 || tree[0].Modules[0].Module != "vod" {
		t.Fatalf("cdn modules = %+v, want [vod]", tree[0].Modules)
	}
	if got := tree[0].Modules[0].Items[0].Key; got != "cdn/vod/bitrate" {
		t.Fatalf("cdn item key = %q", got)
	}

	// debug：单段 key 归入 module=""
	if len(tree[1].Modules) != 1 || tree[1].Modules[0].Module != "" {
		t.Fatalf("debug modules = %+v, want [module=\"\"]", tree[1].Modules)
	}

	// pay：模块 gateway, order（字典序 gateway < order）
	if len(tree[2].Modules) != 2 {
		t.Fatalf("pay modules = %d, want 2", len(tree[2].Modules))
	}
	if tree[2].Modules[0].Module != "gateway" || tree[2].Modules[1].Module != "order" {
		t.Fatalf("pay module order = %q, %q; want gateway, order",
			tree[2].Modules[0].Module, tree[2].Modules[1].Module)
	}
	// gateway 下项按 key 排序
	gw := tree[2].Modules[0].Items
	if len(gw) != 2 || gw[0].Key != "pay/gateway/retry" || gw[1].Key != "pay/gateway/timeout_ms" {
		t.Fatalf("gateway items = %+v", gw)
	}
}

func TestGroupByBusiness(t *testing.T) {
	items := []ConfigItem{
		{Key: "pay/gateway/timeout_ms"},
		{Key: "pay/order/limit"},
		{Key: "cdn/vod/bitrate"},
	}

	mods, ok := GroupByBusiness(items, "pay")
	if !ok {
		t.Fatal("GroupByBusiness(pay) should be found")
	}
	if len(mods) != 2 || mods[0].Module != "gateway" || mods[1].Module != "order" {
		t.Fatalf("pay modules = %+v", mods)
	}

	if _, ok := GroupByBusiness(items, "unknown"); ok {
		t.Fatal("GroupByBusiness(unknown) should be missing")
	}
}
