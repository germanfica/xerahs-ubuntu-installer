package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	XerahSDevelopRepositoryURL = "https://github.com/ShareX/XerahS.git"
	XerahSDevelopBranch        = "develop"
	MicrosoftPackagesDEBURL    = "https://packages.microsoft.com/config/ubuntu/24.04/packages-microsoft-prod.deb"
	MicrosoftPackagesDEBName   = "packages-microsoft-prod.deb"
	NodeSourceSetupURL         = "https://deb.nodesource.com/setup_22.x"
	NodeSourceSetupFileName    = "nodesource_setup_22.sh"
)

func RunXerahSDevelopInstallation(InstallerConfigurationValue InstallerConfiguration, OperationLogValue OperationLog, CheckResultsValue *CheckResults) error {
	if DestinationValidationError := ValidateDestinationRepositoryPath(InstallerConfigurationValue); DestinationValidationError != nil {
		return DestinationValidationError
	}
	RecordPassedCheck(CheckResultsValue)
	if PrerequisiteError := InstallXerahSDevelopPrerequisites(InstallerConfigurationValue, OperationLogValue); PrerequisiteError != nil {
		return PrerequisiteError
	}
	if InstallerConfigurationValue.InstallChanges {
		if ToolVersionError := ValidateXerahSDevelopToolVersions(); ToolVersionError != nil {
			return ToolVersionError
		}
		RecordPassedCheck(CheckResultsValue)
	}
	if SourcePreparationError := PrepareXerahSDevelopSourceRepository(InstallerConfigurationValue, OperationLogValue); SourcePreparationError != nil {
		return SourcePreparationError
	}
	if InstallerConfigurationValue.InstallChanges {
		if TestFixError := ApplyWindowsModernCaptureTestFix(InstallerConfigurationValue, OperationLogValue); TestFixError != nil {
			return TestFixError
		}
		RecordPassedCheck(CheckResultsValue)
	}
	return BuildXerahSDevelop(InstallerConfigurationValue, OperationLogValue)
}

func PrintXerahSDevelopInstallationTarget(InstallerConfigurationValue InstallerConfiguration) {
	fmt.Println("XerahS repository:", XerahSDevelopRepositoryURL)
	fmt.Println("XerahS branch:", XerahSDevelopBranch)
	fmt.Println("Destination path:", InstallerConfigurationValue.DestinationRepositoryPath)
}

func PrintXerahSDevelopDryRunValidationSuccess(InstallerConfigurationValue InstallerConfiguration, CheckResultsValue CheckResults) {
	CheckSummary := fmt.Sprintf("Checks passed: %d/%d", CheckResultsValue.PassedChecks, CheckResultsValue.TotalChecks)
	fmt.Println(TerminalColorGreen + CheckSummary + TerminalColorReset)
	fmt.Println(TerminalColorGreen + "Ubuntu 24.04 and the destination path were validated." + TerminalColorReset)
	fmt.Println(TerminalColorGreen + "It is safe to execute the validated installation plan with:" + TerminalColorReset)
	fmt.Println(TerminalColorGreen + "  ./xerahs-ubuntu-installer --install --dev" + TerminalColorReset)
	fmt.Println(TerminalColorGreen + "This will install and build " + XerahSDevelopRepositoryURL + " (branch " + XerahSDevelopBranch + ")." + TerminalColorReset)
}

