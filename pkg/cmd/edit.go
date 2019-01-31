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
	"sort"

	"github.com/Ridecell/ridectl/pkg/cmd/edit"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/kms/kmsiface"
	"github.com/pkg/errors"
	"github.com/shurcooL/httpfs/vfsutil"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	afterEnc *secretsv1beta1.EncryptedSecret
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
				templateData, err := vfsutil.ReadFile(Templates, "new_instance.yml.tpl")
				if err != nil {
					return errors.Wrap(err, "error reading new instance template")
				}
				template, err := template.New("new_instance.yml.tpl").Parse(string(templateData))
				if err != nil {
					return errors.Wrap(err, "error parsing new instance template")
				}
				match := regexp.MustCompile(`^([a-z0-9]+)-([a-z]+)$`).FindStringSubmatch(args[0])
				if match == nil {
					return errors.Errorf("unable to parse instance name %s", args[0])
				}
				buffer := &bytes.Buffer{}
				err = template.Execute(buffer, struct {
					Name      string
					Namespace string
				}{Name: match[1], Namespace: match[2]})
				if err != nil {
					return errors.Wrap(err, "error rendering new instance template")
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
		secrets := []encryptedSecretContext{}
		inEverythingElse := []runtime.Object{}
		for _, obj := range inObjs {
			enc, ok := obj.(*secretsv1beta1.EncryptedSecret)
			if ok {
				secrets = append(secrets, encryptedSecretContext{origEnc: enc})
			} else {
				inEverythingElse = append(inEverythingElse, obj)
			}
		}

		// Create a KMS session
		// TODO error handling for AWS creds
		sess := session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
		}))
		kmsService := kms.New(sess)

		// Decrypt all the encrypted secrets.
		objectsToEdit := []runtime.Object{}
		for _, sec := range secrets {
			dec, err := decryptSecret(sec.origEnc, kmsService)
			if err != nil {
				return errors.Wrapf(err, "error decrypting %s/%s", sec.origEnc.Namespace, sec.origEnc.Name)
			}
			sec.origDec = dec
			objectsToEdit = append(objectsToEdit, dec)
		}
		objectsToEdit = append(objectsToEdit, inEverythingElse...)

		// Make the YAML to show in the editor.
		tmpfile, err := ioutil.TempFile("", ".*.yml")
		if err != nil {
			return errors.Wrap(err, "error making tempfile")
		}
		defer os.Remove(tmpfile.Name())
		encodeYaml(tmpfile, objectsToEdit)

		// Show the editor.
		err = runEditor(tmpfile.Name())
		if err != nil {
			return errors.Wrap(err, "error running editor")
		}

		// Re-read the edited file.
		tmpfile.Seek(0, 0)
		afterObjs, err := decodeYaml(tmpfile)
		if err != nil {
			// TODO this should re-open the editor and show the error.
			return errors.Wrap(err, "error decoding edited YAML")
		}

		// Match up the new objects with the existing state.
		afterEverythingElse := []runtime.Object{}
		for _, afterObj := range afterObjs {
			dec, ok := afterObj.(*edit.DecryptedSecret)
			if ok {
				found := false
				for _, sec := range secrets {
					if sec.origDec.Name == dec.Name && sec.origDec.Namespace == dec.Namespace {
						sec.afterDec = dec
						found = true
						break
					}
				}
				// No match, must be new.
				if !found {
					secrets = append(secrets, encryptedSecretContext{afterDec: dec})
				}
			} else {
				afterEverythingElse = append(afterEverythingElse, afterObj)
			}
		}

		// Re-encrypt anything that needs it.
		// TODO real key logic
		keyId := os.Getenv("KEY")
		for _, sec := range secrets {
			enc, err := encryptSecret(sec.afterDec, sec.origDec, sec.origEnc, keyId, kmsService)
			if err != nil {
				return errors.Wrapf(err, "error encrypting %s/%s", sec.afterDec.Namespace, sec.afterDec.Name)
			}
			sec.afterEnc = enc
		}

		// Try to rebuild things in the same order as it was originally.
		orderedObjects := []runtime.Object{}
		for _, sec := range secrets {
			orderedObjects = append(orderedObjects, sec.afterEnc)
		}
		orderedObjects = append(orderedObjects, afterEverythingElse...)
		sortObjects(orderedObjects, inObjs, afterObjs)

		// Write out the file again.
		// TODO make sure the file is writable before doing all this.
		outFile, err := os.Create(filename)
		if err != nil {
			return errors.Wrapf(err, "error opening %s for writing", filename)
		}
		defer outFile.Close()
		encodeYaml(outFile, orderedObjects)

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
	encoder := yaml.NewEncoder(out)
	for _, obj := range objs {
		err := encoder.Encode(obj)
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
	keyId := origDec.KeyId
	if keyId == "" {
		keyId = defaultKeyId
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

type orderedObjects struct {
	orig  map[string]int
	after map[string]int
	out   []runtime.Object
}

func (o orderedObjects) Len() int      { return len(o.out) }
func (o orderedObjects) Swap(i, j int) { o.out[i], o.out[j] = o.out[j], o.out[i] }
func (o orderedObjects) Less(i, j int) bool {
	iKey := o.objKey(o.out[i])
	jKey := o.objKey(o.out[j])
	iOrig, iOk := o.orig[iKey]
	jOrig, jOk := o.orig[jKey]
	if iOk && jOk {
		// Both in orig.
		return iOrig < jOrig
	} else if iOk {
		// i in orig, j not (so assumed at infinity).
		return true
	} else if jOk {
		// j in orig, i not (so assumed at infinity).
		return false
	} else {
		// Neither in orig, check after.
		iAfter, iOk := o.after[iKey]
		jAfter, jOk := o.after[jKey]
		if iOk && jOk {
			// Both in orig.
			return iAfter < jAfter
		} else if iOk {
			// i in after, j not (so assumed at infinity).
			return true
		} else if jOk {
			// j in after, i not (so assumed at infinity).
			return false
		}
	}
	panic("unable to compare")
}
func (_ orderedObjects) objKey(obj runtime.Object) string {
	gvk := obj.GetObjectKind().GroupVersionKind()
	meta := obj.(metav1.Object)
	return fmt.Sprintf("%s/%s/%s/%s/%s", gvk.Group, gvk.Version, gvk.Kind, meta.GetNamespace(), meta.GetName())
}

func sortObjects(all []runtime.Object, orig []runtime.Object, after []runtime.Object) error {
	o := orderedObjects{out: all}
	for n, obj := range orig {
		o.orig[o.objKey(obj)] = n
	}
	for n, obj := range after {
		o.after[o.objKey(obj)] = n
	}
	sort.Sort(o)
	return nil
}
