package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = (*clusterResource)(nil)
var _ resource.ResourceWithConfigure = (*clusterResource)(nil)
var _ resource.ResourceWithModifyPlan = (*clusterResource)(nil)

type clusterResource struct {
	provider ProviderConfig
}

func NewClusterResource() resource.Resource {
	return &clusterResource{}
}

func (r *clusterResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster"
}

func (r *clusterResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	cfg, ok := req.ProviderData.(ProviderConfig)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("Expected ProviderConfig, got %T", req.ProviderData))
		return
	}
	r.provider = cfg
}

func (r *clusterResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = clusterSchema(ctx)
}

func (r *clusterResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	var plan ClusterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(validatePlan(ctx, plan)...)
}

func (r *clusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ClusterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var empty ClusterResourceModel
	cfg, diags := expandClusterConfig(ctx, plan, empty, r.provider)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(validateConfig(cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	kubeconfig, serverURL, adminConfig, adminConfigs, statuses, err := r.applyCluster(ctx, cfg)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create cluster", sanitizeError(err))
		return
	}
	state, diags := clusterToState(ctx, cfg, plan, kubeconfig, serverURL, adminConfig, adminConfigs, statuses)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *clusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ClusterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// The provider intentionally does not SSH during refresh. Remote state is checked during apply.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *clusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ClusterResourceModel
	var prior ClusterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg, diags := expandClusterConfig(ctx, plan, prior, r.provider)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(validateConfig(cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	kubeconfig, serverURL, adminConfig, adminConfigs, statuses, err := r.applyCluster(ctx, cfg)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update cluster", sanitizeError(err))
		return
	}
	state, diags := clusterToState(ctx, cfg, plan, kubeconfig, serverURL, adminConfig, adminConfigs, statuses)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *clusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ClusterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg, diags := expandClusterConfig(ctx, state, state, r.provider)
	resp.Diagnostics.Append(diags...)
	if !resp.Diagnostics.HasError() {
		_ = r.bestEffortDelete(ctx, cfg)
	}
	resp.State.RemoveResource(ctx)
}

func (r *clusterResource) applyCluster(ctx context.Context, cfg ClusterConfig) (string, string, string, map[string]string, map[string]string, error) {
	statuses := map[string]string{}
	if cfg.WireGuard.Enabled {
		for _, key := range cfg.NodeNames {
			n := cfg.Nodes[key]
			client, err := DialSSH(n.PublicIP, n.SSH)
			if err != nil {
				return "", "", "", nil, statuses, err
			}
			err = configureHostnameAndHosts(client, cfg, n)
			if err == nil {
				err = configureWireGuard(client, cfg, n)
			}
			client.Close()
			if err != nil {
				return "", "", "", nil, statuses, err
			}
			statuses[key] = "wireguard-configured"
		}
		for _, key := range cfg.NodeNames {
			n := cfg.Nodes[key]
			client, err := DialSSH(n.PublicIP, n.SSH)
			if err != nil {
				return "", "", "", nil, statuses, err
			}
			err = checkWireGuardConnectivity(client, cfg, n)
			client.Close()
			if err != nil {
				return "", "", "", nil, statuses, err
			}
		}
	} else {
		for _, key := range cfg.NodeNames {
			n := cfg.Nodes[key]
			client, err := DialSSH(n.PublicIP, n.SSH)
			if err != nil {
				return "", "", "", nil, statuses, err
			}
			err = configureHostnameAndHosts(client, cfg, n)
			client.Close()
			if err != nil {
				return "", "", "", nil, statuses, err
			}
		}
	}

	if cfg.Firewall.Enabled {
		for _, key := range cfg.NodeNames {
			n := cfg.Nodes[key]
			client, err := DialSSH(n.PublicIP, n.SSH)
			if err != nil {
				return "", "", "", nil, statuses, err
			}
			err = configureFirewall(client, cfg, n)
			client.Close()
			if err != nil {
				return "", "", "", nil, statuses, err
			}
		}
	}

	for _, key := range cfg.NodeNames {
		n := cfg.Nodes[key]
		client, err := DialSSH(n.PublicIP, n.SSH)
		if err != nil {
			return "", "", "", nil, statuses, err
		}
		err = installK3s(client, cfg, n)
		client.Close()
		if err != nil {
			return "", "", "", nil, statuses, err
		}
		statuses[key] = "ready"
	}

	first, err := firstServer(cfg)
	if err != nil {
		return "", "", "", nil, statuses, err
	}
	client, err := DialSSH(first.PublicIP, first.SSH)
	if err != nil {
		return "", "", "", nil, statuses, err
	}
	defer client.Close()
	if err := installAddons(client, cfg); err != nil {
		return "", "", "", nil, statuses, err
	}
	kubeconfig, err := fetchKubeconfig(client, cfg)
	if err != nil {
		return "", "", "", nil, statuses, err
	}
	serverURL, err := serverURLFor(cfg)
	if err != nil {
		return "", "", "", nil, statuses, err
	}
	adminConfig := ""
	if cfg.AdminPeer.Enabled {
		adminConfig, err = renderAdminWireGuardConfig(cfg)
		if err != nil {
			return "", "", "", nil, statuses, err
		}
	}
	adminConfigs, err := renderAdminWireGuardConfigs(cfg)
	if err != nil {
		return "", "", "", nil, statuses, err
	}
	return kubeconfig, serverURL, adminConfig, adminConfigs, statuses, nil
}

func (r *clusterResource) bestEffortDelete(ctx context.Context, cfg ClusterConfig) error {
	for _, key := range cfg.NodeNames {
		n := cfg.Nodes[key]
		client, err := DialSSH(n.PublicIP, n.SSH)
		if err != nil {
			continue
		}
		_, _ = client.Run("systemctl stop k3s k3s-agent 2>/dev/null || true")
		_, _ = client.Run("systemctl stop wg-quick@" + shellQuote(cfg.WireGuard.Interface) + " 2>/dev/null || true")
		client.Close()
	}
	return nil
}

func sanitizeError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	for _, marker := range []string{"PRIVATE KEY", "password", "token"} {
		msg = strings.ReplaceAll(msg, marker, "[redacted]")
	}
	return msg
}

func clusterSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		MarkdownDescription: "K3s cluster installed on existing Debian/Ubuntu VPS nodes over SSH.",
		Attributes: map[string]schema.Attribute{
			"id":                           schema.StringAttribute{Computed: true},
			"name":                         schema.StringAttribute{Required: true, MarkdownDescription: "Cluster name."},
			"nodes":                        nodesAttribute(),
			"wireguard":                    wireGuardAttribute(),
			"k3s":                          k3sAttribute(),
			"firewall":                     firewallAttribute(),
			"addons":                       addonsAttribute(),
			"admin_peer":                   adminPeerAttribute(),
			"admin_peers":                  adminPeersAttribute(),
			"kubeconfig":                   schema.StringAttribute{Computed: true, Sensitive: true},
			"server_url":                   schema.StringAttribute{Computed: true},
			"admin_wireguard_config":       schema.StringAttribute{Computed: true, Sensitive: true},
			"admin_wireguard_configs":      schema.MapAttribute{Computed: true, Sensitive: true, ElementType: types.StringType, MarkdownDescription: "Generated WireGuard configs for admin_peers entries where public_key was not provided."},
			"wireguard_enabled":            schema.BoolAttribute{Computed: true},
			"firewall_enabled":             schema.BoolAttribute{Computed: true},
			"addons_status":                schema.MapAttribute{Computed: true, ElementType: types.StringType},
			"k3s_token":                    schema.StringAttribute{Computed: true, Sensitive: true, MarkdownDescription: "Internal generated K3s token stored in Terraform state."},
			"wireguard_private_keys":       schema.MapAttribute{Computed: true, Sensitive: true, ElementType: types.StringType, MarkdownDescription: "Internal WireGuard private keys stored in Terraform state."},
			"admin_wireguard_private_key":  schema.StringAttribute{Computed: true, Sensitive: true, MarkdownDescription: "Internal admin peer private key stored in Terraform state."},
			"admin_wireguard_public_key":   schema.StringAttribute{Computed: true, MarkdownDescription: "Internal admin peer public key."},
			"admin_wireguard_private_keys": schema.MapAttribute{Computed: true, Sensitive: true, ElementType: types.StringType, MarkdownDescription: "Internal generated admin_peers private keys. Empty for peers that provide public_key."},
			"admin_wireguard_public_keys":  schema.MapAttribute{Computed: true, ElementType: types.StringType, MarkdownDescription: "Admin peer public keys used on cluster nodes."},
		},
	}
}
