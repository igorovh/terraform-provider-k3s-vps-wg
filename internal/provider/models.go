package provider

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type ProviderModel struct {
	SSHUser               types.String `tfsdk:"ssh_user"`
	SSHPort               types.Int64  `tfsdk:"ssh_port"`
	SSHPassword           types.String `tfsdk:"ssh_password"`
	SSHPrivateKey         types.String `tfsdk:"ssh_private_key"`
	SSHPrivateKeyPath     types.String `tfsdk:"ssh_private_key_path"`
	SSHTimeout            types.String `tfsdk:"ssh_timeout"`
	InsecureIgnoreHostKey types.Bool   `tfsdk:"insecure_ignore_host_key"`
}

type ProviderConfig struct {
	SSHUser               string
	SSHPort               int
	SSHPassword           string
	SSHPrivateKey         string
	SSHPrivateKeyPath     string
	SSHTimeout            time.Duration
	InsecureIgnoreHostKey bool
}

type ClusterResourceModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	Nodes      types.List   `tfsdk:"nodes"`
	WireGuard  types.Object `tfsdk:"wireguard"`
	K3s        types.Object `tfsdk:"k3s"`
	Firewall   types.Object `tfsdk:"firewall"`
	Addons     types.Object `tfsdk:"addons"`
	AdminPeer  types.Object `tfsdk:"admin_peer"`
	AdminPeers types.Map    `tfsdk:"admin_peers"`

	Kubeconfig                types.String `tfsdk:"kubeconfig"`
	ServerURL                 types.String `tfsdk:"server_url"`
	AdminWireGuardConfig      types.String `tfsdk:"admin_wireguard_config"`
	AdminWireGuardConfigs     types.Map    `tfsdk:"admin_wireguard_configs"`
	WireGuardEnabled          types.Bool   `tfsdk:"wireguard_enabled"`
	FirewallEnabled           types.Bool   `tfsdk:"firewall_enabled"`
	AddonsStatus              types.Map    `tfsdk:"addons_status"`
	K3sToken                  types.String `tfsdk:"k3s_token"`
	WireGuardPrivateKeys      types.Map    `tfsdk:"wireguard_private_keys"`
	AdminWireGuardPrivate     types.String `tfsdk:"admin_wireguard_private_key"`
	AdminWireGuardPublic      types.String `tfsdk:"admin_wireguard_public_key"`
	AdminWireGuardPrivateKeys types.Map    `tfsdk:"admin_wireguard_private_keys"`
	AdminWireGuardPublicKeys  types.Map    `tfsdk:"admin_wireguard_public_keys"`
}

type NodeModel struct {
	PublicIP           types.String `tfsdk:"public_ip"`
	Role               types.String `tfsdk:"role"`
	Hostname           types.String `tfsdk:"hostname"`
	SSHUser            types.String `tfsdk:"ssh_user"`
	SSHPort            types.Int64  `tfsdk:"ssh_port"`
	SSHPassword        types.String `tfsdk:"ssh_password"`
	SSHPrivateKey      types.String `tfsdk:"ssh_private_key"`
	SSHPrivateKeyPath  types.String `tfsdk:"ssh_private_key_path"`
	Name               types.String `tfsdk:"name"`
	PrivateIP          types.String `tfsdk:"private_ip"`
	WireGuardPublicKey types.String `tfsdk:"wireguard_public_key"`
	Status             types.String `tfsdk:"status"`
}

type WireGuardModel struct {
	Enabled   types.Bool   `tfsdk:"enabled"`
	Interface types.String `tfsdk:"interface"`
	Subnet    types.String `tfsdk:"subnet"`
	Port      types.Int64  `tfsdk:"port"`
	MTU       types.Int64  `tfsdk:"mtu"`
}

