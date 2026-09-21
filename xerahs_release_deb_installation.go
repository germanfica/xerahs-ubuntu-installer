package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	XerahSReleaseVersion              = "0.25.5"
	XerahSReleasePackageURL           = "https://github.com/ShareX/XerahS/releases/download/v0.25.5/XerahS-0.25.5-linux-x64.deb"
	XerahSReleaseExpectedPackageName  = "xerahs"
	XerahSReleaseExpectedArchitecture = "amd64"
)

type DebianPackageMetadata struct {
	PackageName    string
	PackageVersion string
	Architecture   string
}

func RunSelectedInstallation(InstallerConfigurationValue InstallerConfiguration, OperationLogValue OperationLog, CheckResultsValue *CheckResults) error {
	if InstallerConfigurationValue.UseDevelopInstallation {
		return RunXerahSDevelopInstallation(InstallerConfigurationValue, OperationLogValue, CheckResultsValue)
	}
	return RunXerahSReleaseDebInstallation(InstallerConfigurationValue, OperationLogValue, CheckResultsValue)
}

func PrintSelectedInstallationTarget(InstallerConfigurationValue InstallerConfiguration) {
	if InstallerConfigurationValue.UseDevelopInstallation {
		PrintXerahSDevelopInstallationTarget(InstallerConfigurationValue)
		return
	}
	PrintXerahSReleaseDebInstallationTarget()
}

func PrintSelectedDryRunValidationSuccess(InstallerConfigurationValue InstallerConfiguration, CheckResultsValue CheckResults) {
	if InstallerConfigurationValue.UseDevelopInstallation {
		PrintXerahSDevelopDryRunValidationSuccess(InstallerConfigurationValue, CheckResultsValue)
		return
	}
	PrintXerahSReleaseDebDryRunValidationSuccess(CheckResultsValue)
}

func RunXerahSReleaseDebInstallation(InstallerConfigurationValue InstallerConfiguration, OperationLogValue OperationLog, CheckResultsValue *CheckResults) error {
	SourceDebPath := filepath.Join(os.TempDir(), "xerahs-"+XerahSReleaseVersion+"-"+OperationLogValue.OperationID+".deb")
	DownloadOperation := CommandOperation{Name: "Download XerahS release Debian package", ExecutablePath: "curl", Arguments: []string{"--fail", "--show-error", "--location", "--progress-bar", "--output", SourceDebPath, XerahSReleasePackageURL}}
	if DownloadError := ExecuteCommandOperations(InstallerConfigurationValue, OperationLogValue, []CommandOperation{DownloadOperation}); DownloadError != nil {
		return DownloadError
	}
	if !InstallerConfigurationValue.InstallChanges {
		LogMessage(OperationLogValue, "The downloaded Debian package will be validated before installation")
		LogMessage(OperationLogValue, "The validated Debian package will be installed with sudo dpkg --install")
		return nil
	}

	PackageMetadata, PackageValidationError := ValidateXerahSReleaseDebianPackage(SourceDebPath)
	if PackageValidationError != nil {
		return PackageValidationError
	}
	RecordPassedCheck(CheckResultsValue)
	LogMessage(OperationLogValue, "Validated Debian package: "+PackageMetadata.PackageName+" "+PackageMetadata.PackageVersion+" "+PackageMetadata.Architecture)

	InstallOperation := CommandOperation{Name: "Install XerahS release Debian package", ExecutablePath: "sudo", Arguments: []string{"dpkg", "--install", SourceDebPath}}
	return ExecuteCommandOperations(InstallerConfigurationValue, OperationLogValue, []CommandOperation{InstallOperation})
}

func PrintXerahSReleaseDebInstallationTarget() {
	fmt.Println("XerahS release:", "v"+XerahSReleaseVersion)
	fmt.Println("XerahS Debian package:", XerahSReleasePackageURL)
}

func PrintXerahSReleaseDebDryRunValidationSuccess(CheckResultsValue CheckResults) {
	CheckSummary := fmt.Sprintf("Checks passed: %d/%d", CheckResultsValue.PassedChecks, CheckResultsValue.TotalChecks)
	fmt.Println(TerminalColorGreen + CheckSummary + TerminalColorReset)
	fmt.Println(TerminalColorGreen + "Ubuntu 24.04 was validated." + TerminalColorReset)
	fmt.Println(TerminalColorGreen + "It is safe to download, validate, and install the selected release with:" + TerminalColorReset)
	fmt.Println(TerminalColorGreen + "  ./xerahs-ubuntu-installer --install --release" + TerminalColorReset)
}

func ValidateXerahSReleaseDebianPackage(SourceDebPath string) (DebianPackageMetadata, error) {
	SourceDebInformation, SourceDebError := os.Stat(SourceDebPath)
	if SourceDebError != nil {
		return DebianPackageMetadata{}, fmt.Errorf("inspect downloaded Debian package %q: %w", SourceDebPath, SourceDebError)
	}
	if SourceDebInformation.IsDir() {
		return DebianPackageMetadata{}, fmt.Errorf("downloaded Debian package path %q is a directory", SourceDebPath)
	}
	if SourceDebInformation.Size() == 0 {
		return DebianPackageMetadata{}, fmt.Errorf("downloaded Debian package %q is empty", SourceDebPath)
	}

	DebianPackageMetadataOutput, DebianPackageMetadataError := exec.Command("dpkg-deb", "--showformat=${Package}\\n${Version}\\n${Architecture}\\n", "--show", SourceDebPath).Output()
	if DebianPackageMetadataError != nil {
		return DebianPackageMetadata{}, fmt.Errorf("read Debian package metadata from %q: %w", SourceDebPath, DebianPackageMetadataError)
	}
	DebianPackageMetadataLines := strings.Split(strings.TrimSpace(string(DebianPackageMetadataOutput)), "\n")
	if len(DebianPackageMetadataLines) != 3 {
		return DebianPackageMetadata{}, fmt.Errorf("read incomplete Debian package metadata from %q", SourceDebPath)
	}
	PackageMetadata := DebianPackageMetadata{PackageName: DebianPackageMetadataLines[0], PackageVersion: DebianPackageMetadataLines[1], Architecture: DebianPackageMetadataLines[2]}
	if PackageMetadata.PackageName != XerahSReleaseExpectedPackageName {
		return DebianPackageMetadata{}, fmt.Errorf("expected Debian package name %q but found %q", XerahSReleaseExpectedPackageName, PackageMetadata.PackageName)
	}
	if PackageMetadata.PackageVersion != XerahSReleaseVersion {
		return DebianPackageMetadata{}, fmt.Errorf("expected Debian package version %q but found %q", XerahSReleaseVersion, PackageMetadata.PackageVersion)
	}
	if PackageMetadata.Architecture != XerahSReleaseExpectedArchitecture {
		return DebianPackageMetadata{}, fmt.Errorf("expected Debian package architecture %q but found %q", XerahSReleaseExpectedArchitecture, PackageMetadata.Architecture)
	}
	return PackageMetadata, nil
}
