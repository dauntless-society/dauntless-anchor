package ipfs

import (
	"bytes"
	"os/exec"
	"strings"
)

type Client struct {
	Bin string
}

func New(bin string) *Client {
	return &Client{Bin: bin}
}

func (c *Client) Prepare(data []byte) (string, error) {
	cmd := exec.Command(c.Bin, "add", "--pin=true", "--quiet")
	cmd.Stdin = bytes.NewReader(data)

	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

func (c *Client) Abort(cid string) error {
	cmd := exec.Command(c.Bin, "pin", "rm", cid)
	return cmd.Run()
}

