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
	"crypto/rand"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/kms/kmsiface"
	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"golang.org/x/crypto/nacl/secretbox"
	"k8s.io/client-go/kubernetes/scheme"

	secretsv1beta2 "github.com/Ridecell/ridecell-controllers/apis/secrets/v1beta2"
	hacksecretsv1beta2 "github.com/Ridecell/ridectl/pkg/apis/secrets/v1beta2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	nonceLength = 24
)

var dataRegexp *regexp.Regexp
var keyRegexp *regexp.Regexp
var nonStringRegexp *regexp.Regexp

type Payload struct {
	Key     []byte
	Nonce   *[nonceLength]byte
	Message []byte
}

func init() {
	dataRegexp = regexp.MustCompile(`(?ms)kind: (EncryptedSecret|DecryptedSecret).*?(^data:.*?)\z`)
	keyRegexp = regexp.MustCompile("" +
		// Turn on multiline mode for the whole pattern, ^ and $ will match on lines rather than start and end of whole string.
		`(?m)` +
		// Look for the key, some whitespace, then some non-space-or-:, then :
		`^[ \t]+([^:\n\r]+):` +
		// Whitespace between the key's : and the value
		`[ \t]+` +
		// Start an alternation for block scalars and normal values.
		`(?:` +
		// Match block scalars first because they would otherwise match the normal value pattern.
		// Looks for the | or >, optional flags, then lines with 4 spaces of indentation. A better version of this
		// would look more like ([|>]\n([ \t]+).+?\n(?:\3.+?\n)*) and would use a backreference instead of hardwiring
		// things but Go, or rather RE2, refuses to support backrefs because they can be slow. Blaaaaaaah.
		`([|>].*?(?:\n    .+?$)+)` +
		// Alternation between block scalar and normal values.
		`|` +
		// Look for a normal value, something on a single line with optional trailing whitespace.
		`(.+?)[ \t]*$` +
		// Close the block vs. normal alternation.
		`)`,
	)
	nonStringRegexp = regexp.MustCompile(`^(\d+(\.\d+)?|true|false|null|\[.*\]|)$`)
}

func NewObject(raw []byte) (*Object, error) {

	// Here, we need to be able to edit the objects which are not registered in ridectl
	// So we need to use the unstructured object to serialize it.
	// decode YAML into unstructured.Unstructured
	// get object's GroupVersionKind (GVK) from the raw YAML. We don't need to worry about
	// runtime.Object because we are not going to use it for anything here
	// Ignored return value is runtime.Object
	o := &Object{Raw: raw}
	obj, _, err := scheme.Codecs.UniversalDeserializer().Decode(raw, nil, nil)
	if err != nil {
		
		return o, nil
	}

	o.Object = obj
	o.Meta = obj.(metav1.Object)
	obj, _, err = scheme.Codecs.UniversalDeserializer().Decode(raw, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing YAML")
	}

	o.Object = obj
	o.Meta = obj.(metav1.Object)

	// Check if this an EncryptedSecret.
	enc, ok := obj.(*secretsv1beta2.EncryptedSecret)
	if ok {
		o.OrigEnc = enc
		o.Kind = "EncryptedSecret"
		o.Data = enc.Data
	}
	// or a DecryptedSecret.
	dec, ok := obj.(*hacksecretsv1beta2.DecryptedSecret)
	if ok {
		o.AfterDec = dec
		o.Kind = "DecryptedSecret"
		o.Data = dec.Data
	}

	if o.Kind != "" {
		// Run the regex parse. If you are reading this code, I am sorry and yes I
		// feel bad about it. This is used when re-encoding to allow output that
		// preserves comments, whitespace, key ordering, etc.
		match := dataRegexp.FindSubmatchIndex(raw)
		if match == nil {
			// This shouldn't happen.
			panic("EncryptedSecret or DecryptedSecret didn't match dataRegexp")
		}
		// match[0] and [1] are for the whole regexp, we don't need that.
		o.KindLoc.Start = match[2]
		o.KindLoc.End = match[3]
		o.DataLoc.Start = match[4]
		o.DataLoc.End = match[5]
		if len(o.Data) > 0 {
			locs, err := newKeysLocations(raw[o.DataLoc.Start:o.DataLoc.End], o.DataLoc.Start)
			if err != nil {
				// Also shouldn't happen.
				panic(err.Error())
			}
			o.KeyLocs = locs
		}

		// A safety check for now.
		if len(o.Data) != len(o.KeyLocs) {
			panic("key count mismatch")
		}
	}
	return o, nil
}

