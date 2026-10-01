package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIngressClassification(t *testing.T) {
	cases := []struct {
		host string
		want string
	}{
		{"gyst.zaylolabs.com", bucketPublic},
		{"GYST.ZAYLOLABS.COM", bucketPublic},
		{"gyst.zaylolabs.com:443", bucketPublic},
		{"gyst.zaylolabs.com.", bucketPublic},
		{"x.ts.net", bucketTailnet},
		{"X.TS.NET.:3443", bucketTailnet},
		{"gyst-test.example.ts.net:3443", bucketTailnet},
		{"ts.net", bucketPublic},
		{".ts.net", bucketPublic},
		{"evil.ts.net.example.com", bucketPublic},
		{"example.com", bucketPublic},
		{"localhost:8080", bucketPublic},
		{"", bucketPublic},
	}
	for _, c := range cases {
		r := httptest.NewRequest(http.MethodPost, "/login", nil)
		r.Host = c.host
		if got := classifyIngress(r); got != c.want {
			t.Errorf("classifyIngress(%q) = %s, want %s", c.host, got, c.want)
		}
	}
}

// Only the exact public hostname may trust CF-Connecting-IP.
func TestCFTrustedOnlyOnPublicHost(t *testing.T) {
	cases := []struct {
		host string
		want bool
	}{
		{"gyst.zaylolabs.com", true},
		{"Gyst.Zaylolabs.com:8081", true},
		{"gyst.zaylolabs.com.", true},
		{"x.ts.net", false},
		{"example.com", false},
		{"evilgyst.zaylolabs.com", false},
		{"gyst.zaylolabs.com.evil.example", false},
		{"", false},
	}
	for _, c := range cases {
		r := httptest.NewRequest(http.MethodPost, "/login", nil)
		r.Host = c.host
		r.Header.Set("X-Forwarded-For", "192.0.2.7")
		r.Header.Set("CF-Connecting-IP", "203.0.113.9")
		_, key, _ := loginClient(r)
		usedCF := key == "pub|203.0.113.9"
		if usedCF != c.want {
			t.Errorf("host %q: key %q, CF used = %v, want %v", c.host, key, usedCF, c.want)
		}
	}
}

func TestIgnoredHeaders(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/login", nil)
	r.Host = "gyst.zaylolabs.com"
	r.Header.Set("X-Forwarded-Host", "gyst.zaylolabs.com")
	r.Header.Set("X-Real-IP", "198.51.100.1")
	r.Header.Set("True-Client-IP", "198.51.100.2")
	r.Header.Set("X-Forwarded-For", "198.51.100.3, 192.0.2.7")
	_, key, _ := loginClient(r)
	if key != "pub-peer|192.0.2.7" {
		t.Errorf("key = %q, want the rightmost X-Forwarded-For only", key)
	}
	r = httptest.NewRequest(http.MethodPost, "/login", nil)
	r.Host = "example.com"
	r.Header.Set("X-Forwarded-Host", "gyst.zaylolabs.com")
	r.Header.Set("CF-Connecting-IP", "203.0.113.9")
	if bucket, key, _ := loginClient(r); bucket != bucketPublic || key == "pub|203.0.113.9" {
		t.Errorf("X-Forwarded-Host must not make a foreign Host trusted: %s %q", bucket, key)
	}
}

func TestClientKeyParsing(t *testing.T) {
	type req struct {
		host, xff, cf, remote string
	}
	cases := []struct {
		name      string
		in        req
		wantKey   string
		wantLabel string
	}{
		{"rightmost XFF on tailnet", req{"x.ts.net", "1.2.3.4, 100.64.1.2", "", "172.18.0.2:1"}, "ts|100.64.1.2", "100.64.1.0/24"},
		{"garbage XFF falls back to RemoteAddr", req{"x.ts.net", "not-an-ip", "", "172.18.0.2:5000"}, "ts|172.18.0.2", "172.18.0.0/24"},
		{"XFF with port is not an address", req{"x.ts.net", "1.2.3.4:80", "", "172.18.0.2:5000"}, "ts|172.18.0.2", "172.18.0.0/24"},
		{"empty rightmost element", req{"x.ts.net", "1.2.3.4,", "", "172.18.0.2:5000"}, "ts|172.18.0.2", "172.18.0.0/24"},
		{"4-in-6 is unmapped", req{"x.ts.net", "::ffff:203.0.113.5", "", "172.18.0.2:1"}, "ts|203.0.113.5", "203.0.113.0/24"},
		{"IPv6 peer is not grouped", req{"x.ts.net", "2001:db8:1:2:3:4:5:6", "", "172.18.0.2:1"}, "ts|2001:db8:1:2:3:4:5:6", "2001:db8:1::/48"},
		{"zone is dropped", req{"x.ts.net", "fe80::1%eth0", "", "172.18.0.2:1"}, "ts|fe80::1", "fe80::/48"},
		{"nothing parseable", req{"x.ts.net", "", "", "garbage"}, "ts|unknown", "unknown"},
		{"public CF IPv4", req{"gyst.zaylolabs.com", "192.0.2.7", "203.0.113.9", ""}, "pub|203.0.113.9", "203.0.113.0/24"},
		{"public CF 4-in-6", req{"gyst.zaylolabs.com", "192.0.2.7", "::ffff:203.0.113.9", ""}, "pub|203.0.113.9", "203.0.113.0/24"},
		{"public CF IPv6 grouped by /64", req{"gyst.zaylolabs.com", "192.0.2.7", "2001:db8:aaaa:bbbb:1:2:3:4", ""}, "pub|2001:db8:aaaa:bbbb::", "2001:db8:aaaa::/48"},
		{"public CF garbage uses the peer", req{"gyst.zaylolabs.com", "192.0.2.7", "garbage", ""}, "pub-peer|192.0.2.7", "192.0.2.0/24"},
		{"public without CF uses the peer", req{"gyst.zaylolabs.com", "192.0.2.7", "", ""}, "pub-peer|192.0.2.7", "192.0.2.0/24"},
		{"unknown host ignores CF", req{"example.com", "192.0.2.7", "203.0.113.9", ""}, "pub-peer|192.0.2.7", "192.0.2.0/24"},
		{"nothing at all on the public bucket", req{"gyst.zaylolabs.com", "", "", ""}, "pub-peer|unknown", "unknown"},
	}
	for _, c := range cases {
		r := httptest.NewRequest(http.MethodPost, "/login", nil)
		r.Host = c.in.host
		r.RemoteAddr = c.in.remote
		if c.in.xff != "" {
			r.Header.Set("X-Forwarded-For", c.in.xff)
		}
		if c.in.cf != "" {
			r.Header.Set("CF-Connecting-IP", c.in.cf)
		}
		_, key, label := loginClient(r)
		if key != c.wantKey || label != c.wantLabel {
			t.Errorf("%s: key=%q label=%q, want key=%q label=%q", c.name, key, label, c.wantKey, c.wantLabel)
		}
	}
}

func TestTwoCFAddressesInOneSlash64ShareAKey(t *testing.T) {
	key := func(cf string) string {
		r := httptest.NewRequest(http.MethodPost, "/login", nil)
		r.Host = "gyst.zaylolabs.com"
		r.Header.Set("CF-Connecting-IP", cf)
		_, k, _ := loginClient(r)
		return k
	}
	if key("2001:db8:1:2::1") != key("2001:db8:1:2:ffff:ffff:ffff:ffff") {
		t.Error("addresses in one /64 must share a key")
	}
	if key("2001:db8:1:2::1") == key("2001:db8:1:3::1") {
		t.Error("different /64s must not share a key")
	}
}
