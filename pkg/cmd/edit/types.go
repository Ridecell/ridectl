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

package edit

import (
	secretsv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/secrets/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	hacksecretsv1beta1 "github.com/Ridecell/ridectl/pkg/apis/secrets/v1beta1"
)

type Manifest []*Object

type Object struct {
	// The original text as parsed by NewYAMLOrJSONDecoder.
	Raw []byte
	// The original object as decoded by UniversalDeserializer.
	Object runtime.Object
	Meta   metav1.Object

	// Tracking for the various stages of encryption and decryption.
	OrigEnc  *secretsv1beta1.EncryptedSecret
	OrigDec  *hacksecretsv1beta1.DecryptedSecret
	AfterDec *hacksecretsv1beta1.DecryptedSecret
	AfterEnc *secretsv1beta1.EncryptedSecret
	Kind     string
	Data     map[string]string

	// The KMS KeyId used for this object, if known. If nil, it might be a new
	// object.
	KeyId string

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
