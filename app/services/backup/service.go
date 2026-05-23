package backup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"ricemill/app/config"
)

// BackupFile represents a single backup file.
type BackupFile struct {
	FileName      string    `json:"file_name"`
	Path          string    `json:"path"`
	FileSizeBytes int64     `json:"file_size_bytes"`
	CreatedAt     time.Time `json:"created_at"`
}

// GetBackupDir returns a stable backups directory under the user's home folder.
// Using the home directory avoids the temp-path problem in Wails dev mode.
func GetBackupDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback: next to executable
		exe, _ := os.Executable()
		home = filepath.Dir(exe)
	}
	dir := filepath.Join(home, "EggLayerERP", "backups")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		_ = os.MkdirAll(dir, 0755)
	}
	return dir
}

// SuggestedFilename returns a timestamped backup filename.
func SuggestedFilename() string {
	return fmt.Sprintf("egglayererp_backup_%s.sql", time.Now().Format("20060102_150405"))
}

// CreateBackup runs mysqldump to the default backups directory.
func CreateBackup(cfg *config.Config) (BackupFile, error) {
	return CreateBackupToPath(cfg, filepath.Join(GetBackupDir(), SuggestedFilename()))
}

// findMysqldump locates the mysqldump executable.
// It checks PATH first, then falls back to known Windows MySQL installation directories.
func findMysqldump() (string, error) {
	// 1. Try PATH first
	if path, err := exec.LookPath("mysqldump"); err == nil {
		return path, nil
	}

	// 2. Check common Windows MySQL installation paths
	candidates := []string{
		`C:\Program Files\MySQL\MySQL Server 9.6\bin\mysqldump.exe`,
		`C:\Program Files\MySQL\MySQL Server 9.0\bin\mysqldump.exe`,
		`C:\Program Files\MySQL\MySQL Server 8.4\bin\mysqldump.exe`,
		`C:\Program Files\MySQL\MySQL Server 8.0\bin\mysqldump.exe`,
		`C:\Program Files (x86)\MySQL\MySQL Server 8.0\bin\mysqldump.exe`,
		`C:\xampp\mysql\bin\mysqldump.exe`,
		`C:\wamp64\bin\mysql\mysql8.0\bin\mysqldump.exe`,
	}

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	return "", fmt.Errorf(
		"mysqldump not found. Add MySQL's bin folder to your system PATH, " +
			"or install MySQL Server. Checked: PATH and common installation directories",
	)
}

// CreateBackupToPath runs mysqldump and writes the dump to destPath.
// If destPath is inside the managed backups directory, rotation is applied.
func CreateBackupToPath(cfg *config.Config, destPath string) (BackupFile, error) {
	dsn := cfg.Database.DSN
	user, pass, host, port, dbname, err := parseDSN(dsn)
	if err != nil {
		return BackupFile{}, fmt.Errorf("cannot parse DSN for backup: %w", err)
	}

	// Locate mysqldump binary
	mysqldumpPath, err := findMysqldump()
	if err != nil {
		return BackupFile{}, err
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return BackupFile{}, fmt.Errorf("cannot create destination directory: %w", err)
	}

	args := []string{
		fmt.Sprintf("--host=%s", host),
		fmt.Sprintf("--port=%s", port),
		fmt.Sprintf("--user=%s", user),
		fmt.Sprintf("--password=%s", pass),
		// --single-transaction requires RELOAD/FLUSH_TABLES privilege (MySQL 8.0.32+).
		// --skip-lock-tables works for InnoDB (MVCC-based) without elevated privileges.
		"--skip-lock-tables",
		"--no-tablespaces",      // avoids PROCESS privilege error
		"--set-gtid-purged=OFF", // suppresses GTID consistency warning
		"--routines",
		"--triggers",
		"--result-file=" + destPath,
		dbname,
	}

	cmd := exec.Command(mysqldumpPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return BackupFile{}, fmt.Errorf("mysqldump failed: %s\n%s", err.Error(), string(output))
	}

	// Rotate managed backups only when saving inside the backups directory
	backupDir, _ := filepath.Abs(GetBackupDir())
	destAbs, _ := filepath.Abs(destPath)
	if strings.HasPrefix(destAbs, backupDir) {
		_ = rotateBackups(GetBackupDir(), 10)
	}

	info, _ := os.Stat(destPath)
	var size int64
	if info != nil {
		size = info.Size()
	}
	return BackupFile{
		FileName:      filepath.Base(destPath),
		Path:          destPath,
		FileSizeBytes: size,
		CreatedAt:     time.Now(),
	}, nil
}

// GetBackupPath returns the full path for a named backup file, validated for safety.
func GetBackupPath(name string) (string, error) {
	if !strings.HasSuffix(name, ".sql") || strings.Contains(name, "..") || strings.ContainsAny(name, `/\`) {
		return "", fmt.Errorf("invalid backup filename")
	}
	path := filepath.Join(GetBackupDir(), name)
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("backup file not found")
	}
	return path, nil
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
			FileName:      e.Name(),
			Path:          filepath.Join(dir, e.Name()),
			FileSizeBytes: info.Size(),
			CreatedAt:     info.ModTime(),
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