type K3sModel struct {
	Channel             types.String `tfsdk:"channel"`
	ClusterCIDR         types.String `tfsdk:"cluster_cidr"`
	ServiceCIDR         types.String `tfsdk:"service_cidr"`
	DisableComponents   types.List   `tfsdk:"disable_components"`
	WriteKubeconfigMode types.String `tfsdk:"write_kubeconfig_mode"`
	InstallOpenISCSI    types.Bool   `tfsdk:"install_open_iscsi"`
	InstallNFSCommon    types.Bool   `tfsdk:"install_nfs_common"`
	ExtraServerArgs     types.List   `tfsdk:"extra_server_args"`
	ExtraAgentArgs      types.List   `tfsdk:"extra_agent_args"`
}

type FirewallModel struct {
	Enabled         types.Bool   `tfsdk:"enabled"`
	Backend         types.String `tfsdk:"backend"`
	SSHAllowedCIDRs types.List   `tfsdk:"ssh_allowed_cidrs"`
	AdminCIDRs      types.List   `tfsdk:"admin_cidrs"`
	AllowHTTP       types.Bool   `tfsdk:"allow_http"`
	AllowHTTPS      types.Bool   `tfsdk:"allow_https"`
	AllowKubeAPI    types.Bool   `tfsdk:"allow_kube_api"`
	ExtraTCPPorts   types.List   `tfsdk:"extra_tcp_ports"`
	ExtraUDPPorts   types.List   `tfsdk:"extra_udp_ports"`
}

type AddonsModel struct {
	Traefik  types.Object `tfsdk:"traefik"`
	Longhorn types.Object `tfsdk:"longhorn"`
}

type TraefikModel struct {
	Mode         types.String `tfsdk:"mode"`
	Namespace    types.String `tfsdk:"namespace"`
	ChartVersion types.String `tfsdk:"chart_version"`
	ValuesYAML   types.String `tfsdk:"values_yaml"`
}

type LonghornModel struct {
	Enabled        types.Bool   `tfsdk:"enabled"`
	Namespace      types.String `tfsdk:"namespace"`
	ChartVersion   types.String `tfsdk:"chart_version"`
	StorageClasses types.Map    `tfsdk:"storage_classes"`
}

type StorageClassModel struct {
	Name           types.String `tfsdk:"name"`
	NumberReplicas types.Int64  `tfsdk:"number_replicas"`
	ReclaimPolicy  types.String `tfsdk:"reclaim_policy"`
	DefaultClass   types.Bool   `tfsdk:"default_class"`
}

type AdminPeerModel struct {
	Enabled types.Bool   `tfsdk:"enabled"`
	Name    types.String `tfsdk:"name"`
	WGIP    types.String `tfsdk:"wg_ip"`
}

type AdminPeerMapModel struct {
	PublicKey types.String `tfsdk:"public_key"`
	WGIP      types.String `tfsdk:"wg_ip"`
}

type ClusterConfig struct {
	Name               string
	Nodes              map[string]NodeConfig
	NodeNames          []string
	WireGuard          WireGuardConfig
	K3s                K3sConfig
	Firewall           FirewallConfig
	Addons             AddonsConfig
	AdminPeer          AdminPeerConfig
	AdminPeers         map[string]AdminPeerConfig
	AdminPeerNames     []string
	K3sToken           string
	WGPrivateKeys      map[string]string
	AdminWGPriv        string
	AdminWGPub         string
	AdminWGPrivateKeys map[string]string
	AdminWGPublicKeys  map[string]string
}

type NodeConfig struct {
	Key                 string
	Name                string
	PublicIP            string
	PrivateIP           string
	Role                string
	Hostname            string
	WireGuardPrivateKey string
	WireGuardPublicKey  string
	SSH                 SSHConfig
}

type WireGuardConfig struct {
	Enabled   bool
	Interface string
	Subnet    string
	Port      int
	MTU       int
}

type K3sConfig struct {
	Channel             string
	ClusterCIDR         string
	ServiceCIDR         string
	DisableComponents   []string
	WriteKubeconfigMode string
	InstallOpenISCSI    bool
	InstallNFSCommon    bool
	ExtraServerArgs     []string
	ExtraAgentArgs      []string
}

