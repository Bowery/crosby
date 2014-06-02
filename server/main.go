// Copyright 2013-2014 Bowery, Inc.
// Contains the main entry point
package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"os"
)

func main() {
	router := mux.NewRouter()
	router.NotFoundHandler = NotFoundHandler
	for _, r := range Routes {
		route := router.NewRoute()
		route.Path(r.Path).Methods(r.Methods...)
		route.HandlerFunc(r.Handler)
	}

	// Start the server.
	server := &http.Server{
		Addr:    ":3000",
		Handler: &SlashHandler{&LogHandler{os.Stdout, router}},
	}

	err := server.ListenAndServe()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
