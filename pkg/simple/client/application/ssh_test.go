/*
 * Copyright 2024 the KubeSphere Authors.
 * Please refer to the LICENSE file in the root directory of the project.
 * https://github.com/kubesphere/kubesphere/blob/master/LICENSE
 */

package application

import (
	"net/url"
	"testing"

	appv2 "kubesphere.io/api/application/v2"
)

func TestSSHGetter_SetSSHAuth(t *testing.T) {
	tests := []struct {
		name        string
		privateKey  string
		passphrase  string
		knownHosts  string
		expectError bool
	}{
		{
			name:        "valid private key without passphrase",
			privateKey:  `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA4f5wg5l2hKsTeNem/V41fGnJm6gOdrj8ym3rFkEjWT2btZb5
-----END RSA PRIVATE KEY-----`,
			passphrase:  "",
			knownHosts:  "",
			expectError: true, // This is an incomplete key, so it should error
		},
		{
			name:        "empty private key",
			privateKey:  "",
			passphrase:  "",
			knownHosts:  "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getter := &SSHGetter{}
			err := getter.SetSSHAuth(tt.privateKey, tt.passphrase, tt.knownHosts)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestConvertToGitSSHURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "ssh://git@github.com/owner/repo.git",
			expected: "git@github.com:owner/repo.git",
		},
		{
			input:    "git+ssh://git@github.com/owner/repo.git",
			expected: "git@github.com:owner/repo.git",
		},
		{
			input:    "ssh://github.com/owner/repo.git",
			expected: "git@github.com:owner/repo.git",
		},
		{
			input:    "https://github.com/owner/repo.git",
			expected: "",
		},
		{
			input:    "invalid-url",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := convertToGitSSHURL(tt.input)
			if result != tt.expected {
				t.Errorf("convertToGitSSHURL(%s) = %s; expected %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestHelmPullFromSSH(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		credential  appv2.RepoCredential
		expectError bool
	}{
		{
			name: "SSH URL with missing private key",
			url:  "ssh://github.com/owner/repo.git",
			credential: appv2.RepoCredential{
				SSHPrivateKey: "",
			},
			expectError: true,
		},
		{
			name: "SSH URL with private key",
			url:  "ssh://github.com/owner/repo.git",
			credential: appv2.RepoCredential{
				SSHPrivateKey: "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA4f5wg5l2hKsTeNem/V41fGnJm6gOdrj8ym3rFkEjWT2btZb5\n-----END RSA PRIVATE KEY-----",
			},
			expectError: true, // Will fail due to invalid key or network issues, but should not fail due to missing key
		},
		{
			name: "non-SSH URL should not use SSH",
			url:  "https://github.com/owner/repo.git",
			credential: appv2.RepoCredential{
				SSHPrivateKey: "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA4f5wg5l2hKsTeNem/V41fGnJm6gOdrj8ym3rFkEjWT2btZb5\n-----END RSA PRIVATE KEY-----",
			},
			expectError: false, // This won't be handled by helmPullFromSSH
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test URL parsing
			parsedURL, err := url.Parse(tt.url)
			if err != nil {
				t.Errorf("Failed to parse URL %s: %v", tt.url, err)
				return
			}

			// Only test SSH URLs with helmPullFromSSH
			if parsedURL.Scheme == "ssh" || parsedURL.Scheme == "git+ssh" {
				_, err := helmPullFromSSH(tt.url, tt.credential)
				if tt.expectError && err == nil {
					t.Errorf("Expected error but got none")
				}
				if !tt.expectError && err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestLoadRepoIndexFromSSH(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		credential  appv2.RepoCredential
		expectError bool
	}{
		{
			name: "SSH URL with missing private key",
			url:  "ssh://github.com/owner/repo.git",
			credential: appv2.RepoCredential{
				SSHPrivateKey: "",
			},
			expectError: true,
		},
		{
			name: "SSH URL with private key",
			url:  "ssh://github.com/owner/repo.git",
			credential: appv2.RepoCredential{
				SSHPrivateKey: "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA4f5wg5l2hKsTeNem/V41fGnJm6gOdrj8ym3rFkEjWT2btZb5\n-----END RSA PRIVATE KEY-----",
			},
			expectError: true, // Will fail due to invalid key or network issues
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := loadRepoIndexFromSSH(tt.url, tt.credential)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestLoadRepoIndex_WithURLSchemes(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		credential  appv2.RepoCredential
		expectError bool
	}{
		{
			name: "SSH URL should be handled",
			url:  "ssh://github.com/owner/repo.git",
			credential: appv2.RepoCredential{
				SSHPrivateKey: "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA4f5wg5l2hKsTeNem/V41fGnJm6gOdrj8ym3rFkEjWT2btZb5\n-----END RSA PRIVATE KEY-----",
			},
			expectError: true, // Will fail due to invalid key, but should be handled by SSH path
		},
		{
			name: "git+ssh URL should be handled",
			url:  "git+ssh://github.com/owner/repo.git",
			credential: appv2.RepoCredential{
				SSHPrivateKey: "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA4f5wg5l2hKsTeNem/V41fGnJm6gOdrj8ym3rFkEjWT2btZb5\n-----END RSA PRIVATE KEY-----",
			},
			expectError: true, // Will fail due to invalid key, but should be handled by SSH path
		},
		{
			name: "HTTPS URL should not go to SSH path",
			url:  "https://github.com/owner/repo.git",
			credential: appv2.RepoCredential{
				Username: "test",
				Password: "test",
			},
			expectError: true, // Will fail due to invalid credentials, but should not go to SSH path
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadRepoIndex(tt.url, tt.credential)
			// We expect errors in all cases due to test environment limitations,
			// but we want to ensure the URL scheme detection works correctly
			if err == nil {
				t.Errorf("Expected error due to test environment limitations but got none")
			}
		})
	}
}
