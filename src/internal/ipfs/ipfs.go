package ipfs

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Client struct {
	Bin      string
	RepoPath string
}

func New(bin string, repoPath string) *Client {
	if repoPath == "" {
		repoPath = "/var/lib/dauntless/ipfs"
	}
	return &Client{Bin: bin, RepoPath: repoPath}
}

func (c *Client) Prepare(data []byte) (string, error) {
	cmd := exec.Command(c.Bin, "add", "--pin=true", "--quiet", "--cid-version=1")
	cmd.Stdin = bytes.NewReader(data)

	// Explicit repo location (required for no-login users)
	cmd.Env = append(os.Environ(), "IPFS_PATH="+c.RepoPath)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ipfs add failed: %s", strings.TrimSpace(string(out)))
	}

	return strings.TrimSpace(string(out)), nil
}

func (c *Client) Abort(cid string) error {
	cmd := exec.Command(c.Bin, "pin", "rm", cid)
	cmd.Env = append(os.Environ(), "IPFS_PATH="+c.RepoPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ipfs pin rm failed: %s", strings.TrimSpace(string(out)))
	}
	return nil
}
