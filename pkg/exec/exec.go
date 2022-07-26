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
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func Exec(command []string) error {
	binary, err := exec.LookPath(command[0])
	if err != nil {
		return err
	}
	err = syscall.Exec(binary, command, os.Environ())
	// Panic rather than returning since this should never happen.
	panic(err)
}

// ExecuteCommand uses os/exec Command fucntion to execute command,
// which returns the process output/error to parent process,
// unlike syscall.Exec()
func ExecuteCommand(binary string, args []string) error {
	binaryPath, err := exec.LookPath(binary)
	if err != nil {
		return err
	}
	var stderr bytes.Buffer
	c := exec.Command(binaryPath, args...)
	c.Stderr = &stderr
	c.Stdin = os.Stdin
	err = c.Run()
	if err != nil {
		return fmt.Errorf(stderr.String())
	}
	return nil
}
