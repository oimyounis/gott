package utils

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
)

func Run(cmdString string) []string {
	cmd := exec.Command("bash", "-c", cmdString)
	output, err := cmd.Output()
	if err != nil {
		switch cmd.ProcessState.ExitCode() {
		case 127:
			log.Fatalln(fmt.Sprintf("Dependency not found. Executing: (%v)", cmdString))
		}
	}
	return strings.Split(string(output), "\n")
}
