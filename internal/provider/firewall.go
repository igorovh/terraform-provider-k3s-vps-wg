package provider

import (
	"fmt"
	"strings"
)

func configureFirewall(client *SSHClient, cfg ClusterConfig, node NodeConfig) error {
	if cfg.Firewall.Backend != "ufw" {
		return fmt.Errorf("unsupported firewall backend %q", cfg.Firewall.Backend)
	}
	commands := []string{
		"apt-get update -y && DEBIAN_FRONTEND=noninteractive apt-get install -y ufw",
		"ufw --force reset",
		"ufw default deny incoming",
		"ufw default allow outgoing",
	}
	for _, cidr := range cfg.Firewall.SSHAllowedCIDRs {
		commands = append(commands, fmt.Sprintf("ufw allow from %s to any port %d proto tcp", shellQuote(cidr), node.SSH.Port))
	}
	if cfg.WireGuard.Enabled {
		for _, key := range cfg.NodeNames {
			peer := cfg.Nodes[key]
			if peer.Key == node.Key {
				continue
			}
			commands = append(commands, fmt.Sprintf("ufw allow from %s to any port %d proto udp", shellQuote(peer.PublicIP), cfg.WireGuard.Port))
		}
		commands = append(commands, fmt.Sprintf("ufw allow in on %s", shellQuote(cfg.WireGuard.Interface)))
	}
	if cfg.Firewall.AllowHTTP {
		commands = append(commands, "ufw allow 80/tcp")
	}
	if cfg.Firewall.AllowHTTPS {
		commands = append(commands, "ufw allow 443/tcp")
	}
	if cfg.Firewall.AllowKubeAPI {
		if len(cfg.Firewall.AdminCIDRs) == 0 {
			commands = append(commands, "ufw allow 6443/tcp")
		} else {
			for _, cidr := range cfg.Firewall.AdminCIDRs {
				commands = append(commands, fmt.Sprintf("ufw allow from %s to any port 6443 proto tcp", shellQuote(cidr)))
			}
		}
	}
	for _, port := range cfg.Firewall.ExtraTCPPorts {
		commands = append(commands, fmt.Sprintf("ufw allow %d/tcp", port))
	}
	for _, port := range cfg.Firewall.ExtraUDPPorts {
		commands = append(commands, fmt.Sprintf("ufw allow %d/udp", port))
	}
	commands = append(commands, "ufw --force enable")
	_, err := client.Run(strings.Join(commands, " && "))
	return err
}
