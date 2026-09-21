package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type OperationLog struct {
	OperationID string
	LogPath     string
	LogFile     *os.File
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
