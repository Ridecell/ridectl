/*
Copyright 2011 Ridecell, Inc.

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
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Ridecell/ridectl/pkg/cmd/edit"
	"github.com/pterm/pterm"

	//"github.com/Ridecell/ridectl/pkg/kubernetes"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(editCmd)
}

var filenameFlag string
var keyIdFlag string
var recrypt bool

var whitespaceRegexp *regexp.Regexp

func init() {
	editCmd.Flags().BoolVarP(&recrypt, "recrypt", "r", false, "(optional) re-encrypts all secrets in file")
	editCmd.Flags().StringVarP(&filenameFlag, "file", "f", "", "(optional) Path to the file to edit")
	editCmd.Flags().StringVarP(&keyIdFlag, "key", "k", "", "(optional) KMS key ID to use for encrypting")

	whitespaceRegexp = regexp.MustCompile(`\s+`)
}

/*

An explanation of the overall edit process:

1. The existing file is loaded and parsed.
2. That parsed data is decrypted using KMS.
3. A new YAML document is written to a tempfile with the decrypted data.
4. The tempfile is opened in $EDITOR.
5. The tempfile is re-read and parsed.
6. The old and new data is correlated to match up any objects that exist in both.
7. The parsed data is encrypted using KMS if the value changed.
8. A new YAML document is written to the original file.

*/

var editCmd = &cobra.Command{
	Use:   "edit [flags] <cluster_name>",
	Short: "Edit an instance manifest",
	Long:  `Edit a Summon instance manifest file that contains encrypted secret values`,
	Args: func(_ *cobra.Command, args []string) error {
		if filenameFlag == "" {
			if len(args) == 0 {
				pterm.Error.Println("cluster name argument is required")
				os.Exit(1)
			}
			if len(args) > 1 {
				pterm.Error.Println("too many arguments")
				os.Exit(1)
			}
		} else {
			if len(args) > 0 {
				pterm.Error.Println("too many arguments")
				os.Exit(1)
			}
		}
		return nil
	},
	RunE: func(_ *cobra.Command, args []string) error {
		// Work out which file we are editing.
		filename := filenameFlag
		if filename == "" {
			match := regexp.MustCompile(`^([a-z0-9]+)-([a-z]+)$`).FindStringSubmatch(args[0])
			if match == nil {
				return errors.Errorf("unable to parse instance name %s", args[0])
			}

			filenames, err := filepath.Glob(fmt.Sprintf(`*%s/%s.yml`, match[2], match[1]))
			if len(filenames) > 1 {
				return errors.New("found multiple matches for filepath")
			}
			if err != nil {
				return err
			}

			if filenames != nil {
				filename = filenames[0]
			} else {
				// Prompt user for region when creating new file
				regionPrompt := promptui.Prompt{
					Label: "Enter region (eu, us, in, etc.)",
				}
				fileRegion, err := regionPrompt.Run()
				if err != nil {
					return err
				}

				filename = fmt.Sprintf("%s-%s/%s.yml", fileRegion, match[2], match[1])

			}
		}

		// Read the file in.
		var inStream io.Reader
		inFile, err := os.Open(filename)
		if err != nil {
			if os.IsNotExist(err) && filenameFlag == "" {
				// No file, render the template with the default content.
				//return error for now
				return errors.Wrapf(err, "error reading input file %s", filename)
				// buffer, err := createDefaultData(args[0])
				// if err != nil {
				// 	return errors.Wrap(err, "error creating default data")
				// }
				// inStream = buffer
			} else {
				return errors.Wrapf(err, "error reading input file %s", filename)
			}
		} else {
			defer inFile.Close()
			inStream = inFile
		}

		// Parse the input file to objects.
		inManifest, err := edit.NewManifest(inStream)
		if err != nil {
			return errors.Wrap(err, "error decoding input YAML")
		}

		// Create a KMS session
		// TODO error handling for AWS creds
		sess := session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
			Config: aws.Config{
				Region: aws.String("us-west-1"),
			},
		}))
		kmsService := kms.New(sess)

		// Decrypt all the encrypted secrets.
		err = inManifest.Decrypt(kmsService)
		if err != nil {
			return errors.Wrap(err, "error decrypting input manifest")
		}

		// Edit!
		afterManifest, err := editObjects(inManifest, "")
		if err != nil {
			return errors.Wrap(err, "error editing objects")
		}

		// Match up the new objects with the old.
		_ = afterManifest.CorrelateWith(inManifest)

		// Re-encrypt anything that needs it.
		keyId := keyIdFlag
		if keyId == "" {
			keyId, err = edit.FindKeyId(filename)
			if err != nil {
				return errors.Wrap(err, "error finding key ID")
			}
		}

		err = afterManifest.Encrypt(kmsService, keyId, keyIdFlag != "" || recrypt, recrypt)
		if err != nil {
			return errors.Wrap(err, "error encrypting after manifest")
		}

		// Write out the file again.
		// TODO make sure the file is writable before doing all this.
		outFile, err := os.Create(filename)
		if err != nil {
			return errors.Wrapf(err, "error opening %s for writing", filename)
		}
		defer outFile.Close()
		_ = afterManifest.Serialize(outFile)

		pterm.Info.Printf("Encrypted using %s\n", keyId)
		return nil
	},
}

