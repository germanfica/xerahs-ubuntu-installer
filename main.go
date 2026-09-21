// xerahs-ubuntu-installer installs the build prerequisites for XerahS and
// builds the develop branch on Ubuntu 24.04.
//
// The program performs a dry run by default. Pass --install to execute the
// validated commands.
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	XerahSRepositoryURL      = "https://github.com/ShareX/XerahS.git"
	XerahSDevelopBranch      = "develop"
	MicrosoftPackagesDEBURL  = "https://packages.microsoft.com/config/ubuntu/24.04/packages-microsoft-prod.deb"
	MicrosoftPackagesDEBName = "packages-microsoft-prod.deb"
	NodeSourceSetupURL       = "https://deb.nodesource.com/setup_22.x"
	NodeSourceSetupFileName  = "nodesource_setup_22.sh"
	TerminalColorGreen        = "\033[1;32m"
	TerminalColorRed          = "\033[1;31m"
	TerminalColorReset        = "\033[0m"
)

type InstallerConfiguration struct {
	DestinationRepositoryPath string
	OperationLogDirectoryPath string
	InstallChanges            bool
	UpdateExistingSource      bool
	BuildLinuxPackages        bool
}

type OperationLog struct {
	OperationID string
	LogPath     string
	LogFile     *os.File
}

type CommandOperation struct {
	Name            string
	ExecutablePath  string
	Arguments       []string
	WorkingDirectory string
}

type CheckResults struct {
	PassedChecks int
	TotalChecks  int
}

func main() {
	InstallerConfigurationValue, ConfigurationError := ParseInstallerConfiguration()
	if ConfigurationError != nil {
		fmt.Fprintln(os.Stderr, "Configuration error:", ConfigurationError)
		os.Exit(2)
	}

	OperationLogValue, OperationLogError := CreateOperationLog(InstallerConfigurationValue)
	if OperationLogError != nil {
		fmt.Fprintln(os.Stderr, "Could not create operation log:", OperationLogError)
		os.Exit(1)
	}
	defer OperationLogValue.LogFile.Close()

	if InstallerConfigurationValue.InstallChanges {
		LogMessage(OperationLogValue, "Mode: install changes")
	} else {
		LogMessage(OperationLogValue, "Mode: dry run; no commands will be executed")
	}
	PrintInstallationTarget(InstallerConfigurationValue)
	CheckResultsValue := NewCheckResults(InstallerConfigurationValue)

	if ValidationError := ValidateUbuntu2404Host(); ValidationError != nil {
		FailOperation(OperationLogValue, CheckResultsValue, ValidationError)
	}
	RecordPassedCheck(&CheckResultsValue)
	if ValidationError := ValidateDestinationRepositoryPath(InstallerConfigurationValue); ValidationError != nil {
		FailOperation(OperationLogValue, CheckResultsValue, ValidationError)
	}
	RecordPassedCheck(&CheckResultsValue)
	if InstallerConfigurationValue.InstallChanges {
		if ValidationError := ValidateSudoAccess(); ValidationError != nil {
			FailOperation(OperationLogValue, CheckResultsValue, ValidationError)
		}
		RecordPassedCheck(&CheckResultsValue)
	}

	if ExecutionError := InstallBuildPrerequisites(InstallerConfigurationValue, OperationLogValue); ExecutionError != nil {
		FailOperation(OperationLogValue, CheckResultsValue, ExecutionError)
	}
	if InstallerConfigurationValue.InstallChanges {
		if ValidationError := ValidateInstalledToolVersions(); ValidationError != nil {
			FailOperation(OperationLogValue, CheckResultsValue, ValidationError)
		}
		RecordPassedCheck(&CheckResultsValue)
	}
	if ExecutionError := PrepareSourceRepository(InstallerConfigurationValue, OperationLogValue); ExecutionError != nil {
		FailOperation(OperationLogValue, CheckResultsValue, ExecutionError)
	}
	if InstallerConfigurationValue.InstallChanges {
		if FixError := ApplyWindowsModernCaptureTestFix(InstallerConfigurationValue, OperationLogValue); FixError != nil {
			FailOperation(OperationLogValue, CheckResultsValue, FixError)
		}
		RecordPassedCheck(&CheckResultsValue)
	}
	if ExecutionError := BuildXerahS(InstallerConfigurationValue, OperationLogValue); ExecutionError != nil {
		FailOperation(OperationLogValue, CheckResultsValue, ExecutionError)
	}

	LogMessage(OperationLogValue, "Completed successfully")
	fmt.Println("Completed successfully. Operation log:", OperationLogValue.LogPath)
	if !InstallerConfigurationValue.InstallChanges {
		PrintDryRunValidationSuccess(InstallerConfigurationValue, CheckResultsValue)
	}
}

