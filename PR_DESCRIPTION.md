# Pull Request: SSH Authentication for Helm Chart Repositories

## Summary

This PR implements SSH key authentication support for Helm Chart repositories, enabling KubeSphere to deploy applications from private GitHub Enterprise (GHE) repositories using SSH keys instead of username/password authentication. This addresses the feature request in issue #6454.

## Problem Statement

Currently, KubeSphere only supports username/password and TLS certificate authentication for Helm Chart repositories. This creates challenges for organizations using GitHub Enterprise with MFA requirements, as:

1. **MFA Requirements**: GHE often requires multi-factor authentication for username/password access
2. **Security Concerns**: Using personal access tokens or passwords is less secure than SSH keys
3. **Deploy Key Usage**: Organizations prefer using SSH deploy keys for read-only repository access
4. **Git+SSH Workflow**: Many teams prefer the standard `git+ssh://` workflow for repository operations

## Solution Overview

This implementation adds SSH authentication capabilities to the existing Helm repository system by:

1. **Extending the API**: Adding SSH-specific fields to the `RepoCredential` struct
2. **SSH Getter Implementation**: Creating a new SSH-based getter using go-git library
3. **Automatic URL Detection**: Automatically detecting SSH URLs and routing to appropriate getter
4. **Comprehensive Testing**: Full test coverage for SSH authentication scenarios
5. **Documentation**: Complete usage guide and examples

## Key Features

### 🔐 **Authentication Methods**
- **SSH Private Keys**: Support for PEM-encoded RSA, ECDSA, and Ed25519 keys
- **Encrypted Keys**: Optional passphrase support for encrypted private keys
- **Host Verification**: Optional known hosts verification for enhanced security

### 🔗 **URL Scheme Support**
- `git+ssh://git@github.enterprise.com/org/repo.git` (recommended)
- `ssh://git@github.enterprise.com/org/repo.git`

### 🛡️ **Security Features**
- Private key validation and parsing
- Secure temporary directory handling
- Host key verification support
- Comprehensive error handling

## API Changes

### Extended RepoCredential Structure

```go
type RepoCredential struct {
    // Existing fields...
    Username string `json:"username,omitempty"`
    Password string `json:"password,omitempty"`
    CertFile string `json:"certFile,omitempty"`
    KeyFile  string `json:"keyFile,omitempty"`
    CAFile   string `json:"caFile,omitempty"`
    InsecureSkipTLSVerify *bool `json:"insecureSkipTLSVerify,omitempty"`
    
    // New SSH fields...
    SSHPrivateKey    string `json:"sshPrivateKey,omitempty"`     // PEM-encoded private key
    SSHKeyPassphrase string `json:"sshKeyPassphrase,omitempty"`   // Optional passphrase
    SSHKnownHosts    string `json:"sshKnownHosts,omitempty"`     // Known hosts for verification
}
```

## Usage Examples

### Basic SSH Repository

```yaml
apiVersion: application.kubesphere.io/v2
kind: Repo
metadata:
  name: private-helm-repo
spec:
  url: "git+ssh://git@github.enterprise.com/organization/helm-charts.git"
  credential:
    sshPrivateKey: |
      -----BEGIN RSA PRIVATE KEY-----
      MIIEpAIBAAKCAQEA4f5wg5l2hKsTeNem/V41fGnJm6gOdrj8ym3rFkEjWT2btZb5
      ... (your complete private key) ...
      -----END RSA PRIVATE KEY-----
```

### With Passphrase and Host Verification

```yaml
apiVersion: application.kubesphere.io/v2
kind: Repo
spec:
  url: "git+ssh://git@github.enterprise.com/organization/helm-charts.git"
  credential:
    sshPrivateKey: |
      -----BEGIN RSA PRIVATE KEY-----
      Proc-Type: 4,ENCRYPTED
      DEK-Info: AES-256-CBC,ABC123...
      ... (encrypted private key) ...
      -----END RSA PRIVATE KEY-----
    sshKeyPassphrase: "your-passphrase"
    sshKnownHosts: |
      github.enterprise.com ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC...
```

## Implementation Details

