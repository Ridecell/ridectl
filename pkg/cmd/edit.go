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
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Ridecell/ridectl/pkg/cmd/edit"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/kms/kmsiface"
	"github.com/pkg/errors"
	"github.com/shurcooL/httpfs/vfsutil"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	kyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"

	secretsv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/secrets/v1beta1"
)

func init() {
	rootCmd.AddCommand(editCmd)
}

const Root = "/Users/coderanger/src/rc/kubernetes-summon"

var filenameFlag string

func init() {
	editCmd.Flags().StringVarP(&filenameFlag, "file", "f", "", "(optional) Path to the file to edit")
}

type encryptedSecretContext struct {
	origEnc  *secretsv1beta1.EncryptedSecret
	origDec  *edit.DecryptedSecret
	afterDec *edit.DecryptedSecret
	other    runtime.Object
}

var editCmd = &cobra.Command{
	Use:   "edit [flags] <cluster_name>",
	Short: "Edit an instance manifest",
	Long:  `Edit a Summon instance manifest file that contains encrypted secret values`,
	Args: func(_ *cobra.Command, args []string) error {
		if filenameFlag == "" {
			if len(args) == 0 {
				return fmt.Errorf("Cluster name argument is required")
			}
			if len(args) > 1 {
				return fmt.Errorf("Too many arguments")
			}
		} else {
			if len(args) > 0 {
				return fmt.Errorf("Too many arguments")
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
			filename = filepath.Join(Root, match[2], match[1]+".yml")
		}

		// Read the file in.
		var inStream io.Reader
		inFile, err := os.Open(filename)
		if err != nil {
			if os.IsNotExist(err) && filenameFlag == "" {
				// No file, render the template with the default content.
				buffer, err := createDefaultData(args[0])
				if err != nil {
					return errors.Wrap(err, "error creating default data")
				}
				inStream = buffer
			} else {
				return errors.Wrapf(err, "error reading input file %s", filename)
			}
		} else {
			defer inFile.Close()
			inStream = inFile
		}

		// Parse the input file to objects.
		inObjs, err := decodeYaml(inStream)
		if err != nil {
			return errors.Wrap(err, "error decoding input YAML")
		}

		// Pull out the EncryptedSecret objects.
		objects := make([]encryptedSecretContext, 0, len(inObjs))
		for _, obj := range inObjs {
			ctx := encryptedSecretContext{}
			enc, ok := obj.(*secretsv1beta1.EncryptedSecret)
			if ok {
				ctx.origEnc = enc
			} else {
				ctx.other = obj
			}
			objects = append(objects, ctx)
		}

		// Create a KMS session
		// TODO error handling for AWS creds
		sess := session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
		}))
		kmsService := kms.New(sess)

		// Decrypt all the encrypted secrets.
		for _, ctx := range objects {
			if ctx.origEnc == nil {
				continue
			}
			dec, err := decryptSecret(ctx.origEnc, kmsService)
			if err != nil {
				return errors.Wrapf(err, "error decrypting %s/%s", ctx.origEnc.Namespace, ctx.origEnc.Name)
			}
			ctx.origDec = dec
		}

		// Make the list of objects to edit.
		objectsToEdit := make([]runtime.Object, 0, len(objects))
		for _, ctx := range objects {
			if ctx.origDec != nil {
				objectsToEdit = append(objectsToEdit, ctx.origDec)
			} else if ctx.other != nil {
				objectsToEdit = append(objectsToEdit, ctx.other)
			} else {
				panic("invalid context")
			}
		}

		// Edit!
		afterObjs, err := editObjects(objectsToEdit, "")
		if err != nil {
			return errors.Wrap(err, "error editing objects")
		}

		// Match up the new objects with the existing state.
		afterCtxs := make([]encryptedSecretContext, 0, len(afterObjs))
		for _, afterObj := range afterObjs {
			afterCtx := encryptedSecretContext{}

			dec, ok := afterObj.(*edit.DecryptedSecret)
			if ok {
				afterCtx.afterDec = dec
				for _, ctx := range objects {
					if ctx.origDec != nil && ctx.origDec.Name == dec.Name && ctx.origDec.Namespace == dec.Namespace {
						afterCtx.origDec = ctx.origDec
						afterCtx.origEnc = ctx.origEnc
						break
					}
				}
			} else {
				afterCtx.other = afterObj
			}
			afterCtxs = append(afterCtxs, afterCtx)
		}

		// Re-encrypt anything that needs it.
		// TODO real key logic
		keyId := os.Getenv("KEY")
		outObjs := make([]runtime.Object, 0, len(afterCtxs))
		for _, ctx := range afterCtxs {
			if ctx.afterDec != nil {
				enc, err := encryptSecret(ctx.afterDec, ctx.origDec, ctx.origEnc, keyId, kmsService)
				if err != nil {
					return errors.Wrapf(err, "error encrypting %s/%s", ctx.afterDec.Namespace, ctx.afterDec.Name)
				}
				outObjs = append(outObjs, enc)
			} else {
				outObjs = append(outObjs, ctx.other)
			}
		}

		// Write out the file again.
		// TODO make sure the file is writable before doing all this.
		outFile, err := os.Create(filename)
		if err != nil {
			return errors.Wrapf(err, "error opening %s for writing", filename)
		}
		defer outFile.Close()
		encodeYaml(outFile, outObjs)

		return nil
	},
}

