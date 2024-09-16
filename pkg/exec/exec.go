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

package exec

import (
	"bytes"
	"crypto/md5"
	_ "embed"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/pterm/pterm"
)

var (
	//go:embed bin/tsh
	tsh           []byte
	tshMD5        string
	tshInstallDir = "/usr/local/bin/"
)

func InstallOrUpgradeTsh() error {
	binPath, installed := CheckBinary("tsh")
	if !installed {
		pterm.Info.Println("Tsh binary not found, installing using sudo...")
		// First write tsh binary to tmp
		err := os.WriteFile("/tmp/tsh", tsh, 0755)
		if err != nil {
			return err
		}
		// Copy tsh binary to Bin Path
		cp := []string{"cp", "/tmp/tsh", tshInstallDir}
		err = ExecuteCommand("sudo", cp, false)
		if err == nil {
			pterm.Info.Println("Tsh installation completed.")
			return nil
		}
		return err
	}

	//Generate MD5 hash of installed tsh binary
	f, err := os.Open(binPath)
	if err != nil {
		return errors.Wrapf(err, "Error opening tsh")
	}
	defer f.Close()

	hash := md5.New()
	_, err = io.Copy(hash, f)
	if err != nil {
		return errors.Wrapf(err, "Error generating hash for tsh")
	}
	// Check if tsh binary's md5 is same; if not, install tsh
	if hex.EncodeToString(hash.Sum(nil)) != GetTshMd5Hash() {
		pterm.Info.Println("Tsh version not matched, re-installing using sudo...")

		// First remove old tsh binary
		rm := []string{"rm", "-rf", binPath}
		err = ExecuteCommand("sudo", rm, false)
		if err != nil {
			return err
		}

		// Now write tsh binary to tmp
		err = os.WriteFile("/tmp/tsh", tsh, 0755)
		if err != nil {
			return err
		}
		dir, err := filepath.Abs(filepath.Dir(binPath))
		if err != nil {
			return err
		}
		// Copy tsh binary to Bin Path
		cp := []string{"cp", "/tmp/tsh", dir}
		err = ExecuteCommand("sudo", cp, false)
		if err == nil {
			pterm.Info.Println("Tsh upgrade completed.")
			return nil
		}
		return err
	}
	return nil
}

func GetTshMd5Hash() string {
	return tshMD5
}

func CheckBinary(binary string) (string, bool) {
	binaryPath, err := exec.LookPath(binary)
	return binaryPath, err == nil
}

// ExecuteCommand uses os/exec Command fucntion to execute command,
// which returns the process output/error to parent process,
// If detachProcess flag set to true, then ridectl will exit with
// no error irrespective of given command's exit code.
func ExecuteCommand(binary string, args []string, detachProcess bool) error {
	binaryPath, err := exec.LookPath(binary)
	if err != nil {
		return err
	}

	c := exec.Command(binaryPath, args...)
	c.Stdin = os.Stdin

	// Execute a process by seperating it's Stderr and Stdout streams from ridectl code
	// Here we will just execute the command, and complete ridectl command with no error
	// Mainly used by "kubectl exec" and "psql" commands
	if detachProcess {
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		_ = c.Run()
		return nil
	}

	// Here we will capture Stderr from given command output
	// Mainly used by "tsh status" and "tsh db login" commands
	var stderr bytes.Buffer
	c.Stderr = &stderr
	err = c.Run()
	if err != nil {
		if stderr.String() != "" {
			
			return fmt.Errorf("%s", stderr.String())
		}
		return fmt.Errorf("Error while executing command: %s", err.Error())
	}
	return nil
}
