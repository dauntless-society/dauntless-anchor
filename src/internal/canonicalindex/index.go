package canonicalindex

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"api.dauntless-society.com/anchor/internal/ipfs"
)

const SchemaV1 = "dauntless-anchor.canonical-index.v1"
const SchemaV2 = "dauntless-anchor.canonical-index.v2"

type Index struct {
	Schema           string  `json:"schema"`
	IndexVersion     int     `json:"index_version"`
	CreatedAt        string  `json:"created_at"`
	PreviousIndexCID string  `json:"previous_index_cid,omitempty"`
	Entries          []Entry `json:"entries"`
}

type Entry struct {
	DocumentID       string `json:"document_id"`
	DocumentVersion  int    `json:"document_version"`
	DocumentSHA256   string `json:"document_sha256"`
	DocumentSHA512   string `json:"document_sha512"`
	DocumentSHA3_512 string `json:"document_sha3_512"`
	DocumentCID      string `json:"document_cid"`
	Author           string `json:"author,omitempty"`
	Timestamp        string `json:"timestamp"`
}

type Store struct {
	Dir string
}

func NewStore(dir string) *Store {
	return &Store{Dir: dir}
}

func (s *Store) Ensure() error {
	return os.MkdirAll(s.Dir, 0755)
}

func (s *Store) latestJSONPath() string { return filepath.Join(s.Dir, "latest.json") }
func (s *Store) latestCIDPath() string  { return filepath.Join(s.Dir, "latest.cid") }
func (s *Store) lockPath() string       { return filepath.Join(s.Dir, "index.lock") }

func (s *Store) LoadLatest() (idx *Index, indexCID string, err error) {
	data, err := os.ReadFile(s.latestJSONPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, "", nil
		}
		return nil, "", err
	}

	var parsed Index
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, "", err
	}

	cidBytes, err := os.ReadFile(s.latestCIDPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &parsed, "", nil
		}
		return nil, "", err
	}

	return &parsed, strings.TrimSpace(string(cidBytes)), nil
}

func (s *Store) Append(ipfsClient *ipfs.Client, entry Entry) (newIndexCID string, newIndexVersion int, err error) {
	if ipfsClient == nil {
		return "", 0, errors.New("ipfs client required")
	}
	if entry.DocumentID == "" || entry.DocumentSHA256 == "" || entry.DocumentSHA512 == "" || entry.DocumentSHA3_512 == "" || entry.DocumentCID == "" || entry.Timestamp == "" {
		return "", 0, errors.New("entry missing required fields")
	}

	if err := s.Ensure(); err != nil {
		return "", 0, err
	}

	lockFile, err := os.OpenFile(s.lockPath(), os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return "", 0, err
	}
	defer lockFile.Close()

	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX); err != nil {
		return "", 0, err
	}
	defer func() { _ = syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN) }()

	current, currentCID, err := s.LoadLatest()
	if err != nil {
		return "", 0, err
	}

	previousCID := ""
	previousVersion := 0
	previousEntries := []Entry{}
	if current != nil {
		previousVersion = current.IndexVersion
		previousEntries = append(previousEntries, current.Entries...)
		previousCID = currentCID
	}

	newIndexVersion = previousVersion + 1

	newIndex := Index{
		Schema:           SchemaV2,
		IndexVersion:     newIndexVersion,
		CreatedAt:        time.Now().UTC().Format(time.RFC3339Nano),
		PreviousIndexCID: previousCID,
		Entries:          append(previousEntries, entry),
	}

	jsonBytes, err := json.MarshalIndent(newIndex, "", "  ")
	if err != nil {
		return "", 0, err
	}
	jsonBytes = append(jsonBytes, '\n')

	versionBase := fmt.Sprintf("index-v%06d", newIndexVersion)
	versionJSONFinal := filepath.Join(s.Dir, versionBase+".json")
	versionCIDFinal := filepath.Join(s.Dir, versionBase+".cid")

	versionJSONTmp := versionJSONFinal + ".tmp"
	versionCIDTmp := versionCIDFinal + ".tmp"
	latestJSONTmp := s.latestJSONPath() + ".tmp"
	latestCIDTmp := s.latestCIDPath() + ".tmp"

	cleanupTemps := func() {
		_ = os.Remove(versionJSONTmp)
		_ = os.Remove(versionCIDTmp)
		_ = os.Remove(latestJSONTmp)
		_ = os.Remove(latestCIDTmp)
	}
	cleanupTemps()

	if err := os.WriteFile(versionJSONTmp, jsonBytes, 0644); err != nil {
		cleanupTemps()
		return "", 0, err
	}
	if err := os.WriteFile(latestJSONTmp, jsonBytes, 0644); err != nil {
		cleanupTemps()
		return "", 0, err
	}

	newIndexCID, err = ipfsClient.Prepare(jsonBytes)
	if err != nil {
		cleanupTemps()
		return "", 0, err
	}

	if err := os.WriteFile(versionCIDTmp, []byte(newIndexCID+"\n"), 0644); err != nil {
		_ = ipfsClient.Abort(newIndexCID)
		cleanupTemps()
		return "", 0, err
	}
	if err := os.WriteFile(latestCIDTmp, []byte(newIndexCID+"\n"), 0644); err != nil {
		_ = ipfsClient.Abort(newIndexCID)
		cleanupTemps()
		return "", 0, err
	}

	if err := os.Rename(versionJSONTmp, versionJSONFinal); err != nil {
		_ = ipfsClient.Abort(newIndexCID)
		cleanupTemps()
		return "", 0, err
	}
	if err := os.Rename(versionCIDTmp, versionCIDFinal); err != nil {
		_ = ipfsClient.Abort(newIndexCID)
		cleanupTemps()
		return "", 0, err
	}
	if err := os.Rename(latestJSONTmp, s.latestJSONPath()); err != nil {
		_ = ipfsClient.Abort(newIndexCID)
		cleanupTemps()
		return "", 0, err
	}
	if err := os.Rename(latestCIDTmp, s.latestCIDPath()); err != nil {
		_ = ipfsClient.Abort(newIndexCID)
		cleanupTemps()
		return "", 0, err
	}

	return newIndexCID, newIndexVersion, nil
}
