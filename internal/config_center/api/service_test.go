package api

import (
	"context"
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

// TestServiceRevisionAndChangesSince 验证全局版本号单调递增，以及增量查询 ChangesSince。
func TestServiceRevisionAndChangesSince(t *testing.T) {
	svc := NewService()
	if rev := svc.Revision(context.Background()); rev != 0 {
		t.Fatalf("initial revision = %d, want 0", rev)
	}
	if _, err := svc.Create("pay/gateway/timeout_ms", "100", "a"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create("pay/gateway/retry", "3", "a"); err != nil {
		t.Fatal(err)
	}
	if rev := svc.Revision(context.Background()); rev != 2 {
		t.Fatalf("revision after 2 creates = %d, want 2", rev)
	}
	if _, err := svc.Update("pay/gateway/timeout_ms", "200", "b"); err != nil {
		t.Fatal(err)
	}
	rev2 := svc.Revision(context.Background())
	if rev2 != 3 {
		t.Fatalf("revision after update = %d, want 3", rev2)
	}

	// ChangesSince(0) 聚合出各 key 最后一次变更
	changed := svc.ChangesSince(0)
	if ch := changed["pay/gateway/timeout_ms"]; ch.Action != "update" || ch.After != "200" {
		t.Fatalf("timeout_ms last change = %+v", ch)
	}
	if ch := changed["pay/gateway/retry"]; ch.Action != "create" {
		t.Fatalf("retry last change = %+v", ch)
	}
	if len(svc.ChangesSince(rev2)) != 0 {
		t.Fatalf("changes since latest should be empty, got %v", svc.ChangesSince(rev2))
	}

	// 删除：ChangesSince 能看到 delete，且 revision 继续递增
	if err := svc.Delete("pay/gateway/retry"); err != nil {
		t.Fatal(err)
	}
	rev3 := svc.Revision(context.Background())
	if rev3 != 4 {
		t.Fatalf("revision after delete = %d, want 4", rev3)
	}
	if ch := svc.ChangesSince(rev2)["pay/gateway/retry"]; ch.Action != "delete" {
		t.Fatalf("retry last change = %+v", ch)
	}
	if _, exists := svc.Get("pay/gateway/retry"); exists {
		t.Fatal("retry should be deleted")
	}
}

// TestPullKeyMatcher 验证各过滤参数的匹配语义。
func TestPullKeyMatcher(t *testing.T) {
	cases := []struct {
		name       string
		business   string
		module     string
		nameParam  string
		path       string
		key        string
		keyToMatch string
		want       bool
	}{
		{"business only", "pay", "", "", "", "", "pay/gateway/timeout_ms", true},
		{"business mismatch", "pay", "", "", "", "", "cdn/vod/bitrate", false},
		{"module", "pay", "gateway", "", "", "", "pay/gateway/retry", true},
		{"module mismatch", "pay", "order", "", "", "", "pay/gateway/retry", false},
		{"item name", "pay", "gateway", "timeout_ms", "", "", "pay/gateway/timeout_ms", true},
		{"item name mismatch", "pay", "gateway", "timeout_ms", "", "", "pay/gateway/retry", false},
		{"path prefix", "", "", "", "pay/gateway", "", "pay/gateway/timeout_ms", true},
		{"path exact", "", "", "", "cdn/vod/bitrate", "", "cdn/vod/bitrate", true},
		{"path not under", "", "", "", "pay/gateway", "", "pay/order/limit", false},
		{"key exact", "", "", "", "", "cdn/vod/bitrate", "cdn/vod/bitrate", true},
		{"key mismatch", "", "", "", "", "cdn/vod/bitrate", "cdn/vod/other", false},
	}
	for _, c := range cases {
		m := pullKeyMatcher(c.business, c.module, c.nameParam, c.path, c.key)
		if got := m(c.keyToMatch); got != c.want {
			t.Errorf("%s: match(%q) = %v, want %v", c.name, c.keyToMatch, got, c.want)
		}
	}
}
