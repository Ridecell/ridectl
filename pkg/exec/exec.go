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
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

var (
	//go:embed bin/tsh
	tsh []byte
	//go:embed bin/tsh.md5
	tshMD5 string
)

func InstallTsh() error {
	executablePath, _ := os.Executable()
	dir, err := filepath.Abs(filepath.Dir(executablePath))
	if err != nil {
		return err
	}
	return os.WriteFile(dir+"/tsh", tsh, 0755)
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
			return fmt.Errorf(stderr.String())
		}
		return fmt.Errorf("Error while executing command: %s", err.Error())
	}
	return nil
}
