# Copyright 2019 Ridecell, Inc.
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

all: test build

# Run tests
test: generate fmt vet
	ginkgo --randomizeAllSpecs --randomizeSuites --cover --trace --progress ./pkg/... ./cmd/...
	gover

# Build command binary
build: generate fmt vet
	go build -o bin/ridectl github.com/Ridecell/ridectl/cmd/ridectl

# Build command binary, for macOS
build_macos: generate fmt vet
	GOOS=darwin GOARCH=amd64 go build -o bin/ridectl.macos github.com/Ridecell/ridectl/cmd/ridectl

# Build command binary, for Linux
build_linux: generate fmt vet
	GOOS=linux GOARCH=amd64 go build -o bin/ridectl.linux github.com/Ridecell/ridectl/cmd/ridectl

# Run go fmt against code
fmt:
	go fmt ./pkg/... ./cmd/...

# Run go vet against code
vet:
	go vet ./pkg/... ./cmd/...

# Generate code
generate:
	go generate ./pkg/... ./cmd/...

# Install tools
tools:
	if ! type dep >/dev/null; then go get github.com/golang/dep/cmd/dep; fi
	go get -u github.com/onsi/ginkgo/ginkgo github.com/modocache/gover github.com/mattn/goveralls github.com/matryer/moq

# Install dependencies
dep: tools
	dep ensure

# Display a coverage report
cover:
	go tool cover -html=gover.coverprofile
