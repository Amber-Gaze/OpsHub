package casbinx

import (
	"testing"
)

func TestNormalizeConfigObject(t *testing.T) {
	cases := map[string]string{
		"":                         "config/*",
		"debug":                    "config/default/debug",
		"pay/gateway/timeout_ms":   "config/pay/gateway/timeout_ms",
		"/pay/gateway/timeout_ms/": "config/pay/gateway/timeout_ms",
	}
	for key, want := range cases {
		if got := NormalizeConfigObject(key); got != want {
			t.Errorf("NormalizeConfigObject(%q) = %q, want %q", key, got, want)
		}
	}
}

func TestGrantsMatch(t *testing.T) {
	grants := []Grant{
		{Obj: "config/pay/**", Act: "read"},
		{Obj: "config/pay/gateway/timeout_ms", Act: "write"},
	}
	cases := []struct {
		obj  string
		act  string
		want bool
	}{
		{"config/pay/gateway/x", "read", true},             // pay/** read 覆盖
		{"config/pay/gateway/x", "write", false},           // pay/** 仅 read
		{"config/pay/gateway/timeout_ms", "write", true},   // 精确项 write
		{"config/pay/gateway/timeout_ms", "delete", false}, // 无 delete
		{"config/cdn/x", "read", false},                    // 越权
	}
	for _, c := range cases {
		if got := GrantsMatch(grants, c.obj, c.act); got != c.want {
			t.Errorf("GrantsMatch(%q, %q) = %v, want %v", c.obj, c.act, got, c.want)
		}
	}
}

func TestCanReadWriteDelete(t *testing.T) {
	readOnly := []Grant{{Obj: "config/pay/**", Act: "read"}}
	if !CanRead(readOnly, "pay/gateway/timeout_ms") {
		t.Error("CanRead(pay/...) should be true")
	}
	if CanWrite(readOnly, "pay/gateway/timeout_ms") {
		t.Error("CanWrite on read-only should be false")
	}

	all := AdminGrant
	if !CanRead(all, "cdn/vod/x") || !CanWrite(all, "cdn/vod/x") || !CanDelete(all, "cdn/vod/x") {
		t.Error("AdminGrant should cover read/write/delete")
	}
}

func TestBuildConfigObj(t *testing.T) {
	cases := []struct {
		b, m, i string
		want    string
	}{
		{"pay", "", "", "config/pay/**"},
		{"pay", "gateway", "", "config/pay/gateway/**"},
		{"pay", "gateway", "timeout", "config/pay/gateway/timeout"},
		{"", "", "", "config/**"},
	}
	for _, c := range cases {
		if got := BuildConfigObj(c.b, c.m, c.i); got != c.want {
			t.Errorf("BuildConfigObj(%q,%q,%q) = %q, want %q", c.b, c.m, c.i, got, c.want)
		}
	}
}

func TestScopePayloadRoundTrip(t *testing.T) {
	grants := []Grant{{Obj: "config/pay/**", Act: "read"}, {Obj: "config/cdn/**", Act: "write"}}
	payload, _ := BuildScopePayload("alice", grants)
	subject, parsed, err := ParseScopePayload(payload)
	if err != nil {
		t.Fatalf("ParseScopePayload: %v", err)
	}
	if subject != "alice" {
		t.Fatalf("subject = %q, want alice", subject)
	}
	if len(parsed) != 2 || parsed[0].Obj != "config/pay/**" || parsed[1].Act != "write" {
		t.Fatalf("parsed = %+v", parsed)
	}
	if _, _, err := ParseScopePayload("allow|alice|x|1"); err == nil {
		t.Fatal("non-scope payload should fail")
	}
}

func TestEncodeDecodeGrants(t *testing.T) {
	grants := []Grant{{Obj: "config/pay/**", Act: "read"}}
	enc := EncodeGrants(grants)
	if enc == "" {
		t.Fatal("encode empty")
	}
	dec := DecodeGrants(enc)
	if len(dec) != 1 || dec[0].Obj != "config/pay/**" {
		t.Fatalf("decode = %+v", dec)
	}
	if DecodeGrants("!!not-base64!!") != nil {
		t.Fatal("invalid base64 should return nil")
	}
	if DecodeGrants("") != nil {
		t.Fatal("empty should return nil")
	}
}
