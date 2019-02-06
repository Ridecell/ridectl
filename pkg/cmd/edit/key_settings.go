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
	"os"
	"path"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

func FindKeyId(manifestPath string) (string, error) {
	keysPath := path.Join(manifestPath, "..", ".keys.yml")
	keysF, err := os.Open(keysPath)
	if err != nil {
		if os.IsNotExist(err) {
			// If the file doesn't exist, the key ID from the file is "". This allows
			// editing a file with existing encrypted data without worrying about the key file.
			return "", nil
		}
		return "", errors.Wrapf(err, "error loading key settings file %s", keysPath)
	}
	decoder := yaml.NewDecoder(keysF)
	keys := yaml.MapSlice{}
	err = decoder.Decode(&keys)
	if err != nil {
		return "", errors.Wrap(err, "error decoding key settings YAML")
	}
	matchKey := ""
	matchValue := ""
	defaultValue := ""
	matchTarget := path.Base(manifestPath)
	for _, m := range keys {
		mKey := m.Key.(string)
		mValue := m.Value.(string)
		if mKey == "default" {
			// Handled later.
			defaultValue = mValue
			continue
		}
		matched, err := path.Match("*"+mKey+"*", matchTarget)
		if err != nil {
			return "", errors.Wrapf(err, "error matching key %s", mKey)
		}
		if matched {
			if len(mKey) > len(matchKey) {
				matchKey = mKey
				matchValue = mValue
			}
		}
	}
	if matchValue == "" {
		return defaultValue, nil
	}
	return matchValue, nil
}
