package bitcoin

import (
	"os/exec"
	"strings"
)

type Client struct {
	CLI     string
	DataDir string
	Address string
	FeeBTC  string
}

func New(cli, datadir, address, fee string) *Client {
	return &Client{
		CLI:     cli,
		DataDir: datadir,
		Address: address,
		FeeBTC:  fee,
	}
}

func (c *Client) Commit(hash string) (string, error) {
	cmd := exec.Command(
		c.CLI,
		"-datadir="+c.DataDir,
		"sendtoaddress",
		c.Address,
		c.FeeBTC,
		"",
		"",
		"false",
		"true",
		"1",
		"UNSET",
		hash,
	)

	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}