func InstallXerahSDevelopPrerequisites(InstallerConfigurationValue InstallerConfiguration, OperationLogValue OperationLog) error {
	PackageDownloadPath := filepath.Join(os.TempDir(), MicrosoftPackagesDEBName)
	NodeSourceSetupPath := filepath.Join(os.TempDir(), NodeSourceSetupFileName)
	Operations := []CommandOperation{
		{Name: "Update Ubuntu package metadata", ExecutablePath: "sudo", Arguments: []string{"apt-get", "update"}},
		{Name: "Install base tools", ExecutablePath: "sudo", Arguments: []string{"apt-get", "install", "--yes", "ca-certificates", "curl", "git", "gnupg", "wget", "dpkg-dev", "build-essential", "libfontconfig1", "libfreetype6", "libgtk-3-0", "libnss3", "libx11-6", "libxcomposite1", "libxcursor1", "libxdamage1", "libxext6", "libxi6", "libxrandr2", "libxrender1", "libxtst6", "wl-clipboard", "xclip"}},
		{Name: "Download Microsoft package repository configuration", ExecutablePath: "wget", Arguments: []string{"--output-document", PackageDownloadPath, MicrosoftPackagesDEBURL}},
		{Name: "Install Microsoft package repository configuration", ExecutablePath: "sudo", Arguments: []string{"dpkg", "--install", PackageDownloadPath}},
		{Name: "Update package metadata after adding Microsoft repository", ExecutablePath: "sudo", Arguments: []string{"apt-get", "update"}},
		{Name: "Install .NET 10 SDK", ExecutablePath: "sudo", Arguments: []string{"apt-get", "install", "--yes", "dotnet-sdk-10.0"}},
		{Name: "Download NodeSource 22 repository setup", ExecutablePath: "curl", Arguments: []string{"--fail", "--silent", "--show-error", "--location", "--output", NodeSourceSetupPath, NodeSourceSetupURL}},
		{Name: "Configure NodeSource 22 package repository", ExecutablePath: "sudo", Arguments: []string{"bash", NodeSourceSetupPath}},
		{Name: "Install Node.js 22", ExecutablePath: "sudo", Arguments: []string{"apt-get", "install", "--yes", "nodejs"}},
	}
	return ExecuteCommandOperations(InstallerConfigurationValue, OperationLogValue, Operations)
}

func ValidateXerahSDevelopToolVersions() error {
	DotNetVersionOutput, DotNetVersionError := exec.Command("dotnet", "--version").Output()
	if DotNetVersionError != nil {
		return fmt.Errorf("read installed .NET SDK version: %w", DotNetVersionError)
	}
	if !strings.HasPrefix(strings.TrimSpace(string(DotNetVersionOutput)), "10.") {
		return fmt.Errorf(".NET SDK 10 is required; found version %q", strings.TrimSpace(string(DotNetVersionOutput)))
	}
	NodeVersionOutput, NodeVersionError := exec.Command("node", "--version").Output()
	if NodeVersionError != nil {
		return fmt.Errorf("read installed Node.js version: %w", NodeVersionError)
	}
	NodeVersion := strings.TrimPrefix(strings.TrimSpace(string(NodeVersionOutput)), "v")
	NodeVersionParts := strings.Split(NodeVersion, ".")
	if len(NodeVersionParts) < 2 {
		return fmt.Errorf("Node.js 20.19+ or 22.12+ is required; found version %q", strings.TrimSpace(string(NodeVersionOutput)))
	}
	NodeMajorVersion, NodeMajorVersionError := strconv.Atoi(NodeVersionParts[0])
	NodeMinorVersion, NodeMinorVersionError := strconv.Atoi(NodeVersionParts[1])
	if NodeMajorVersionError != nil || NodeMinorVersionError != nil || NodeMajorVersion < 20 || NodeMajorVersion == 21 || (NodeMajorVersion == 20 && NodeMinorVersion < 19) || (NodeMajorVersion == 22 && NodeMinorVersion < 12) {
		return fmt.Errorf("Node.js 20.19+ or 22.12+ is required; found version %q", strings.TrimSpace(string(NodeVersionOutput)))
	}
	if _, NpmVersionError := exec.Command("npm", "--version").Output(); NpmVersionError != nil {
		return fmt.Errorf("read installed npm version: %w", NpmVersionError)
	}
	return nil
}

