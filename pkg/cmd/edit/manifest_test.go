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
	"strings"

	"github.com/Ridecell/ridectl/pkg/cmd/edit"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Manifest", func() {

	decrypted := `apiVersion: secrets.ridecell.io/v1beta1
kind: DecryptedSecret
metadata:
  name: default
  namespace: default
data:
  KEY: val
---
apiVersion: secrets.ridecell.io/v1beta1
kind: DecryptedSecret
metadata:
  # This name is important.
  name: mixedstuff
  namespace: commentsland
data:
  # Critical secret.
  MYKEY: myvalue
  # HTTP key!
  tls.key: |
    -----BEGIN PRIVATE KEY-----
    MIIEvAIBADANasdfasdfasdfasdfasdfasdf
    qwerqwerqwerqwerqwerqwerqwerqwerqwer
    -----END PRIVATE KEY-----
  # Determined by fair dice roll.
  RANDOM_VALUE: "4"
`

	encrypted := `apiVersion: secrets.ridecell.io/v1beta1
kind: EncryptedSecret
metadata:
  name: default
  namespace: default
data:
  KEY: a21zdmFs
---
apiVersion: secrets.ridecell.io/v1beta1
kind: EncryptedSecret
metadata:
  # This name is important.
  name: mixedstuff
  namespace: commentsland
data:
  # Critical secret.
  MYKEY: a21zbXl2YWx1ZQ==
  # HTTP key!
  tls.key: a21zLS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCk1JSUV2QUlCQURBTmFzZGZhc2RmYXNkZmFzZGZhc2RmYXNkZgpxd2VycXdlcnF3ZXJxd2VycXdlcnF3ZXJxd2VycXdlcnF3ZXIKLS0tLS1FTkQgUFJJVkFURSBLRVktLS0tLQo=
  # Determined by fair dice roll.
  RANDOM_VALUE: a21zNA==
`

	Context("with decrypted data", func() {
		It("loads the data", func() {
			m, err := edit.NewManifest(strings.NewReader(decrypted))
			Expect(err).ToNot(HaveOccurred())
			Expect(m).To(HaveLen(2))
			Expect(m[0].Meta.GetName()).To(Equal("default"))
			Expect(m[1].Meta.GetName()).To(Equal("mixedstuff"))
		})

		It("re-serializes correctly", func() {
			m, err := edit.NewManifest(strings.NewReader(decrypted))
			Expect(err).ToNot(HaveOccurred())
			var buf strings.Builder
			err = m.Serialize(&buf)
			Expect(err).ToNot(HaveOccurred())
			Expect(buf.String()).To(Equal(decrypted))
		})

		It("encrypts and serializes correctly", func() {
			m, err := edit.NewManifest(strings.NewReader(decrypted))
			Expect(err).ToNot(HaveOccurred())
			err = m.Encrypt(kmsMock(), "12345", false)
			Expect(err).ToNot(HaveOccurred())
			var buf strings.Builder
			err = m.Serialize(&buf)
			Expect(err).ToNot(HaveOccurred())
			Expect(buf.String()).To(Equal(encrypted))
		})
	})

	Context("with encrypted data", func() {
		It("loads the data", func() {
			m, err := edit.NewManifest(strings.NewReader(encrypted))
			Expect(err).ToNot(HaveOccurred())
			Expect(m).To(HaveLen(2))
			Expect(m[0].Meta.GetName()).To(Equal("default"))
			Expect(m[1].Meta.GetName()).To(Equal("mixedstuff"))
		})

		It("re-serializes correctly", func() {
			m, err := edit.NewManifest(strings.NewReader(encrypted))
			Expect(err).ToNot(HaveOccurred())
			var buf strings.Builder
			err = m.Serialize(&buf)
			Expect(err).ToNot(HaveOccurred())
			Expect(buf.String()).To(Equal(encrypted))
		})

		It("decrypts and serializes correctly", func() {
			m, err := edit.NewManifest(strings.NewReader(encrypted))
			Expect(err).ToNot(HaveOccurred())
			err = m.Decrypt(kmsMock())
			Expect(err).ToNot(HaveOccurred())
			var buf strings.Builder
			err = m.Serialize(&buf)
			Expect(err).ToNot(HaveOccurred())
			Expect(buf.String()).To(Equal(decrypted))
		})
	})
})
