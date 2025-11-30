package application

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	appv2 "kubesphere.io/api/application/v2"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// TestSSHAuthenticationFlow demonstrates the complete SSH authentication workflow
func TestSSHAuthenticationFlow(t *testing.T) {
	// Step 1: Generate test SSH key pair
	privateKey, publicKey, err := generateTestSSHKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate test SSH key pair: %v", err)
	}

	// Step 2: Create a temporary Git repository with Helm chart
	tempRepo, err := createTestHelmRepo(t, publicKey)
	if err != nil {
		t.Fatalf("Failed to create test Helm repository: %v", err)
	}
	defer os.RemoveAll(tempRepo)

	// Step 3: Test SSH authentication with the repository
	sshURL := fmt.Sprintf("git+ssh://git@localhost:%s/%s", getTestSSHPort(), filepath.Base(tempRepo))
	
	credential := appv2.RepoCredential{
		SSHPrivateKey:    privateKey,
		SSHKeyPassphrase: "",
		SSHKnownHosts:    getTestKnownHosts(),
	}

	// Step 4: Test HelmPull with SSH
	t.Run("HelmPull with SSH", func(t *testing.T) {
		result, err := helmPullFromSSH(sshURL, credential)
		if err != nil {
			t.Errorf("HelmPull with SSH failed: %v", err)
			return
		}

		if result == nil {
			t.Error("Expected non-nil result from HelmPull")
			return
		}

		// Verify the content contains chart information
		content := result.String()
		if !strings.Contains(content, "apiVersion") || !strings.Contains(content, "name") {
			t.Error("Result does not contain valid Helm chart content")
		}

		t.Logf("Successfully pulled Helm chart via SSH: %d bytes", len(content))
	})

	// Step 5: Test LoadRepoIndex with SSH
	t.Run("LoadRepoIndex with SSH", func(t *testing.T) {
		indexURL := fmt.Sprintf("%s/index.yaml", sshURL)
		idx, err := loadRepoIndexFromSSH(indexURL, credential)
		if err != nil {
			t.Errorf("LoadRepoIndex with SSH failed: %v", err)
			return
		}

		if len(idx.Entries) == 0 {
			t.Error("Expected at least one chart entry in index")
		}

		t.Logf("Successfully loaded repository index with %d chart entries", len(idx.Entries))
	})
}

// TestSSHGetterIntegration tests the SSH getter implementation
func TestSSHGetterIntegration(t *testing.T) {
	// Generate test credentials
	privateKey, publicKey, err := generateTestSSHKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate test SSH key pair: %v", err)
	}

	// Create test repository
	tempRepo, err := createTestHelmRepo(t, publicKey)
	if err != nil {
		t.Fatalf("Failed to create test Helm repository: %v", err)
	}
	defer os.RemoveAll(tempRepo)

	// Create SSH getter
	sshGetter, err := NewSSHGetter()
	if err != nil {
		t.Fatalf("Failed to create SSH getter: %v", err)
	}

	// Set SSH authentication
	err = sshGetter.SetSSHAuth(privateKey, "", getTestKnownHosts())
	if err != nil {
		t.Fatalf("Failed to set SSH authentication: %v", err)
	}

	// Test URL conversion
	testURL := fmt.Sprintf("git+ssh://git@localhost:%s/%s", getTestSSHPort(), filepath.Base(tempRepo))
	convertedURL := convertToGitSSHURL(testURL)
	expectedURL := fmt.Sprintf("git@localhost:%s:%s", getTestSSHPort(), filepath.Base(tempRepo))

	if convertedURL != expectedURL {
		t.Errorf("URL conversion failed: got %s, expected %s", convertedURL, expectedURL)
	}

	// Test getting chart content
	result, err := sshGetter.Get(testURL)
	if err != nil {
		t.Errorf("SSH getter Get failed: %v", err)
		return
	}

	if result == nil {
		t.Error("Expected non-nil result from SSH getter")
		return
	}

	t.Logf("SSH getter successfully retrieved %d bytes", result.Len())
}

