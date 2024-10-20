package backup

import (
	"io"
	"os"

	"golang.org/x/sys/unix"
)

func BackupDatabaseLocally(dbPath, backupPath string) error {
	dbFile, err := os.OpenFile(dbPath, os.O_RDONLY, 0666)
	if err != nil {
		return err
	}
	defer dbFile.Close()

	// Acquire a lock on the database file
	if err := unix.Flock(int(dbFile.Fd()), unix.LOCK_EX); err != nil {
		return err
	}
	defer unix.Flock(int(dbFile.Fd()), unix.LOCK_UN) // Ensure the lock is released

	backupFile, err := os.Create(backupPath)
	if err != nil {
		return err
	}
	defer backupFile.Close()

	_, err = io.Copy(backupFile, dbFile)
	return err
}
