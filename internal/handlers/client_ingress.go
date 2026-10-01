package handlers

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// Login throttle: which door a request came through, and who it came from
// (GYST-VN-006).
//
// The app sits behind Caddy, so r.RemoteAddr is always Caddy. Two Caddy sites
// serve it (see the Caddyfile): a public one that only proxies the Cloudflare
// hostname, and a tailnet one on the MagicDNS name. Caddy replaces any
// client-supplied X-Forwarded-For with its immediate peer (probe P1, recorded
// in the PR), so the rightmost X-Forwarded-For element is the address Caddy
// saw. CF-Connecting-IP is passed through untouched, which makes it forgeable
// on any path that is not actually fronted by Cloudflare. It is therefore read
// ONLY when the request was served by the public site. That is the trust
// boundary accepted in decision D1; its residual (a container on the shared
// proxy network can forge the header) is documented in docs/ARCHITECTURE.md.
//
// The key is for rate limiting only. It is never used for authorisation.
//
// Local development has no Caddy in front, so the rightmost X-Forwarded-For is
// client-controlled there and localhost classifies as public, keyed by peer.

const (
	// publicHost is the committed public hostname, the same literal as the
	// Caddyfile's http://gyst.zaylolabs.com:8081 site.
	publicHost = "gyst.zaylolabs.com"
	// tailnetSuffix marks the MagicDNS host served by Caddy's tailnet site.
	tailnetSuffix = ".ts.net"

	bucketPublic  = "public"
	bucketTailnet = "tailnet"

	unknownClient = "unknown"
)

// normalizeHost lowercases the Host, drops any port and a trailing dot. It
// reads only r.Host, never X-Forwarded-Host.
func normalizeHost(host string) string {
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	return strings.TrimSuffix(strings.ToLower(host), ".")
}

// classifyIngress returns the throttle bucket for a request. Only a host that
// ends in a ".ts.net" label is tailnet. Everything else, including an empty or
// unrecognised Host, is public: that fails safe and keeps the tailnet recovery
// bucket clean.
func classifyIngress(r *http.Request) string {
	h := normalizeHost(r.Host)
	if len(h) > len(tailnetSuffix) && strings.HasSuffix(h, tailnetSuffix) {
		return bucketTailnet
	}
	return bucketPublic
}

// isPublicSite reports whether CF-Connecting-IP may be trusted: the normalised
// Host is exactly the committed public hostname.
func isPublicSite(r *http.Request) bool {
	return normalizeHost(r.Host) == publicHost
}

// parseAddr parses an IP with netip, drops any zone and unmaps 4-in-6.
func parseAddr(s string) (netip.Addr, bool) {
	a, err := netip.ParseAddr(strings.TrimSpace(s))
	if err != nil {
		return netip.Addr{}, false
	}
	return a.WithZone("").Unmap(), true
}

// clientPeer returns the client address as recorded by Caddy: the rightmost
// X-Forwarded-For element, falling back to the connection address.
func clientPeer(r *http.Request) (netip.Addr, bool) {
	if values := r.Header.Values("X-Forwarded-For"); len(values) > 0 {
		joined := strings.Join(values, ",")
		last := joined[strings.LastIndexByte(joined, ',')+1:]
		if a, ok := parseAddr(last); ok {
			return a, true
		}
	}
	if ap, err := netip.ParseAddrPort(r.RemoteAddr); err == nil {
		return ap.Addr().WithZone("").Unmap(), true
	}
	if a, ok := parseAddr(r.RemoteAddr); ok {
		return a, true
	}
	return netip.Addr{}, false
}

// cfAddr parses CF-Connecting-IP. An IPv6 address is masked to its /64 so one
// subscriber's rotating addresses share a key; IPv4 is used as is.
func cfAddr(r *http.Request) (netip.Addr, bool) {
	a, ok := parseAddr(r.Header.Get("CF-Connecting-IP"))
	if !ok {
		return netip.Addr{}, false
	}
	if a.Is6() {
		return netip.PrefixFrom(a, 64).Masked().Addr(), true
	}
	return a, true
}

// logLabel truncates an address for log lines: IPv4 to /24, IPv6 to /48.
func logLabel(a netip.Addr) string {
	if !a.IsValid() {
		return unknownClient
	}
	bits := 24
	if a.Is6() {
		bits = 48
	}
	return netip.PrefixFrom(a, bits).Masked().String()
}

// loginClient returns the throttle bucket, the map key and the truncated log
// label for a request. Keys are built only from normalised addresses; raw
// header text is never used.
//
//	pub|<cf>        public site with a parseable CF-Connecting-IP
//	pub-peer|<ip>   public bucket without a usable CF header, or an unknown Host
//	ts|<ip>         tailnet site (CF-Connecting-IP is never read)
//	unknown         nothing parseable at all: one shared key, fail closed
func loginClient(r *http.Request) (bucket, key, label string) {
	bucket = classifyIngress(r)
	if bucket == bucketPublic && isPublicSite(r) {
		if cf, ok := cfAddr(r); ok {
			return bucket, "pub|" + cf.String(), logLabel(cf)
		}
	}
	prefix := "pub-peer|"
	if bucket == bucketTailnet {
		prefix = "ts|"
	}
	peer, ok := clientPeer(r)
	if !ok {
		return bucket, prefix + unknownClient, unknownClient
	}
	return bucket, prefix + peer.String(), logLabel(peer)
}
