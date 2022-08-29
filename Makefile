# Copyright 2021 Ridecell, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

TSH_VERSION = 10.1.4

# Build command binary, for macOS
build_macos:
	wget https://get.gravitational.com/teleport-v$(TSH_VERSION)-darwin-amd64-bin.tar.gz
	tar -xf teleport-v$(TSH_VERSION)-darwin-amd64-bin.tar.gz
	cp teleport/tsh pkg/exec/bin/
	# Adding double $ in awk command to escape single $
	GOOS=darwin GOARCH=amd64 go build -o bin/ridectl.macos -ldflags "-X github.com/Ridecell/ridectl/pkg/exec.tshMD5=$(shell md5sum teleport/tsh | awk '{ print $$1 }'| tr -d '\n') -X github.com/Ridecell/ridectl/pkg/cmd.version=$(shell git describe --tags)" -tags release github.com/Ridecell/ridectl/cmd/ridectl

# Build command binary, for Linux
build_linux:
	wget https://get.gravitational.com/teleport-v$(TSH_VERSION)-linux-amd64-bin.tar.gz
	tar -xf teleport-v$(TSH_VERSION)-linux-amd64-bin.tar.gz
	cp teleport/tsh pkg/exec/bin/
	# Adding double $ in awk command to escape single $
	GOOS=linux GOARCH=amd64 go build -o bin/ridectl.linux -ldflags "-X github.com/Ridecell/ridectl/pkg/exec.tshMD5=$(shell md5sum teleport/tsh | awk '{ print $$1 }'| tr -d '\n') -X github.com/Ridecell/ridectl/pkg/cmd.version=$(shell git describe --tags)" -tags release github.com/Ridecell/ridectl/cmd/ridectl
