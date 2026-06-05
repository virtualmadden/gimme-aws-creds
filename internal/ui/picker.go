package ui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"golang.org/x/term"
)

var stdin io.Reader = os.Stdin
var stdout io.Writer = os.Stdout

// SetIO overrides stdin/stdout for testing.
func SetIO(in io.Reader, out io.Writer) {
	stdin = in
	stdout = out
}

// ResetIO restores default stdin/stdout.
func ResetIO() {
	stdin = os.Stdin
	stdout = os.Stdout
}

// Pick presents a selectable list and returns the selected index.
// Uses arrow-key navigation when stdin is a terminal; falls back to numbered input otherwise.
func Pick(prompt string, options []string) (int, error) {
	if len(options) == 0 {
		return 0, fmt.Errorf("no options to pick from")
	}
	if len(options) == 1 {
		fmt.Fprintf(stdout, "%s\n  → %s\n", prompt, options[0])
		return 0, nil
	}

	if f, ok := stdin.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		return pickSurvey(prompt, options)
	}

	return pickNumbered(prompt, options)
}

func pickSurvey(prompt string, options []string) (int, error) {
	var answer string
	question := &survey.Select{
		Message: prompt,
		Options: options,
	}
	if err := survey.AskOne(question, &answer); err != nil {
		return 0, fmt.Errorf("selection cancelled: %w", err)
	}

	for i, opt := range options {
		if opt == answer {
			return i, nil
		}
	}
	return 0, fmt.Errorf("selected option %q not found", answer)
}

func pickNumbered(prompt string, options []string) (int, error) {
	fmt.Fprintln(stdout, prompt)
	for i, opt := range options {
		fmt.Fprintf(stdout, "  [%d] %s\n", i, opt)
	}

	reader := bufio.NewReader(stdin)
	for {
		fmt.Fprint(stdout, "Selection: ")
		line, err := reader.ReadString('\n')
		if err != nil {
			return 0, fmt.Errorf("read selection: %w", err)
		}
		line = strings.TrimSpace(line)
		idx, err := strconv.Atoi(line)
		if err != nil || idx < 0 || idx >= len(options) {
			fmt.Fprintln(stdout, "Invalid selection, try again.")
			continue
		}
		return idx, nil
	}
}

// Prompt reads a single line of input with a label.
func Prompt(label string) (string, error) {
	fmt.Fprintf(stdout, "%s", label)
	reader := bufio.NewReader(stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("read input: %w", err)
	}
	return strings.TrimSpace(line), nil
}

// PromptDefault reads input with a default value.
func PromptDefault(label, defaultValue string) (string, error) {
	if defaultValue != "" {
		fmt.Fprintf(stdout, "%s [%s]: ", label, defaultValue)
	} else {
		fmt.Fprintf(stdout, "%s: ", label)
	}
	reader := bufio.NewReader(stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("read input: %w", err)
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultValue, nil
	}
	return line, nil
}

// Confirm asks a yes/no question.
func Confirm(label string) (bool, error) {
	answer, err := PromptDefault(label+" (y/n)", "n")
	if err != nil {
		return false, err
	}
	answer = strings.ToLower(answer)
	return answer == "y" || answer == "yes", nil
}
