package assurance

import (
	"net"
	"net/url"
	"strings"

	"github.com/Ozgurisikdamar/Membrane-AI/pkg/errs"
)

// Allowlist is the server-side set of AgentTwin destinations Membrane is allowed
// to call. It is the SSRF defense for the assurance adapter (work order §7.2):
// the AgentTwin base URL comes from operator configuration, never from a request
// or an agent attribute, and every outbound URL is checked against this list
// before use. The list is pure data; the Phase-2 HTTP adapter also re-checks
// each redirect target with CheckURL and refuses redirects to new hosts.
type Allowlist struct {
	// hosts is the set of exact lowercase host[:port] values that are allowed.
	hosts map[string]struct{}
	// allowLoopback permits http/https to loopback addresses (localhost,
	// 127.0.0.0/8, ::1) for local development and the compose stack. It is off
	// in production configuration.
	allowLoopback bool
}

// NewAllowlist builds an allowlist from a set of allowed host[:port] values. A
// bare host (no port) matches that host on any port. Hosts are compared
// case-insensitively. allowLoopback additionally permits loopback destinations.
func NewAllowlist(allowLoopback bool, hosts ...string) *Allowlist {
	a := &Allowlist{hosts: make(map[string]struct{}, len(hosts)), allowLoopback: allowLoopback}
	for _, h := range hosts {
		h = strings.ToLower(strings.TrimSpace(h))
		if h != "" {
			a.hosts[h] = struct{}{}
		}
	}
	return a
}

// CheckURL parses raw and reports whether it is an acceptable AgentTwin
// destination. It rejects: non-http(s) schemes, URLs with embedded credentials,
// a missing host, and any host that is neither loopback (when enabled) nor in
// the allowlist. It does not perform DNS resolution; name-to-IP rebinding is
// defended at dial time by the Phase-2 adapter.
func (a *Allowlist) CheckURL(raw string) error {
	const op = Op + ".Allowlist.CheckURL"
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return errs.Validation(op, "destination is not a valid URL", err)
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return errs.Validation(op, "destination scheme must be http or https, got "+u.Scheme, nil)
	}
	if u.User != nil {
		return errs.Validation(op, "destination must not embed credentials", nil)
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return errs.Validation(op, "destination has no host", nil)
	}

	if isLoopbackHost(host) {
		if a.allowLoopback {
			return nil
		}
		return errs.Validation(op, "loopback destination is not allowed in this configuration", nil)
	}
	if scheme != "https" {
		return errs.Validation(op, "non-loopback destination must use https", nil)
	}

	hostPort := host
	if p := u.Port(); p != "" {
		hostPort = host + ":" + p
	}
	if _, ok := a.hosts[hostPort]; ok {
		return nil
	}
	if _, ok := a.hosts[host]; ok {
		return nil
	}
	return errs.Validation(op, "destination host is not in the allowlist: "+host, nil)
}

// isLoopbackHost reports whether host is localhost or a loopback IP literal.
func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}
	return false
}