type FirewallConfig struct {
	Enabled         bool
	Backend         string
	SSHAllowedCIDRs []string
	AdminCIDRs      []string
	AllowHTTP       bool
	AllowHTTPS      bool
	AllowKubeAPI    bool
	ExtraTCPPorts   []int
	ExtraUDPPorts   []int
}

type AddonsConfig struct {
	Traefik  TraefikConfig
	Longhorn LonghornConfig
}

type TraefikConfig struct {
	Mode         string
	Namespace    string
	ChartVersion string
	ValuesYAML   string
}

type LonghornConfig struct {
	Enabled        bool
	Namespace      string
	ChartVersion   string
	StorageClasses map[string]StorageClassConfig
}

type StorageClassConfig struct {
	Name           string
	NumberReplicas int
	ReclaimPolicy  string
	DefaultClass   bool
}

type AdminPeerConfig struct {
	Enabled         bool
	Name            string
	WGIP            string
	PrivateKey      string
	PublicKey       string
	WireGuardConfig string
}

const legacyAdminPeerKey = "__admin_peer"

type SSHConfig struct {
	User           string
	Port           int
	Password       string
	PrivateKey     string
	PrivateKeyPath string
	Timeout        time.Duration
	Insecure       bool
}

func expandProviderConfig(ctx context.Context, model ProviderModel) (ProviderConfig, diag.Diagnostics) {
	var diags diag.Diagnostics
	timeout := "5m"
	if !model.SSHTimeout.IsNull() && !model.SSHTimeout.IsUnknown() {
		timeout = model.SSHTimeout.ValueString()
	}
	d, err := time.ParseDuration(timeout)
	if err != nil {
		diags.AddAttributeError(pathRoot("ssh_timeout"), "Invalid SSH timeout", err.Error())
		d = 5 * time.Minute
	}
	return ProviderConfig{
		SSHUser:               valueStringDefault(model.SSHUser, "root"),
		SSHPort:               valueIntDefault(model.SSHPort, 22),
		SSHPassword:           valueStringDefault(model.SSHPassword, ""),
		SSHPrivateKey:         valueStringDefault(model.SSHPrivateKey, ""),
		SSHPrivateKeyPath:     valueStringDefault(model.SSHPrivateKeyPath, ""),
		SSHTimeout:            d,
		InsecureIgnoreHostKey: valueBoolDefault(model.InsecureIgnoreHostKey, true),
	}, diags
}

