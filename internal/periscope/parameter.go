package periscope

import (
	"os"
	"strconv"
	"strings"
)

var scanThreads = envGetInt("PERISCOPE_SCAN_THREADS", 32)
var testDebug = envGetBool("PERISCOPE_TEST_DEBUG", false)

func envGetInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		if i, err := strconv.ParseInt(value, 10, 0); err == nil {
			return int(i)
		}
	}
	return fallback
}

var stringToBool map[string]bool = map[string]bool{
	"1":     true,
	"t":     true,
	"true":  true,
	"y":     true,
	"yes":   true,
	"0":     false,
	"f":     false,
	"false": false,
	"n":     false,
	"no":    false,
}

func envGetBool(key string, fallback bool) bool {
	if value, ok := os.LookupEnv(key); ok {
		if b, ok := stringToBool[strings.ToLower(value)]; ok {
			return b
		}
	}
	return fallback
}
