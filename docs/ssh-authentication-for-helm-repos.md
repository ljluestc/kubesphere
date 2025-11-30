# SSH Authentication for Helm Chart Repositories

This document describes the SSH authentication feature for accessing Helm Chart repositories hosted on GitHub Enterprise (GHE) or other Git servers using SSH keys.

## Overview

KubeSphere now supports SSH-based authentication for Helm Chart repositories, allowing you to:
- Access private Git repositories using SSH keys
- Deploy applications from `git+ssh://` and `ssh://` URLs
- Use encrypted SSH private keys with passphrases
- Verify SSH host keys using known hosts

## Configuration

### RepoCredential Structure

The `RepoCredential` structure has been extended with SSH authentication fields:

```yaml
apiVersion: application.kubesphere.io/v2
kind: Repo
metadata:
  name: my-ssh-repo
spec:
  url: "git+ssh://git@github.enterprise.com/my-org/my-helm-charts.git"
  credential:
    # SSH private key for authentication (required for SSH URLs)
    sshPrivateKey: |
      -----BEGIN RSA PRIVATE KEY-----
      MIIEpAIBAAKCAQEA4f5wg5l2hKsTeNem/V41fGnJm6gOdrj8ym3rFkEjWT2btZb5
      ... (your private key content) ...
      -----END RSA PRIVATE KEY-----
    
    # SSH key passphrase (optional, for encrypted private keys)
    sshKeyPassphrase: "your-passphrase"
    
    # Known hosts content for host key verification (optional)
    sshKnownHosts: |
      github.enterprise.com ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC...
    
    # Traditional authentication fields (still supported for HTTP/HTTPS)
    username: ""  # Not used for SSH URLs
    password: ""  # Not used for SSH URLs
```

### Supported URL Formats

The following URL formats are supported for SSH authentication:

1. **git+ssh://** (recommended)
   ```
   git+ssh://git@github.enterprise.com/organization/helm-charts.git
   ```

2. **ssh://**
   ```
   ssh://git@github.enterprise.com/organization/helm-charts.git
   ```

### SSH Key Requirements

- **Private Key Format**: PEM format (RSA, ECDSA, or Ed25519)
- **Key Type**: Must support SSH authentication
- **Permissions**: Private key should have appropriate file permissions (600 or 400)
- **Passphrase**: Optional, supported for encrypted private keys

## Usage Examples

### Example 1: Basic SSH Repository

```yaml
apiVersion: application.kubesphere.io/v2
kind: Repo
metadata:
  name: private-helm-repo
  namespace: default
spec:
  url: "git+ssh://git@github.com/organization/helm-charts.git"
  credential:
    sshPrivateKey: |
      -----BEGIN RSA PRIVATE KEY-----
      MIIEpAIBAAKCAQEA4f5wg5l2hKsTeNem/V41fGnJm6gOdrj8ym3rFkEjWT2btZb5
      ... (your complete private key) ...
      -----END RSA PRIVATE KEY-----
```

### Example 2: SSH with Passphrase

```yaml
apiVersion: application.kubesphere.io/v2
kind: Repo
metadata:
  name: encrypted-ssh-repo
  namespace: default
spec:
  url: "git+ssh://git@github.enterprise.com/my-org/my-charts.git"
  credential:
    sshPrivateKey: |
      -----BEGIN RSA PRIVATE KEY-----
      Proc-Type: 4,ENCRYPTED
      DEK-Info: AES-256-CBC,ABC123...
      ... (your encrypted private key) ...
      -----END RSA PRIVATE KEY-----
    sshKeyPassphrase: "my-secret-passphrase"
```

### Example 3: SSH with Known Hosts Verification

```yaml
apiVersion: application.kubesphere.io/v2
kind: Repo
metadata:
  name: secure-ssh-repo
  namespace: default
spec:
  url: "git+ssh://git@github.enterprise.com/organization/helm-charts.git"
  credential:
    sshPrivateKey: |
      -----BEGIN RSA PRIVATE KEY-----
      MIIEpAIBAAKCAQEA4f5wg5l2hKsTeNem/V41fGnJm6gOdrj8ym3rFkEjWT2btZb5
      ... (your private key content) ...
      -----END RSA PRIVATE KEY-----
    sshKnownHosts: |
      github.enterprise.com ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC...
      github.enterprise.com ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTY...
```

## Security Considerations

### Private Key Management

1. **Store Securely**: Always store SSH private keys in Kubernetes secrets, not directly in YAML files
2. **Use Deploy Keys**: Create read-only deploy keys in your Git repository for better security
3. **Key Rotation**: Regularly rotate your SSH keys
4. **Minimum Permissions**: Use keys with the minimum necessary permissions

