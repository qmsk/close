package main

import (
    "github.com/qmsk/close/http"
)

func init() {
    Options.Register("http", &http.Config{})
}

