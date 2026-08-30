package api

import (
	"testing"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	svc := NewService()
	keys := []string{
		"pay/gateway/timeout_ms",
		"pay/gateway/retry",
		"pay/order/limit",
		"cdn/vod/bitrate",
		"debug",
	}
	for _, k := range keys {
		if _, err := svc.Create(k, "v", "tester"); err != nil {
			t.Fatalf("create %s: %v", k, err)
		}
	}
	return svc
}

func TestServiceTree(t *testing.T) {
	svc := newTestService(t)
	tree := svc.Tree()
	if len(tree) != 3 {
		t.Fatalf("tree len = %d, want 3", len(tree))
	}
	if tree[0].Business != "cdn" || tree[2].Business != "pay" {
		t.Fatalf("business order = %q, %q, %q", tree[0].Business, tree[1].Business, tree[2].Business)
	}
}

func TestServiceGetBusiness(t *testing.T) {
	svc := newTestService(t)
	mods, ok := svc.GetBusiness("pay")
	if !ok {
		t.Fatal("GetBusiness(pay) should be found")
	}
	if len(mods) != 2 {
		t.Fatalf("pay modules = %d, want 2", len(mods))
	}
	if _, ok := svc.GetBusiness("missing"); ok {
		t.Fatal("GetBusiness(missing) should be missing")
	}
	if _, ok := svc.GetBusiness(""); ok {
		t.Fatal("GetBusiness('') should be missing")
	}
}

func TestServiceGetModule(t *testing.T) {
	svc := newTestService(t)
	items, ok := svc.GetModule("pay", "gateway")
	if !ok {
		t.Fatal("GetModule(pay, gateway) should be found")
	}
	if len(items) != 2 {
		t.Fatalf("pay/gateway items = %d, want 2", len(items))
	}
	if _, ok := svc.GetModule("pay", "missing"); ok {
		t.Fatal("GetModule(pay, missing) should be missing")
	}
	if _, ok := svc.GetModule("", "gateway"); ok {
		t.Fatal("GetModule('', gateway) should be missing")
	}
}

func TestServiceGetItem(t *testing.T) {
	svc := newTestService(t)
	cfg, ok := svc.GetItem("pay", "gateway", "timeout_ms")
	if !ok {
		t.Fatal("GetItem(pay, gateway, timeout_ms) should be found")
	}
	if cfg.Key != "pay/gateway/timeout_ms" {
		t.Fatalf("item key = %q", cfg.Key)
	}
	if _, ok := svc.GetItem("pay", "gateway", "missing"); ok {
		t.Fatal("GetItem missing item should not be found")
	}
	if _, ok := svc.GetItem("pay", "gateway", ""); ok {
		t.Fatal("GetItem with empty name should not be found")
	}
}

func TestServiceCreateInvalidKey(t *testing.T) {
	svc := NewService()
	if _, err := svc.Create("pay//timeout", "v", "t"); err == nil {
		t.Error("Create with empty segment should fail")
	}
	if _, err := svc.Create("", "v", "t"); err == nil {
		t.Error("Create with empty key should fail")
	}
}

func TestServiceUpdateItemVersionBump(t *testing.T) {
	svc := newTestService(t)
	cfg, err := svc.Update("pay/gateway/timeout_ms", "200", "tester")
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if cfg.Version != 2 || cfg.Value != "200" {
		t.Fatalf("updated cfg = %+v, want version 2 value 200", cfg)
	}
	if cfg.UpdatedBy != "tester" {
		t.Fatalf("updated_by = %q", cfg.UpdatedBy)
	}
}

// TestServiceHistory 验证创建/更新/删除都会留下审计历史，且删除后 current 不存在。
func TestServiceHistory(t *testing.T) {
	svc := NewService()
	if _, err := svc.Create("pay/gateway/timeout_ms", "100", "alice"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := svc.Update("pay/gateway/timeout_ms", "200", "bob"); err != nil {
		t.Fatalf("update: %v", err)
	}
	if err := svc.Delete("pay/gateway/timeout_ms"); err != nil {
		t.Fatalf("delete: %v", err)
	}

	hist := svc.History("pay/gateway/timeout_ms")
	if len(hist) != 3 {
		t.Fatalf("history len = %d, want 3", len(hist))
	}
	if hist[0].Action != "create" || hist[0].Before != "" || hist[0].After != "100" || hist[0].Operator != "alice" || hist[0].Version != 1 {
		t.Fatalf("hist[0] = %+v", hist[0])
	}
	if hist[1].Action != "update" || hist[1].Before != "100" || hist[1].After != "200" || hist[1].Version != 2 || hist[1].Operator != "bob" {
		t.Fatalf("hist[1] = %+v", hist[1])
	}
	if hist[2].Action != "delete" || hist[2].Before != "200" || hist[2].After != "" || hist[2].Version != 3 {
		t.Fatalf("hist[2] = %+v", hist[2])
	}
	if _, exists := svc.Get("pay/gateway/timeout_ms"); exists {
		t.Fatal("config should be deleted")
	}
	if h := svc.History("never-existed"); len(h) != 0 {
		t.Fatalf("history of unknown key = %+v", h)
	}
}

// TestServiceHistoryCompare 更新后当前值应与最新历史一致，便于对比排障。
func TestServiceHistoryCompare(t *testing.T) {
	svc := NewService()
	if _, err := svc.Create("cdn/vod/bitrate", "1000", "alice"); err != nil {
		t.Fatalf("create: %v", err)
	}
	cfg, _ := svc.Update("cdn/vod/bitrate", "2000", "bob")

	hist := svc.History("cdn/vod/bitrate")
	last := hist[len(hist)-1]
	if last.After != cfg.Value || last.Version != cfg.Version {
		t.Fatalf("last history %+v != current %+v", last, cfg)
	}
}
