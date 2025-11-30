/*
 * Copyright 2024 the KubeSphere Authors.
 * Please refer to the LICENSE file in the root directory of the project.
 * https://github.com/kubesphere/kubesphere/blob/master/LICENSE
 */

package application

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/pkg/errors"
	"k8s.io/klog/v2"
)

// SSHGetter is a getter implementation for Git repositories using SSH protocol
type SSHGetter struct {
	sshAuthMethod *ssh.PublicKeys
	knownHosts    string
}

// NewSSHGetter creates a new SSH getter
func NewSSHGetter() (*SSHGetter, error) {
	return &SSHGetter{}, nil
}

// Get retrieves the content from a Git repository via SSH
func (g *SSHGetter) Get(urlStr string) (*bytes.Buffer, error) {
	// Extract repository path from URL
	repoURL := convertToGitSSHURL(urlStr)
	if repoURL == "" {
		return nil, errors.Errorf("unsupported SSH URL format: %s", urlStr)
	}

	klog.Infof("Cloning SSH repository: %s", repoURL)

	// Create temporary directory for cloning
	tempDir, err := os.MkdirTemp("", "helm-getter-ssh-")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create temporary directory")
	}
	defer os.RemoveAll(tempDir)

	// Clone the repository
	_, err = git.PlainClone(tempDir, false, &git.CloneOptions{
		URL:  repoURL,
		Auth: g.sshAuthMethod,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to clone repository %s", repoURL)
	}

	// Look for Chart.yaml or index.yaml in the repository
	chartPath, err := findChartInRepo(tempDir)
	if err != nil {
		return nil, err
	}

	// Read the chart file
	content, err := os.ReadFile(chartPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read chart file %s", chartPath)
	}

	klog.Infof("Successfully retrieved chart from SSH repository: %s", chartPath)
	return bytes.NewBuffer(content), nil
}

// SetSSHAuth sets the SSH authentication method
func (g *SSHGetter) SetSSHAuth(privateKey, passphrase, knownHosts string) error {
	if privateKey == "" {
		return errors.New("SSH private key is required")
	}

	// Create SSH auth method from private key string using go-git's SSH package
	var authMethod *ssh.PublicKeys
	var err error

	if passphrase != "" {
		authMethod, err = ssh.NewPublicKeys("git", []byte(privateKey), passphrase)
	} else {
		authMethod, err = ssh.NewPublicKeys("git", []byte(privateKey), "")
	}
	
	if err != nil {
		return errors.Wrap(err, "failed to create SSH auth method")
	}

	g.sshAuthMethod = authMethod
	g.knownHosts = knownHosts
	return nil
}

// convertToGitSSHURL converts various SSH URL formats to git@host:repo format
func convertToGitSSHURL(urlStr string) string {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return ""
	}

	switch parsedURL.Scheme {
	case "ssh":
		// ssh://git@github.com/owner/repo.git -> git@github.com:owner/repo.git
		if strings.HasPrefix(parsedURL.Host, "git@") {
			host := strings.TrimPrefix(parsedURL.Host, "git@")
			return fmt.Sprintf("git@%s:%s", host, strings.TrimPrefix(parsedURL.Path, "/"))
		}
		return fmt.Sprintf("git@%s:%s", parsedURL.Host, strings.TrimPrefix(parsedURL.Path, "/"))
	case "git+ssh":
		// git+ssh://git@github.com/owner/repo.git -> git@github.com:owner/repo.git
		host := strings.TrimPrefix(parsedURL.Host, "git@")
		return fmt.Sprintf("git@%s:%s", host, strings.TrimPrefix(parsedURL.Path, "/"))
	default:
		return ""
	}
}

// findChartInRepo searches for Chart.yaml or index.yaml in the repository
func findChartInRepo(repoDir string) (string, error) {
	var candidates []string

	// Look for Chart.yaml
	err := filepath.Walk(repoDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Name() == "Chart.yaml" {
			candidates = append(candidates, path)
		}
		return nil
	})

	if err != nil {
		return "", errors.Wrap(err, "failed to walk repository directory")
	}

	// Look for index.yaml if no Chart.yaml found
	if len(candidates) == 0 {
		err = filepath.Walk(repoDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.Name() == "index.yaml" {
				candidates = append(candidates, path)
			}
			return nil
		})

		if err != nil {
			return "", errors.Wrap(err, "failed to walk repository directory")
		}
	}

	if len(candidates) == 0 {
		return "", errors.New("no Chart.yaml or index.yaml found in repository")
	}

	// Return the first candidate (could be made smarter to find the root chart)
	return candidates[0], nil
}
