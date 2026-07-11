package provider

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var nodeAttrTypes = map[string]attr.Type{
	"public_ip":            types.StringType,
	"role":                 types.StringType,
	"hostname":             types.StringType,
	"ssh_user":             types.StringType,
	"ssh_port":             types.Int64Type,
	"ssh_password":         types.StringType,
	"ssh_private_key":      types.StringType,
	"ssh_private_key_path": types.StringType,
	"name":                 types.StringType,
	"private_ip":           types.StringType,
	"wireguard_public_key": types.StringType,
	"status":               types.StringType,
}

var nodeObjectType = types.ObjectType{AttrTypes: nodeAttrTypes}

func nodesAttribute() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		Required:            true,
		MarkdownDescription: "Cluster nodes. Stable node identity is hostname when set, otherwise public_ip.",
		NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
			"public_ip":            schema.StringAttribute{Required: true},
			"role":                 schema.StringAttribute{Required: true, MarkdownDescription: "Node role: server or agent."},
			"hostname":             schema.StringAttribute{Optional: true},
			"ssh_user":             schema.StringAttribute{Optional: true},
			"ssh_port":             schema.Int64Attribute{Optional: true},
			"ssh_password":         schema.StringAttribute{Optional: true, Sensitive: true},
			"ssh_private_key":      schema.StringAttribute{Optional: true, Sensitive: true},
			"ssh_private_key_path": schema.StringAttribute{Optional: true},
			"name":                 schema.StringAttribute{Computed: true},
			"private_ip":           schema.StringAttribute{Computed: true},
			"wireguard_public_key": schema.StringAttribute{Computed: true},
			"status":               schema.StringAttribute{Computed: true},
		}},
	}
}

func wireGuardAttribute() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional: true,
		Attributes: map[string]schema.Attribute{
			"enabled":   schema.BoolAttribute{Optional: true},
			"interface": schema.StringAttribute{Optional: true},
			"subnet":    schema.StringAttribute{Optional: true},
			"port":      schema.Int64Attribute{Optional: true},
			"mtu":       schema.Int64Attribute{Optional: true},
		},
	}
}

func k3sAttribute() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional: true,
		Attributes: map[string]schema.Attribute{
			"channel":               schema.StringAttribute{Optional: true},
			"cluster_cidr":          schema.StringAttribute{Optional: true},
			"service_cidr":          schema.StringAttribute{Optional: true},
			"disable_components":    schema.ListAttribute{Optional: true, ElementType: types.StringType},
			"write_kubeconfig_mode": schema.StringAttribute{Optional: true},
			"install_open_iscsi":    schema.BoolAttribute{Optional: true},
			"install_nfs_common":    schema.BoolAttribute{Optional: true},
			"extra_server_args":     schema.ListAttribute{Optional: true, ElementType: types.StringType},
			"extra_agent_args":      schema.ListAttribute{Optional: true, ElementType: types.StringType},
		},
	}
}

func firewallAttribute() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional: true,
		Attributes: map[string]schema.Attribute{
			"enabled":           schema.BoolAttribute{Optional: true},
			"backend":           schema.StringAttribute{Optional: true},
			"ssh_allowed_cidrs": schema.ListAttribute{Optional: true, ElementType: types.StringType},
			"admin_cidrs":       schema.ListAttribute{Optional: true, ElementType: types.StringType},
			"allow_http":        schema.BoolAttribute{Optional: true},
			"allow_https":       schema.BoolAttribute{Optional: true},
			"allow_kube_api":    schema.BoolAttribute{Optional: true},
			"extra_tcp_ports":   schema.ListAttribute{Optional: true, ElementType: types.Int64Type},
			"extra_udp_ports":   schema.ListAttribute{Optional: true, ElementType: types.Int64Type},
		},
	}
}