func decodeYaml(in io.Reader) ([]runtime.Object, error) {
	objects := []runtime.Object{}
	// Code based on https://github.com/kubernetes/kubernetes/blob/0f93328c7a051e28a097270daaf7a7ff6f90bae0/staging/src/k8s.io/cli-runtime/pkg/genericclioptions/resource/visitor.go#L534-L561
	decoder := kyaml.NewYAMLOrJSONDecoder(in, 4096)
	for {
		ext := runtime.RawExtension{}
		err := decoder.Decode(&ext)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, errors.Wrap(err, "error parsing YAML")
		}
		ext.Raw = bytes.TrimSpace(ext.Raw)
		if len(ext.Raw) == 0 || bytes.Equal(ext.Raw, []byte("null")) {
			continue
		}
		// Ignored return value is a GVK.
		obj, _, err := scheme.Codecs.UniversalDeserializer().Decode(ext.Raw, nil, nil)
		if err != nil {
			return nil, errors.Wrap(err, "error decoding object")
		}
		objects = append(objects, obj)
	}
	return objects, nil
}

func encodeYaml(out io.Writer, objs []runtime.Object) error {
	first := true
	for _, obj := range objs {
		if !first {
			out.Write([]byte("---\n"))
		}
		first = false

		groupVersion := obj.GetObjectKind().GroupVersionKind().GroupVersion()
		info, ok := runtime.SerializerInfoForMediaType(scheme.Codecs.SupportedMediaTypes(), "application/yaml")
		if !ok {
			return errors.New("unable to find serializer info")
		}
		encoder := scheme.Codecs.EncoderForVersion(info.Serializer, groupVersion)
		err := encoder.Encode(obj, out)
		if err != nil {
			return errors.Wrap(err, "error encoding object")
		}
	}

	return nil
}

func decryptSecret(enc *secretsv1beta1.EncryptedSecret, kmsService kmsiface.KMSAPI) (*edit.DecryptedSecret, error) {
	dec := &edit.DecryptedSecret{ObjectMeta: enc.ObjectMeta}
	for key, value := range enc.Data {
		decodedValue := make([]byte, base64.StdEncoding.DecodedLen(len(value)))
		_, err := base64.StdEncoding.Decode(decodedValue, []byte(value))
		if err != nil {
			return nil, errors.Wrapf(err, "error base64 decoding value for %s", key)
		}
		decryptedValue, err := kmsService.Decrypt(&kms.DecryptInput{CiphertextBlob: decodedValue})
		if err != nil {
			return nil, errors.Wrapf(err, "error decrypting value for %s", key)
		}
		// Check if values in this secret were encrypted with more than one key.
		if dec.KeyId != "" && dec.KeyId != *decryptedValue.KeyId {
			return nil, errors.Errorf("key mismatch between %s and %s for %s", dec.KeyId, *decryptedValue.KeyId, key)
		}
		dec.KeyId = *decryptedValue.KeyId
		dec.Data[key] = string(decryptedValue.Plaintext)
	}
	return dec, nil
}

