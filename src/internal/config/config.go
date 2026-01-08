package config

import (
	"bufio"
	"os"
	"strings"
)

type Config struct {
	ListenAddr     string
	ListenPort     string
	IPFSBin        string
	BitcoinCLI     string
	BitcoinDataDir string
	BitcoinAddress string
	BitcoinFeeBTC  string
	StagingDir     string
	LogDir         string
}

func Load(path string) (*Config, error) {
	cfg := &Config{}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch key {
		case "LISTEN_ADDR":
			cfg.ListenAddr = val
		case "LISTEN_PORT":
			cfg.ListenPort = val
		case "IPFS_BIN":
			cfg.IPFSBin = val
		case "BITCOIN_CLI":
			cfg.BitcoinCLI = val
		case "BITCOIN_DATADIR":
			cfg.BitcoinDataDir = val
		case "BITCOIN_ADDRESS":
			cfg.BitcoinAddress = val
		case "BITCOIN_FEE_BTC":
			cfg.BitcoinFeeBTC = val
		case "STAGING_DIR":
			cfg.StagingDir = val
		case "LOG_DIR":
			cfg.LogDir = val
		}
	}

	return cfg, scanner.Err()
}
