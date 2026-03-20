package helper

import "fmt"

type LogPosition string

const (
	LogPositionHandler    LogPosition = "handler"
	LogPositionRepo       LogPosition = "repo"
	LogPositionService    LogPosition = "service"
	LogPositionServer     LogPosition = "server"
	LogPositionMiddleware LogPosition = "middleware"
	LogPositionConfig     LogPosition = "config"
)

func NewAPIPath(method string, path string) string {
	return fmt.Sprintf("%s %s", method, path)
}

func PrintLog(domain string, position LogPosition, msg string) {
	fmt.Printf("[%s][%s] %s\n", domain, position, msg)
}
