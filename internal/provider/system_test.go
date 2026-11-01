package provider

import (
	"strings"
	"testing"
)

func TestUpstreamNameservers(t *testing.T) {
	resolved := "# This is /run/systemd/resolve/resolv.conf\n" +
		"nameserver 127.0.0.53\n" +
		"nameserver 1.1.1.1\n" +
		"nameserver ::1\n" +
		"nameserver 2606:4700:4700::1111\n" +
		"nameserver not-an-ip\n" +
		"nameserver 8.8.8.8\n" +
		"nameserver 9.9.9.9\n" +
		"search vps-provider.example\n"
	got := strings.Join(upstreamNameservers(resolved), ",")
	want := "1.1.1.1,2606:4700:4700::1111,8.8.8.8"
	if got != want {
		t.Fatalf("upstreamNameservers() = %q, want %q", got, want)
	}
	if got := upstreamNameservers("nameserver 127.0.0.53\nsearch vps-provider.example\n"); len(got) != 0 {
		t.Fatalf("stub-only resolv.conf returned %v, want none", got)
	}
}

func TestRenderResolvConfDropsHostSearchDomains(t *testing.T) {
	got := renderResolvConf([]string{"1.1.1.1", "8.8.8.8"}, nil)
	if strings.Contains(got, "search") {
		t.Fatalf("resolv.conf without search_domains must not contain a search line:\n%s", got)
	}
	if !strings.Contains(got, "nameserver 1.1.1.1\nnameserver 8.8.8.8\n") {
		t.Fatalf("missing nameservers:\n%s", got)
	}
	if got := renderResolvConf([]string{"1.1.1.1"}, []string{"example.com", "corp.example.com"}); !strings.HasSuffix(got, "search example.com corp.example.com\n") {
		t.Fatalf("missing configured search domains:\n%s", got)
	}
}

func TestRenderSysctlConfIsSorted(t *testing.T) {
	got := renderSysctlConf(map[string]string{"vm.swappiness": " 10 ", "net.mptcp.enabled": "0"})
	want := "# Managed by terraform-provider-k3s-vps-wg.\nnet.mptcp.enabled = 0\nvm.swappiness = 10\n"
	if got != want {
		t.Fatalf("renderSysctlConf() = %q, want %q", got, want)
	}
}

func TestRenderK3sConfigResolvConf(t *testing.T) {
	cfg := ClusterConfig{
		Nodes: map[string]NodeConfig{
			"s1": {Key: "s1", Name: "s1", Role: "server", PublicIP: "192.0.2.1"},
			"a1": {Key: "a1", Name: "a1", Role: "agent", PublicIP: "192.0.2.2"},
		},
		NodeNames: []string{"a1", "s1"},
	}
	if strings.Contains(renderK3sConfig(cfg, cfg.Nodes["s1"]), "resolv-conf") {
		t.Fatal("resolv-conf must stay unset without a dns block, so existing clusters do not restart K3s")
	}
	cfg.DNS = DNSConfig{Enabled: true}
	line := "resolv-conf: \"" + k3sResolvConfPath + "\"\n"
	for _, key := range cfg.NodeNames {
		if got := renderK3sConfig(cfg, cfg.Nodes[key]); !strings.Contains(got, line) {
			t.Fatalf("%s config is missing %q:\n%s", key, line, got)
		}
	}
}

func TestValidateDNS(t *testing.T) {
	cases := []struct {
		name    string
		dns     DNSConfig
		k3s     K3sConfig
		wantErr bool
	}{
		{"disabled block skips checks", DNSConfig{Nameservers: []string{"bad"}}, K3sConfig{}, false},
		{"auto-detected nameservers", DNSConfig{Enabled: true}, K3sConfig{}, false},
		{"explicit values", DNSConfig{Enabled: true, Nameservers: []string{"1.1.1.1"}, SearchDomains: []string{"example.com"}}, K3sConfig{}, false},
		{"loopback nameserver", DNSConfig{Enabled: true, Nameservers: []string{"127.0.0.53"}}, K3sConfig{}, true},
		{"too many nameservers", DNSConfig{Enabled: true, Nameservers: []string{"1.1.1.1", "1.0.0.1", "8.8.8.8", "8.8.4.4"}}, K3sConfig{}, true},
		{"root search domain", DNSConfig{Enabled: true, SearchDomains: []string{"."}}, K3sConfig{}, true},
		{"duplicate resolv-conf arg", DNSConfig{Enabled: true}, K3sConfig{ExtraAgentArgs: []string{"--resolv-conf=/etc/r.conf"}}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := validateDNS(tc.dns, tc.k3s).HasError(); got != tc.wantErr {
				t.Fatalf("validateDNS() error = %v, want %v", got, tc.wantErr)
			}
		})
	}
}

func TestValidateSysctl(t *testing.T) {
	cases := []struct {
		name    string
		values  map[string]string
		wantErr bool
	}{
		{"dotted key", map[string]string{"net.mptcp.enabled": "0"}, false},
		{"slash key with interface", map[string]string{"net/ipv4/conf/eth0.100/rp_filter": "1"}, false},
		{"single word key", map[string]string{"mptcp": "0"}, true},
		{"newline injects another line", map[string]string{"net.mptcp.enabled": "0\nkernel.panic = 1"}, true},
		{"empty value", map[string]string{"net.mptcp.enabled": " "}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := validateSysctl(tc.values).HasError(); got != tc.wantErr {
				t.Fatalf("validateSysctl() error = %v, want %v", got, tc.wantErr)
			}
		})
	}
}
