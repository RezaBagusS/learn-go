package helper

import (
	"fmt"
	// "github.com/golang-jwt/jwt/v5"
)

type LogPosition string

func NewAPIPath(method, version, path string) string {
	return fmt.Sprintf("%s /api/%s%s", method, version, path)
}

func PrintLog(domain string, position LogPosition, msg string) {
	fmt.Printf("[%s][%s] %s\n", domain, position, msg)
}

// func GetClaims(ctx context.Context) jwt.MapClaims {
// 	return ctx.Value("claims").(jwt.MapClaims)
// }
