package provider

import "testing"

func TestWireGuardNeedsRestart(t *testing.T) {
	link := "3: wg0    inet 10.20.0.4/32 scope global wg0\\       valid_lft forever preferred_lft forever\n" +
		"3: wg0: <POINTOPOINT,NOARP,UP,LOWER_UP> mtu 1420 qdisc noqueue state UNKNOWN mode DEFAULT group default qlen 1000\\    link/none"
	cases := []struct {
		name      string
		active    bool
		link      string
		privateIP string
		mtu       int
		want      bool
	}{
		{"running with same address and mtu syncs in place", true, link, "10.20.0.4", 1420, false},
		{"inactive service starts", false, link, "10.20.0.4", 1420, true},
		{"missing interface starts", true, "", "10.20.0.4", 1420, true},
		{"changed address restarts", true, link, "10.20.0.5", 1420, true},
		{"address prefix does not match", true, link, "10.20.0.", 1420, true},
		{"changed mtu restarts", true, link, "10.20.0.4", 1380, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := wireGuardNeedsRestart(tc.active, tc.link, tc.privateIP, tc.mtu); got != tc.want {
				t.Fatalf("wireGuardNeedsRestart() = %v, want %v", got, tc.want)
			}
		})
	}
}
