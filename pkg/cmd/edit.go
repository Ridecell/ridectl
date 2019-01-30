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
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"
)

func init() {
	rootCmd.AddCommand(editCmd)
}

const Root = "/Users/coderanger/src/rc/kubernetes-summon"

var editCmd = &cobra.Command{
	Use:   "edit [flags] <cluster_name>",
	Short: "",
	Long:  ``,
	Args: func(_ *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("Cluster name argument is required")
		}
		if len(args) > 1 {
			return fmt.Errorf("Too many arguments")
		}
		return nil
	},
	RunE: func(_ *cobra.Command, args []string) error {
		match := regexp.MustCompile(`^([a-z0-9]+)-([a-z]+)$`).FindStringSubmatch(args[0])
		if match == nil {
			return errors.Errorf("unable to parse instance name %s", args[0])
		}

		clusterFile := filepath.Join(Root, match[2], match[1]+".yml")

		// Check if the cluster file exists.
		_, err := os.Stat(clusterFile)
		if os.IsNotExist(err) {
			// Make template, render, etc
			// TODO
			fmt.Printf("File doesn't exist, TODO: %s\n", clusterFile)
		} else {
			dat, err := ioutil.ReadFile(clusterFile)
			if err != nil {
				return errors.Wrap(err, "unable to read cluster file")
			}
			objects := []map[interface{}]interface{}{}
			dec := yaml.NewDecoder(bytes.NewReader(dat))
			for {
				m := map[interface{}]interface{}{}
				err = dec.Decode(&m)
				if err == io.EOF {
					// all done
					break
				}
				if err != nil {
					return errors.Wrap(err, "error parsing cluster file")
				}
				objects = append(objects, m)
			}

			var everythingElse []map[interface{}]interface{}
			var encryptedSecret map[interface{}]interface{}
			for _, obj := range objects {
				if obj["kind"] == "EncryptedSecret" {
					encryptedSecret = obj
				} else {
					everythingElse = append(everythingElse, obj)
				}
			}
			if encryptedSecret == nil {
				encryptedSecret = map[interface{}]interface{}{
					"apiVersion": "secrets.ridecell.io/v1beta1",
					"kind":       "EncryptedSecret",
					"metadata": map[string]string{
						"name":      "",
						"namespace": "",
					},
					"data": map[string]string{},
				}
			}

			decryptedData := map[interface{}]string{}
			decryptedSecret := map[interface{}]interface{}{
				"kind":     "decryptedSecret",
				"metadata": encryptedSecret["metadata"],
				"data":     decryptedData,
			}

			sess := session.Must(session.NewSessionWithOptions(session.Options{
				SharedConfigState: session.SharedConfigEnable,
			}))
			kmsService := kms.New(sess)
			for k, v := range encryptedSecret["data"].(map[interface{}]interface{}) {
				decryptedValue, err := kmsService.Decrypt(&kms.DecryptInput{CiphertextBlob: []byte(v.(string))})
				if err != nil {
					return errors.Wrapf(err, "error decrypting value for %s", k)
				}
				decryptedData[k] = string(decryptedValue.Plaintext)
			}

			tmpfile, err := ioutil.TempFile("", args[0]+".*.yml")
			if err != nil {
				return errors.Wrap(err, "unable to make tempfile")
			}
			defer os.Remove(tmpfile.Name())

			encoder := yaml.NewEncoder(tmpfile)
			for _, obj := range everythingElse {
				encoder.Encode(obj)
			}
			encoder.Encode(decryptedSecret)

			editor := os.Getenv("EDITOR")
			if editor == "" {
				return errors.New("No $EDITOR set")
			}

			cmd := exec.Command(editor, tmpfile.Name())
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Run()

			tmpfile.Seek(0, 0)

			afterObjects := []map[interface{}]interface{}{}
			dec = yaml.NewDecoder(tmpfile)
			for {
				m := map[interface{}]interface{}{}
				err = dec.Decode(&m)
				if err == io.EOF {
					// all done
					break
				}
				if err != nil {
					return errors.Wrap(err, "error parsing after data")
				}
				afterObjects = append(afterObjects, m)
			}

			var afterEverythingElse []map[interface{}]interface{}
			var afterDecryptedSecret map[interface{}]interface{}
			for _, obj := range afterObjects {
				if obj["kind"] == "decryptedSecret" {
					afterDecryptedSecret = obj
				} else {
					afterEverythingElse = append(afterEverythingElse, obj)
				}
			}

			fmt.Printf("%#v\n", afterDecryptedSecret)

			reencryptedData := map[interface{}]string{}
			reencryptedSecret := map[interface{}]interface{}{
				"apiVersion": encryptedSecret["apiVersion"],
				"kind":       encryptedSecret["kind"],
				"metadata":   afterDecryptedSecret["metadata"],
				"data":       reencryptedData,
			}

			for k, v := range afterDecryptedSecret["data"].(map[interface{}]interface{}) {
				encryptedValue, err := kmsService.Encrypt(&kms.EncryptInput{
					KeyId:     aws.String(os.Getenv("KEY")),
					Plaintext: []byte(v.(string)),
				})
				if err != nil {
					return errors.Wrapf(err, "error encypted value for %s", k)
				}
				reencryptedData[k] = base64.StdEncoding.EncodeToString(encryptedValue.CiphertextBlob)
			}

			fmt.Printf("%#v\n", reencryptedSecret)

			buf := strings.Builder{}
			encoder = yaml.NewEncoder(&buf)
			for _, obj := range afterEverythingElse {
				encoder.Encode(obj)
			}
			encoder.Encode(reencryptedSecret)

			fmt.Printf("%s\n", buf.String())
		}

		return nil
	},
}
