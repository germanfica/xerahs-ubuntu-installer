package main

type CheckResults struct {
	PassedChecks int
	TotalChecks  int
}

func NewCheckResults(InstallerConfigurationValue InstallerConfiguration) CheckResults {
	TotalChecks := 1
	if InstallerConfigurationValue.UseDevelopInstallation {
		TotalChecks++
	}
	if InstallerConfigurationValue.InstallChanges {
		TotalChecks++
		if InstallerConfigurationValue.UseDevelopInstallation {
			TotalChecks = TotalChecks + 2
		} else {
			TotalChecks++
		}
	}
	return CheckResults{TotalChecks: TotalChecks}
}

func RecordPassedCheck(CheckResultsValue *CheckResults) {
	CheckResultsValue.PassedChecks++
}
