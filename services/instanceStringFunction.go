package services

import "fmt"

func NameStringer(ipAddr string, loc, numNodes int) string {
	return fmt.Sprintf("%s - (%d/%d)", ipAddr, loc+1, numNodes)
}
