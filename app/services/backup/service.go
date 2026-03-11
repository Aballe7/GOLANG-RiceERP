package backup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"egglayererp/app/config"
)

// BackupFile represents a single backup file.
type BackupFile struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
}

// GetBackupDir returns the backups directory next to the executable.
func GetBackupDir() string {
	exe, _ := os.Executable()
	dir := filepath.Join(filepath.Dir(exe), "backups")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		_ = os.MkdirAll(dir, 0755)
	}
	return dir
}

// CreateBackup runs mysqldump and saves the output to the backups directory.
// Returns the path of the created backup file.
func CreateBackup(cfg *config.Config) (string, error) {
	// Parse DSN to extract host, port, user, password, dbname
	// DSN format: user:pass@tcp(host:port)/dbname?...
	dsn := cfg.Database.DSN
	user, pass, host, port, dbname, err := parseDSN(dsn)
	if err != nil {
		return "", fmt.Errorf("cannot parse DSN for backup: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("egglayererp_backup_%s.sql", timestamp)
	backupPath := filepath.Join(GetBackupDir(), filename)

	args := []string{
		fmt.Sprintf("--host=%s", host),
		fmt.Sprintf("--port=%s", port),
		fmt.Sprintf("--user=%s", user),
		fmt.Sprintf("--password=%s", pass),
		"--single-transaction",
		"--routines",
		"--triggers",
		"--result-file=" + backupPath,
		dbname,
	}

	cmd := exec.Command("mysqldump", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("mysqldump failed: %s\n%s", err.Error(), string(output))
	}

	// Rotate: keep only last 10 backups
	_ = rotateBackups(GetBackupDir(), 10)

	return backupPath, nil
}

// ListBackups returns all backup files sorted by creation time (newest first).
func ListBackups() ([]BackupFile, error) {
	dir := GetBackupDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []BackupFile
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		info, _ := e.Info()
		files = append(files, BackupFile{
			Name:      e.Name(),
			Path:      filepath.Join(dir, e.Name()),
			Size:      info.Size(),
			CreatedAt: info.ModTime(),
		})
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].CreatedAt.After(files[j].CreatedAt)
	})
	return files, nil
}

// DeleteBackup removes a backup file by name.
func DeleteBackup(name string) error {
	// Security: only allow .sql files in the backups directory
	if !strings.HasSuffix(name, ".sql") || strings.Contains(name, "..") {
		return fmt.Errorf("invalid backup filename")
	}
	path := filepath.Join(GetBackupDir(), name)
	return os.Remove(path)
}

func rotateBackups(dir string, keep int) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	var files []os.FileInfo
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			info, _ := e.Info()
			files = append(files, info)
		}
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].ModTime().After(files[j].ModTime())
	})
	for i := keep; i < len(files); i++ {
		_ = os.Remove(filepath.Join(dir, files[i].Name()))
	}
	return nil
}

// parseDSN extracts connection parameters from a GORM MySQL DSN string.
// Format: user:pass@tcp(host:port)/dbname?...
func parseDSN(dsn string) (user, pass, host, port, dbname string, err error) {
	// Strip query params
	if idx := strings.Index(dsn, "?"); idx >= 0 {
		dsn = dsn[:idx]
	}

	// user:pass@tcp(host:port)/dbname
	atIdx := strings.LastIndex(dsn, "@")
	if atIdx < 0 {
		err = fmt.Errorf("missing @ in DSN")
		return
	}
	userPass := dsn[:atIdx]
	rest := dsn[atIdx+1:]

	colonIdx := strings.Index(userPass, ":")
	if colonIdx < 0 {
		user = userPass
	} else {
		user = userPass[:colonIdx]
		pass = userPass[colonIdx+1:]
	}

	// rest: tcp(host:port)/dbname
	if !strings.HasPrefix(rest, "tcp(") {
		err = fmt.Errorf("expected tcp(host:port) in DSN")
		return
	}
	rest = rest[4:] // strip "tcp("
	closeIdx := strings.Index(rest, ")")
	if closeIdx < 0 {
		err = fmt.Errorf("missing ) in DSN")
		return
	}
	hostPort := rest[:closeIdx]
	rest = rest[closeIdx+2:] // strip ")/", dbname follows

	hpIdx := strings.LastIndex(hostPort, ":")
	if hpIdx < 0 {
		host = hostPort
		port = "3306"
	} else {
		host = hostPort[:hpIdx]
		port = hostPort[hpIdx+1:]
	}
	dbname = rest
	return
}
