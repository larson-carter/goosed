//go:build windows

package main

import (
	"log"

	agent "goosed/services/agents/windows"
)

func main() {
	if err := agent.Run(); err != nil {
		log.Fatalf("windows agent exited with error: %v", err)
	}
}