// TestEncryptedSSHKey tests passphrase-protected SSH keys
func TestEncryptedSSHKey(t *testing.T) {
	// Generate encrypted test SSH key pair
	privateKey, publicKey, passphrase, err := generateEncryptedTestSSHKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate encrypted test SSH key pair: %v", err)
	}

	// Create test repository
	tempRepo, err := createTestHelmRepo(t, publicKey)
	if err != nil {
		t.Fatalf("Failed to create test Helm repository: %v", err)
	}
	defer os.RemoveAll(tempRepo)

	// Test with correct passphrase
	credential := appv2.RepoCredential{
		SSHPrivateKey:    privateKey,
		SSHKeyPassphrase: passphrase,
		SSHKnownHosts:    getTestKnownHosts(),
	}

	sshURL := fmt.Sprintf("git+ssh://git@localhost:%s/%s", getTestSSHPort(), filepath.Base(tempRepo))
	
	_, err = helmPullFromSSH(sshURL, credential)
	if err != nil {
		t.Errorf("HelmPull with encrypted SSH key failed: %v", err)
	}

	// Test with incorrect passphrase
	wrongCredential := appv2.RepoCredential{
		SSHPrivateKey:    privateKey,
		SSHKeyPassphrase: "wrong-passphrase",
		SSHKnownHosts:    getTestKnownHosts(),
	}

	_, err = helmPullFromSSH(sshURL, wrongCredential)
	if err == nil {
		t.Error("Expected authentication to fail with wrong passphrase")
	}
}

// TestURLSchemeDetection tests automatic SSH URL detection
func TestURLSchemeDetection(t *testing.T) {
	testCases := []struct {
		url      string
		isSSH    bool
		expected string
	}{
		{
			url:      "git+ssh://git@github.com/owner/repo.git",
			isSSH:    true,
			expected: "git@github.com:owner/repo.git",
		},
		{
			url:      "ssh://git@github.com/owner/repo.git",
			isSSH:    true,
			expected: "git@github.com:owner/repo.git",
		},
		{
			url:      "https://github.com/owner/repo.git",
			isSSH:    false,
			expected: "",
		},
		{
			url:      "http://github.com/owner/repo.git",
			isSSH:    false,
			expected: "",
		},
		{
			url:      "oci://registry.example.com/chart:1.0.0",
			isSSH:    false,
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.url, func(t *testing.T) {
			parsedURL, err := url.Parse(tc.url)
			if err != nil {
				t.Errorf("Failed to parse URL %s: %v", tc.url, err)
				return
			}

			isSSH := parsedURL.Scheme == "ssh" || parsedURL.Scheme == "git+ssh"
			if isSSH != tc.isSSH {
				t.Errorf("SSH detection failed for %s: got %v, expected %v", tc.url, isSSH, tc.isSSH)
			}

			converted := convertToGitSSHURL(tc.url)
			if converted != tc.expected {
				t.Errorf("URL conversion failed for %s: got %s, expected %s", tc.url, converted, tc.expected)
			}
		})
	}
}

// Helper functions for testing

func generateTestSSHKeyPair() (string, string, error) {
	// Mock SSH key pair for testing
	privateKey := `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA4f5wg5l2hKsTeNem/V41fGnJm6gOdrj8ym3rFkEjWT2btZb5
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA4f5wg5l2hKsTeNem/V41f
GnJm6gOdrj8ym3rFkEjWT2btZb5J+2rG1jyLQAiVJ8o6KsXlU8mKv3pNYuKJz4q
-----END RSA PRIVATE KEY-----`
	
	publicKey := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC7VZV test@example.com"
	
	return privateKey, publicKey, nil
}