func expandClusterConfig(ctx context.Context, plan ClusterResourceModel, prior ClusterResourceModel, provider ProviderConfig) (ClusterConfig, diag.Diagnostics) {
	var diags diag.Diagnostics
	cfg := ClusterConfig{
		Name:               plan.Name.ValueString(),
		Nodes:              map[string]NodeConfig{},
		AdminPeers:         map[string]AdminPeerConfig{},
		WGPrivateKeys:      map[string]string{},
		AdminWGPrivateKeys: map[string]string{},
		AdminWGPublicKeys:  map[string]string{},
	}

	cfg.WireGuard = expandWireGuard(ctx, plan.WireGuard)
	cfg.K3s = expandK3s(ctx, plan.K3s)
	cfg.Firewall = expandFirewall(ctx, plan.Firewall)
	cfg.Addons = expandAddons(ctx, plan.Addons)
	cfg.AdminPeer = expandAdminPeer(ctx, plan.AdminPeer)
	cfg.AdminPeers, cfg.AdminPeerNames = expandAdminPeers(ctx, plan.AdminPeers)

	priorKeys := map[string]string{}
	if !prior.WireGuardPrivateKeys.IsNull() && !prior.WireGuardPrivateKeys.IsUnknown() {
		diags.Append(prior.WireGuardPrivateKeys.ElementsAs(ctx, &priorKeys, false)...)
	}
	cfg.K3sToken = valueStringDefault(prior.K3sToken, "")
	if cfg.K3sToken == "" {
		token, err := randomSecret(32)
		if err != nil {
			diags.AddError("Failed to generate K3s token", err.Error())
		} else {
			cfg.K3sToken = token
		}
	}
	cfg.AdminWGPriv = valueStringDefault(prior.AdminWireGuardPrivate, "")
	cfg.AdminWGPub = valueStringDefault(prior.AdminWireGuardPublic, "")
	priorAdminPrivateKeys := map[string]string{}
	if !prior.AdminWireGuardPrivateKeys.IsNull() && !prior.AdminWireGuardPrivateKeys.IsUnknown() {
		diags.Append(prior.AdminWireGuardPrivateKeys.ElementsAs(ctx, &priorAdminPrivateKeys, false)...)
	}

	var nodeModels []NodeModel
	diags.Append(plan.Nodes.ElementsAs(ctx, &nodeModels, false)...)
	if diags.HasError() {
		return cfg, diags
	}

	cfg.NodeNames = make([]string, 0, len(nodeModels))
	for i, n := range nodeModels {
		hostname := valueStringDefault(n.Hostname, "")
		key := nodeIdentity(n, i)
		name := fmt.Sprintf("node-%d", i+1)
		if hostname != "" {
			name = hostname
		}
		sshCfg := SSHConfig{
			User:           valueStringDefault(n.SSHUser, provider.SSHUser),
			Port:           valueIntDefault(n.SSHPort, provider.SSHPort),
			Password:       valueStringDefault(n.SSHPassword, provider.SSHPassword),
			PrivateKey:     valueStringDefault(n.SSHPrivateKey, provider.SSHPrivateKey),
			PrivateKeyPath: valueStringDefault(n.SSHPrivateKeyPath, provider.SSHPrivateKeyPath),
			Timeout:        provider.SSHTimeout,
			Insecure:       provider.InsecureIgnoreHostKey,
		}
		cfg.Nodes[key] = NodeConfig{
			Key:      key,
			Name:     name,
			PublicIP: n.PublicIP.ValueString(),
			Role:     n.Role.ValueString(),
			Hostname: hostname,
			SSH:      sshCfg,
		}
		cfg.NodeNames = append(cfg.NodeNames, key)
	}
	sort.Strings(cfg.NodeNames)

	if cfg.WireGuard.Enabled {
		adminNames, requestedAdminIPs := cfg.adminPeerIPRequests()
		assigned, err := assignWireGuardIPs(cfg.WireGuard.Subnet, cfg.NodeNames, adminNames, requestedAdminIPs)
		if err != nil {
			diags.AddError("Failed to assign WireGuard IPs", err.Error())
			return cfg, diags
		}
		for _, key := range cfg.NodeNames {
			n := cfg.Nodes[key]
			n.PrivateIP = assigned.NodeIPs[key]
			priv := priorKeys[key]
			pub := ""
			if priv == "" {
				var err error
				priv, pub, err = generateWireGuardKeyPair()
				if err != nil {
					diags.AddError("Failed to generate WireGuard key", err.Error())
					return cfg, diags
				}
			} else {
				var err error
				pub, err = publicKeyFromPrivate(priv)
				if err != nil {
					diags.AddError("Invalid stored WireGuard private key", err.Error())
					return cfg, diags
				}
			}
			n.WireGuardPrivateKey = priv
			n.WireGuardPublicKey = pub
			cfg.WGPrivateKeys[key] = priv
			cfg.Nodes[key] = n
		}
		if cfg.AdminPeer.Enabled {
			cfg.AdminPeer.WGIP = assigned.AdminIPs[legacyAdminPeerKey]
			if cfg.AdminWGPriv == "" {
				priv, pub, err := generateWireGuardKeyPair()
				if err != nil {
					diags.AddError("Failed to generate admin WireGuard key", err.Error())
					return cfg, diags
				}
				cfg.AdminWGPriv, cfg.AdminWGPub = priv, pub
			}
			cfg.AdminPeer.PrivateKey = cfg.AdminWGPriv
			cfg.AdminPeer.PublicKey = cfg.AdminWGPub
		}
		for _, name := range cfg.AdminPeerNames {
			peer := cfg.AdminPeers[name]
			peer.WGIP = assigned.AdminIPs[name]
			if peer.PublicKey == "" {
				priv := priorAdminPrivateKeys[name]
				pub := ""
				if priv == "" {
					var err error
					priv, pub, err = generateWireGuardKeyPair()
					if err != nil {
						diags.AddError("Failed to generate admin WireGuard key", err.Error())
						return cfg, diags
					}
				} else {
					var err error
					pub, err = publicKeyFromPrivate(priv)
					if err != nil {
						diags.AddError("Invalid stored admin WireGuard private key", err.Error())
						return cfg, diags
					}
				}
				peer.PrivateKey = priv
				peer.PublicKey = pub
				cfg.AdminWGPrivateKeys[name] = priv
			} else {
				peer.PrivateKey = ""
			}
			cfg.AdminWGPublicKeys[name] = peer.PublicKey
			cfg.AdminPeers[name] = peer
		}
	}

	cfg.K3s.DisableComponents = mergedDisableComponents(cfg.K3s.DisableComponents, cfg.Addons.Traefik.Mode)
	return cfg, diags
}