func newKeysLocations(raw []byte, offset int) ([]KeysLocation, error) {
	matches := keyRegexp.FindAllSubmatchIndex(raw, -1)
	if matches == nil {
		return nil, errors.New("unable to parse keys")
	}
	locs := []KeysLocation{}
	for _, match := range matches {
		keyStart := match[2]
		keyEnd := match[3]
		blockValueStart := match[4]
		blockValueEnd := match[5]
		normalValueStart := match[6]
		normalValueEnd := match[7]
		var valueLoc TextLocation
		if normalValueStart == -1 {
			if raw[blockValueStart] != '|' {
				return nil, errors.New("only | block scalars are supported")
			}
			valueLoc.Start = blockValueStart + offset
			valueLoc.End = blockValueEnd + offset
		} else {
			valueLoc.Start = normalValueStart + offset
			valueLoc.End = normalValueEnd + offset
		}
		key := string(raw[keyStart:keyEnd])
		if key[0] == '#' {
			// Go doesn't do negative lookaheads to easier to filter comments out here.
			continue
		}
		locs = append(locs, KeysLocation{TextLocation: valueLoc, Key: key})
	}
	return locs, nil
}

func (o *Object) Decrypt(kmsService kmsiface.KMSAPI) error {
	if o.Kind == "" {
		return nil
	}

	// Key map for holding plainDataKey to avoid repetative KMS decrypt calls for single cipherDataKey
	keyMap := map[string]*[32]byte{}

	dec := &hacksecretsv1beta2.DecryptedSecret{ObjectMeta: o.OrigEnc.ObjectMeta, Data: map[string]string{}}

	for key, value := range o.OrigEnc.Data {
		useDataKey := false

		if strings.HasPrefix(value, "crypto") {
			useDataKey = true
			array := strings.Split(value, " ")
			value = array[len(array)-1]
		}

		decodedValue := make([]byte, base64.StdEncoding.DecodedLen(len(value)))
		l, err := base64.StdEncoding.Decode(decodedValue, []byte(value))
		if err != nil {
			return errors.Wrapf(err, "error base64 decoding value for %s", key)
		}

		// If True, decrypt using data key
		if useDataKey {
			var p Payload
			_ = gob.NewDecoder(bytes.NewReader(decodedValue)).Decode(&p)

			plainDataKey, ok := keyMap[string(p.Key)]
			if !ok {
				// Decrypt cipherdatakey
				plainDataKey, err = DecryptCipherDataKey(kmsService, p.Key)
				if err != nil {
					return errors.Wrapf(err, "error decrypting value for cipherDatakey")
				}
				keyMap[string(p.Key)] = plainDataKey
			}

			// Decrypt message
			var plaintext []byte
			plaintext, ok = secretbox.Open(plaintext, p.Message, p.Nonce, plainDataKey)
			if !ok {
				return errors.Wrapf(err, "error decrypting value with data key for %s", key)
			}
			dec.Data[key] = string(plaintext)
			continue
		}

		// Decrypt using KMS service
		decryptedValue, err := kmsService.Decrypt(&kms.DecryptInput{
			CiphertextBlob: decodedValue[:l],
			EncryptionContext: map[string]*string{
				"RidecellOperator": aws.String("true"),
			},
		})
		if err != nil {
			return errors.Wrapf(err, "error decrypting value for %s", key)
		}
		// Check if values in this secret were encrypted with more than one key.
		if o.KeyId != "" && o.KeyId != *decryptedValue.KeyId {
			return errors.Errorf("key mismatch between %s and %s for %s", o.KeyId, *decryptedValue.KeyId, key)
		}
		o.KeyId = *decryptedValue.KeyId

		decryptedString := string(decryptedValue.Plaintext)
		if decryptedString == secretsv1beta2.EncryptedSecretEmptyKey {
			decryptedString = ""
		}
		dec.Data[key] = decryptedString

	}
	o.OrigDec = dec
	o.Kind = "DecryptedSecret"
	o.Data = dec.Data
	return nil
}

