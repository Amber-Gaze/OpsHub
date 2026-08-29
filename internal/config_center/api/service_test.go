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