func clusterToState(ctx context.Context, cfg ClusterConfig, plan ClusterResourceModel, kubeconfig, serverURL, adminConfig string, adminConfigs map[string]string, statuses map[string]string) (ClusterResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	state := plan
	state.ID = types.StringValue(cfg.Name)
	state.Kubeconfig = types.StringValue(kubeconfig)
	state.ServerURL = types.StringValue(serverURL)
	state.AdminWireGuardConfig = nullOrString(adminConfig)
	state.WireGuardEnabled = types.BoolValue(cfg.WireGuard.Enabled)
	state.FirewallEnabled = types.BoolValue(cfg.Firewall.Enabled)
	state.K3sToken = types.StringValue(cfg.K3sToken)
	state.AdminWireGuardPrivate = nullOrString(cfg.AdminWGPriv)
	state.AdminWireGuardPublic = nullOrString(cfg.AdminWGPub)

	adminConfigMap, d := types.MapValueFrom(ctx, types.StringType, adminConfigs)
	diags.Append(d...)
	state.AdminWireGuardConfigs = adminConfigMap
	adminPrivMap, d := types.MapValueFrom(ctx, types.StringType, cfg.AdminWGPrivateKeys)
	diags.Append(d...)
	state.AdminWireGuardPrivateKeys = adminPrivMap
	adminPubMap, d := types.MapValueFrom(ctx, types.StringType, cfg.AdminWGPublicKeys)
	diags.Append(d...)
	state.AdminWireGuardPublicKeys = adminPubMap

	wgPriv, d := types.MapValueFrom(ctx, types.StringType, cfg.WGPrivateKeys)
	diags.Append(d...)
	state.WireGuardPrivateKeys = wgPriv

	addonStatus := map[string]string{"traefik": cfg.Addons.Traefik.Mode, "longhorn": boolStatus(cfg.Addons.Longhorn.Enabled)}
	addonsMap, d := types.MapValueFrom(ctx, types.StringType, addonStatus)
	diags.Append(d...)
	state.AddonsStatus = addonsMap

	nodeState := make([]NodeModel, 0, len(cfg.NodeNames))
	planNodes := map[string]NodeModel{}
	if !plan.Nodes.IsNull() && !plan.Nodes.IsUnknown() {
		var inputNodes []NodeModel
		diags.Append(plan.Nodes.ElementsAs(ctx, &inputNodes, false)...)
		for i, n := range inputNodes {
			planNodes[nodeIdentity(n, i)] = n
		}
	}
	for _, key := range cfg.NodeNames {
		n := cfg.Nodes[key]
		input := planNodes[key]
		nodeState = append(nodeState, NodeModel{
			PublicIP:           types.StringValue(n.PublicIP),
			Role:               types.StringValue(n.Role),
			Hostname:           input.Hostname,
			SSHUser:            input.SSHUser,
			SSHPort:            input.SSHPort,
			SSHPassword:        input.SSHPassword,
			SSHPrivateKey:      input.SSHPrivateKey,
			SSHPrivateKeyPath:  input.SSHPrivateKeyPath,
			Name:               types.StringValue(n.Name),
			PrivateIP:          nullOrString(n.PrivateIP),
			WireGuardPublicKey: nullOrString(n.WireGuardPublicKey),
			Status:             types.StringValue(statuses[key]),
		})
	}
	nodesMap, d := types.ListValueFrom(ctx, nodeObjectType, nodeState)
	diags.Append(d...)
	state.Nodes = nodesMap
	return state, diags
}

