package cli

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/felix-hatr/goto-browser/internal/config"
)

type profileBackup struct {
	ProfileName string
	Timestamp   string
	Path        string
}

// parseBackupEntry parses a backup directory name into profile name and timestamp.
// Format: {profileName}.{YYYYMMDD-HHMMSS}
func parseBackupEntry(name string) (profileName, timestamp string, ok bool) {
	const tsLen = 15 // "20060102-150405"
	if len(name) <= tsLen+1 {
		return "", "", false
	}
	ts := name[len(name)-tsLen:]
	if ts[8] != '-' {
		return "", "", false
	}
	return name[:len(name)-tsLen-1], ts, true
}

// listAllBackups returns all backups sorted by profile name, then timestamp descending.
func listAllBackups() ([]profileBackup, error) {
	bakDir := filepath.Join(config.ProfilesDir(), ".bak")
	entries, err := os.ReadDir(bakDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var backups []profileBackup
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		profileName, ts, ok := parseBackupEntry(e.Name())
		if !ok {
			continue
		}
		backups = append(backups, profileBackup{
			ProfileName: profileName,
			Timestamp:   ts,
			Path:        filepath.Join(bakDir, e.Name()),
		})
	}
	sort.Slice(backups, func(i, j int) bool {
		if backups[i].ProfileName != backups[j].ProfileName {
			return backups[i].ProfileName < backups[j].ProfileName
		}
		return backups[i].Timestamp > backups[j].Timestamp // newest first
	})
	return backups, nil
}

// findBackupsFor returns backups for a specific profile, newest first.
func findBackupsFor(profileName string) ([]profileBackup, error) {
	all, err := listAllBackups()
	if err != nil {
		return nil, err
	}
	var result []profileBackup
	for _, b := range all {
		if b.ProfileName == profileName {
			result = append(result, b)
		}
	}
	return result, nil
}
