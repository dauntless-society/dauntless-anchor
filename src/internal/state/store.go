package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var jobDir = "/var/lib/dauntless/anchor/jobs"

func SetJobDir(dir string) {
	if dir != "" {
		jobDir = dir
	}
}

func ensureJobDir() error {
	return os.MkdirAll(jobDir, 0755)
}

func Save(job AnchorJob) error {
	if err := ensureJobDir(); err != nil {
		return err
	}

	job.UpdatedAt = time.Now().UTC()

	path := filepath.Join(jobDir, fmt.Sprintf("%s.json", job.ID))
	data, err := json.MarshalIndent(job, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
