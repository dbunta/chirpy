package main

import (
	"fmt"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(".")))

	server := http.Server{
		Handler: mux,
		Addr:    ":8080",
	}

	//go server.ListenAndServe()

	err := server.ListenAndServe()
	if err != nil {
		fmt.Printf("%w", err)
	}
}