func (o *Object) Encrypt(kmsService kmsiface.KMSAPI, defaultKeyId string, forceKeyId bool, reEncrypt bool) error {
	if o.Kind == "" {
		return nil
	}

	// Work out which key to use.
	keyId := defaultKeyId
	if o.KeyId != "" && !forceKeyId {
		keyId = o.KeyId
	}
	if keyId == "" {
		return errors.New("Key ID cannot be blank")
	}

	enc := &secretsv1beta2.EncryptedSecret{ObjectMeta: o.AfterDec.ObjectMeta, Data: map[string]string{}}

	plainDataKeyPresent := false
	plainDataKey := &[32]byte{}
	var cipherDataKey []byte

	// check if any value is already encrypted using data key before
	// If true, then use same data key to encrypt new values
	if o.OrigEnc != nil {
		for key, value := range o.OrigEnc.Data {
			if !strings.HasPrefix(value, "crypto") {
				continue
			}
			array := strings.Split(value, " ")
			value = array[len(array)-1]

			decodedValue := make([]byte, base64.StdEncoding.DecodedLen(len(value)))
			_, err := base64.StdEncoding.Decode(decodedValue, []byte(value))
			if err != nil {
				return errors.Wrapf(err, "error base64 decoding value for %s", key)
			}
			var p Payload
			_ = gob.NewDecoder(bytes.NewReader(decodedValue)).Decode(&p)
			plainDataKey, err = DecryptCipherDataKey(kmsService, p.Key)
			if err != nil {
				return errors.Wrapf(err, "error decrypting value for cipherDatakey")
			}
			cipherDataKey = p.Key
			break
		}
	}

	for key, value := range o.AfterDec.Data {

		// Check if this key has changed.
		if o.OrigDec != nil && o.OrigEnc != nil && !reEncrypt {
			origDecValue, ok := o.OrigDec.Data[key]
			if ok && value == origDecValue {
				// Key was not changed, reuse the old encrypted value.
				enc.Data[key] = o.OrigEnc.Data[key]

				continue
			}
		}

		// check if plainDataKey is populated, if not create data key
		if !plainDataKeyPresent {
			var err error
			plainDataKey, cipherDataKey, err = GenerateDataKey(kmsService, keyId)
			if err != nil {
				return errors.Wrapf(err, "error generating data key")
			}
			plainDataKeyPresent = true
		}

		// Initialize Payload
		p := &Payload{
			Key:   cipherDataKey,
			Nonce: &[nonceLength]byte{},
		}

		// Set nonce
		if _, err := rand.Read(p.Nonce[:]); err != nil {
			return errors.Wrapf(err, "error generating nonce for %s", key)
		}

		// Encrypt message
		p.Message = secretbox.Seal(p.Message, []byte(value), p.Nonce, plainDataKey)
		buf := &bytes.Buffer{}
		if err := gob.NewEncoder(buf).Encode(p); err != nil {
			return errors.Wrapf(err, "error encrypting value using data key for %s", key)
		}

		enc.Data[key] = fmt.Sprintf("crypto %s", string(base64.StdEncoding.EncodeToString(buf.Bytes())))
	}

	pterm.Info.Printf("Encrypted using %s\n", getAliasByKey(kmsService, keyId))
	o.AfterEnc = enc
	o.Kind = "EncryptedSecret"
	o.Data = enc.Data
	return nil
}

