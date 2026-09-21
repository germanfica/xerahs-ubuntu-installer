package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

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
		if len(OperatingSystemParts) == 2 {
			OperatingSystemValues[OperatingSystemParts[0]] = strings.Trim(OperatingSystemParts[1], "\"")
		}
	}
	if ScannerError := OperatingSystemScanner.Err(); ScannerError != nil {
		return fmt.Errorf("scan %s: %w", OperatingSystemReleasePath, ScannerError)
	}
	if OperatingSystemValues["ID"] != "ubuntu" || OperatingSystemValues["VERSION_ID"] != "24.04" {
		return fmt.Errorf("this script supports Ubuntu 24.04 only; detected ID=%q VERSION_ID=%q", OperatingSystemValues["ID"], OperatingSystemValues["VERSION_ID"])
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
