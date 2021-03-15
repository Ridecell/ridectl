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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Ridecell/ridectl/pkg/cmd/edit"
)

var _ = Describe("Object", func() {
	simpleEncryptedSecret := `apiVersion: secrets.ridecell.io/v1beta1
kind: EncryptedSecret
metadata:
  name: simple
  namespace: default
data:
  MYKEY: a21zbXl2YWx1ZQ==
`

	simpleDecryptedSecret := `apiVersion: secrets.ridecell.io/v1beta1
kind: DecryptedSecret
metadata:
  name: simple
  namespace: default
data:
  MYKEY: myvalue
`

	withComments := `apiVersion: secrets.ridecell.io/v1beta1
kind: DecryptedSecret
metadata:
  # This name is important.
  name: comments
  namespace: commentsland
data:
  # Critical secret.
  MYKEY: myvalue
  # Determined by fair dice roll.
  RANDOM_VALUE: "4"
`

	withBlockScalar := `apiVersion: secrets.ridecell.io/v1beta1
kind: DecryptedSecret
metadata:
  name: block
  namespace: default
data:
  MYKEY: |
    -----BEGIN PRIVATE KEY-----
    MIIEvAIBADANasdfasdfasdfasdfasdfasdf
    qwerqwerqwerqwerqwerqwerqwerqwerqwer
    -----END PRIVATE KEY-----
`

	complexMixedContext := `apiVersion: secrets.ridecell.io/v1beta1
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
  # Might look like an array, but it's not.
  FAKE_ARRAY: "[1234]"
  # No value!
  Empty: ""
`

	complexEncryptedContent := `apiVersion: secrets.ridecell.io/v1beta1
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
  # Might look like an array, but it's not.
  FAKE_ARRAY: a21zWzEyMzRd
  # No value!
  Empty: a21zX19fZW1wdHlfc3RyaW5nX19f
`

	Context("with a simple encrypted secret", func() {
		It("loads the data", func() {
			obj, err := edit.NewObject([]byte(simpleEncryptedSecret))
			Expect(err).ToNot(HaveOccurred())
			Expect(obj.Kind).To(Equal("EncryptedSecret"))
			Expect(obj.Data).To(HaveKeyWithValue("MYKEY", "a21zbXl2YWx1ZQ=="))
		})

		It("serializes the data", func() {
			obj, err := edit.NewObject([]byte(simpleEncryptedSecret))
			Expect(err).ToNot(HaveOccurred())
			var buf strings.Builder
			err = obj.Serialize(&buf)
			Expect(err).ToNot(HaveOccurred())
			Expect(buf.String()).To(Equal(simpleEncryptedSecret))
		})
	})

	Context("with a simple decrypted secret", func() {
		It("loads the data", func() {
			obj, err := edit.NewObject([]byte(simpleDecryptedSecret))
			Expect(err).ToNot(HaveOccurred())
			Expect(obj.Kind).To(Equal("DecryptedSecret"))
			Expect(obj.Data).To(HaveKeyWithValue("MYKEY", "myvalue"))
		})

		It("serializes the data", func() {
			obj, err := edit.NewObject([]byte(simpleDecryptedSecret))
			Expect(err).ToNot(HaveOccurred())
			var buf strings.Builder
			err = obj.Serialize(&buf)
			Expect(err).ToNot(HaveOccurred())
			Expect(buf.String()).To(Equal(simpleDecryptedSecret))
		})
	})

	Context("with comments", func() {
		It("loads the data", func() {
			obj, err := edit.NewObject([]byte(withComments))
			Expect(err).ToNot(HaveOccurred())
			Expect(obj.Kind).To(Equal("DecryptedSecret"))
			Expect(obj.Data).To(HaveKeyWithValue("MYKEY", "myvalue"))
			Expect(obj.Data).To(HaveKeyWithValue("RANDOM_VALUE", "4"))
		})

		It("serializes the data", func() {
			obj, err := edit.NewObject([]byte(withComments))
			Expect(err).ToNot(HaveOccurred())
			var buf strings.Builder
			err = obj.Serialize(&buf)
			Expect(err).ToNot(HaveOccurred())
			Expect(buf.String()).To(Equal(withComments))
		})
	})

	Context("with a block scalar", func() {
		It("loads the data", func() {
			obj, err := edit.NewObject([]byte(withBlockScalar))
			Expect(err).ToNot(HaveOccurred())
			Expect(obj.Kind).To(Equal("DecryptedSecret"))
			Expect(obj.Data).To(HaveKeyWithValue("MYKEY", "-----BEGIN PRIVATE KEY-----\nMIIEvAIBADANasdfasdfasdfasdfasdfasdf\nqwerqwerqwerqwerqwerqwerqwerqwerqwer\n-----END PRIVATE KEY-----\n"))
		})

		It("serializes the data", func() {
			obj, err := edit.NewObject([]byte(withBlockScalar))
			Expect(err).ToNot(HaveOccurred())
			var buf strings.Builder
			err = obj.Serialize(&buf)
			Expect(err).ToNot(HaveOccurred())
			Expect(buf.String()).To(Equal(withBlockScalar))
		})
	})

	Context("with complex content", func() {
		It("loads the data", func() {
			obj, err := edit.NewObject([]byte(complexMixedContext))
			Expect(err).ToNot(HaveOccurred())
			Expect(obj.Kind).To(Equal("DecryptedSecret"))
			Expect(obj.Data).To(HaveKeyWithValue("MYKEY", "myvalue"))
			Expect(obj.Data).To(HaveKeyWithValue("RANDOM_VALUE", "4"))
			Expect(obj.Data).To(HaveKeyWithValue("tls.key", "-----BEGIN PRIVATE KEY-----\nMIIEvAIBADANasdfasdfasdfasdfasdfasdf\nqwerqwerqwerqwerqwerqwerqwerqwerqwer\n-----END PRIVATE KEY-----\n"))
		})

		It("serializes the data", func() {
			obj, err := edit.NewObject([]byte(complexMixedContext))
			Expect(err).ToNot(HaveOccurred())
			var buf strings.Builder
			err = obj.Serialize(&buf)
			Expect(err).ToNot(HaveOccurred())
			Expect(buf.String()).To(Equal(complexMixedContext))
		})

		It("encrypts the data", func() {
			obj, err := edit.NewObject([]byte(complexMixedContext))
			Expect(err).ToNot(HaveOccurred())
			err = obj.Encrypt(kmsMock(), "12345", false, false)
			Expect(obj.Kind).To(Equal("EncryptedSecret"))
			Expect(obj.Data["MYKEY"]).To(HavePrefix("crypto"))
			Expect(obj.Data["RANDOM_VALUE"]).To(HavePrefix("crypto"))
			Expect(obj.Data["tls.key"]).To(HavePrefix("crypto"))
		})

		// It("serializes the data after encryption", func() {
		// 	obj, err := edit.NewObject([]byte(complexMixedContext))
		// 	Expect(err).ToNot(HaveOccurred())
		// 	err = obj.Encrypt(kmsMock(), "12345", false, false)
		// 	var buf strings.Builder
		// 	err = obj.Serialize(&buf)
		// 	Expect(err).ToNot(HaveOccurred())
		// 	Expect(buf.String()).To(Equal(complexEncryptedContent))
		// })
	})

	Context("with complex encryped content", func() {
		It("loads the data", func() {
			obj, err := edit.NewObject([]byte(complexEncryptedContent))
			Expect(err).ToNot(HaveOccurred())
			Expect(obj.Kind).To(Equal("EncryptedSecret"))
			Expect(obj.Data).To(HaveKeyWithValue("MYKEY", "a21zbXl2YWx1ZQ=="))
			Expect(obj.Data).To(HaveKeyWithValue("RANDOM_VALUE", "a21zNA=="))
			Expect(obj.Data).To(HaveKeyWithValue("tls.key", "a21zLS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCk1JSUV2QUlCQURBTmFzZGZhc2RmYXNkZmFzZGZhc2RmYXNkZgpxd2VycXdlcnF3ZXJxd2VycXdlcnF3ZXJxd2VycXdlcnF3ZXIKLS0tLS1FTkQgUFJJVkFURSBLRVktLS0tLQo="))
		})

		It("serializes the data", func() {
			obj, err := edit.NewObject([]byte(complexEncryptedContent))
			Expect(err).ToNot(HaveOccurred())
			var buf strings.Builder
			err = obj.Serialize(&buf)
			Expect(err).ToNot(HaveOccurred())
			Expect(buf.String()).To(Equal(complexEncryptedContent))
		})

		It("decrypts the data", func() {
			obj, err := edit.NewObject([]byte(complexEncryptedContent))
			Expect(err).ToNot(HaveOccurred())
			err = obj.Decrypt(kmsMock())
			Expect(obj.Kind).To(Equal("DecryptedSecret"))
			Expect(obj.Data).To(HaveKeyWithValue("MYKEY", "myvalue"))
			Expect(obj.Data).To(HaveKeyWithValue("RANDOM_VALUE", "4"))
			Expect(obj.Data).To(HaveKeyWithValue("tls.key", "-----BEGIN PRIVATE KEY-----\nMIIEvAIBADANasdfasdfasdfasdfasdfasdf\nqwerqwerqwerqwerqwerqwerqwerqwerqwer\n-----END PRIVATE KEY-----\n"))
		})

		It("serializes the data after decryption", func() {
			obj, err := edit.NewObject([]byte(complexEncryptedContent))
			Expect(err).ToNot(HaveOccurred())
			err = obj.Decrypt(kmsMock())
			var buf strings.Builder
			err = obj.Serialize(&buf)
			Expect(err).ToNot(HaveOccurred())
			Expect(buf.String()).To(Equal(complexMixedContext))
		})
	})
})
