package auth

import "os"

// EnvToken returns the GROUNDS_TOKEN value if set. CI flows use this
// to skip keyring/refresh entirely.
func EnvToken() string { return os.Getenv("GROUNDS_TOKEN") }