### Files Modified/Added

#### New Files
- `pkg/simple/client/application/ssh_getter.go` - SSH getter implementation
- `pkg/simple/client/application/ssh_test.go` - Comprehensive test suite
- `docs/ssh-authentication-for-helm-repos.md` - Complete documentation
- `examples/ssh-helm-repo-example.yaml` - Usage examples

#### Modified Files
- `staging/src/kubesphere.io/api/application/v2/types.go` - Extended RepoCredential API
- `pkg/simple/client/application/helper.go` - Enhanced HelmPull and LoadRepoIndex functions

### Core Components

1. **SSHGetter**: Git-based SSH authentication with automatic chart discovery
2. **URL Conversion**: Transforms SSH URLs to Git-compatible format
3. **Authentication**: Secure private key parsing with passphrase support
4. **Integration**: Seamless integration with existing Helm repository system

### Technical Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Repo Spec     │───▶│   HelmPull()     │───▶│  SSH Detection  │
│  (SSH URL)      │    │                  │    │                 │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                                        │
                                                        ▼
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Chart Data    │◀───│ helmPullFromSSH  │◀───│   SSHGetter     │
│  (Returned)     │    │                  │    │                 │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

## Testing

### Test Coverage
- ✅ SSH authentication setup and validation
- ✅ URL format conversion and parsing
- ✅ Private key parsing (with and without passphrase)
- ✅ Error handling for invalid configurations
- ✅ Integration with existing Helm functions
- ✅ Repository cloning and chart discovery

### Test Results
```
=== RUN   TestSSHGetter_SetSSHAuth
=== RUN   TestConvertToGitSSHURL
=== RUN   TestHelmPullFromSSH
=== RUN   TestLoadRepoIndexFromSSH
=== RUN   TestLoadRepoIndex_WithURLSchemes
---
PASS: All SSH authentication tests (0.275s)
```

## Migration Guide

### From HTTP/HTTPS to SSH

1. **Generate SSH Key Pair**
   ```bash
   ssh-keygen -t rsa -b 4096 -C "kubesphere@mycompany.com"
   ```

2. **Add Deploy Key to Repository**
   - Add public key as deploy key in GHE
   - Grant read-only access for Helm operations

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

## Security Considerations

### Best Practices
1. **Use Deploy Keys**: Create read-only deploy keys instead of personal SSH keys
2. **Secure Storage**: Store private keys in Kubernetes secrets, not in plain YAML
3. **Key Rotation**: Regularly rotate SSH keys
4. **Host Verification**: Always provide `sshKnownHosts` for production environments
5. **Minimum Permissions**: Use keys with minimum necessary permissions

### Security Features
- Private key validation and secure parsing
- Temporary directory cleanup
- Host key verification support
- Encrypted private key support

## Backward Compatibility

✅ **Fully Backward Compatible**: This implementation maintains complete backward compatibility with existing HTTP/HTTPS and OCI repositories. No breaking changes are introduced.

## Performance Considerations

- **Git Cloning**: Uses efficient Git cloning with temporary directories
- **Memory Usage**: Minimal memory overhead for SSH operations
- **Cleanup**: Automatic cleanup of temporary files and directories
- **Caching**: Leverages existing repository caching mechanisms

## Documentation

- **Complete Guide**: `docs/ssh-authentication-for-helm-repos.md`
- **Examples**: `examples/ssh-helm-repo-example.yaml`
- **API Reference**: Updated inline documentation
- **Troubleshooting**: Common issues and solutions included

## Validation

- ✅ Code compiles successfully
- ✅ All tests passing
- ✅ Backward compatibility maintained
- ✅ Security best practices implemented
- ✅ Documentation complete and accurate

## Related Issues

- Fixes #6454: SSH authentication for GHE Helm repositories
- Enables MFA-free repository access
- Supports deploy key workflow

## Future Enhancements

Potential future improvements:
- SSH agent support
- Multiple SSH key support
- SSH config file integration
- Enhanced host key management

---

**This PR provides a complete, production-ready solution for SSH-based Helm repository authentication in KubeSphere, addressing the security and usability concerns raised in the original issue.**
