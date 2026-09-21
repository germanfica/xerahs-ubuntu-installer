package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const WindowsModernCaptureServiceTestsRelativePath = "tests/XerahS.Tests/Platform/Windows/WindowsModernCaptureServiceTests.cs"

type SourceRange struct {
	StartOffset int
	EndOffset   int
}

type WindowsTestMethodDefinition struct {
	MethodName        string
	MethodDeclaration string
}

type WindowsModernCaptureTestFixPlan struct {
	SourceTestPath    string
	SourceTestContent string
	MethodSourceRanges []SourceRange
}

func ApplyWindowsModernCaptureTestFix(InstallerConfigurationValue InstallerConfiguration, OperationLogValue OperationLog) error {
	SourceTestPath := filepath.Join(InstallerConfigurationValue.DestinationRepositoryPath, WindowsModernCaptureServiceTestsRelativePath)
	LogMessage(OperationLogValue, "Operation: Apply temporary Linux guard to Windows modern capture tests")
	LogMessage(OperationLogValue, "Source test file: "+SourceTestPath)

	WindowsModernCaptureTestFixPlanValue, ValidationError := ValidateWindowsModernCaptureTestFix(SourceTestPath)
	if ValidationError != nil {
		return ValidationError
	}
	if len(WindowsModernCaptureTestFixPlanValue.MethodSourceRanges) == 0 {
		LogMessage(OperationLogValue, "Temporary Linux guard is already present")
		return nil
	}

	SnapshotPath, SnapshotError := CreateWindowsModernCaptureTestSnapshot(WindowsModernCaptureTestFixPlanValue, OperationLogValue)
	if SnapshotError != nil {
		return SnapshotError
	}
	LogMessage(OperationLogValue, "Snapshot: "+SnapshotPath)

	DestinationTestContent := AddWindowsGuardsToTestMethods(WindowsModernCaptureTestFixPlanValue)
	if WriteError := WriteDestinationTestContent(SourceTestPath, DestinationTestContent); WriteError != nil {
		return WriteError
	}
	LogMessage(OperationLogValue, "Applied temporary Linux guard to 2 Windows-only test methods")
	return nil
}

func ValidateWindowsModernCaptureTestFix(SourceTestPath string) (WindowsModernCaptureTestFixPlan, error) {
	SourceTestBytes, SourceTestReadError := os.ReadFile(SourceTestPath)
	if SourceTestReadError != nil {
		return WindowsModernCaptureTestFixPlan{}, fmt.Errorf("read Windows modern capture test file %q: %w", SourceTestPath, SourceTestReadError)
	}
	SourceTestContent := string(SourceTestBytes)
	WindowsTestMethodDefinitions := []WindowsTestMethodDefinition{
		{
			MethodName:        "ShouldUseModernCapture_UsesResolvedCaptureOption",
			MethodDeclaration: "public void ShouldUseModernCapture_UsesResolvedCaptureOption(bool configuredValue, bool expected)",
		},
		{
			MethodName:        "ShouldUseModernCapture_WithoutOptions_PrefersDxgi",
			MethodDeclaration: "public void ShouldUseModernCapture_WithoutOptions_PrefersDxgi()",
		},
	}
	MethodSourceRanges := make([]SourceRange, 0, len(WindowsTestMethodDefinitions))
	AlreadyGuardedMethodCount := 0

	for _, CurrentMethodDefinition := range WindowsTestMethodDefinitions {
		MethodSourceRange, AlreadyGuarded, MethodRangeError := FindTestMethodSourceRange(SourceTestContent, CurrentMethodDefinition)
		if MethodRangeError != nil {
			return WindowsModernCaptureTestFixPlan{}, MethodRangeError
		}
		if AlreadyGuarded {
			AlreadyGuardedMethodCount++
			continue
		}
		MethodSourceRanges = append(MethodSourceRanges, MethodSourceRange)
	}
	if AlreadyGuardedMethodCount != 0 && AlreadyGuardedMethodCount != len(WindowsTestMethodDefinitions) {
		return WindowsModernCaptureTestFixPlan{}, fmt.Errorf("refusing to modify %q because only some target methods are already guarded", SourceTestPath)
	}
	if AlreadyGuardedMethodCount == len(WindowsTestMethodDefinitions) {
		return WindowsModernCaptureTestFixPlan{SourceTestPath: SourceTestPath, SourceTestContent: SourceTestContent}, nil
	}
	return WindowsModernCaptureTestFixPlan{SourceTestPath: SourceTestPath, SourceTestContent: SourceTestContent, MethodSourceRanges: MethodSourceRanges}, nil
}

func FindTestMethodSourceRange(SourceTestContent string, WindowsTestMethodDefinitionValue WindowsTestMethodDefinition) (SourceRange, bool, error) {
	MethodDeclarationOffset := strings.Index(SourceTestContent, WindowsTestMethodDefinitionValue.MethodDeclaration)
	if MethodDeclarationOffset < 0 {
		return SourceRange{}, false, fmt.Errorf("could not find required test method %q with its expected signature", WindowsTestMethodDefinitionValue.MethodName)
	}
	if strings.Index(SourceTestContent[MethodDeclarationOffset+len(WindowsTestMethodDefinitionValue.MethodDeclaration):], WindowsTestMethodDefinitionValue.MethodDeclaration) >= 0 {
		return SourceRange{}, false, fmt.Errorf("found more than one test method with the expected signature for %q", WindowsTestMethodDefinitionValue.MethodName)
	}

	MethodStartOffset := FindTestMethodStartOffset(SourceTestContent, MethodDeclarationOffset)
	MethodEndOffset, MethodEndError := FindTestMethodEndOffset(SourceTestContent, MethodDeclarationOffset)
	if MethodEndError != nil {
		return SourceRange{}, false, fmt.Errorf("find end of test method %q: %w", WindowsTestMethodDefinitionValue.MethodName, MethodEndError)
	}
	if IsWindowsGuardedMethod(SourceTestContent, MethodStartOffset, MethodEndOffset) {
		return SourceRange{}, true, nil
	}
	return SourceRange{StartOffset: MethodStartOffset, EndOffset: MethodEndOffset}, false, nil
}

