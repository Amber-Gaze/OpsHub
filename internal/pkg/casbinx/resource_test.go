package casbinx

import "testing"

func TestNormalizeConfigResource(t *testing.T) {
	cases := []struct {
		path   string
		method string
		obj    string
		act    string
	}{
		{"/configs", "GET", "config/*", "read"},
		{"/configs/tree", "GET", "config/*", "read"},
		{"/configs/debug", "GET", "config/default/debug", "read"},
		{"/configs/pay", "GET", "config/default/pay", "read"},
		{"/configs/pay/gateway", "GET", "config/pay/gateway", "read"},
		{"/configs/pay/gateway/timeout_ms", "GET", "config/pay/gateway/timeout_ms", "read"},
		{"/configs/tree/pay", "GET", "config/default/pay", "read"},
		{"/configs/tree/pay/gateway", "GET", "config/pay/gateway", "read"},
		{"/configs/tree/pay/gateway/timeout_ms", "GET", "config/pay/gateway/timeout_ms", "read"},
		{"/internal/configs/tree/pay/gateway", "GET", "config/pay/gateway", "read"},
		{"/configs/pay/gateway/timeout_ms", "PUT", "config/pay/gateway/timeout_ms", "write"},
		{"/configs/tree/pay/gateway/timeout_ms", "DELETE", "config/pay/gateway/timeout_ms", "delete"},
	}
	for _, c := range cases {
		obj, act := NormalizeConfigResource(c.path, c.method)
		if obj != c.obj || act != c.act {
			t.Errorf("NormalizeConfigResource(%q, %q) = (%q, %q), want (%q, %q)",
				c.path, c.method, obj, act, c.obj, c.act)
		}
	}
}
