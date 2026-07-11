package provider

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type SSHClient struct {
	client *ssh.Client
}

func DialSSH(host string, cfg SSHConfig) (*SSHClient, error) {
	auth, err := sshAuthMethods(cfg)
	if err != nil {
		return nil, err
	}
	hostKeyCallback := ssh.FixedHostKey(nil)
	if cfg.Insecure {
		hostKeyCallback = ssh.InsecureIgnoreHostKey()
	}
	clientConfig := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            auth,
		HostKeyCallback: hostKeyCallback,
		Timeout:         cfg.Timeout,
	}
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", cfg.Port))
	client, err := ssh.Dial("tcp", addr, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("ssh dial %s: %w", addr, err)
	}
	return &SSHClient{client: client}, nil
}

func (c *SSHClient) Close() error {
	return c.client.Close()
}

func (c *SSHClient) Run(command string) (string, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr
	err = session.Run("set -euo pipefail; " + command)
	out := strings.TrimSpace(stdout.String())
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = out
		}
		return out, fmt.Errorf("remote command failed: %s: %w", msg, err)
	}
	return out, nil
}

func (c *SSHClient) RunNoPipefail(command string) (string, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr
	err = session.Run(command)
	out := strings.TrimSpace(stdout.String())
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = out
		}
		return out, fmt.Errorf("remote command failed: %s: %w", msg, err)
	}
	return out, nil
}

func (c *SSHClient) Upload(path string, content []byte, mode os.FileMode) error {
	s, err := sftp.NewClient(c.client)
	if err != nil {
		return err
	}
	defer s.Close()
	tmp := fmt.Sprintf("/tmp/vpsk3s-%d", time.Now().UnixNano())
	f, err := s.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY)
	if err != nil {
		return err
	}
	if _, err := f.Write(content); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := s.Chmod(tmp, mode); err != nil {
		return err
	}
	_, err = c.Run(fmt.Sprintf("install -D -m %04o %s %s && rm -f %s", mode.Perm(), shellQuote(tmp), shellQuote(path), shellQuote(tmp)))
	return err
}

func (c *SSHClient) Download(path string) ([]byte, error) {
	s, err := sftp.NewClient(c.client)
	if err != nil {
		return nil, err
	}
	defer s.Close()
	f, err := s.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var buf bytes.Buffer
	_, err = buf.ReadFrom(f)
	return buf.Bytes(), err
}

func sshAuthMethods(cfg SSHConfig) ([]ssh.AuthMethod, error) {
	if cfg.PrivateKey != "" {
		signer, err := ssh.ParsePrivateKey([]byte(cfg.PrivateKey))
		if err != nil {
			return nil, fmt.Errorf("parse ssh_private_key: %w", err)
		}
		return []ssh.AuthMethod{ssh.PublicKeys(signer)}, nil
	}
	if cfg.PrivateKeyPath != "" {
		keyPath := expandPath(cfg.PrivateKeyPath)
		key, err := os.ReadFile(keyPath)
		if err != nil {
			return nil, fmt.Errorf("read ssh_private_key_path %s: %w", keyPath, err)
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("parse ssh_private_key_path %s: %w", keyPath, err)
		}
		return []ssh.AuthMethod{ssh.PublicKeys(signer)}, nil
	}
	if cfg.Password != "" {
		return []ssh.AuthMethod{ssh.Password(cfg.Password)}, nil
	}
	return nil, fmt.Errorf("no SSH authentication configured; set ssh_private_key, ssh_private_key_path, or ssh_password")
}

func expandPath(path string) string {
	if path == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
	}
	if strings.HasPrefix(path, "~/") || strings.HasPrefix(path, `~\`) {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