func expandWireGuard(ctx context.Context, obj types.Object) WireGuardConfig {
	out := WireGuardConfig{Enabled: true, Interface: "wg0", Subnet: "10.10.0.0/24", Port: 51820, MTU: 1420}
	if obj.IsNull() || obj.IsUnknown() {
		return out
	}
	var m WireGuardModel
	obj.As(ctx, &m, basetypes.ObjectAsOptions{})
	out.Enabled = valueBoolDefault(m.Enabled, out.Enabled)
	out.Interface = valueStringDefault(m.Interface, out.Interface)
	out.Subnet = valueStringDefault(m.Subnet, out.Subnet)
	out.Port = valueIntDefault(m.Port, out.Port)
	out.MTU = valueIntDefault(m.MTU, out.MTU)
	return out
}

func expandK3s(ctx context.Context, obj types.Object) K3sConfig {
	out := K3sConfig{Channel: "stable", ClusterCIDR: "10.42.0.0/16", ServiceCIDR: "10.43.0.0/16", WriteKubeconfigMode: "0644", InstallOpenISCSI: true, InstallNFSCommon: true}
	if obj.IsNull() || obj.IsUnknown() {
		return out
	}
	var m K3sModel
	obj.As(ctx, &m, basetypes.ObjectAsOptions{})
	out.Channel = valueStringDefault(m.Channel, out.Channel)
	out.ClusterCIDR = valueStringDefault(m.ClusterCIDR, out.ClusterCIDR)
	out.ServiceCIDR = valueStringDefault(m.ServiceCIDR, out.ServiceCIDR)
	out.WriteKubeconfigMode = valueStringDefault(m.WriteKubeconfigMode, out.WriteKubeconfigMode)
	out.InstallOpenISCSI = valueBoolDefault(m.InstallOpenISCSI, out.InstallOpenISCSI)
	out.InstallNFSCommon = valueBoolDefault(m.InstallNFSCommon, out.InstallNFSCommon)
	out.DisableComponents = stringList(ctx, m.DisableComponents)
	out.ExtraServerArgs = stringList(ctx, m.ExtraServerArgs)
	out.ExtraAgentArgs = stringList(ctx, m.ExtraAgentArgs)
	return out
}

func expandFirewall(ctx context.Context, obj types.Object) FirewallConfig {
	out := FirewallConfig{Enabled: false, Backend: "ufw", SSHAllowedCIDRs: []string{"0.0.0.0/0"}, AllowHTTP: true, AllowHTTPS: true}
	if obj.IsNull() || obj.IsUnknown() {
		return out
	}
	var m FirewallModel
	obj.As(ctx, &m, basetypes.ObjectAsOptions{})
	out.Enabled = valueBoolDefault(m.Enabled, out.Enabled)
	out.Backend = valueStringDefault(m.Backend, out.Backend)
	out.SSHAllowedCIDRs = defaultStringList(ctx, m.SSHAllowedCIDRs, out.SSHAllowedCIDRs)
	out.AdminCIDRs = stringList(ctx, m.AdminCIDRs)
	out.AllowHTTP = valueBoolDefault(m.AllowHTTP, out.AllowHTTP)
	out.AllowHTTPS = valueBoolDefault(m.AllowHTTPS, out.AllowHTTPS)
	out.AllowKubeAPI = valueBoolDefault(m.AllowKubeAPI, out.AllowKubeAPI)
	out.ExtraTCPPorts = intList(ctx, m.ExtraTCPPorts)
	out.ExtraUDPPorts = intList(ctx, m.ExtraUDPPorts)
	return out
}

