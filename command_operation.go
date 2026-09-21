package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type CommandOperation struct {
	Name             string
	ExecutablePath   string
	Arguments        []string
	WorkingDirectory string
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
