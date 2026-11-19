package filedist

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// SFTPClient wraps SFTP connection for file transfers.
type SFTPClient struct {
	host            string
	port            int
	username        string
	password        string
	privateKey      string
	hostFingerprint string
	sshClient       *ssh.Client
	sftpClient      *sftp.Client
	mu              sync.Mutex
	closed          bool
}

// SFTPConfig represents SFTP connection configuration.
type SFTPConfig struct {
	Host            string `json:"host"`
	Port            int    `json:"port"`
	Username        string `json:"username"`
	Password        string `json:"password"`
	PrivateKey      string `json:"privateKey"`
	HostFingerprint string `json:"hostFingerprint"` // required SSH host key fingerprint (SHA256)
	Timeout         int    `json:"timeout"`         // seconds
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
	return c.ConnectContext(context.Background())
}

// ConnectContext bounds dialing, SSH handshake and SFTP subsystem setup.
func (c *SFTPClient) ConnectContext(ctx context.Context) error {
	if ctx == nil {
		return errors.New("business.filedist.task.invalidContext")
	}
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

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	addr := net.JoinHostPort(c.host, fmt.Sprint(c.port))
	connection, err := (&net.Dialer{}).DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("ssh_dial_failed: %w", err)
	}
	stop := context.AfterFunc(ctx, func() { _ = connection.Close() })
	defer stop()
	clientConn, channels, requests, err := ssh.NewClientConn(connection, addr, config)
	if err != nil {
		_ = connection.Close()
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("ssh_dial_failed: %w", err)
	}
	sshClient := ssh.NewClient(clientConn, channels, requests)

	// Open SFTP session
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		_ = sshClient.Close()
		return fmt.Errorf("sftp_session_failed: %w", err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || ctx.Err() != nil {
		_ = sshClient.Close()
		_ = sftpClient.Close()
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errors.New("not_connected")
	}
	c.sshClient, c.sftpClient = sshClient, sftpClient
	return nil
}

// Close closes SFTP connection.
func (c *SFTPClient) Close() error {
	c.mu.Lock()
	client, sshClient := c.sftpClient, c.sshClient
	c.closed = true
	c.sftpClient, c.sshClient = nil, nil
	c.mu.Unlock()
	// Close the transport first to interrupt blocked SFTP requests.
	if sshClient != nil {
		_ = sshClient.Close()
	}
	if client != nil {
		return client.Close()
	}
	return nil
}

// Upload uploads a file to remote server.
func (c *SFTPClient) Upload(localPath, remotePath string) error {
	return c.UploadContext(context.Background(), localPath, remotePath)
}

// UploadContext interrupts the owned transport on cancellation. Already
// accepted remote bytes may remain; cancelling is not remote-file rollback.
func (c *SFTPClient) UploadContext(ctx context.Context, localPath, remotePath string) error {
	if ctx == nil {
		return errors.New("business.filedist.task.invalidContext")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	client := c.sftpClient
	c.mu.Unlock()
	if client == nil {
		return fmt.Errorf("not_connected")
	}
	stop := context.AfterFunc(ctx, func() { _ = c.Close() })
	defer stop()

	// Open local file
	localFile, err := os.Open(localPath) // #nosec G304 -- localPath comes from the transfer engine, which derives it from a stored filedist record path under the configured storage root.
	if err != nil {
		return fmt.Errorf("open_local_file_failed: %w", err)
	}
	defer func() { _ = localFile.Close() }()

	// Create remote directory
	remoteDir := filepath.Dir(remotePath)
	if err := client.MkdirAll(remoteDir); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("create_remote_dir_failed: %w", err)
	}

	// Create remote file
	remoteFile, err := client.Create(remotePath)
	if err != nil {
		return fmt.Errorf("create_remote_file_failed: %w", err)
	}

	// Copy data
	if _, err := io.Copy(remoteFile, &contextReader{ctx: ctx, reader: localFile}); err != nil {
		_ = remoteFile.Close()
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("copy_failed: %w", err)
	}
	// Closing flushes the remote handle; a failed close means the upload did
	// not land, so it must not be reported as success.
	if err := remoteFile.Close(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("close_remote_file_failed: %w", err)
	}

	return ctx.Err()
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(buffer []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(buffer)
}

// Download downloads a file from remote server.
func (c *SFTPClient) Download(remotePath, localPath string) error {
	c.mu.Lock()
	client := c.sftpClient
	c.mu.Unlock()
	if client == nil {
		return fmt.Errorf("not_connected")
	}

	// Open remote file
	remoteFile, err := client.Open(remotePath)
	if err != nil {
		return fmt.Errorf("open_remote_file_failed: %w", err)
	}
	defer func() { _ = remoteFile.Close() }()

	// Create local directory
	localDir := filepath.Dir(localPath)
	if err := os.MkdirAll(localDir, 0750); err != nil {
		return fmt.Errorf("create_local_dir_failed: %w", err)
	}

	// Create local file
	localFile, err := os.Create(localPath) // #nosec G304 -- localPath comes from the transfer engine, which derives it from a stored filedist record path under the configured storage root.
	if err != nil {
		return fmt.Errorf("create_local_file_failed: %w", err)
	}

	// Copy data
	if _, err := io.Copy(localFile, remoteFile); err != nil {
		_ = localFile.Close()
		return fmt.Errorf("copy_failed: %w", err)
	}
	// A failed close means the local file may be truncated, so surface it
	// instead of reporting a successful download.
	if err := localFile.Close(); err != nil {
		return fmt.Errorf("close_local_file_failed: %w", err)
	}

	return nil
}

// Stat gets file information.
func (c *SFTPClient) Stat(remotePath string) (os.FileInfo, error) {
	c.mu.Lock()
	client := c.sftpClient
	c.mu.Unlock()
	if client == nil {
		return nil, fmt.Errorf("not_connected")
	}
	return client.Stat(remotePath)
}

// Remove deletes a remote file.
func (c *SFTPClient) Remove(remotePath string) error {
	c.mu.Lock()
	client := c.sftpClient
	c.mu.Unlock()
	if client == nil {
		return fmt.Errorf("not_connected")
	}
	return client.Remove(remotePath)
}

// List lists files in a remote directory.
func (c *SFTPClient) List(remotePath string) ([]os.FileInfo, error) {
	c.mu.Lock()
	client := c.sftpClient
	c.mu.Unlock()
	if client == nil {
		return nil, fmt.Errorf("not_connected")
	}
	return client.ReadDir(remotePath)
}

// TestSFTPConnection tests SFTP connectivity.
func TestSFTPConnection(config SFTPConfig) error {
	client := NewSFTPClient(config)
	if err := client.Connect(); err != nil {
		return err
	}
	defer func() { _ = client.Close() }()

	// Try to stat the home directory
	_, err := client.sftpClient.Getwd()
	return err
}
