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
  wget https://get.gravitational.com/teleport-v$TSH_VERSION-darwin-arm64-bin.tar.gz
  tar -xf teleport-v$TSH_VERSION-darwin-arm64-bin.tar.gz
  cp teleport/tsh.app/Contents/MacOS/tsh pkg/exec/bin/
  # Re-sign with ad-hoc signature so tsh runs standalone outside the .app bundle.
  # The bundle binary carries a keychain-access-groups entitlement that requires
  # its embedded.provisionprofile; without this macOS sends SIGKILL on execution.
  codesign --remove-signature pkg/exec/bin/tsh
  codesign -s - pkg/exec/bin/tsh
  GOOS=darwin GOARCH=arm64 go build -o bin/ridectl.macos -ldflags "-X github.com/Ridecell/ridectl/pkg/exec.tshMD5=$(md5sum pkg/exec/bin/tsh | awk '{ print $1 }'| tr -d '\n') -X github.com/Ridecell/ridectl/pkg/cmd.version=$(git describe --tags)" -tags release github.com/Ridecell/ridectl/cmd/ridectl
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
