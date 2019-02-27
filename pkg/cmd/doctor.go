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
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
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

		//tests := []*doctorTest{
		//	doctorTestKubectlContext,
		//}

		tests := []*doctorTest{
			doctorTestEditorEnvVar,
			doctorTestHomebrew,
			doctorTestCaskroom,
			doctorTestGcloud,
			doctorTestKubectl,
			doctorTestKubectlCommand,
			doctorTestGoogleCredentials,
			doctorTestKubectlContext,
			doctorTestAWSCredentials,
			doctorTestS3Access,
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
		buf.WriteString("? ")
		t.fixMsg = buf.String()
	}
	yes, err := getUserConfirmation(t.fixMsg)
	if err != nil {
		return err
	}
	if !yes {
		return nil
	}

	if t.fixFn != nil {
		return t.fixFn()
	}
	words, err := shellwords.Parse(t.fixCmd)
	if err != nil {
		return err
	}
	cmd := exec.Command(words[0], words[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()

}

// Check if EDITOR environment variable is set for the edit command
var doctorTestEditorEnvVar = &doctorTest{
	subject: "$EDITOR Environment Variable",
	checkFn: func() bool {
		return os.Getenv("EDITOR") != ""
	},
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
	fixCmd: `gcloud auth login`,
}

// Check for Kubernetes context for noah-test.
var doctorTestKubectlContext = &doctorTest{
	subject: `Kubenerets Config "noah-test"`,
	checkFn: func() bool {
		cmd := exec.Command("kubectl", "config", "current-context")
		var buf strings.Builder
		cmd.Stdout = &buf
		err := cmd.Run()
		if err != nil {
			return false
		}
		return strings.Contains(buf.String(), "noah-test")
	},
}

// Check example Kubernetes command.
var doctorTestKubectlCommand = &doctorTest{
	subject: "Kubernetes Test",
	checkFn: func() bool {
		cmd := exec.Command("kubectl", "version")
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			return false
		}
		return true
	},
}

// Check for AWS credentials.
var doctorTestAWSCredentials = &doctorTest{
	subject: "AWS Credentials",
	checkFn: func() bool {
		sess, err := session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
		})
		if err != nil {
			return false
		}
		_, err = sess.Config.Credentials.Get()
		if err != nil {
			return false
		}
		return true
	},
	fixFn: func() error {
		yes, err := getUserConfirmation("Do you have an AWS Access Key ")
		if err != nil {
			return err
		}
		if !yes {
			fmt.Println("Please contact devops/infra team for assistance.")
			return nil
		}

		awsDir := fmt.Sprintf(`%s/.aws`, os.Getenv("HOME"))
		credentialsPath := fmt.Sprintf("%s/credentials", awsDir)
		// Check if the credentials file exists
		_, err = os.Stat(credentialsPath)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		// If the credentials file exists exit, we aren't editing that.
		if !os.IsNotExist(err) {
			fmt.Printf("%s Already exists. This file should be configured manually.\n", credentialsPath)
			return err
		}

		accessKeyPrompt := promptui.Prompt{
			Label: "Enter aws_access_key_id: ",
			Validate: func(input string) error {
				if !strings.HasPrefix(input, "AKIA") {
					return errors.New("Access key must have prefix of AKIA")
				}
				if len(input) < 16 {
					return errors.New("Access Key must be at least 16 digits long")
				}
				return nil
			},
		}
		accessKey, err := accessKeyPrompt.Run()
		if err != nil {
			return err
		}

		secretKeyPrompt := promptui.Prompt{
			Label: "Enter aws_secret_access_key: ",
			Validate: func(input string) error {
				if len(input) < 16 {
					return errors.New("Secret key must be at least 16 digits long")
				}
				return nil
			},
			Mask: 'X',
		}
		secretKey, err := secretKeyPrompt.Run()
		if err != nil {
			return err
		}

		// Test that credentials are valid before we write them to file.
		sess, err := session.NewSessionWithOptions(session.Options{
			Config: aws.Config{
				Credentials: credentials.NewStaticCredentials(accessKey, secretKey, ""),
			},
		})
		if err != nil {
			return err
		}
		svc := sts.New(sess)
		// This call will succeed with valid credntials regardless of permissions.
		_, err = svc.GetCallerIdentity(&sts.GetCallerIdentityInput{})
		if err != nil {
			fmt.Println("Provided AWS credentials are not valid.")
			return err
		}

		// Make sure our .aws directory exists
		_, err = os.Stat(awsDir)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		// Directory doesn't exist, create it.
		if os.IsNotExist(err) {
			err = os.Mkdir(awsDir, 0755)
			if err != nil {
				return err
			}
		}

		// This will not error if the file exists, we need to check before we get here.
		file, err := os.Create(credentialsPath)
		if err != nil {
			return err
		}
		defer file.Close()

		// Write our new credentials to the file.
		_, err = file.WriteString(fmt.Sprintf("[default]\naws_access_key_id = %s\naws_secret_access_key = %s\n", accessKey, secretKey))
		if err != nil {
			return err
		}
		return nil
	},
}

// Check for access to the flavors S3 bucket.
var doctorTestS3Access = &doctorTest{
	subject: "S3 Flavors Access",
	checkFn: func() bool {
		sess, err := session.NewSessionWithOptions(session.Options{
			Config: aws.Config{
				Region: aws.String("us-west-2"),
			},
			SharedConfigState: session.SharedConfigEnable,
		})
		if err != nil {
			return false
		}
		svc := s3.New(sess)

		_, err = svc.ListObjects(&s3.ListObjectsInput{
			Bucket:  aws.String("ridecell-flavors"),
			MaxKeys: aws.Int64(1),
		})
		if err != nil {
			return false
		}
		return true
	},
}

// Ask user [y/n] return [true/false]
func getUserConfirmation(label string) (bool, error) {
	prompt := promptui.Prompt{
		Label:     label,
		IsConfirm: true,
	}
	_, err := prompt.Run()
	if err != nil {
		// ErrAbort = N, do not return err
		if err == promptui.ErrAbort {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