func generateEncryptedTestSSHKeyPair() (string, string, string, error) {
	// Mock encrypted SSH key pair for testing
	privateKey := `-----BEGIN RSA PRIVATE KEY-----
Proc-Type: 4,ENCRYPTED
DEK-Info: AES-256-CBC,ABC123DEF456GHI789JKL012MNO345PQR
MIIEpAIBAAKCAQEA4f5wg5l2hKsTeNem/V41fGnJm6gOdrj8ym3rFkEjWT2btZb5
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA4f5wg5l2hKsTeNem/V41f
GnJm6gOdrj8ym3rFkEjWT2btZb5J+2rG1jyLQAiVJ8o6KsXlU8mKv3pNYuKJz4q
-----END RSA PRIVATE KEY-----`
	
	publicKey := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC7VZV encrypted@example.com"
	passphrase := "test-passphrase-123"
	
	return privateKey, publicKey, passphrase, nil
}

func createTestHelmRepo(t *testing.T, publicKey string) (string, error) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "test-helm-repo-")
	if err != nil {
		return "", err
	}

	// Initialize Git repository
	repo, err := git.PlainInit(tempDir, false)
	if err != nil {
		os.RemoveAll(tempDir)
		return "", err
	}

	// Create .ssh directory and add public key
	sshDir := filepath.Join(tempDir, ".ssh")
	err = os.MkdirAll(sshDir, 0755)
	if err != nil {
		os.RemoveAll(tempDir)
		return "", err
	}

	authorizedKeysPath := filepath.Join(sshDir, "authorized_keys")
	err = os.WriteFile(authorizedKeysPath, []byte(publicKey), 0644)
	if err != nil {
		os.RemoveAll(tempDir)
		return "", err
	}

	// Create a sample Helm chart
	chartDir := filepath.Join(tempDir, "test-chart")
	err = os.MkdirAll(chartDir, 0755)
	if err != nil {
		os.RemoveAll(tempDir)
		return "", err
	}

	chartYaml := `apiVersion: v2
name: test-chart
description: A test Helm chart
type: application
version: 0.1.0
appVersion: "1.0.0"`

	err = os.WriteFile(filepath.Join(chartDir, "Chart.yaml"), []byte(chartYaml), 0644)
	if err != nil {
		os.RemoveAll(tempDir)
		return "", err
	}

	// Create values.yaml
	valuesYaml := `replicaCount: 1
image:
  repository: nginx
  tag: latest`

	err = os.WriteFile(filepath.Join(chartDir, "values.yaml"), []byte(valuesYaml), 0644)
	if err != nil {
		os.RemoveAll(tempDir)
		return "", err
	}

	// Create index.yaml
	indexYaml := `apiVersion: v1
entries:
  test-chart:
  - apiVersion: v2
    appVersion: "1.0.0"
    created: ` + time.Now().UTC().Format(time.RFC3339) + `
    description: A test Helm chart
    digest: ` + generateTestDigest() + `
    name: test-chart
    type: application
    urls:
    - charts/test-chart-0.1.0.tgz
    version: 0.1.0
generated: ` + time.Now().UTC().Format(time.RFC3339)

	err = os.WriteFile(filepath.Join(tempDir, "index.yaml"), []byte(indexYaml), 0644)
	if err != nil {
		os.RemoveAll(tempDir)
		return "", err
	}

	// Git operations
	worktree, err := repo.Worktree()
	if err != nil {
		os.RemoveAll(tempDir)
		return "", err
	}

	// Add all files
	_, err = worktree.Add(".")
	if err != nil {
		os.RemoveAll(tempDir)
		return "", err
	}

	// Commit
	_, err = worktree.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		os.RemoveAll(tempDir)
		return "", err
	}

	return tempDir, nil
}

func getTestSSHPort() string {
	// In a real test environment, this would be the actual SSH port
	// For demonstration, we use a mock port
	return "2222"
}

func getTestKnownHosts() string {
	// In a real test environment, this would be the actual known hosts entry
	// For demonstration, we use a mock entry
	return "localhost ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC..."
}

func generateTestDigest() string {
	// Generate a mock digest for testing
	return "a1b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef123456"
}
