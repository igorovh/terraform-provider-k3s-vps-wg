package provider

func installAddons(client *SSHClient, cfg ClusterConfig) error {
	if cfg.Addons.Traefik.Mode == "install" {
		if err := installTraefik(client, cfg); err != nil {
			return err
		}
	}
	if cfg.Addons.Longhorn.Enabled {
		if err := installLonghorn(client, cfg); err != nil {
			return err
		}
	}
	return nil
}
