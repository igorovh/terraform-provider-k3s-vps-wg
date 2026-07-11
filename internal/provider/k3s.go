package provider

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
)

func randomSecret(size int) (string, error) {
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func configureHostnameAndHosts(client *SSHClient, cfg ClusterConfig, node NodeConfig) error {
	if node.Hostname != "" {
		if _, err := client.Run("hostnamectl set-hostname " + shellQuote(node.Hostname)); err != nil {
			return err
		}
	}
	content := renderHostsBlock(cfg)
	cmd := "python3 - <<'PY'\n" +
		"from pathlib import Path\n" +
		"p = Path('/etc/hosts')\n" +
		"text = p.read_text() if p.exists() else ''\n" +
		"start = '# BEGIN vpsk3s hosts'\n" +
		"end = '# END vpsk3s hosts'\n" +
		"block = " + pythonString(content) + "\n" +
		"if start in text and end in text:\n" +
		"    before = text.split(start)[0].rstrip()\n" +
		"    after = text.split(end, 1)[1].lstrip()\n" +
		"    text = before + '\\n' + block + '\\n' + after\n" +
		"else:\n" +
		"    text = text.rstrip() + '\\n' + block + '\\n'\n" +
		"p.write_text(text)\n" +
		"PY"
	_, err := client.RunNoPipefail(cmd)
	return err
}

func renderHostsBlock(cfg ClusterConfig) string {
	var b strings.Builder
	b.WriteString("# BEGIN vpsk3s hosts\n")
	for _, key := range cfg.NodeNames {
		n := cfg.Nodes[key]
		ip := n.PublicIP
		if cfg.WireGuard.Enabled {
			ip = n.PrivateIP
		}
		fmt.Fprintf(&b, "%s %s %s\n", ip, n.Name, n.Key)
	}
	b.WriteString("# END vpsk3s hosts")
	return b.String()
}

func installK3s(client *SSHClient, cfg ClusterConfig, node NodeConfig) error {
	packages := []string{"curl", "ca-certificates"}
	if cfg.K3s.InstallOpenISCSI {
		packages = append(packages, "open-iscsi")
	}
	if cfg.K3s.InstallNFSCommon {
		packages = append(packages, "nfs-common")
	}
	if _, err := client.Run("apt-get update -y && DEBIAN_FRONTEND=noninteractive apt-get install -y " + strings.Join(packages, " ")); err != nil {
		return err
	}
	config := renderK3sConfig(cfg, node)
	if err := client.Upload("/etc/rancher/k3s/config.yaml", []byte(config), 0644); err != nil {
		return err
	}
	service := "k3s"
	installArg := "server"
	if node.Role == "agent" {
		service = "k3s-agent"
		installArg = "agent"
	}
	active, _ := client.RunNoPipefail("systemctl is-active " + shellQuote(service) + " 2>/dev/null || true")
	if strings.TrimSpace(active) == "active" {
		_, err := client.Run("systemctl restart " + shellQuote(service))
		return err
	}
	cmd := fmt.Sprintf("curl -sfL https://get.k3s.io | INSTALL_K3S_CHANNEL=%s sh -s - %s", shellQuote(cfg.K3s.Channel), shellQuote(installArg))
	if _, err := client.Run(cmd); err != nil {
		return err
	}
	_, err := client.Run("for i in $(seq 1 60); do systemctl is-active " + shellQuote(service) + " >/dev/null 2>&1 && exit 0; sleep 5; done; systemctl status " + shellQuote(service) + " --no-pager; exit 1")
	return err
}

func renderK3sConfig(cfg ClusterConfig, node NodeConfig) string {
	first, _ := firstServer(cfg)
	serverURL, _ := serverURLFor(cfg)
	var b strings.Builder
	if node.Role == "server" && node.Key == first.Key {
		b.WriteString("cluster-init: true\n")
	} else {
		fmt.Fprintf(&b, "server: %q\n", serverURL)
	}
	fmt.Fprintf(&b, "token: %q\n", cfg.K3sToken)
	fmt.Fprintf(&b, "node-name: %q\n", node.Name)
	if cfg.WireGuard.Enabled {
		fmt.Fprintf(&b, "node-ip: %q\n", node.PrivateIP)
		if node.Role == "server" {
			fmt.Fprintf(&b, "advertise-address: %q\n", node.PrivateIP)
		}
		fmt.Fprintf(&b, "flannel-iface: %q\n", cfg.WireGuard.Interface)
	} else {
		fmt.Fprintf(&b, "node-ip: %q\n", node.PublicIP)
		if node.Role == "server" {
			fmt.Fprintf(&b, "advertise-address: %q\n", node.PublicIP)
		}
	}
	if node.Role == "server" {
		fmt.Fprintf(&b, "cluster-cidr: %q\n", cfg.K3s.ClusterCIDR)
		fmt.Fprintf(&b, "service-cidr: %q\n", cfg.K3s.ServiceCIDR)
		fmt.Fprintf(&b, "write-kubeconfig-mode: %q\n", cfg.K3s.WriteKubeconfigMode)
		if len(cfg.K3s.DisableComponents) > 0 {
			b.WriteString("disable:\n")
			for _, component := range cfg.K3s.DisableComponents {
				fmt.Fprintf(&b, "  - %q\n", component)
			}
		}
		b.WriteString("tls-san:\n")
		if cfg.WireGuard.Enabled {
			fmt.Fprintf(&b, "  - %q\n", node.PrivateIP)
		} else {
			fmt.Fprintf(&b, "  - %q\n", node.PublicIP)
		}
		if node.Hostname != "" {
			fmt.Fprintf(&b, "  - %q\n", node.Hostname)
		}
		for _, arg := range cfg.K3s.ExtraServerArgs {
			appendK3sArg(&b, arg)
		}
	} else {
		for _, arg := range cfg.K3s.ExtraAgentArgs {
			appendK3sArg(&b, arg)
		}
	}
	return b.String()
}

func appendK3sArg(b *strings.Builder, arg string) {
	arg = strings.TrimSpace(strings.TrimPrefix(arg, "--"))
	if arg == "" {
		return
	}
	parts := strings.SplitN(arg, "=", 2)
	if len(parts) == 2 {
		fmt.Fprintf(b, "%s: %q\n", parts[0], parts[1])
	} else {
		fmt.Fprintf(b, "%s: true\n", parts[0])
	}
}

func fetchKubeconfig(client *SSHClient, cfg ClusterConfig) (string, error) {
	data, err := client.Download("/etc/rancher/k3s/k3s.yaml")
	if err != nil {
		return "", err
	}
	serverURL, err := serverURLFor(cfg)
	if err != nil {
		return "", err
	}
	out := strings.ReplaceAll(string(data), "https://127.0.0.1:6443", serverURL)
	out = strings.ReplaceAll(out, "https://localhost:6443", serverURL)
	return out, nil
}

func pythonString(s string) string {
	return "'''" + strings.ReplaceAll(s, "'''", "'''\"'\"'''") + "'''"
}
