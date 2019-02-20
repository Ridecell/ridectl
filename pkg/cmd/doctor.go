/*
Copyright 2019 Ridecell, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/fatih/color"
	"github.com/mattn/go-shellwords"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var interactiveFlag bool

func init() {
	doctorCmd.Flags().BoolVarP(&interactiveFlag, "interactive", "i", false, "enable interactive mode")

	rootCmd.AddCommand(doctorCmd)
}

var doctorCmd = &cobra.Command{
	Use:   "doctor [flags]",
	Short: "Find common environment issues",
	Long:  `Find, and correct, common setup issues with ridectl`,
	Args: func(_ *cobra.Command, args []string) error {
		if len(args) > 0 {
			return fmt.Errorf("Too many arguments")
		}
		return nil
	},
	RunE: func(_ *cobra.Command, args []string) error {
		// Interactive mode only on Mac for now.
		if interactiveFlag && runtime.GOOS != "darwin" {
			return errors.New("Interactive mode is only supported on macOS for now")
		}

		tests := []*doctorTest{
			doctorTestHomebrew,
			doctorTestCaskroom,
			doctorTestGcloud,
			doctorTestKubectl,
			doctorTestGoogleCredentials,
			doctorTestAsdf,
		}

		for _, test := range tests {
			err := test.run()
			if err != nil {
				return err
			}
		}

		return nil
	},
}

type doctorTest struct {
	subject  string
	checkCmd string
	checkFn  func() bool
	fixCmd   string
	fixFn    func() error

	checkingMsg string
	fixMsg      string
}

func (t *doctorTest) run() error {
	if t.runCheck() {
		t.showPass()
	} else {
		t.showFail()
		if interactiveFlag {
			return t.tryFix()
		}
	}
	return nil
}

func (t *doctorTest) runCheck() bool {
	t.showChecking()
	var ok bool
	if t.checkFn != nil {
		ok = t.checkFn()
	} else if t.checkCmd != "" {
		_, err := exec.LookPath(t.checkCmd)
		ok = (err == nil)
	} else {
		panic("invalid test")
	}
	t.unshowChecking()
	return ok
}

func (t *doctorTest) showChecking() {
	if t.checkingMsg == "" {
		t.checkingMsg = fmt.Sprintf("  Checking for %s", t.subject)
	}
	fmt.Print(t.checkingMsg)
}

func (t *doctorTest) unshowChecking() {
	fmt.Print(strings.Repeat("\b", len(t.checkingMsg)))
}

func (t *doctorTest) showPass() {
	passMsg := fmt.Sprintf("✅ Found %s       ", t.subject)
	color.Green(passMsg)
}

func (t *doctorTest) showFail() {
	failMsg := fmt.Sprintf("❌ Did not find %s", t.subject)
	color.Red(failMsg)
}

func (t *doctorTest) tryFix() error {
	if t.fixFn == nil && t.fixCmd == "" {
		return nil
	}

	if t.fixMsg == "" {
		var buf strings.Builder
		buf.WriteString(fmt.Sprintf("Would you like to fix %s", t.subject))
		if t.fixCmd != "" {
			buf.WriteString(fmt.Sprintf(" (%s)", t.fixCmd))
		}
		buf.WriteString("? [y/n] ")
		t.fixMsg = buf.String()
	}
	fmt.Print(t.fixMsg)
	reader := bufio.NewReader(os.Stdin)
	char, _, err := reader.ReadRune()
	if err != nil {
		return err
	}
	if char != 'y' && char != 'Y' {
		// Assume everything else is a no
		return nil
	}
	if t.fixFn != nil {
		return t.fixFn()
	} else {
		words, err := shellwords.Parse(t.fixCmd)
		if err != nil {
			return err
		}
		cmd := exec.Command(words[0], words[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
}

// Check for Homebrew.
var doctorTestHomebrew = &doctorTest{
	subject:  "Homebrew",
	checkCmd: "brew",
	fixCmd:   `/usr/bin/ruby -e "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install)"`,
}

// Check for Caskroom.
var doctorTestCaskroom = &doctorTest{
	subject: "Homebrew Caskroom",
	checkFn: func() bool {
		// I think?
		_, err := os.Stat("/usr/local/Caskroom")
		return !os.IsNotExist(err)
	},
	fixCmd: `brew tap caskroom/cask`,
}

// Check for gcloud CLI.
var doctorTestGcloud = &doctorTest{
	subject:  "Google Cloud CLI",
	checkCmd: "gcloud",
	fixCmd:   `brew cask install google-cloud-sdk`,
}

// Check for kubectl.
var doctorTestKubectl = &doctorTest{
	subject:  "Kubectl CLI",
	checkCmd: "kubectl",
	fixCmd:   `brew install kubernetes-cli`,
}

// Check for gcloud credentials.
var doctorTestGoogleCredentials = &doctorTest{
	subject: "Google Cloud CLI credentials",
	checkFn: func() bool {
		cmd := exec.Command("gcloud", "config", "get-value", "account")
		var buf strings.Builder
		cmd.Stdout = &buf
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			return false
		}
		return !strings.HasPrefix(buf.String(), "(unset)")
	},
}

// Check for Kubernetes context for noah-test.

// Check example Kubernetes command.

// Check for AWS credentials.

// Check for access to the flavors S3 bucket.

// Check for test.
var doctorTestAsdf = &doctorTest{
	subject:  "asdf",
	checkCmd: "qwerasdf",
	fixCmd:   `brew help`,
}