func runEditor(filename string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		return errors.New("No $EDITOR set")
	}

	// Deal with an editor that has options.
	editorParts := whitespaceRegexp.Split(editor, -1)
	executable := editorParts[0]
	executable, _ = exec.LookPath(executable)

	editorParts = append(editorParts, filename)
	cmd := exec.Command(executable, editorParts[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return errors.Wrap(err, "error running editor")
	}
	return nil
}

func editObjects(manifest edit.Manifest, comment string) (edit.Manifest, error) {
	manifestBuf := bytes.Buffer{}
	err := manifest.Serialize(&manifestBuf)
	if err != nil {
		return nil, errors.Wrap(err, "error encoding objects to YAML")
	}
	for {
		// Format the comment.
		commentBuf := bytes.Buffer{}
		if comment != "" {
			for _, line := range strings.Split(comment, "\n") {
				commentBuf.WriteString("# ")
				commentBuf.WriteString(line)
				commentBuf.WriteString("\n")
			}
			commentBuf.WriteString("#\n")
		}
		commentReader := bytes.NewReader(commentBuf.Bytes())

		// Make the YAML to show in the editor.
		editorBuf := bytes.Buffer{}
		_, _ = commentReader.WriteTo(&editorBuf)
		_, _ = manifestBuf.WriteTo(&editorBuf)
		editorReader := bytes.NewReader(editorBuf.Bytes())

		// Open a temporary file.
		tmpfile, err := ioutil.TempFile("", ".*.yml")
		if err != nil {
			return nil, errors.Wrap(err, "error making tempfile")
		}
		defer tmpfile.Close()
		defer os.Remove(tmpfile.Name())
		_, _ = editorReader.WriteTo(tmpfile)
		_ = tmpfile.Sync()

		// Show the editor.
		err = runEditor(tmpfile.Name())
		if err != nil {
			return nil, errors.Wrap(err, "error running editor")
		}

		// Re-read the edited file.
		afterTmpfile, err := os.Open(tmpfile.Name())
		if err != nil {
			return nil, errors.Wrapf(err, "error re-opening tempfile %s", tmpfile.Name())
		}
		defer afterTmpfile.Close()
		afterBuf := bytes.Buffer{}
		_, err = afterBuf.ReadFrom(afterTmpfile)
		if err != nil {
			return nil, errors.Wrapf(err, "error reading tempfile %s", tmpfile.Name())
		}

		// If we're reencrypting ignore this equality check.
		// Check if the file was edited at all.
		if bytes.Equal(editorBuf.Bytes(), afterBuf.Bytes()) && !recrypt {
			return nil, errors.New("tempfile not edited, aborting")
		}

		// Try strip off the comment.
		afterReader := bytes.NewReader(afterBuf.Bytes())
		seekPos := int64(0)
		if bytes.Equal(commentBuf.Bytes(), afterBuf.Bytes()[:commentBuf.Len()]) {
			seekPos = int64(commentBuf.Len())
		}
		_, _ = afterReader.Seek(seekPos, 0)

		outManifest, err := edit.NewManifest(afterReader)
		if err == nil {
			// Decode success, we're done!
			return outManifest, nil
		}

		// Some kind decoding error, probably bad syntax, show the editor again.
		comment = fmt.Sprintf("Error parsing file:\n%s", err)
		manifestBuf.Reset()
		_, _ = afterReader.Seek(seekPos, 0)
		_, _ = afterReader.WriteTo(&manifestBuf)
	}
}

// func createDefaultData(instance string) (io.Reader, error) {
// 	target, err := kubernetes.ParseSubject(instance)
// 	if err != nil {
// 		return nil, fmt.Errorf("unable to parse instance name %s", instance)
// 	}
// 	templateData, err := vfsutil.ReadFile(Templates, "new_instance.yml.tpl")
// 	if err != nil {
// 		return nil, errors.Wrap(err, "error reading new instance template")
// 	}
// 	template, err := template.New("new_instance.yml.tpl").Parse(string(templateData))
// 	if err != nil {
// 		return nil, errors.Wrap(err, "error parsing new instance template")
// 	}

// 	// Prompt user for a slack channel to alert to
// 	slackChannelPrompt := promptui.Prompt{
// 		Label: "Enter a slack channel name (#channel-name, blank to skip)",
// 		Validate: func(input string) error {
// 			if !strings.HasPrefix(input, "#") && input != "" {
// 				return errors.New(`Channel name must have prefix "#"`)
// 			}
// 			return nil
// 		},
// 	}
// 	slackChannelName, err := slackChannelPrompt.Run()
// 	if err != nil {
// 		return nil, err
// 	}

// 	buffer := &bytes.Buffer{}
// 	err = template.Execute(buffer, struct {
// 		Name         string
// 		Namespace    string
// 		SlackChannel string
// 	}{
// 		Name:         instance,
// 		Namespace:    target.Namespace,
// 		SlackChannel: slackChannelName,
// 	})
// 	if err != nil {
// 		return nil, errors.Wrap(err, "error rendering new instance template")
// 	}
// 	return buffer, nil
// }
