/*
Copyright 2021 Ridecell, Inc.

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
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/gob"
	"os"

	"github.com/Ridecell/ridectl/pkg/cmd/edit"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/nacl/secretbox"
)

func init() {
	rootCmd.AddCommand(encryptCmd)
}

var recrypt bool

func init() {
	encryptCmd.Flags().BoolVarP(&recrypt, "recrypt", "r", false, "(optional) re-encrypts the file")
	encryptCmd.Flags().StringVarP(&keyIdFlag, "key", "k", "", "(optional) KMS key ID / key alias to use for encrypting")
}

/*
An explanation of the encrypt process:

1. Generated KMS data key using given key id
2. The existing file is loaded.
3. First check if its encrypted copy exists, and has no change in data
4. If file is changed, then encrypt the file data using data key
5. Write encrypted data to file

*/

var encryptCmd = &cobra.Command{
	Use:   "encrypt [-k <kms-key-alias>] [-r] <file-names>",
	Short: "Encrypt files",
	Long:  `encrypt files that has secret values`,
	Args: func(_ *cobra.Command, args []string) error {
		if len(args) == 0 {
			pterm.Error.Printf("Filename(s) are required")
			os.Exit(1)
		}
		return nil
	},
	RunE: func(_ *cobra.Command, fileNames []string) error {
		// Check if key id is provided
		keyId := keyIdFlag
		if len(keyId) == 0 {
			keyId = "alias/microservices_dev"
			pterm.Info.Printf("---------------\nWARNING: Using %s KMS key by default, Please specify other key for Prod/UAT environment using -k option.\n         For example: ridectl encrypt -k alias/<key-alias> [file-names]\n---------------\n", keyId)
		}
		pterm.Info.Println("Encrypting using key: " + keyId)

		// Load the Shared AWS Configuration (~/.aws/config)
		cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-west-1"))
		if err != nil {
			return errors.Wrapf(err, "error creating AWS session")
		}

		// Create an Amazon KMS service client
		kmsService := kms.NewFromConfig(cfg)

		plainDataKey, cipherDataKey, err := edit.GenerateDataKey(kmsService, keyId)
		if err != nil {
			return errors.Wrapf(err, "error generating data key using KMS key: %s", keyId)
		}

		var p *edit.Payload
		for _, filename := range fileNames {
			// read file content
			fileContent, err := os.ReadFile(filename)
			if err != nil {
				return errors.Wrapf(err, "error reading file: %s", filename)
			}

			// Check if there is need to encrypt the file - the file content is changed.
			if !recrypt {
				encryptedFileContent, err := os.ReadFile(filename + ".encrypted")
				if err == nil {
					decryptedFileContent, err := GetDecryptedData(kmsService, encryptedFileContent)
					if err == nil {
						// If file content is not changed, then continue with next file
						if string(fileContent) == string(decryptedFileContent) {
							pterm.Info.Println("No changes: " + filename + ".encrypted")
							continue
						}
					}
				}
			}

			// encrypt file content
			p = &edit.Payload{
				Key:   cipherDataKey,
				Nonce: &[24]byte{},
			}
			// Set nonce
			if _, err = rand.Read(p.Nonce[:]); err != nil {
				return errors.Wrap(err, "error generating nonce.")
			}
			// Encrypt message
			p.Message = secretbox.Seal(p.Message, fileContent, p.Nonce, plainDataKey)
			buf := &bytes.Buffer{}
			if err = gob.NewEncoder(buf).Encode(p); err != nil {
				return errors.Wrapf(err, "error encrypting value using data key for file %s", filename)
			}
			encryptedFileContent := string(base64.StdEncoding.EncodeToString(buf.Bytes()))

			// write encrypted content in <filename>.encrypted
			err = os.WriteFile(filename+".encrypted", []byte(encryptedFileContent), 0644)
			if err != nil {
				return errors.Wrapf(err, "error writing file: %s", filename)
			}
			pterm.Success.Println("Encrypted : " + filename + ".encrypted")
		}

		return nil
	},
}