func PrepareXerahSDevelopSourceRepository(InstallerConfigurationValue InstallerConfiguration, OperationLogValue OperationLog) error {
	GitDirectoryPath := filepath.Join(InstallerConfigurationValue.DestinationRepositoryPath, ".git")
	GitDirectoryInformation, GitDirectoryError := os.Stat(GitDirectoryPath)
	if errors.Is(GitDirectoryError, os.ErrNotExist) {
		CloneOperations := []CommandOperation{{Name: "Clone XerahS develop branch with required submodules", ExecutablePath: "git", Arguments: []string{"clone", "--branch", XerahSDevelopBranch, "--recursive", XerahSDevelopRepositoryURL, InstallerConfigurationValue.DestinationRepositoryPath}}}
		return ExecuteCommandOperations(InstallerConfigurationValue, OperationLogValue, CloneOperations)
	}
	if GitDirectoryError != nil || !GitDirectoryInformation.IsDir() {
		return fmt.Errorf("inspect source repository at %q: %w", InstallerConfigurationValue.DestinationRepositoryPath, GitDirectoryError)
	}
	if !InstallerConfigurationValue.UpdateExistingSource {
		LogMessage(OperationLogValue, "Using existing source repository without updating it")
		return nil
	}
	if InstallerConfigurationValue.InstallChanges {
		if CleanWorkingTreeError := ValidateXerahSDevelopCleanWorkingTree(InstallerConfigurationValue.DestinationRepositoryPath); CleanWorkingTreeError != nil {
			return CleanWorkingTreeError
		}
	}
	UpdateOperations := []CommandOperation{
		{Name: "Fetch develop branch", ExecutablePath: "git", Arguments: []string{"fetch", "origin", XerahSDevelopBranch}, WorkingDirectory: InstallerConfigurationValue.DestinationRepositoryPath},
		{Name: "Fast-forward source repository", ExecutablePath: "git", Arguments: []string{"checkout", XerahSDevelopBranch}, WorkingDirectory: InstallerConfigurationValue.DestinationRepositoryPath},
		{Name: "Pull latest develop branch", ExecutablePath: "git", Arguments: []string{"pull", "--ff-only", "origin", XerahSDevelopBranch}, WorkingDirectory: InstallerConfigurationValue.DestinationRepositoryPath},
		{Name: "Initialize and update submodules", ExecutablePath: "git", Arguments: []string{"submodule", "update", "--init", "--recursive"}, WorkingDirectory: InstallerConfigurationValue.DestinationRepositoryPath},
	}
	return ExecuteCommandOperations(InstallerConfigurationValue, OperationLogValue, UpdateOperations)
}

func ValidateXerahSDevelopCleanWorkingTree(DestinationRepositoryPath string) error {
	StatusCommand := exec.Command("git", "status", "--porcelain")
	StatusCommand.Dir = DestinationRepositoryPath
	StatusOutput, StatusError := StatusCommand.Output()
	if StatusError != nil {
		return fmt.Errorf("inspect existing source repository status: %w", StatusError)
	}
	if strings.TrimSpace(string(StatusOutput)) != "" {
		return fmt.Errorf("refusing to update %q because it has uncommitted changes", DestinationRepositoryPath)
	}
	return nil
}

func BuildXerahSDevelop(InstallerConfigurationValue InstallerConfiguration, OperationLogValue OperationLog) error {
	var BuildOperation CommandOperation
	if InstallerConfigurationValue.BuildLinuxPackages {
		BuildOperation = CommandOperation{Name: "Build XerahS Linux packages", ExecutablePath: "./build/linux/package-linux.sh", WorkingDirectory: InstallerConfigurationValue.DestinationRepositoryPath}
	} else {
		BuildOperation = CommandOperation{Name: "Compile XerahS desktop solution", ExecutablePath: "dotnet", Arguments: []string{"build", "src/desktop/XerahS.sln", "-c", "Release", "-m:1", "-p:nodeReuse=false", "-p:UseSharedCompilation=false"}, WorkingDirectory: InstallerConfigurationValue.DestinationRepositoryPath}
	}
	FrontendDirectoryPath := filepath.Join(InstallerConfigurationValue.DestinationRepositoryPath, "ShareX.VideoEditor", "frontend")
	BuildOperations := []CommandOperation{
		{Name: "Install ShareX.VideoEditor frontend dependencies", ExecutablePath: "npm", Arguments: []string{"ci"}, WorkingDirectory: FrontendDirectoryPath},
		{Name: "Build ShareX.VideoEditor frontend", ExecutablePath: "npm", Arguments: []string{"run", "build"}, WorkingDirectory: FrontendDirectoryPath},
		BuildOperation,
	}
	return ExecuteCommandOperations(InstallerConfigurationValue, OperationLogValue, BuildOperations)
}
