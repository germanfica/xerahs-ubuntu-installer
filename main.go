// xerahs-ubuntu-installer installs and builds XerahS on Ubuntu 24.04.
//
// The program performs a dry run by default. Pass --install to execute the
// validated commands.
package main

import (
	"fmt"
	"os"
)

func main() {
	InstallerConfigurationValue, ConfigurationError := ParseInstallerConfiguration()
	if ConfigurationError != nil {
		fmt.Fprintln(os.Stderr, "Configuration error:", ConfigurationError)
		os.Exit(2)
	}
	if InstallationError := RunInstaller(InstallerConfigurationValue); InstallationError != nil {
		fmt.Fprintln(os.Stderr, "Installation failed:", InstallationError)
		os.Exit(1)
	}
}
