package main

import "fmt"

func RunInstaller(InstallerConfigurationValue InstallerConfiguration) error {
	OperationLogValue, OperationLogError := CreateOperationLog(InstallerConfigurationValue)
	if OperationLogError != nil {
		return fmt.Errorf("create operation log: %w", OperationLogError)
	}
	defer OperationLogValue.LogFile.Close()
	if InstallerConfigurationValue.InstallChanges {
		LogMessage(OperationLogValue, "Mode: install changes")
	} else {
		LogMessage(OperationLogValue, "Mode: dry run; no commands will be executed")
	}
	PrintSelectedInstallationTarget(InstallerConfigurationValue)
	CheckResultsValue := NewCheckResults(InstallerConfigurationValue)
	if ValidationError := ValidateUbuntu2404Host(); ValidationError != nil {
		FailOperation(OperationLogValue, CheckResultsValue, ValidationError)
	}
	RecordPassedCheck(&CheckResultsValue)
	if InstallerConfigurationValue.InstallChanges {
		if ValidationError := ValidateSudoAccess(); ValidationError != nil {
			FailOperation(OperationLogValue, CheckResultsValue, ValidationError)
		}
		RecordPassedCheck(&CheckResultsValue)
	}
	if InstallationError := RunSelectedInstallation(InstallerConfigurationValue, OperationLogValue, &CheckResultsValue); InstallationError != nil {
		FailOperation(OperationLogValue, CheckResultsValue, InstallationError)
	}
	LogMessage(OperationLogValue, "Completed successfully")
	fmt.Println("Completed successfully. Operation log:", OperationLogValue.LogPath)
	if !InstallerConfigurationValue.InstallChanges {
		PrintSelectedDryRunValidationSuccess(InstallerConfigurationValue, CheckResultsValue)
	}
	return nil
}