func NewCheckResults(InstallerConfigurationValue InstallerConfiguration) CheckResults {
	TotalChecks := 2
	if InstallerConfigurationValue.InstallChanges {
		TotalChecks = TotalChecks + 3
	}
	return CheckResults{TotalChecks: TotalChecks}
}

func RecordPassedCheck(CheckResultsValue *CheckResults) {
	CheckResultsValue.PassedChecks++
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
	UpdateExistingSource := flag.Bool("update-source", false, "fast-forward an existing clean clone to origin/develop")
	BuildLinuxPackages := flag.Bool("build-packages", false, "build Linux packages under dist/ instead of only compiling the desktop solution")
	flag.Parse()

	AbsoluteDestinationRepositoryPath, DestinationPathError := filepath.Abs(*DestinationRepositoryPath)
	if DestinationPathError != nil {
		return InstallerConfiguration{}, fmt.Errorf("resolve destination path: %w", DestinationPathError)
	}
	AbsoluteOperationLogDirectoryPath, LogDirectoryPathError := filepath.Abs(*OperationLogDirectoryPath)
	if LogDirectoryPathError != nil {
		return InstallerConfiguration{}, fmt.Errorf("resolve log directory path: %w", LogDirectoryPathError)
	}

	return InstallerConfiguration{
		DestinationRepositoryPath: AbsoluteDestinationRepositoryPath,
		OperationLogDirectoryPath: AbsoluteOperationLogDirectoryPath,
		InstallChanges:            *InstallChanges,
		UpdateExistingSource:      *UpdateExistingSource,
		BuildLinuxPackages:        *BuildLinuxPackages,
	}, nil
}

