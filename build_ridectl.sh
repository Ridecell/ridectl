#!/bin/bash
set -x

OS=$1
TSH_VERSION=$2

if [ -z "$OS" ]
then
      echo "OS not provided, exiting."
      exit 1
fi

if [ -z "$TSH_VERSION" ]
then
      echo "TSH_VERSION not provided, exiting."
      exit 1
fi

# remove any old teleport files/folders
rm -rf teleport*

if [[ "$OS" == "macos" ]]
then
  wget https://get.gravitational.com/teleport-v$TSH_VERSION-darwin-amd64-bin.tar.gz
  tar -xf teleport-v$TSH_VERSION-darwin-amd64-bin.tar.gz
  cp teleport/tsh pkg/exec/bin/
  GOOS=darwin GOARCH=amd64 go build -o bin/ridectl.macos -ldflags "-X github.com/Ridecell/ridectl/pkg/exec.tshMD5=$(md5sum teleport/tsh | awk '{ print $1 }'| tr -d '\n') -X github.com/Ridecell/ridectl/pkg/cmd.version=$(git describe --tags)" -tags release github.com/Ridecell/ridectl/cmd/ridectl
elif [[ "$OS" == "linux" ]]
then
  wget https://get.gravitational.com/teleport-v$TSH_VERSION-linux-amd64-bin.tar.gz
  tar -xf teleport-v$TSH_VERSION-linux-amd64-bin.tar.gz
  cp teleport/tsh pkg/exec/bin/
  GOOS=linux GOARCH=amd64 go build -o bin/ridectl.linux -ldflags "-X github.com/Ridecell/ridectl/pkg/exec.tshMD5=$(md5sum teleport/tsh | awk '{ print $1 }'| tr -d '\n') -X github.com/Ridecell/ridectl/pkg/cmd.version=$(git describe --tags)" -tags release github.com/Ridecell/ridectl/cmd/ridectl
else
  echo "Invalid OS value: $OS, exiting."
  exit 1
fi
