package sshtunnel

import (
	"testing"
)

func TestTunnel_RequiresHostUserKeyPath(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
	}{
		{"missing host", Config{User: "u", KeyPath: "/tmp/k"}},
		{"missing user", Config{Host: "h", KeyPath: "/tmp/k"}},
		{"missing key_path", Config{Host: "h", User: "u"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := Tunnel(tt.cfg, "db:3306")
			if err == nil {
				t.Error("expected error when required SSH config is missing")
			}
		})
	}
}

func TestTunnel_InvalidKeyPath(t *testing.T) {
	_, _, err := Tunnel(Config{
		Host:    "bastion.example.com",
		User:    "deploy",
		KeyPath: "/nonexistent/key",
		Port:    22,
	}, "mysql.internal:3306")
	if err == nil {
		t.Error("expected error for nonexistent key file")
	}
}