func (o *Object) Serialize(out io.Writer) error {
	// Check if this is one of the two types we care about.
	if o.Data == nil {
		// Nope, we're out.
		_, err := out.Write(o.Raw)
		return err
	}

	// Start writing!
	_, err := out.Write(o.Raw[0:o.KindLoc.Start])
	if err != nil {
		return err
	}
	_, err = out.Write([]byte(o.Kind))
	if err != nil {
		return err
	}
	// Track where we are up to.
	carry := o.KindLoc.End
	for _, keyLoc := range o.KeyLocs {
		newValue, ok := o.Data[keyLoc.Key]
		if !ok {
			panic("key from location not found in data")
		}

		// Check for a multiline value.
		if strings.ContainsRune(newValue, '\n') {
			var buf strings.Builder
			buf.WriteString("|")
			lines := strings.Split(newValue, "\n")
			// Trim the trailing newline, basically.
			if lines[len(lines)-1] == "" {
				lines = lines[:len(lines)-1]
			}
			for _, line := range lines {
				// Hardwire to 4 spaces of indentation, which assumes 2 spaces on the keys.
				// I'll probably regret this when someone uses 8 space indents.
				buf.WriteString("\n    ")
				buf.WriteString(line)
			}
			newValue = buf.String()
		}

		// Check for things that YAML thinks aren't strings that might show up in the value.
		if nonStringRegexp.MatchString(newValue) {
			newValue = fmt.Sprintf(`"%s"`, newValue)
		}
		_, err = out.Write(o.Raw[carry:keyLoc.Start])
		if err != nil {
			return err
		}
		_, err = out.Write([]byte(newValue))
		if err != nil {
			return err
		}
		carry = keyLoc.End
	}
	_, err = out.Write(o.Raw[carry:])
	if err != nil {
		return err
	}

	return nil
}

func GenerateDataKey(kmsService kmsiface.KMSAPI, keyId string) (*[32]byte, []byte, error) {
	// Generate data key
	rsp, err := kmsService.GenerateDataKey(&kms.GenerateDataKeyInput{
		KeyId:         aws.String(keyId),
		NumberOfBytes: aws.Int64(32),
		EncryptionContext: map[string]*string{
			"RidecellOperator": aws.String("true"),
		},
	})
	if err != nil {
		return nil, nil, err
	}

	key := &[32]byte{}
	copy(key[:], rsp.Plaintext)

	return key, rsp.CiphertextBlob, nil
}

func DecryptCipherDataKey(kmsService kmsiface.KMSAPI, cipherDataKey []byte) (*[32]byte, error) {
	decryptRsp, err := kmsService.Decrypt(&kms.DecryptInput{
		CiphertextBlob: cipherDataKey,
		EncryptionContext: map[string]*string{
			"RidecellOperator": aws.String("true"),
		},
	})

	if err != nil {
		return nil, err
	}
	plainDataKey := &[32]byte{}
	copy(plainDataKey[:], decryptRsp.Plaintext)

	pterm.Info.Printf("Decrypted using %s\n", getAliasByKey(kmsService, *decryptRsp.KeyId))
	return plainDataKey, nil
}

func getAliasByKey(kmsService kmsiface.KMSAPI, keyId string) string {

	// check if the key is an alias
	if strings.HasPrefix(keyId, "alias") {
		return keyId
	}
	// get aliasname from key id
	var aliases []string
	aliasRsp, err := kmsService.ListAliases(&kms.ListAliasesInput{
		KeyId: aws.String(keyId),
	})
	if err != nil {
		fmt.Println(err.Error())
		pterm.Error.Println("Error getting alias for key")
		return keyId
	}

	aliasList := aliasRsp.Aliases
	if len(aliasList) == 0 {
		pterm.Warning.Println("Error getting alias for key")
		return keyId
	}

	for alias := range aliasList {
		aliases = append(aliases, *aliasList[alias].AliasName)
	}

	return strings.Join(aliases, ",")

}
