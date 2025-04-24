package main

import (
    "fmt"
    "log"
    "net/http"
    "os"
)

func main() {
	settings.ReadEnvironment(context.Background(), &settings.MailAppEnv)
}
