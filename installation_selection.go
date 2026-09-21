package main

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
