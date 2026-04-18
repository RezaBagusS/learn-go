package helper

import (
	"fmt"
)

type LogPosition string

func NewAPIPath(method, version, path string) string {
	return fmt.Sprintf("%s /api/%s%s", method, version, path)
}

func PrintLog(domain string, position LogPosition, msg string) {
	fmt.Printf("[%s][%s] %s\n", domain, position, msg)
}
