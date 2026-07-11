package provider

import (
	"fmt"
	"strings"
)

func ensureHelm(client *SSHClient) error {
	_, err := client.Run("command -v helm >/dev/null 2>&1 || curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash")
	return err
}

func installTraefik(client *SSHClient, cfg ClusterConfig) error {
	if err := ensureHelm(client); err != nil {
		return err
	}
	ns := cfg.Addons.Traefik.Namespace
	if ns == "" {
		ns = "traefik"
	}
	cmds := []string{
		"export KUBECONFIG=/etc/rancher/k3s/k3s.yaml",
		"helm repo add traefik https://traefik.github.io/charts >/dev/null 2>&1 || true",
		"helm repo update",
	}
	args := []string{"upgrade", "--install", "traefik", "traefik/traefik", "--namespace", ns, "--create-namespace"}
	if cfg.Addons.Traefik.ChartVersion != "" {
		args = append(args, "--version", cfg.Addons.Traefik.ChartVersion)
	}
	valuesPath := ""
	if cfg.Addons.Traefik.ValuesYAML != "" {
		valuesPath = "/tmp/vpsk3s-traefik-values.yaml"
		if err := client.Upload(valuesPath, []byte(cfg.Addons.Traefik.ValuesYAML), 0600); err != nil {
			return err
		}
		args = append(args, "-f", valuesPath)
	}
	cmds = append(cmds, shellArgs("helm", args...))
	if valuesPath != "" {
		cmds = append(cmds, "rm -f "+shellQuote(valuesPath))
	}
	_, err := client.Run(strings.Join(cmds, " && "))
	return err
}

func installLonghorn(client *SSHClient, cfg ClusterConfig) error {
	if err := ensureHelm(client); err != nil {
		return err
	}
	ns := cfg.Addons.Longhorn.Namespace
	if ns == "" {
		ns = "longhorn-system"
	}
	cmds := []string{
		"export KUBECONFIG=/etc/rancher/k3s/k3s.yaml",
		"helm repo add longhorn https://charts.longhorn.io >/dev/null 2>&1 || true",
		"helm repo update",
	}
	args := []string{"upgrade", "--install", "longhorn", "longhorn/longhorn", "--namespace", ns, "--create-namespace"}
	if cfg.Addons.Longhorn.ChartVersion != "" {
		args = append(args, "--version", cfg.Addons.Longhorn.ChartVersion)
	}
	cmds = append(cmds, shellArgs("helm", args...))
	cmds = append(cmds, "kubectl -n "+shellQuote(ns)+" rollout status deployment/longhorn-driver-deployer --timeout=10m || true")
	for _, sc := range cfg.Addons.Longhorn.StorageClasses {
		manifest := renderLonghornStorageClass(sc)
		path := fmt.Sprintf("/tmp/vpsk3s-sc-%s.yaml", sc.Name)
		if err := client.Upload(path, []byte(manifest), 0600); err != nil {
			return err
		}
		cmds = append(cmds, "kubectl apply -f "+shellQuote(path), "rm -f "+shellQuote(path))
	}
	_, err := client.Run(strings.Join(cmds, " && "))
	return err
}

func renderLonghornStorageClass(sc StorageClassConfig) string {
	defaultClass := "false"
	if sc.DefaultClass {
		defaultClass = "true"
	}
	return fmt.Sprintf(`apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: %s
  annotations:
    storageclass.kubernetes.io/is-default-class: "%s"
provisioner: driver.longhorn.io
allowVolumeExpansion: true
reclaimPolicy: %s
volumeBindingMode: Immediate
parameters:
  numberOfReplicas: "%d"
  staleReplicaTimeout: "30"
  fromBackup: ""
`, sc.Name, defaultClass, sc.ReclaimPolicy, sc.NumberReplicas)
}

func shellArgs(command string, args ...string) string {
	parts := []string{command}
	for _, arg := range args {
		parts = append(parts, shellQuote(arg))
	}
	return strings.Join(parts, " ")
}
