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
	"k8s.io/apimachinery/pkg/runtime"

	secretsv1beta2 "github.com/Ridecell/ridecell-controllers/apis/secrets/v1beta2"
	hacksecretsv1beta2 "github.com/Ridecell/ridectl/pkg/apis/secrets/v1beta2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Manifest []*Object

type Object struct {
	// The original text as parsed by NewYAMLOrJSONDecoder.
	Raw []byte
	// The original object as decoded by UniversalDeserializer.
	Object runtime.Object
	Meta   metav1.Object

	// Tracking for the various stages of encryption and decryption.
	OrigEnc  *secretsv1beta2.EncryptedSecret
	OrigDec  *hacksecretsv1beta2.DecryptedSecret
	AfterDec *hacksecretsv1beta2.DecryptedSecret
	AfterEnc *secretsv1beta2.EncryptedSecret
	Kind     string
	Data     map[string]string

	// The KMS KeyId used for this object, if known. If nil, it might be a new
	// object.
	KeyId string
	// The Plaintext Data key and Cipher Key generated using KMS Key ID
	PlainDataKey  *[32]byte
	CipherDataKey []byte

	// Byte coordinates for areas of the raw text we need to edit when re-serializing.
	KindLoc TextLocation
	DataLoc TextLocation
	KeyLocs []KeysLocation
}

type TextLocation struct {
	Start int
	End   int
}

type KeysLocation struct {
	TextLocation
	Key string
}
