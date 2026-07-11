package provider

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/curve25519"
)

func generateWireGuardKeyPair() (string, string, error) {
	private := make([]byte, 32)
	if _, err := rand.Read(private); err != nil {
		return "", "", err
	}
	private[0] &= 248
	private[31] &= 127
	private[31] |= 64
	public, err := curve25519.X25519(private, curve25519.Basepoint)
	if err != nil {
		return "", "", err
	}
	return base64.StdEncoding.EncodeToString(private), base64.StdEncoding.EncodeToString(public), nil
}

func publicKeyFromPrivate(privateKey string) (string, error) {
	private, err := base64.StdEncoding.DecodeString(privateKey)
	if err != nil {
		return "", err
	}
	if len(private) != 32 {
		return "", fmt.Errorf("WireGuard private key must be 32 bytes")
	}
	public, err := curve25519.X25519(private, curve25519.Basepoint)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(public), nil
}

func configureWireGuard(client *SSHClient, cfg ClusterConfig, node NodeConfig) error {
	if _, err := client.Run("apt-get update -y && DEBIAN_FRONTEND=noninteractive apt-get install -y wireguard iproute2 iputils-ping"); err != nil {
		return err
	}
	conf := renderNodeWireGuardConfig(cfg, node)
	remote := fmt.Sprintf("/etc/wireguard/%s.conf", cfg.WireGuard.Interface)
	if err := client.Upload(remote, []byte(conf), 0600); err != nil {
		return err
	}
	cmd := fmt.Sprintf("systemctl enable wg-quick@%s && systemctl restart wg-quick@%s", shellQuote(cfg.WireGuard.Interface), shellQuote(cfg.WireGuard.Interface))
	_, err := client.Run(cmd)
	return err
}

func renderNodeWireGuardConfig(cfg ClusterConfig, node NodeConfig) string {
	var b strings.Builder
	fmt.Fprintf(&b, "[Interface]\n")
	fmt.Fprintf(&b, "PrivateKey = %s\n", node.WireGuardPrivateKey)
	fmt.Fprintf(&b, "Address = %s/32\n", node.PrivateIP)
	fmt.Fprintf(&b, "ListenPort = %d\n", cfg.WireGuard.Port)
	fmt.Fprintf(&b, "MTU = %d\n\n", cfg.WireGuard.MTU)
	for _, key := range cfg.NodeNames {
		peer := cfg.Nodes[key]
		if peer.Key == node.Key {
			continue
		}
		fmt.Fprintf(&b, "[Peer]\n")
		fmt.Fprintf(&b, "PublicKey = %s\n", peer.WireGuardPublicKey)
		fmt.Fprintf(&b, "AllowedIPs = %s/32\n", peer.PrivateIP)
		fmt.Fprintf(&b, "Endpoint = %s:%d\n", peer.PublicIP, cfg.WireGuard.Port)
		fmt.Fprintf(&b, "PersistentKeepalive = 25\n\n")
	}
	if cfg.AdminPeer.Enabled {
		fmt.Fprintf(&b, "[Peer]\n")
		fmt.Fprintf(&b, "# %s\n", cfg.AdminPeer.Name)
		fmt.Fprintf(&b, "PublicKey = %s\n", cfg.AdminPeer.PublicKey)
		fmt.Fprintf(&b, "AllowedIPs = %s/32\n", cfg.AdminPeer.WGIP)
		fmt.Fprintf(&b, "PersistentKeepalive = 25\n\n")
	}
	for _, name := range cfg.AdminPeerNames {
		peer := cfg.AdminPeers[name]
		fmt.Fprintf(&b, "[Peer]\n")
		fmt.Fprintf(&b, "# %s\n", peer.Name)
		fmt.Fprintf(&b, "PublicKey = %s\n", peer.PublicKey)
		fmt.Fprintf(&b, "AllowedIPs = %s/32\n", peer.WGIP)
		fmt.Fprintf(&b, "PersistentKeepalive = 25\n\n")
	}
	return b.String()
}

func checkWireGuardConnectivity(client *SSHClient, cfg ClusterConfig, node NodeConfig) error {
	for _, key := range cfg.NodeNames {
		peer := cfg.Nodes[key]
		if peer.Key == node.Key {
			continue
		}
		_, err := client.Run(fmt.Sprintf("ping -c 1 -W 3 %s >/dev/null", shellQuote(peer.PrivateIP)))
		if err != nil {
			return fmt.Errorf("WireGuard connectivity from %s to %s (%s) failed: %w", node.Key, peer.Key, peer.PrivateIP, err)
		}
	}
	return nil
}

func renderAdminWireGuardConfig(cfg ClusterConfig) (string, error) {
	if !cfg.AdminPeer.Enabled {
		return "", nil
	}
	if cfg.AdminPeer.PrivateKey == "" || cfg.AdminPeer.WGIP == "" {
		return "", fmt.Errorf("admin peer keys or IP are missing")
	}
	var b strings.Builder
	fmt.Fprintf(&b, "[Interface]\n")
	fmt.Fprintf(&b, "PrivateKey = %s\n", cfg.AdminPeer.PrivateKey)
	fmt.Fprintf(&b, "Address = %s/32\n", cfg.AdminPeer.WGIP)
	fmt.Fprintf(&b, "MTU = %d\n\n", cfg.WireGuard.MTU)
	for _, key := range cfg.NodeNames {
		n := cfg.Nodes[key]
		fmt.Fprintf(&b, "[Peer]\n")
		fmt.Fprintf(&b, "# %s\n", n.Name)
		fmt.Fprintf(&b, "PublicKey = %s\n", n.WireGuardPublicKey)
		fmt.Fprintf(&b, "AllowedIPs = %s/32\n", n.PrivateIP)
		fmt.Fprintf(&b, "Endpoint = %s:%d\n", n.PublicIP, cfg.WireGuard.Port)
		fmt.Fprintf(&b, "PersistentKeepalive = 25\n\n")
	}
	return b.String(), nil
}

func renderAdminWireGuardConfigs(cfg ClusterConfig) (map[string]string, error) {
	out := map[string]string{}
	for _, name := range cfg.AdminPeerNames {
		peer := cfg.AdminPeers[name]
		if peer.PrivateKey == "" {
			continue
		}
		conf, err := renderAdminPeerConfig(cfg, peer)
		if err != nil {
			return nil, err
		}
		out[name] = conf
	}
	return out, nil
}

func renderAdminPeerConfig(cfg ClusterConfig, peer AdminPeerConfig) (string, error) {
	if peer.PrivateKey == "" || peer.WGIP == "" {
		return "", fmt.Errorf("admin peer %q generated keys or IP are missing", peer.Name)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "[Interface]\n")
	fmt.Fprintf(&b, "PrivateKey = %s\n", peer.PrivateKey)
	fmt.Fprintf(&b, "Address = %s/32\n", peer.WGIP)
	fmt.Fprintf(&b, "MTU = %d\n\n", cfg.WireGuard.MTU)
	for _, key := range cfg.NodeNames {
		n := cfg.Nodes[key]
		fmt.Fprintf(&b, "[Peer]\n")
		fmt.Fprintf(&b, "# %s\n", n.Name)
		fmt.Fprintf(&b, "PublicKey = %s\n", n.WireGuardPublicKey)
		fmt.Fprintf(&b, "AllowedIPs = %s/32\n", n.PrivateIP)
		fmt.Fprintf(&b, "Endpoint = %s:%d\n", n.PublicIP, cfg.WireGuard.Port)
		fmt.Fprintf(&b, "PersistentKeepalive = 25\n\n")
	}
	return b.String(), nil
}
