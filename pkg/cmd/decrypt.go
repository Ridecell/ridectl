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
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/Ridecell/ridectl/pkg/cmd/edit"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/kms/kmsiface"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/nacl/secretbox"
)

var (
	keyMap = map[string]*[32]byte{}
)

func init() {
	rootCmd.AddCommand(decryptCmd)
}

/*

An explanation of the overall decrypt process:

1. The existing file is loaded
2. Then its decoded using encoding/gob library
3. The cipher data key is extracted from decoded file data, and then using KMS decrypt, we obtain plainData key from cipher data key
4. Then, encrypted file data is decrypted using plain data key, and written to file.

*/

var decryptCmd = &cobra.Command{
	Use:   "decrypt <file-names>",
	Short: "Decrypt files",
	Long:  `decrypt files that has secret values`,
	Args: func(_ *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("Filename(s) are required")
		}
		return nil
	},
	RunE: func(_ *cobra.Command, fileNames []string) error {
		// Create AWS KMS session
		sess := session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
			Config: aws.Config{
				Region: aws.String("us-west-1"),
			},
		}))
		kmsService := kms.New(sess)

		for _, filename := range fileNames {
			// read file content
			fileContent, err := ioutil.ReadFile(filename)
			if err != nil {
				return errors.Wrapf(err, "error reading file: %s", filename)
			}

			// call function here
			plaintext, err := GetDecryptedData(kmsService, fileContent)
			if err != nil {
				return errors.Wrapf(err, "filename: %s", filename)
			}

			// output file name
			out_filename := strings.TrimSuffix(filename, ".encrypted")

			// Check if out_filename exists and has same decrypted data
			// If true, don't need to write file
			decryptedFileContent, err := ioutil.ReadFile(out_filename)
			if err == nil {
				if string(decryptedFileContent) == string(plaintext) {
					fmt.Println("No changes: " + out_filename)
					continue
				}
			}

			// write decrypted content in <filename>.decrypted
			err = ioutil.WriteFile(out_filename, plaintext, 0644)
			if err != nil {
				return errors.Wrapf(err, "error writing file: %s", filename)
			}
			fmt.Println("Decrypted : " + out_filename)
		}

		return nil
	},
}

func GetDecryptedData(kmsService kmsiface.KMSAPI, encryptedData []byte) ([]byte, error) {
	var p edit.Payload
	var plaintext []byte

	decodedData := make([]byte, base64.StdEncoding.DecodedLen(len(encryptedData)))
	_, err := base64.StdEncoding.Decode(decodedData, encryptedData)
	if err != nil {
		return plaintext, errors.Wrap(err, "error base64 decoding value")
	}

	gob.NewDecoder(bytes.NewReader(decodedData)).Decode(&p)
	plainDataKey, ok := keyMap[string(p.Key)]
	if !ok {
		// Decrypt cipherdatakey
		plainDataKey, err = edit.DecryptCipherDataKey(kmsService, p.Key)
		if err != nil {
			return plaintext, errors.Wrap(err, "error decrypting value for cipherDatakey")
		}
		keyMap[string(p.Key)] = plainDataKey
	}

	// Decrypt file content
	plaintext, ok = secretbox.Open(plaintext, p.Message, p.Nonce, plainDataKey)
	if !ok {
		return plaintext, errors.Wrap(err, "error decrypting value with data key")
	}
	return plaintext, nil
}