func addonsAttribute() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional: true,
		Attributes: map[string]schema.Attribute{
			"traefik": schema.SingleNestedAttribute{Optional: true, Attributes: map[string]schema.Attribute{
				"mode":          schema.StringAttribute{Optional: true},
				"namespace":     schema.StringAttribute{Optional: true},
				"chart_version": schema.StringAttribute{Optional: true},
				"values_yaml":   schema.StringAttribute{Optional: true, Sensitive: true},
			}},
			"longhorn": schema.SingleNestedAttribute{Optional: true, Attributes: map[string]schema.Attribute{
				"enabled":       schema.BoolAttribute{Optional: true},
				"namespace":     schema.StringAttribute{Optional: true},
				"chart_version": schema.StringAttribute{Optional: true},
				"storage_classes": schema.MapNestedAttribute{Optional: true, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"name":            schema.StringAttribute{Required: true},
					"number_replicas": schema.Int64Attribute{Required: true},
					"reclaim_policy":  schema.StringAttribute{Required: true},
					"default_class":   schema.BoolAttribute{Optional: true},
				}}},
			}},
		},
	}
}

func adminPeerAttribute() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional: true,
		Attributes: map[string]schema.Attribute{
			"enabled": schema.BoolAttribute{Optional: true},
			"name":    schema.StringAttribute{Optional: true},
			"wg_ip":   schema.StringAttribute{Optional: true},
		},
	}
}

func adminPeersAttribute() schema.MapNestedAttribute {
	return schema.MapNestedAttribute{
		Optional:            true,
		MarkdownDescription: "Additional WireGuard admin peers keyed by peer name. If public_key is provided, no private key is generated or stored for that peer.",
		NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
			"public_key": schema.StringAttribute{Optional: true, MarkdownDescription: "Existing WireGuard public key for this peer. If omitted, the provider generates a keypair and returns a config in admin_wireguard_configs."},
			"wg_ip":      schema.StringAttribute{Optional: true, MarkdownDescription: "Optional fixed WireGuard IP for this peer."},
		}},
	}
}

func validatePlan(ctx context.Context, plan ClusterResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	var nodes []NodeModel
	diags.Append(plan.Nodes.ElementsAs(ctx, &nodes, false)...)
	if diags.HasError() {
		return diags
	}
	if len(nodes) == 0 {
		diags.AddAttributeError(pathRoot("nodes"), "No nodes configured", "At least one node with role server is required.")
		return diags
	}
	servers := 0
	publicIPs := map[string]int{}
	hostnames := map[string]int{}
	names := make([]string, 0, len(nodes))
	for i, n := range nodes {
		identity := nodeIdentity(n, i)
		names = append(names, identity)
		role := n.Role.ValueString()
		if role != "server" && role != "agent" {
			diags.AddAttributeError(pathRoot("nodes"), "Invalid node role", fmt.Sprintf("Node %d role must be server or agent.", i))
		}
		if role == "server" {
			servers++
		}
		ip := n.PublicIP.ValueString()
		if net.ParseIP(ip) == nil {
			diags.AddAttributeError(pathRoot("nodes"), "Invalid public IP", fmt.Sprintf("Node %d public_ip must be a valid IP address.", i))
		}
		if other, ok := publicIPs[ip]; ok {
			diags.AddAttributeError(pathRoot("nodes"), "Duplicate public IP", fmt.Sprintf("Nodes %d and %d use the same public_ip %s.", other, i, ip))
		}
		publicIPs[ip] = i
		h := valueStringDefault(n.Hostname, "")
		if h != "" {
			if !validHostname(h) {
				diags.AddAttributeError(pathRoot("nodes"), "Invalid hostname", fmt.Sprintf("Node %d hostname %q is not valid.", i, h))
			}
			if other, ok := hostnames[h]; ok {
				diags.AddAttributeError(pathRoot("nodes"), "Duplicate hostname", fmt.Sprintf("Nodes %d and %d use hostname %q.", other, i, h))
			}
			hostnames[h] = i
		}
	}
	if servers == 0 {
		diags.AddAttributeError(pathRoot("nodes"), "No server node", "At least one node must have role = \"server\".")
	}
	if servers > 0 && servers%2 == 0 {
		diags.AddAttributeError(pathRoot("nodes"), "Even number of server nodes", fmt.Sprintf("K3s server node count must be odd for embedded etcd quorum. Got %d; use 1, 3, 5, etc.", servers))
	}
	sort.Strings(names)

	wg := expandWireGuard(ctx, plan.WireGuard)
	admin := expandAdminPeer(ctx, plan.AdminPeer)
	adminPeers, adminPeerNames := expandAdminPeers(ctx, plan.AdminPeers)
	if admin.Enabled && !wg.Enabled {
		diags.AddAttributeError(pathRoot("admin_peer"), "admin_peer requires WireGuard", "admin_peer.enabled can be true only when wireguard.enabled is true.")
	}
	if len(adminPeerNames) > 0 && !wg.Enabled {
		diags.AddAttributeError(pathRoot("admin_peers"), "admin_peers requires WireGuard", "admin_peers can be configured only when wireguard.enabled is true.")
	}
	if admin.Enabled && len(adminPeerNames) > 0 {
		diags.AddAttributeError(pathRoot("admin_peers"), "admin_peer conflicts with admin_peers", "Use either the single admin_peer shortcut or the admin_peers map, not both.")
	}
	if wg.Enabled {
		if _, _, err := net.ParseCIDR(wg.Subnet); err != nil {
			diags.AddAttributeError(pathRoot("wireguard"), "Invalid WireGuard subnet", err.Error())
		} else if _, err := assignWireGuardIPs(wg.Subnet, names, adminIPNames(admin, adminPeerNames), adminIPRequests(admin, adminPeers)); err != nil {
			diags.AddAttributeError(pathRoot("wireguard"), "Invalid WireGuard addressing", err.Error())
		}
	}
	addons := expandAddons(ctx, plan.Addons)
	if addons.Traefik.Mode != "default" && addons.Traefik.Mode != "disabled" && addons.Traefik.Mode != "install" {
		diags.AddAttributeError(pathRoot("addons"), "Invalid Traefik mode", "addons.traefik.mode must be one of default, disabled, install.")
	}
	return diags
}

