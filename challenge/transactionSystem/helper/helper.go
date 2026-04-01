package helper

import "fmt"

type LogPosition string

func NewAPIPath(method string, path string) string {
	return fmt.Sprintf("%s %s", method, path)
}

func PrintLog(domain string, position LogPosition, msg string) {
	fmt.Printf("[%s][%s] %s\n", domain, position, msg)
}
