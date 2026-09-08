package core

import "testing"

func TestDNSDefaultsRuntimeQueryPolicy(t *testing.T) {
	cases := []struct {
		name, domain, mode string
		direct, remote     int32
	}{
		{name: "unknown defaults to proxy", domain: "unknown.test", remote: 1},
		{name: "proxy list", domain: "proxy.test", remote: 1},
		{name: "direct list", domain: "direct.test", direct: 1},
		{name: "overlapping lists prefer proxy", domain: "overlap.test", remote: 1},
		{name: "advertisements never leave process", domain: "ads.test"},
		{name: "explicit Direct mode", domain: "unknown.test", mode: "Direct", direct: 1},
		{name: "Global mode overrides direct list", domain: "direct.test", mode: "Global", remote: 1},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			runtime := newPolicyRuntime(t, policyDefaultsFixture())
			if test.mode != "" {
				runtime.mode.mode = test.mode
			}
			if _, err := runtime.query(test.domain); err != nil {
				t.Fatal(err)
			}
			if runtime.direct.calls.Load() != test.direct || runtime.remote.calls.Load() != test.remote {
				t.Fatalf("direct/remote DNS queries = %d/%d, want %d/%d",
					runtime.direct.calls.Load(), runtime.remote.calls.Load(), test.direct, test.remote)
			}
		})
	}
}

func TestDNSDefaultsRuntimeRemoteFailureDoesNotFallback(t *testing.T) {
	runtime := newPolicyRuntime(t, policyDefaultsFixture())
	runtime.remote.fail = true
	if _, err := runtime.query("unknown.test"); err == nil {
		t.Fatal("remote DNS failure was hidden")
	}
	if runtime.direct.calls.Load() != 0 || runtime.remote.calls.Load() != 1 {
		t.Fatalf("remote failure leaked DNS: direct=%d remote=%d", runtime.direct.calls.Load(), runtime.remote.calls.Load())
	}
}

func TestDNSDefaultsRuntimeSeparateResolverCaches(t *testing.T) {
	runtime := newPolicyRuntime(t, policyDefaultsFixture())
	if _, err := runtime.query("direct.test"); err != nil {
		t.Fatal(err)
	}
	runtime.mode.mode = "Global"
	if _, err := runtime.query("direct.test"); err != nil {
		t.Fatal(err)
	}
	if runtime.direct.calls.Load() != 1 || runtime.remote.calls.Load() != 1 {
		t.Fatal("proxy DNS reused the direct resolver cache")
	}
}
