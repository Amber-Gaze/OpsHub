package api

import "testing"

func TestNormalizeModulePath(t *testing.T) {
	cases := []struct{ in, want string }{
		{"pay/gateway", "pay/gateway"},
		{"/pay/gateway/", "pay/gateway"},
		{"pay", "pay"},
		{"  common/ratelimit ", "common/ratelimit"},
		{"a/b/c/d", "a/b"}, // 只保留 2 段
		{"//", ""},
		{"", ""},
	}
	for _, c := range cases {
		if got := normalizeModulePath(c.in); got != c.want {
			t.Errorf("normalizeModulePath(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestModulePathToObj(t *testing.T) {
	if obj, ok := modulePathToObj("pay/gateway"); !ok || obj != "config/pay/gateway/**" {
		t.Fatalf("modulePathToObj(pay/gateway) = %q, %v", obj, ok)
	}
	if obj, ok := modulePathToObj("pay"); !ok || obj != "config/pay/**" {
		t.Fatalf("modulePathToObj(pay) = %q, %v", obj, ok)
	}
	if _, ok := modulePathToObj(""); ok {
		t.Fatal("empty path should be invalid")
	}
}