### Example with Kubernetes Secret

```yaml
# Create a secret with SSH key
apiVersion: v1
kind: Secret
metadata:
  name: ssh-repo-credentials
type: Opaque
data:
  sshPrivateKey: <base64-encoded-private-key>
  sshKeyPassphrase: <base64-encoded-passphrase>  # optional
  sshKnownHosts: <base64-encoded-known-hosts>  # optional
```

```yaml
# Reference the secret in your Repo
apiVersion: application.kubesphere.io/v2
kind: Repo
metadata:
  name: secure-repo
spec:
  url: "git+ssh://git@github.enterprise.com/organization/helm-charts.git"
  credential:
    sshPrivateKey: |
      {{ .Values.sshPrivateKey | quote }}
    sshKeyPassphrase: |
      {{ .Values.sshKeyPassphrase | quote }}
```

### Host Key Verification

- Always provide `sshKnownHosts` for production environments
- Obtain known hosts entries using: `ssh-keyscan -t rsa,ecdsa github.enterprise.com`
- Avoid disabling host key verification in production

## Troubleshooting

### Common Issues

1. **Private Key Format Error**
   ```
   Error: failed to parse SSH private key
   ```
   - Ensure the key is in valid PEM format
   - Check for extra whitespace or line breaks
   - Verify the key is not corrupted

2. **Authentication Failed**
   ```
   Error: failed to clone repository
   ```
   - Verify the SSH key has access to the repository
   - Check if the deploy key is properly configured
   - Ensure the URL format is correct

3. **Host Key Verification Failed**
   ```
   Error: SSH host key verification failed
   ```
   - Add the correct host key to `sshKnownHosts`
   - Verify the hostname matches the known hosts entry
   - Check for man-in-the-middle attacks

### Debug Commands

```bash
# Test SSH connection manually
ssh -T git@github.enterprise.com

# Check SSH key format
ssh-keygen -l -f ~/.ssh/id_rsa

# Get known hosts entry
ssh-keyscan -t rsa,ecdsa github.enterprise.com
```

## Migration from HTTP/HTTPS

To migrate existing HTTP/HTTPS repositories to SSH:

1. **Generate SSH Key Pair**
   ```bash
   ssh-keygen -t rsa -b 4096 -C "kubesphere@mycompany.com"
   ```

2. **Add Deploy Key to Repository**
   - Add the public key as a deploy key in your Git repository
   - Grant read-only access for Helm chart operations

3. **Update Repository Configuration**
   ```yaml
   # Before
   url: "https://github.enterprise.com/organization/helm-charts.git"
   credential:
     username: "git"
     password: "personal-access-token"
   
   # After
   url: "git+ssh://git@github.enterprise.com/organization/helm-charts.git"
   credential:
     sshPrivateKey: |
       -----BEGIN RSA PRIVATE KEY-----
       ... (your private key) ...
       -----END RSA PRIVATE KEY-----
   ```

## API Reference

### RepoCredential Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `sshPrivateKey` | string | Yes (for SSH URLs) | PEM-encoded SSH private key |
| `sshKeyPassphrase` | string | No | Passphrase for encrypted private keys |
| `sshKnownHosts` | string | No | Known hosts entries for verification |
| `username` | string | No | Username for HTTP/HTTPS repositories |
| `password` | string | No | Password/token for HTTP/HTTPS repositories |

### Supported Schemes

| Scheme | Example | Authentication |
|--------|---------|----------------|
| `git+ssh` | `git+ssh://git@host.com/repo.git` | SSH private key |
| `ssh` | `ssh://git@host.com/repo.git` | SSH private key |
| `https` | `https://host.com/repo.git` | Username/password or TLS certs |
| `http` | `http://host.com/repo.git` | Username/password or TLS certs |
| `oci` | `oci://host.com/repo/chart` | Basic auth or TLS certs |

## Implementation Details

The SSH authentication implementation:

1. **URL Detection**: Automatically detects SSH URLs and routes them to SSH getter
2. **Key Parsing**: Supports both encrypted and unencrypted private keys
3. **Git Integration**: Uses `go-git` library for Git operations
4. **Temporary Storage**: Uses temporary directories for cloning and cleanup
5. **Chart Discovery**: Automatically finds `Chart.yaml` or `index.yaml` in repositories

For more technical details, refer to the implementation in:
- `pkg/simple/client/application/ssh_getter.go`
- `pkg/simple/client/application/helper.go`
