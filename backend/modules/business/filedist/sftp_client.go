package filedist

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// SFTPClient wraps SFTP connection for file transfers.
type SFTPClient struct {
	host          string
	port          int
	username      string
	password      string
	privateKey    string
	hostFingerprint string
	sshClient     *ssh.Client
	sftpClient    *sftp.Client
}

// SFTPConfig represents SFTP connection configuration.
type SFTPConfig struct {
	Host           string `json:"host"`
	Port           int    `json:"port"`
	Username       string `json:"username"`
	Password       string `json:"password"`
	PrivateKey     string `json:"privateKey"`
	HostFingerprint string `json:"hostFingerprint"` // required SSH host key fingerprint (SHA256)
	Timeout        int    `json:"timeout"`          // seconds
}

// NewSFTPClient creates a new SFTP client.
func NewSFTPClient(config SFTPConfig) *SFTPClient {
	if config.Port == 0 {
		config.Port = 22
	}
	return &SFTPClient{
		host:            config.Host,
		port:            config.Port,
		username:        config.Username,
		password:        config.Password,
		privateKey:      config.PrivateKey,
		hostFingerprint: strings.TrimSpace(config.HostFingerprint),
	}
}

// hostFingerprintCallback verifies the remote SSH host key against the
// expected SHA256 fingerprint, preventing man-in-the-middle attacks.
func hostFingerprintCallback(expectedFingerprint string) ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		if strings.TrimSpace(ssh.FingerprintSHA256(key)) != strings.TrimSpace(expectedFingerprint) {
			return errors.New("filedist.host_key_mismatch")
		}
		return nil
	}
}

// Connect establishes SFTP connection.
func (c *SFTPClient) Connect() error {
	var authMethods []ssh.AuthMethod

	// Password authentication
	if c.password != "" {
		authMethods = append(authMethods, ssh.Password(c.password))
	}

	// Private key authentication
	if c.privateKey != "" {
		signer, err := ssh.ParsePrivateKey([]byte(c.privateKey))
		if err != nil {
			return fmt.Errorf("parse_private_key_failed: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	// Require a host key fingerprint and verify it; without it we cannot
	// authenticate the remote and refuse to connect (fail-closed).
	if c.hostFingerprint == "" {
		return errors.New("filedist.host_fingerprint_required")
	}

	config := &ssh.ClientConfig{
		User:            c.username,
		Auth:            authMethods,
		HostKeyCallback: hostFingerprintCallback(c.hostFingerprint),
		Timeout:         30 * time.Second,
	}

	// Connect to SSH
	addr := fmt.Sprintf("%s:%d", c.host, c.port)
	sshClient, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return fmt.Errorf("ssh_dial_failed: %w", err)
	}
	c.sshClient = sshClient

	// Open SFTP session
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		_ = sshClient.Close()
		return fmt.Errorf("sftp_session_failed: %w", err)
	}
	c.sftpClient = sftpClient

	return nil
}

// Close closes SFTP connection.
func (c *SFTPClient) Close() error {
	if c.sftpClient != nil {
		_ = c.sftpClient.Close()
	}
	if c.sshClient != nil {
		return c.sshClient.Close()
	}
	return nil
}

// Upload uploads a file to remote server.
func (c *SFTPClient) Upload(localPath, remotePath string) error {
	if c.sftpClient == nil {
		return fmt.Errorf("not_connected")
	}

	// Open local file
	localFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open_local_file_failed: %w", err)
	}
	defer localFile.Close()

	// Create remote directory
	remoteDir := filepath.Dir(remotePath)
	if err := c.sftpClient.MkdirAll(remoteDir); err != nil {
		return fmt.Errorf("create_remote_dir_failed: %w", err)
	}

	// Create remote file
	remoteFile, err := c.sftpClient.Create(remotePath)
	if err != nil {
		return fmt.Errorf("create_remote_file_failed: %w", err)
	}
	defer remoteFile.Close()

	// Copy data
	if _, err := io.Copy(remoteFile, localFile); err != nil {
		return fmt.Errorf("copy_failed: %w", err)
	}

	return nil
}

// Download downloads a file from remote server.
func (c *SFTPClient) Download(remotePath, localPath string) error {
	if c.sftpClient == nil {
		return fmt.Errorf("not_connected")
	}

	// Open remote file
	remoteFile, err := c.sftpClient.Open(remotePath)
	if err != nil {
		return fmt.Errorf("open_remote_file_failed: %w", err)
	}
	defer remoteFile.Close()

	// Create local directory
	localDir := filepath.Dir(localPath)
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return fmt.Errorf("create_local_dir_failed: %w", err)
	}

	// Create local file
	localFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("create_local_file_failed: %w", err)
	}
	defer localFile.Close()

	// Copy data
	if _, err := io.Copy(localFile, remoteFile); err != nil {
		return fmt.Errorf("copy_failed: %w", err)
	}

	return nil
}

// Stat gets file information.
func (c *SFTPClient) Stat(remotePath string) (os.FileInfo, error) {
	if c.sftpClient == nil {
		return nil, fmt.Errorf("not_connected")
	}
	return c.sftpClient.Stat(remotePath)
}

// Remove deletes a remote file.
func (c *SFTPClient) Remove(remotePath string) error {
	if c.sftpClient == nil {
		return fmt.Errorf("not_connected")
	}
	return c.sftpClient.Remove(remotePath)
}

// List lists files in a remote directory.
func (c *SFTPClient) List(remotePath string) ([]os.FileInfo, error) {
	if c.sftpClient == nil {
		return nil, fmt.Errorf("not_connected")
	}
	return c.sftpClient.ReadDir(remotePath)
}

// TestConnection tests SFTP connectivity.
func TestSFTPConnection(config SFTPConfig) error {
	client := NewSFTPClient(config)
	if err := client.Connect(); err != nil {
		return err
	}
	defer client.Close()

	// Try to stat the home directory
	_, err := client.sftpClient.Getwd()
	return err
}
