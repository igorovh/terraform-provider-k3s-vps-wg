package provider

import "testing"

func TestAssignWireGuardIPsStable(t *testing.T) {
	assignments, err := assignWireGuardIPs("10.10.0.0/24", []string{"vps1", "vps2", "vps3"}, []string{"admin"}, map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if assignments.NodeIPs["vps1"] != "10.10.0.1" || assignments.NodeIPs["vps2"] != "10.10.0.2" || assignments.NodeIPs["vps3"] != "10.10.0.3" {
		t.Fatalf("unexpected node IPs: %#v", assignments.NodeIPs)
	}
	if assignments.AdminIPs["admin"] != "10.10.0.254" {
		t.Fatalf("unexpected admin IP: %s", assignments.AdminIPs["admin"])
	}
}

func TestAssignWireGuardIPsTooSmall(t *testing.T) {
	_, err := assignWireGuardIPs("10.10.0.0/30", []string{"a", "b"}, []string{"admin"}, map[string]string{})
	if err == nil {
		t.Fatal("expected subnet capacity error")
	}
}
