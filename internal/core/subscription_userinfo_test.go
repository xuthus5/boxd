package core

import (
	"net/http"
	"testing"
	"time"
)

func TestParseSubscriptionUserinfo(t *testing.T) {
	header := http.Header{}
	header.Set("subscription-userinfo", "upload=100; download=200; total=1024; expire=1719859200")
	traffic := parseSubscriptionUserinfo(header)
	if traffic == nil {
		t.Fatal("expected traffic")
	}
	if traffic.Upload != 100 || traffic.Download != 200 || traffic.Total != 1024 {
		t.Fatalf("traffic = %+v", traffic)
	}
	if traffic.Expire == nil || !traffic.Expire.Equal(time.Unix(1719859200, 0).UTC()) {
		t.Fatalf("expire = %v", traffic.Expire)
	}
}

func TestParseSubscriptionUserinfoPartialAndInvalid(t *testing.T) {
	if parseSubscriptionUserinfo(nil) != nil {
		t.Fatal("nil header should yield nil")
	}
	header := http.Header{}
	if parseSubscriptionUserinfo(header) != nil {
		t.Fatal("empty header should yield nil")
	}
	header.Set("Subscription-Userinfo", "upload=abc; download=-1; total=; expire=0")
	if parseSubscriptionUserinfo(header) != nil {
		t.Fatal("invalid values should yield nil")
	}
	header.Set("Subscription-Userinfo", "upload=12; foo=bar")
	traffic := parseSubscriptionUserinfo(header)
	if traffic == nil || traffic.Upload != 12 || traffic.Download != 0 {
		t.Fatalf("partial traffic = %+v", traffic)
	}
}