func validateConfig(cfg ClusterConfig) diag.Diagnostics {
	var diags diag.Diagnostics
	if cfg.AdminPeer.Enabled && !cfg.WireGuard.Enabled {
		diags.AddError("Invalid admin_peer", "admin_peer.enabled requires wireguard.enabled = true.")
	}
	if len(cfg.AdminPeerNames) > 0 && !cfg.WireGuard.Enabled {
		diags.AddError("Invalid admin_peers", "admin_peers requires wireguard.enabled = true.")
	}
	if cfg.AdminPeer.Enabled && len(cfg.AdminPeerNames) > 0 {
		diags.AddError("Invalid admin peer configuration", "Use either admin_peer or admin_peers, not both.")
	}
	if cfg.Firewall.Enabled && cfg.Firewall.Backend != "ufw" {
		diags.AddError("Unsupported firewall backend", "Only backend = \"ufw\" is implemented.")
	}
	if cfg.Firewall.Enabled && cfg.Firewall.AllowKubeAPI && len(cfg.Firewall.AdminCIDRs) == 0 {
		diags.AddWarning("Kubernetes API exposed publicly", "firewall.allow_kube_api is true and admin_cidrs is empty, so 6443/tcp will be allowed from 0.0.0.0/0.")
	}
	return diags
}

func adminIPNames(admin AdminPeerConfig, adminPeerNames []string) []string {
	names := make([]string, 0, len(adminPeerNames)+1)
	if admin.Enabled {
		names = append(names, legacyAdminPeerKey)
	}
	names = append(names, adminPeerNames...)
	return names
}

func adminIPRequests(admin AdminPeerConfig, adminPeers map[string]AdminPeerConfig) map[string]string {
	out := map[string]string{}
	if admin.Enabled {
		out[legacyAdminPeerKey] = admin.WGIP
	}
	for name, peer := range adminPeers {
		out[name] = peer.WGIP
	}
	return out
}

var hostnamePattern = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)

func validHostname(v string) bool {
	return len(v) <= 253 && hostnamePattern.MatchString(v)
}
