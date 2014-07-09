// Copyright 2013-2014 Bowery, Inc.
// Contains the routes for crosby server.
package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Bowery/gopackages/keen"
	"github.com/bradrydzewski/go.stripe"
)

// 32 MB, same as http.
const httpMaxMem = 32 << 10

var STATIC_DIR string = TEMPLATE_DIR
var keenC *keen.Client

// Route is a single named route with a http.HandlerFunc.
type Route struct {
	Path    string
	Methods []string
	Handler http.HandlerFunc
}

// List of named routes.
var Routes = []*Route{
	&Route{"/", []string{"GET"}, HomeHandler},
	&Route{"/download", []string{"GET"}, DownloadHandler},
	&Route{"/signup", []string{"GET"}, SignUpHandler},
	&Route{"/thanks!", []string{"GET"}, ThanksHandler},
	&Route{"/healthz", []string{"GET"}, HealthzHandler},
	&Route{"/static/{rest}", []string{"GET"}, http.StripPrefix("/static/", http.FileServer(http.Dir(STATIC_DIR))).ServeHTTP},
}

func init() {
	stripeKey := "sk_test_BKnPoMNUWSGHJsLDcSGeV8I9"
	var cwd, _ = filepath.Abs(filepath.Dir(os.Args[0]))
	if os.Getenv("ENV") == "production" {
		STATIC_DIR = cwd + "/" + STATIC_DIR
		stripeKey = "sk_live_fx0WR9yUxv6JLyOcawBdNEgj"
	}
	stripe.SetKey(stripeKey)

	keenC = &keen.Client{
		WriteKey:  "8bbe0d9425a22a6c31e6da9ae3012c738ee21000b533c351a419bb0e3d08431456359d1bea654a39c2065df0b1df997ecde7e3cf49a9be0cd44341b15c1ff5523f13d26d8060373390f47bcc6a33b80e69e2b2c1101cde4ddb3d20b16a53a439a98043919e809c09c30e4856dedc963f",
		ProjectID: "52c08d6736bf5a4a4b000005",
	}

}

// GET /, Introduction to Crosby
func HomeHandler(rw http.ResponseWriter, req *http.Request) {
	if err := RenderTemplate(rw, "home", map[string]string{"Name": "Crosby"}); err != nil {
		RenderTemplate(rw, "error", map[string]string{"Error": err.Error()})
	}
}

// GET /download, Renders Page to Download Crosby
func DownloadHandler(rw http.ResponseWriter, req *http.Request) {
	if err := RenderTemplate(rw, "download", map[string]string{"Name": "Crosby"}); err != nil {
		RenderTemplate(rw, "error", map[string]string{"Error": err.Error()})
	}
}

// GET /signup, Renders signup find. Will also handle billing
func SignUpHandler(w http.ResponseWriter, req *http.Request) {
	stripePubKey := "pk_test_m8TQEAkYWSc1jZh7czo8xhA7"
	if os.Getenv("ENV") == "production" {
		stripePubKey = "pk_live_LOngSSK6d3qwW0aBEhWSVEcF"
	}

	if err := RenderTemplate(w, "signup", map[string]interface{}{
		"isSignup":     true,
		"stripePubKey": stripePubKey,
	}); err != nil {
		RenderTemplate(w, "error", map[string]string{"Error": err.Error()})
	}
}

// Get /thanks!, Renders a thank you/confirmation message stored in static/thanks.html
func ThanksHandler(w http.ResponseWriter, req *http.Request) {
	if err := RenderTemplate(w, "thanks", map[string]interface{}{}); err != nil {
		RenderTemplate(w, "error", map[string]string{"Error": err.Error()})
	}
}
func HealthzHandler(res http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(res, "ok")
}
