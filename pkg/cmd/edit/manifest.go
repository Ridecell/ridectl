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

package edit

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"regexp"

	"github.com/aws/aws-sdk-go/service/kms/kmsiface"
	"github.com/pkg/errors"
)

var emptyRegexp *regexp.Regexp
var splitRegexp *regexp.Regexp

func init() {
	emptyRegexp = regexp.MustCompile(`(?m)\A(^(\s*#.*|\s*)$\s*)*\z`)
	splitRegexp = regexp.MustCompile(`(?m)^---$(\n)?`)
}

func NewManifest(in io.Reader) (Manifest, error) {

	// Read in the whole file.
	buf := bytes.Buffer{}
	_, err := buf.ReadFrom(in)
	if err != nil {
		return nil, errors.Wrap(err, "error reading manifest")
	}

	objects := []*Object{}
	for _, chunk := range splitRegexp.Split(buf.String(), -1) {
		if emptyRegexp.MatchString(chunk) {
			continue
		}
		obj, err := NewObject([]byte(chunk))
		// return the error if we have one which is not ignorable
		if err != nil {
			// Dont append the object if its of old SummonPlatform or EncryptedSecret version
			if strings.Contains(err.Error(), "no kind \"SummonPlatform\" is registered for version \"summon.ridecell.io/v1beta1\"") || strings.Contains(err.Error(), "no kind \"EncryptedSecret\" is registered for version \"secrets.ridecell.io/v1beta1\"") {
				continue
			}
			// Ignore unregistered object errors, and only return other errors
			if ok, _ := regexp.MatchString("no kind(.*)is registered for version", err.Error()); !ok {
				return nil, errors.Wrap(err, "error decoding object")
			}
		}
		objects = append(objects, obj)
	}
	return objects, nil
}

func (m Manifest) Decrypt(kmsService kmsiface.KMSAPI) error {
	for _, obj := range m {
		err := obj.Decrypt(kmsService)
		if err != nil {
			return errors.Wrapf(err, "error decrypting %s/%s", obj.Meta.GetNamespace(), obj.Meta.GetName())
		}
	}
	return nil
}

func (m Manifest) Encrypt(kmsService kmsiface.KMSAPI, defaultKeyId string, forceKeyId bool, reEncrypt bool) error {
	for _, obj := range m {
		err := obj.Encrypt(kmsService, defaultKeyId, forceKeyId, reEncrypt)
		if err != nil {
			return errors.Wrapf(err, "error encrypting %s/%s", obj.Meta.GetNamespace(), obj.Meta.GetName())
		}
	}
	return nil
}

func (m Manifest) Serialize(out io.Writer) error {
	first := true
	for _, obj := range m {
		if !first {
			_, _ = out.Write([]byte("---\n"))
		}
		first = false
		err := obj.Serialize(out)
		if err != nil {
			return errors.Wrapf(err, "error serializing %s/%s", obj.Meta.GetNamespace(), obj.Meta.GetName())
		}
	}
	return nil
}

func (m Manifest) CorrelateWith(origManifest Manifest) error {
	// Build a map of the input secrets.
	origByName := map[string]*Object{}
	for _, obj := range origManifest {
		if obj.Kind == "" {
			continue
		}
		origByName[fmt.Sprintf("%s/%s", obj.Meta.GetNamespace(), obj.Meta.GetName())] = obj
	}

	// Find the original objects.
	for _, obj := range m {
		if obj.Kind == "" {
			continue
		}
		origObj, ok := origByName[fmt.Sprintf("%s/%s", obj.Meta.GetNamespace(), obj.Meta.GetName())]
		if ok {
			obj.OrigEnc = origObj.OrigEnc
			obj.OrigDec = origObj.OrigDec
			obj.KeyId = origObj.KeyId
		}
	}
	return nil
}
