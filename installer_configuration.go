package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

type InstallerConfiguration struct {
	DestinationRepositoryPath     string
	OperationLogDirectoryPath     string
	InstallChanges                bool
	UseDevelopInstallation        bool
	UseReleasePackageInstallation bool
	UpdateExistingSource          bool
	BuildLinuxPackages            bool
}

func ParseInstallerConfiguration() (InstallerConfiguration, error) {
	CurrentWorkingDirectoryPath, CurrentWorkingDirectoryError := os.Getwd()
	if CurrentWorkingDirectoryError != nil {
		return InstallerConfiguration{}, fmt.Errorf("get current working directory: %w", CurrentWorkingDirectoryError)
	}
	DefaultDestinationRepositoryPath := filepath.Join(CurrentWorkingDirectoryPath, "XerahS")
	DefaultOperationLogDirectoryPath := filepath.Join(CurrentWorkingDirectoryPath, "xerahs-installer-logs")
	DestinationRepositoryPath := flag.String("destination", DefaultDestinationRepositoryPath, "destination directory for the XerahS repository")
	OperationLogDirectoryPath := flag.String("log-directory", DefaultOperationLogDirectoryPath, "directory for operation logs")
	InstallChanges := flag.Bool("install", false, "install prerequisites, clone XerahS, and build it; without this flag the program only prints the plan")
	UseDevelopInstallation := flag.Bool("dev", false, "select the XerahS develop branch installation")
	UseReleasePackageInstallation := flag.Bool("release", false, "select installation from the XerahS v0.25.5 Debian package")
	UpdateExistingSource := flag.Bool("update-source", false, "fast-forward an existing clean clone to origin/develop")
	BuildLinuxPackages := flag.Bool("build-packages", false, "build Linux packages under dist/ instead of only compiling the desktop solution")
	flag.Parse()
	if *UseDevelopInstallation && *UseReleasePackageInstallation {
		return InstallerConfiguration{}, fmt.Errorf("select only one installation target: --dev or --release")
	}
	if !*UseDevelopInstallation && !*UseReleasePackageInstallation {
		return InstallerConfiguration{}, fmt.Errorf("select an installation target with --dev or --release")
	}
	if *UseReleasePackageInstallation && (*UpdateExistingSource || *BuildLinuxPackages) {
		return InstallerConfiguration{}, fmt.Errorf("--update-source and --build-packages require --dev")
	}
	AbsoluteDestinationRepositoryPath, DestinationPathError := filepath.Abs(*DestinationRepositoryPath)
	if DestinationPathError != nil {
		return InstallerConfiguration{}, fmt.Errorf("resolve destination path: %w", DestinationPathError)
	}
	AbsoluteOperationLogDirectoryPath, LogDirectoryPathError := filepath.Abs(*OperationLogDirectoryPath)
	if LogDirectoryPathError != nil {
		return InstallerConfiguration{}, fmt.Errorf("resolve log directory path: %w", LogDirectoryPathError)
	}
	return InstallerConfiguration{DestinationRepositoryPath: AbsoluteDestinationRepositoryPath, OperationLogDirectoryPath: AbsoluteOperationLogDirectoryPath, InstallChanges: *InstallChanges, UseDevelopInstallation: *UseDevelopInstallation, UseReleasePackageInstallation: *UseReleasePackageInstallation, UpdateExistingSource: *UpdateExistingSource, BuildLinuxPackages: *BuildLinuxPackages}, nil
}
