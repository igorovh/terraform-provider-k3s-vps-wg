package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = (*vpsk3sProvider)(nil)

type vpsk3sProvider struct {
	version string
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &vpsk3sProvider{version: version}
	}
}

func (p *vpsk3sProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "vpsk3s"
	resp.Version = p.version
}

func (p *vpsk3sProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Terraform provider for installing K3s on existing Debian/Ubuntu VPS nodes over SSH.",
		Attributes: map[string]schema.Attribute{
			"ssh_user":                 schema.StringAttribute{Optional: true, MarkdownDescription: "Default SSH user. Defaults to root."},
			"ssh_port":                 schema.Int64Attribute{Optional: true, MarkdownDescription: "Default SSH port. Defaults to 22."},
			"ssh_password":             schema.StringAttribute{Optional: true, Sensitive: true, MarkdownDescription: "Default SSH password. Used only when private key auth is not configured."},
			"ssh_private_key":          schema.StringAttribute{Optional: true, Sensitive: true, MarkdownDescription: "Default SSH private key content. Takes precedence over ssh_private_key_path and ssh_password."},
			"ssh_private_key_path":     schema.StringAttribute{Optional: true, MarkdownDescription: "Path to default SSH private key. Supports ~ expansion."},
			"ssh_timeout":              schema.StringAttribute{Optional: true, MarkdownDescription: "SSH connection timeout as Go duration, for example 30s or 5m. Defaults to 5m."},
			"insecure_ignore_host_key": schema.BoolAttribute{Optional: true, MarkdownDescription: "Ignore SSH host key verification. Defaults to true for simple VPS bootstrap usage."},
		},
	}
}

func (p *vpsk3sProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var model ProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg, diags := expandProviderConfig(ctx, model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.ResourceData = cfg
}

func (p *vpsk3sProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{NewClusterResource}
}

func (p *vpsk3sProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}

func pathRoot(name string) path.Path {
	return path.Root(name)
}

func stringMap(ctx context.Context, values map[string]string) types.Map {
	v, _ := types.MapValueFrom(ctx, types.StringType, values)
	return v
}