func CreateOperationLog(InstallerConfigurationValue InstallerConfiguration) (OperationLog, error) {
	if DirectoryError := os.MkdirAll(InstallerConfigurationValue.OperationLogDirectoryPath, 0755); DirectoryError != nil {
		return OperationLog{}, fmt.Errorf("create log directory %q: %w", InstallerConfigurationValue.OperationLogDirectoryPath, DirectoryError)
	}
	OperationID := time.Now().UTC().Format("20060102T150405Z")
	LogPath := filepath.Join(InstallerConfigurationValue.OperationLogDirectoryPath, "xerahs-install-"+OperationID+".log")
	LogFile, LogFileError := os.OpenFile(LogPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if LogFileError != nil {
		return OperationLog{}, fmt.Errorf("create operation log %q: %w", LogPath, LogFileError)
	}
	return OperationLog{OperationID: OperationID, LogPath: LogPath, LogFile: LogFile}, nil
}

func ValidateUbuntu2404Host() error {
	OperatingSystemReleasePath := "/etc/os-release"
	OperatingSystemReleaseFile, OperatingSystemReleaseError := os.Open(OperatingSystemReleasePath)
	if OperatingSystemReleaseError != nil {
		return fmt.Errorf("read %s: %w", OperatingSystemReleasePath, OperatingSystemReleaseError)
	}
	defer OperatingSystemReleaseFile.Close()

	OperatingSystemValues := make(map[string]string)
	OperatingSystemScanner := bufio.NewScanner(OperatingSystemReleaseFile)
	for OperatingSystemScanner.Scan() {
		OperatingSystemLine := OperatingSystemScanner.Text()
		OperatingSystemParts := strings.SplitN(OperatingSystemLine, "=", 2)
		if len(OperatingSystemParts) != 2 {
			continue
		}
		OperatingSystemValues[OperatingSystemParts[0]] = strings.Trim(OperatingSystemParts[1], "\"")
	}
	if ScannerError := OperatingSystemScanner.Err(); ScannerError != nil {
		return fmt.Errorf("scan %s: %w", OperatingSystemReleasePath, ScannerError)
	}
	if OperatingSystemValues["ID"] != "ubuntu" || OperatingSystemValues["VERSION_ID"] != "24.04" {
		return fmt.Errorf("this script supports Ubuntu 24.04 only; detected ID=%q VERSION_ID=%q", OperatingSystemValues["ID"], OperatingSystemValues["VERSION_ID"])
	}
	return nil
}

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

func ValidateSudoAccess() error {
	SudoValidationCommand := exec.Command("sudo", "-v")
	SudoValidationCommand.Stdin = os.Stdin
	SudoValidationCommand.Stdout = os.Stdout
	SudoValidationCommand.Stderr = os.Stderr
	if SudoValidationError := SudoValidationCommand.Run(); SudoValidationError != nil {
		return fmt.Errorf("validate sudo access: %w", SudoValidationError)
	}
	return nil
}

func PrintInstallationTarget(InstallerConfigurationValue InstallerConfiguration) {
	fmt.Println("XerahS repository:", XerahSRepositoryURL)
	fmt.Println("XerahS branch:", XerahSDevelopBranch)
	fmt.Println("Destination path:", InstallerConfigurationValue.DestinationRepositoryPath)
}

func PrintDryRunValidationSuccess(InstallerConfigurationValue InstallerConfiguration, CheckResultsValue CheckResults) {
	CheckSummary := fmt.Sprintf("Checks passed: %d/%d", CheckResultsValue.PassedChecks, CheckResultsValue.TotalChecks)
	fmt.Println(TerminalColorGreen + CheckSummary + TerminalColorReset)
	fmt.Println(TerminalColorGreen + "Ubuntu 24.04 and the destination path were validated." + TerminalColorReset)
	fmt.Println(TerminalColorGreen + "It is safe to execute the validated installation plan with:" + TerminalColorReset)
	fmt.Println(TerminalColorGreen + "  ./xerahs-ubuntu-installer --install" + TerminalColorReset)
	fmt.Println(TerminalColorGreen + "This will install and build " + XerahSRepositoryURL + " (branch " + XerahSDevelopBranch + ")." + TerminalColorReset)
}

func InstallBuildPrerequisites(InstallerConfigurationValue InstallerConfiguration, OperationLogValue OperationLog) error {
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

func ValidateInstalledToolVersions() error {
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

func PrepareSourceRepository(InstallerConfigurationValue InstallerConfiguration, OperationLogValue OperationLog) error {
	GitDirectoryPath := filepath.Join(InstallerConfigurationValue.DestinationRepositoryPath, ".git")
	GitDirectoryInformation, GitDirectoryError := os.Stat(GitDirectoryPath)
	if errors.Is(GitDirectoryError, os.ErrNotExist) {
		CloneOperations := []CommandOperation{{Name: "Clone XerahS develop branch with required submodules", ExecutablePath: "git", Arguments: []string{"clone", "--branch", XerahSDevelopBranch, "--recursive", XerahSRepositoryURL, InstallerConfigurationValue.DestinationRepositoryPath}}}
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
		if CleanWorkingTreeError := ValidateCleanWorkingTree(InstallerConfigurationValue.DestinationRepositoryPath); CleanWorkingTreeError != nil {
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

func ValidateCleanWorkingTree(DestinationRepositoryPath string) error {
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

func BuildXerahS(InstallerConfigurationValue InstallerConfiguration, OperationLogValue OperationLog) error {
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

func ExecuteCommandOperations(InstallerConfigurationValue InstallerConfiguration, OperationLogValue OperationLog, Operations []CommandOperation) error {
	for _, CurrentOperation := range Operations {
		LogMessage(OperationLogValue, "Operation: "+CurrentOperation.Name)
		LogMessage(OperationLogValue, "Command: "+FormatCommand(CurrentOperation))
		if !InstallerConfigurationValue.InstallChanges {
			continue
		}
		Command := exec.Command(CurrentOperation.ExecutablePath, CurrentOperation.Arguments...)
		Command.Dir = CurrentOperation.WorkingDirectory
		Command.Stdin = os.Stdin
		Command.Stdout = os.Stdout
		Command.Stderr = os.Stderr
		if CommandError := Command.Run(); CommandError != nil {
			return fmt.Errorf("%s failed: %w", CurrentOperation.Name, CommandError)
		}
	}
	return nil
}

func FormatCommand(CommandOperationValue CommandOperation) string {
	CommandParts := []string{CommandOperationValue.ExecutablePath}
	for _, CurrentArgument := range CommandOperationValue.Arguments {
		CommandParts = append(CommandParts, CurrentArgument)
	}
	if CommandOperationValue.WorkingDirectory != "" {
		return "(cd " + CommandOperationValue.WorkingDirectory + " && " + strings.Join(CommandParts, " ") + ")"
	}
	return strings.Join(CommandParts, " ")
}

func LogMessage(OperationLogValue OperationLog, Message string) {
	Timestamp := time.Now().UTC().Format(time.RFC3339)
	LogLine := Timestamp + " " + Message
	fmt.Println(LogLine)
	_, _ = fmt.Fprintln(OperationLogValue.LogFile, LogLine)
}

func FailOperation(OperationLogValue OperationLog, CheckResultsValue CheckResults, OperationError error) {
	LogMessage(OperationLogValue, "Failed: "+OperationError.Error())
	CheckSummary := fmt.Sprintf("Checks passed: %d/%d", CheckResultsValue.PassedChecks, CheckResultsValue.TotalChecks)
	fmt.Fprintln(os.Stderr, TerminalColorRed+CheckSummary+TerminalColorReset)
	fmt.Fprintln(os.Stderr, TerminalColorRed+"CHECK FAILED"+TerminalColorReset)
	fmt.Fprintln(os.Stderr, TerminalColorRed+"The installation plan was not approved: "+OperationError.Error()+TerminalColorReset)
	fmt.Fprintln(os.Stderr, TerminalColorRed+"Do not run --install until this check is resolved."+TerminalColorReset)
	fmt.Fprintln(os.Stderr, "Operation log:", OperationLogValue.LogPath)
	os.Exit(1)
}