func expandAddons(ctx context.Context, obj types.Object) AddonsConfig {
	out := AddonsConfig{Traefik: TraefikConfig{Mode: "default", Namespace: "traefik"}, Longhorn: LonghornConfig{Namespace: "longhorn-system"}}
	if obj.IsNull() || obj.IsUnknown() {
		return out
	}
	var m AddonsModel
	obj.As(ctx, &m, basetypes.ObjectAsOptions{})
	if !m.Traefik.IsNull() && !m.Traefik.IsUnknown() {
		var t TraefikModel
		m.Traefik.As(ctx, &t, basetypes.ObjectAsOptions{})
		out.Traefik.Mode = valueStringDefault(t.Mode, out.Traefik.Mode)
		out.Traefik.Namespace = valueStringDefault(t.Namespace, out.Traefik.Namespace)
		out.Traefik.ChartVersion = valueStringDefault(t.ChartVersion, "")
		out.Traefik.ValuesYAML = valueStringDefault(t.ValuesYAML, "")
	}
	if !m.Longhorn.IsNull() && !m.Longhorn.IsUnknown() {
		var l LonghornModel
		m.Longhorn.As(ctx, &l, basetypes.ObjectAsOptions{})
		out.Longhorn.Enabled = valueBoolDefault(l.Enabled, false)
		out.Longhorn.Namespace = valueStringDefault(l.Namespace, out.Longhorn.Namespace)
		out.Longhorn.ChartVersion = valueStringDefault(l.ChartVersion, "")
		out.Longhorn.StorageClasses = expandStorageClasses(ctx, l.StorageClasses)
	}
	if out.Longhorn.Enabled && len(out.Longhorn.StorageClasses) == 0 {
		out.Longhorn.StorageClasses = defaultLonghornStorageClasses()
	}
	return out
}

func expandAdminPeer(ctx context.Context, obj types.Object) AdminPeerConfig {
	out := AdminPeerConfig{Enabled: false, Name: "admin"}
	if obj.IsNull() || obj.IsUnknown() {
		return out
	}
	var m AdminPeerModel
	obj.As(ctx, &m, basetypes.ObjectAsOptions{})
	out.Enabled = valueBoolDefault(m.Enabled, false)
	out.Name = valueStringDefault(m.Name, out.Name)
	out.WGIP = valueStringDefault(m.WGIP, "")
	return out
}

func expandAdminPeers(ctx context.Context, m types.Map) (map[string]AdminPeerConfig, []string) {
	out := map[string]AdminPeerConfig{}
	if m.IsNull() || m.IsUnknown() {
		return out, nil
	}
	var peers map[string]AdminPeerMapModel
	m.ElementsAs(ctx, &peers, false)
	names := make([]string, 0, len(peers))
	for name, peer := range peers {
		names = append(names, name)
		out[name] = AdminPeerConfig{
			Enabled:   true,
			Name:      name,
			WGIP:      valueStringDefault(peer.WGIP, ""),
			PublicKey: valueStringDefault(peer.PublicKey, ""),
		}
	}
	sort.Strings(names)
	return out, names
}

