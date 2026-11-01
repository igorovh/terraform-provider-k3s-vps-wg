package provider

import (
	"strings"
	"testing"
)

func TestResolveK3sTokenPrefersConfiguredToken(t *testing.T) {
	if got := resolveK3sToken("rotated", "leaked"); got != "rotated" {
		t.Fatalf("configured token must replace the state token, got %q", got)
	}
	if got := resolveK3sToken("", "leaked"); got != "leaked" {
		t.Fatalf("without a configured token the state token must stay, got %q", got)
	}
}

func TestK3sTokenRotates(t *testing.T) {
	cases := []struct {
		configured, prior string
		want              bool
	}{
		{"new", "old", true},
		{"same", "same", false},
		{"", "old", false},
		{"new", "", false},
	}
	for _, c := range cases {
		if got := k3sTokenRotates(c.configured, c.prior); got != c.want {
			t.Errorf("k3sTokenRotates(%q, %q) = %v, want %v", c.configured, c.prior, got, c.want)
		}
	}
}

func TestRenderK3sConfigWritesSameTokenForServersAndAgents(t *testing.T) {
	cfg := ClusterConfig{
		NodeNames: []string{"s1", "a1"},
		Nodes: map[string]NodeConfig{
			"s1": {Key: "s1", Name: "s1", Role: "server", PublicIP: "203.0.113.10"},
			"a1": {Key: "a1", Name: "a1", Role: "agent", PublicIP: "203.0.113.11"},
		},
		K3sToken: resolveK3sToken("rotated", "leaked"),
	}
	for _, key := range cfg.NodeNames {
		out := renderK3sConfig(cfg, cfg.Nodes[key])
		if !strings.Contains(out, "token: \"rotated\"\n") || strings.Contains(out, "leaked") {
			t.Fatalf("node %s config has the wrong token:\n%s", key, out)
		}
	}
}