func FindTestMethodStartOffset(SourceTestContent string, MethodDeclarationOffset int) int {
	MethodLineStartOffset := strings.LastIndex(SourceTestContent[:MethodDeclarationOffset], "\n") + 1
	CurrentStartOffset := MethodLineStartOffset
	for CurrentStartOffset > 0 {
		PreviousLineEndOffset := CurrentStartOffset - 1
		PreviousLineStartOffset := strings.LastIndex(SourceTestContent[:PreviousLineEndOffset], "\n") + 1
		PreviousLine := strings.TrimSpace(SourceTestContent[PreviousLineStartOffset:PreviousLineEndOffset])
		if PreviousLine == "" || (strings.HasPrefix(PreviousLine, "[") && strings.HasSuffix(PreviousLine, "]")) {
			CurrentStartOffset = PreviousLineStartOffset
			continue
		}
		break
	}
	return CurrentStartOffset
}

func FindTestMethodEndOffset(SourceTestContent string, MethodDeclarationOffset int) (int, error) {
	OpeningBraceRelativeOffset := strings.Index(SourceTestContent[MethodDeclarationOffset:], "{")
	if OpeningBraceRelativeOffset < 0 {
		return 0, fmt.Errorf("opening brace was not found")
	}
	OpeningBraceOffset := MethodDeclarationOffset + OpeningBraceRelativeOffset
	BraceDepth := 0
	for CurrentOffset := OpeningBraceOffset; CurrentOffset < len(SourceTestContent); CurrentOffset++ {
		switch SourceTestContent[CurrentOffset] {
		case '{':
			BraceDepth++
		case '}':
			BraceDepth--
			if BraceDepth == 0 {
				return CurrentOffset + 1, nil
			}
		}
	}
	return 0, fmt.Errorf("closing brace was not found")
}

func IsWindowsGuardedMethod(SourceTestContent string, MethodStartOffset int, MethodEndOffset int) bool {
	BeforeMethod := strings.TrimRight(SourceTestContent[:MethodStartOffset], " \t\r\n")
	AfterMethod := strings.TrimLeft(SourceTestContent[MethodEndOffset:], " \t\r\n")
	return strings.HasSuffix(BeforeMethod, "#if WINDOWS") && strings.HasPrefix(AfterMethod, "#endif")
}

func CreateWindowsModernCaptureTestSnapshot(WindowsModernCaptureTestFixPlanValue WindowsModernCaptureTestFixPlan, OperationLogValue OperationLog) (string, error) {
	SnapshotFileName := "xerahs-windows-modern-capture-tests-" + OperationLogValue.OperationID + ".cs"
	SnapshotPath := filepath.Join(filepath.Dir(OperationLogValue.LogPath), SnapshotFileName)
	if SnapshotWriteError := os.WriteFile(SnapshotPath, []byte(WindowsModernCaptureTestFixPlanValue.SourceTestContent), 0644); SnapshotWriteError != nil {
		return "", fmt.Errorf("create test source snapshot %q: %w", SnapshotPath, SnapshotWriteError)
	}
	return SnapshotPath, nil
}

func AddWindowsGuardsToTestMethods(WindowsModernCaptureTestFixPlanValue WindowsModernCaptureTestFixPlan) string {
	DestinationTestContent := WindowsModernCaptureTestFixPlanValue.SourceTestContent
	for CurrentRangeIndex := len(WindowsModernCaptureTestFixPlanValue.MethodSourceRanges) - 1; CurrentRangeIndex >= 0; CurrentRangeIndex-- {
		CurrentRange := WindowsModernCaptureTestFixPlanValue.MethodSourceRanges[CurrentRangeIndex]
		MethodContent := DestinationTestContent[CurrentRange.StartOffset:CurrentRange.EndOffset]
		GuardedMethodContent := "#if WINDOWS\n" + MethodContent + "\n#endif"
		DestinationTestContent = DestinationTestContent[:CurrentRange.StartOffset] + GuardedMethodContent + DestinationTestContent[CurrentRange.EndOffset:]
	}
	return DestinationTestContent
}

func WriteDestinationTestContent(DestinationTestPath string, DestinationTestContent string) error {
	TemporaryDestinationTestPath := DestinationTestPath + ".xerahs-installer.tmp"
	if TemporaryWriteError := os.WriteFile(TemporaryDestinationTestPath, []byte(DestinationTestContent), 0644); TemporaryWriteError != nil {
		return fmt.Errorf("write temporary fixed test file %q: %w", TemporaryDestinationTestPath, TemporaryWriteError)
	}
	if RenameError := os.Rename(TemporaryDestinationTestPath, DestinationTestPath); RenameError != nil {
		return fmt.Errorf("replace fixed test file %q: %w", DestinationTestPath, RenameError)
	}
	return nil
}