func (cfg ClusterConfig) adminPeerIPRequests() ([]string, map[string]string) {
	requested := map[string]string{}
	names := make([]string, 0, len(cfg.AdminPeerNames)+1)
	if cfg.AdminPeer.Enabled {
		names = append(names, legacyAdminPeerKey)
		requested[legacyAdminPeerKey] = cfg.AdminPeer.WGIP
	}
	for _, name := range cfg.AdminPeerNames {
		names = append(names, name)
		requested[name] = cfg.AdminPeers[name].WGIP
	}
	return names, requested
}

func expandStorageClasses(ctx context.Context, m types.Map) map[string]StorageClassConfig {
	out := map[string]StorageClassConfig{}
	if m.IsNull() || m.IsUnknown() {
		return out
	}
	var classes map[string]StorageClassModel
	m.ElementsAs(ctx, &classes, false)
	for key, c := range classes {
		out[key] = StorageClassConfig{Name: c.Name.ValueString(), NumberReplicas: valueIntDefault(c.NumberReplicas, 1), ReclaimPolicy: valueStringDefault(c.ReclaimPolicy, "Delete"), DefaultClass: valueBoolDefault(c.DefaultClass, false)}
	}
	return out
}

func defaultLonghornStorageClasses() map[string]StorageClassConfig {
	return map[string]StorageClassConfig{
		"critical": {Name: "longhorn-critical-r2", NumberReplicas: 2, ReclaimPolicy: "Retain", DefaultClass: false},
		"cheap":    {Name: "longhorn-cheap-r1", NumberReplicas: 1, ReclaimPolicy: "Delete", DefaultClass: false},
	}
}

func nodeIdentity(n NodeModel, index int) string {
	hostname := valueStringDefault(n.Hostname, "")
	if hostname != "" {
		return hostname
	}
	publicIP := valueStringDefault(n.PublicIP, "")
	if publicIP != "" {
		return publicIP
	}
	return fmt.Sprintf("node-%d", index+1)
}

func firstServer(cfg ClusterConfig) (NodeConfig, error) {
	for _, key := range cfg.NodeNames {
		if cfg.Nodes[key].Role == "server" {
			return cfg.Nodes[key], nil
		}
	}
	return NodeConfig{}, fmt.Errorf("cluster must contain at least one server node")
}

func serverURLFor(cfg ClusterConfig) (string, error) {
	n, err := firstServer(cfg)
	if err != nil {
		return "", err
	}
	host := n.PublicIP
	if cfg.WireGuard.Enabled {
		host = n.PrivateIP
	}
	return "https://" + host + ":6443", nil
}

func mergedDisableComponents(base []string, traefikMode string) []string {
	seen := map[string]bool{}
	var out []string
	for _, v := range base {
		v = strings.TrimSpace(v)
		if v != "" && !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	if (traefikMode == "disabled" || traefikMode == "install") && !seen["traefik"] {
		out = append(out, "traefik")
	}
	return out
}

func valueStringDefault(v types.String, def string) string {
	if v.IsNull() || v.IsUnknown() {
		return def
	}
	return v.ValueString()
}

func valueBoolDefault(v types.Bool, def bool) bool {
	if v.IsNull() || v.IsUnknown() {
		return def
	}
	return v.ValueBool()
}

func valueIntDefault(v types.Int64, def int) int {
	if v.IsNull() || v.IsUnknown() {
		return def
	}
	return int(v.ValueInt64())
}

func nullOrString(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

func stringList(ctx context.Context, list types.List) []string {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}
	var out []string
	list.ElementsAs(ctx, &out, false)
	return out
}

func defaultStringList(ctx context.Context, list types.List, def []string) []string {
	out := stringList(ctx, list)
	if out == nil {
		return def
	}
	return out
}

func intList(ctx context.Context, list types.List) []int {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}
	var vals []int64
	list.ElementsAs(ctx, &vals, false)
	out := make([]int, 0, len(vals))
	for _, v := range vals {
		out = append(out, int(v))
	}
	return out
}

func boolStatus(v bool) string {
	if v {
		return "enabled"
	}
	return "disabled"
}