func encryptSecret(dec *edit.DecryptedSecret, origDec *edit.DecryptedSecret, origEnc *secretsv1beta1.EncryptedSecret, defaultKeyId string, kmsService kmsiface.KMSAPI) (*secretsv1beta1.EncryptedSecret, error) {
	// Work out which key to use.
	keyId := defaultKeyId
	if origDec != nil && origDec.KeyId != "" {
		keyId = origDec.KeyId
	}

	enc := &secretsv1beta1.EncryptedSecret{ObjectMeta: dec.ObjectMeta}
	for key, value := range dec.Data {
		// Check if this key has changed.
		if origDec != nil && origEnc != nil {
			origDecValue, ok := origDec.Data[key]
			if ok && value == origDecValue {
				// Key was not changed, reuse the old encrypted value.
				enc.Data[key] = origEnc.Data[key]
				continue
			}
		}
		// Encrypt the new value.
		encryptedValue, err := kmsService.Encrypt(&kms.EncryptInput{
			KeyId:     aws.String(keyId),
			Plaintext: []byte(value),
			EncryptionContext: map[string]*string{
				"RidecellOperator": aws.String("true"),
			},
		})
		if err != nil {
			return nil, errors.Wrapf(err, "error encrypting value for %s", key)
		}
		enc.Data[key] = base64.StdEncoding.EncodeToString(encryptedValue.CiphertextBlob)
	}
	return enc, nil
}

func runEditor(filename string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		return errors.New("No $EDITOR set")
	}

	cmd := exec.Command(editor, filename)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
	return nil
}

func editObjects(objects []runtime.Object, comment string) ([]runtime.Object, error) {
	objectBuf := bytes.Buffer{}
	err := encodeYaml(&objectBuf, objects)
	if err != nil {
		return nil, errors.Wrap(err, "error encoding objects to YAML")
	}
	for {
		// Make the YAML to show in the editor.
		outBuf := bytes.Buffer{}
		if comment != "" {
			for _, line := range strings.Split(comment, "\n") {
				outBuf.WriteString("# ")
				outBuf.WriteString(line)
				outBuf.WriteString("\n")
			}
			outBuf.WriteString("#\n")
		}
		outBuf.Write(objectBuf.Bytes())

		// Open a temporary file.
		tmpfile, err := ioutil.TempFile("", ".*.yml")
		if err != nil {
			return nil, errors.Wrap(err, "error making tempfile")
		}
		defer os.Remove(tmpfile.Name())
		tmpfile.Write(outBuf.Bytes())

		// Show the editor.
		err = runEditor(tmpfile.Name())
		if err != nil {
			return nil, errors.Wrap(err, "error running editor")
		}

		// Re-read the edited file.
		tmpfile.Seek(0, 0)
		objectBuf.Reset()
		_, err = objectBuf.ReadFrom(tmpfile)
		if err != nil {
			return nil, errors.Wrapf(err, "error reading tempfile %s", tmpfile.Name())
		}

		// Check if the file was edited at all.
		if bytes.Equal(objectBuf.Bytes(), outBuf.Bytes()) {
			return nil, errors.New("tempfile not edited, aborting")
		}

		outObjects, err := decodeYaml(&objectBuf)
		if err == nil {
			// Decode success, we're done!
			return outObjects, nil
		}

		// Some kind decoding error, probably bad syntax, show the editor again.
		comment = fmt.Sprintf("Error parsing file:\n%s", err)
	}
}

func createDefaultData(instance string) (io.Reader, error) {
	templateData, err := vfsutil.ReadFile(Templates, "new_instance.yml.tpl")
	if err != nil {
		return nil, errors.Wrap(err, "error reading new instance template")
	}
	template, err := template.New("new_instance.yml.tpl").Parse(string(templateData))
	if err != nil {
		return nil, errors.Wrap(err, "error parsing new instance template")
	}
	match := regexp.MustCompile(`^([a-z0-9]+)-([a-z]+)$`).FindStringSubmatch(instance)
	if match == nil {
		return nil, errors.Errorf("unable to parse instance name %s", instance)
	}
	buffer := &bytes.Buffer{}
	err = template.Execute(buffer, struct {
		Name      string
		Namespace string
	}{Name: match[1], Namespace: match[2]})
	if err != nil {
		return nil, errors.Wrap(err, "error rendering new instance template")
	}
	return buffer, nil
}
