package main

import (
	"net/http"
)

func main() {
	//nolint
	err := http.ListenAndServe(":8080", http.FileServer(http.Dir(".")))
	if err != nil {
		return
	}
}
