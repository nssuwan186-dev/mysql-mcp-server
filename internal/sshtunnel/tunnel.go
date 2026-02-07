// Package sshtunnel provides SSH port forwarding for MySQL connections through a bastion host.
package sshtunnel

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

const defaultSSHPort = 22

// Config holds SSH bastion connection parameters.
type Config struct {
	Host    string // Bastion hostname
	User    string // SSH username
	KeyPath string // Path to private key file
	Port    int    // SSH port (0 = default 22)
}

// expandTilde returns path with a leading "~" or "~/" expanded to the user's home directory.
func expandTilde(path string) (string, error) {
	if path == "" || (path != "~" && !strings.HasPrefix(path, "~/")) {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("expand ~ in key path: %w", err)
	}
	if path == "~" {
		return home, nil
	}
	return filepath.Join(home, path[2:]), nil
}

// Tunnel starts a local listener that forwards connections to remoteAddr via SSH.
// Returns the local address (e.g. "127.0.0.1:12345") and a close function.
// remoteAddr should be the MySQL server address (e.g. "db.example.com:3306").
// KeyPath may start with "~/" or be "~" to mean the current user's home directory.
func Tunnel(cfg Config, remoteAddr string) (localAddr string, closeFn func(), err error) {
	if cfg.Host == "" || cfg.User == "" || cfg.KeyPath == "" {
		return "", nil, fmt.Errorf("ssh tunnel requires host, user, and key_path")
	}

	keyPath, err := expandTilde(cfg.KeyPath)
	if err != nil {
		return "", nil, err
	}
	key, err := os.ReadFile(keyPath)
	if err != nil {
		return "", nil, fmt.Errorf("read SSH key: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return "", nil, fmt.Errorf("parse SSH key: %w", err)
	}

	port := cfg.Port
	if port <= 0 {
		port = defaultSSHPort
	}
	sshAddr := net.JoinHostPort(cfg.Host, strconv.Itoa(port))

	sshConfig := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Bastion: host key verification not performed; consider known_hosts in future
		Timeout:         15 * time.Second,
	}

	client, err := ssh.Dial("tcp", sshAddr, sshConfig)
	if err != nil {
		return "", nil, fmt.Errorf("ssh dial: %w", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		client.Close()
		return "", nil, fmt.Errorf("listen: %w", err)
	}

	localAddr = listener.Addr().String()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go proxy(client, remoteAddr, conn)
		}
	}()

	closeFn = func() {
		listener.Close()
		client.Close()
	}
	return localAddr, closeFn, nil
}

// proxy opens a connection to remoteAddr via the SSH client and copies data both ways.
func proxy(client *ssh.Client, remoteAddr string, local net.Conn) {
	defer local.Close()
	remote, err := client.Dial("tcp", remoteAddr)
	if err != nil {
		return
	}
	defer remote.Close()
	go io.Copy(remote, local)
	io.Copy(local, remote)
}
