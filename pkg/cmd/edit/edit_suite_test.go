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

package edit_test

import (
	"bytes"
	"testing"

	"github.com/Ridecell/ridecell-operator/pkg/apis"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes/scheme"

	hackapis "github.com/Ridecell/ridectl/pkg/apis"
)

// Sed is a workaround for https://github.com/matryer/moq/issues/86.
//go:generate bash -c "moq -pkg edit_test ../../../vendor/github.com/aws/aws-sdk-go/service/kms/kmsiface KMSAPI | sed 's|github.com/Ridecell/ridectl/vendor/||' > zz_generated.mock_kmsiface_test.go"
func kmsMock() *KMSAPIMock {
	return &KMSAPIMock{
		DecryptFunc: func(in *kms.DecryptInput) (*kms.DecryptOutput, error) {
			if !bytes.HasPrefix(in.CiphertextBlob, []byte("kms")) {
				return nil, errors.Errorf("Value %s is not mack encrypted", in.CiphertextBlob)
			}
			return &kms.DecryptOutput{
				Plaintext: in.CiphertextBlob[3:],
				KeyId:     aws.String("12345"),
			}, nil
		},
		EncryptFunc: func(in *kms.EncryptInput) (*kms.EncryptOutput, error) {
			return &kms.EncryptOutput{
				CiphertextBlob: append([]byte("kms"), in.Plaintext...),
			}, nil
		},
	}
}

func TestEdit(t *testing.T) {
	// Register all types from ridecell-operator.
	apis.AddToScheme(scheme.Scheme)
	hackapis.AddToScheme(scheme.Scheme)

	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Edit Suite")
}
