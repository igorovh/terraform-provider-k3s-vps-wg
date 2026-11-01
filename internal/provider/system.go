package provider

import (
	"fmt"
	"net"
	"sort"
	"strings"
)

const (
	k3sResolvConfPath = "/etc/vpsk3s/resolv.conf"
	sysctlConfPath    = "/etc/sysctl.d/90-vpsk3s.conf"
	maxNameservers    = 3
)

func configureNodeSystem(client *SSHClient, cfg ClusterConfig) error {
	if err := configureSysctl(client, cfg.Sysctl); err != nil {
		return err
	}
	if !cfg.DNS.Enabled {
		return nil
	}
	return configureK3sResolvConf(client, cfg.DNS)
}

func configureSysctl(client *SSHClient, values map[string]string) error {
	if len(values) == 0 {
		_, err := client.Run("rm -f " + shellQuote(sysctlConfPath))
		return err
	}
	content := []byte(renderSysctlConf(values))
	if !client.FileEquals(sysctlConfPath, content) {
		if err := client.Upload(sysctlConfPath, content, 0644); err != nil {
			return err
		}
	}
	// Applying on every run also restores values that changed at runtime.
	_, err := client.Run("sysctl -q -p " + shellQuote(sysctlConfPath))
	return err
}

func renderSysctlConf(values map[string]string) string {
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString("# Managed by terraform-provider-k3s-vps-wg.\n")
	for _, k := range keys {
		fmt.Fprintf(&b, "%s = %s\n", k, strings.TrimSpace(values[k]))
	}
	return b.String()
}

func configureK3sResolvConf(client *SSHClient, dns DNSConfig) error {
	nameservers := dns.Nameservers
	if len(nameservers) == 0 {
		// systemd-resolved keeps the real upstream servers here; /etc/resolv.conf may only list the 127.0.0.53 stub.
		out, err := client.RunNoPipefail("for f in /run/systemd/resolve/resolv.conf /etc/resolv.conf; do if [ -r \"$f\" ]; then cat \"$f\"; exit 0; fi; done")
		if err != nil {
			return err
		}
		nameservers = upstreamNameservers(out)
		if len(nameservers) == 0 {
			return fmt.Errorf("no non-loopback nameserver found on the node; set dns.nameservers")
		}
	}
	content := []byte(renderResolvConf(nameservers, dns.SearchDomains))
	if client.FileEquals(k3sResolvConfPath, content) {
		return nil
	}
	// Kubelet reads this file when it creates a pod, so K3s does not need a restart.
	return client.Upload(k3sResolvConfPath, content, 0644)
}

func upstreamNameservers(resolvConf string) []string {
	var out []string
	for _, line := range strings.Split(resolvConf, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] != "nameserver" {
			continue
		}
		ip := net.ParseIP(fields[1])
		if ip == nil || ip.IsLoopback() || ip.IsUnspecified() {
			continue
		}
		out = append(out, ip.String())
		if len(out) == maxNameservers {
			break
		}
	}
	return out
}

func renderResolvConf(nameservers, searchDomains []string) string {
	var b strings.Builder
	b.WriteString("# Managed by terraform-provider-k3s-vps-wg; K3s uses this instead of the host resolv.conf.\n")
	for _, ns := range nameservers {
		fmt.Fprintf(&b, "nameserver %s\n", ns)
	}
	if len(searchDomains) > 0 {
		fmt.Fprintf(&b, "search %s\n", strings.Join(searchDomains, " "))
	}
	return b.String()
}
