package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func ValidateDestinationRepositoryPath(InstallerConfigurationValue InstallerConfiguration) error {
	DestinationRepositoryInformation, DestinationRepositoryError := os.Stat(InstallerConfigurationValue.DestinationRepositoryPath)
	if errors.Is(DestinationRepositoryError, os.ErrNotExist) {
		return nil
	}
	if DestinationRepositoryError != nil {
		return fmt.Errorf("inspect destination %q: %w", InstallerConfigurationValue.DestinationRepositoryPath, DestinationRepositoryError)
	}
	if !DestinationRepositoryInformation.IsDir() {
		return fmt.Errorf("destination %q exists but is not a directory", InstallerConfigurationValue.DestinationRepositoryPath)
	}
	GitDirectoryPath := filepath.Join(InstallerConfigurationValue.DestinationRepositoryPath, ".git")
	GitDirectoryInformation, GitDirectoryError := os.Stat(GitDirectoryPath)
	if errors.Is(GitDirectoryError, os.ErrNotExist) {
		DestinationEntries, DestinationEntriesError := os.ReadDir(InstallerConfigurationValue.DestinationRepositoryPath)
		if DestinationEntriesError != nil {
			return fmt.Errorf("read destination %q: %w", InstallerConfigurationValue.DestinationRepositoryPath, DestinationEntriesError)
		}
		if len(DestinationEntries) == 0 {
			return nil
		}
	}
	if GitDirectoryError != nil || !GitDirectoryInformation.IsDir() {
		return fmt.Errorf("destination %q exists but is not a Git clone; choose a new empty destination", InstallerConfigurationValue.DestinationRepositoryPath)
	}
	return nil
}
