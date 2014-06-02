// Copyright 2013-2014 Bowery, Inc.
// Contains the routes for crosby server.
package main

import (
	"github.com/bradrydzewski/go.stripe"
	"github.com/gorilla/mux"
	"net/http"
	"time"
)

// 32 MB, same as http.
const httpMaxMem = 32 << 10

// Route is a single named route with a http.HandlerFunc.
type Route struct {
	Path    string
	Methods []string
	Handler http.HandlerFunc
}

// List of named routes.
var Routes = []*Route{
	&Route{"/", []string{"GET"}, HomeHandler},
	&Route{"/session", []string{"POST"}, CreateSessionHandler},
	&Route{"/session/{id}", []string{"GET"}, SessionHandler},
	&Route{"/signup", []string{"GET"}, SignUpHandler},
}

func init() {
	stripe.SetKey("sk_test_BKnPoMNUWSGHJsLDcSGeV8I9")
}

// GET /, Upload new service.
func HomeHandler(rw http.ResponseWriter, req *http.Request) {
	res := NewResponder(rw, req)
	res.Body["status"] = "success"
	res.Body["message"] = "Welcome to Crosby.io"
	res.Send(http.StatusOK)
}

// POST /session, Creates a new user and charges them for the first year.
func CreateSessionHandler(rw http.ResponseWriter, req *http.Request) {
	res := NewResponder(rw, req)
	err := req.ParseForm()

	// Simple Validation
	if err != nil || req.PostFormValue("stripeToken") == "" || req.PostFormValue("stripeEmail") == "" || req.PostFormValue("password") == "" {
		res.Body["status"] = "failed"
		var strErr string
		if err == nil {
			strErr = "Missing Required Field"
		} else {
			strErr = err.Error()
		}
		res.Body["error"] = strErr
		res.Send(http.StatusBadRequest)
		return
	}

	u := &User{
		Name:       req.PostFormValue("name"),
		Email:      req.PostFormValue("stripeEmail"),
		Expiration: time.Now(),
	}

	// Hash Password
	u.Salt, err = HashToken()
	if err != nil {
		res.Body["status"] = "failed"
		res.Body["error"] = err.Error()
		res.Send(http.StatusBadRequest)
		return
	}
	u.Password = HashPassword(req.PostFormValue("password"), u.Salt)

	// Create Stripe Customer
	customerParams := stripe.CustomerParams{
		Email: u.Email,
		Desc:  u.Name,
		Token: req.PostFormValue("stripeToken"),
	}
	customer, err := stripe.Customers.Create(&customerParams)
	if err != nil {
		res.Body["status"] = "failed"
		res.Body["error"] = err.Error()
		res.Send(http.StatusBadRequest)
		return
	}

	// Charge Stripe Customer
	chargeParams := stripe.ChargeParams{
		Desc:     "Crosby Annual License",
		Amount:   2500,
		Currency: "usd",
		Customer: customer.Id,
	}
	_, err = stripe.Charges.Create(&chargeParams)
	if err != nil {
		res.Body["status"] = "failed"
		res.Body["error"] = err.Error()
		res.Send(http.StatusBadRequest)
		return
	}

	// Update Stripe Info and Persist to Orchestrate
	u.StripeToken = customer.Id
	if err := u.Save(); err != nil {
		res.Body["status"] = "failed"
		res.Body["error"] = err.Error()
		res.Send(http.StatusBadRequest)
		return
	}

	res.Body["status"] = "success"
	res.Body["user"] = u
	res.Send(http.StatusOK)
}

// GET /session/{id}, Gets user by ID. If their license has expired it attempts
// to charge them again. It is called everytime crosby is run.
func SessionHandler(rw http.ResponseWriter, req *http.Request) {
	res := NewResponder(rw, req)
	u, err := GetUser(mux.Vars(req)["id"])
	if err != nil {
		res.Body["status"] = "failed"
		res.Body["error"] = err.Error()
		res.Send(http.StatusBadRequest)
		return
	}

	if u.Expiration.Before(time.Now()) {
		res.Body["status"] = "found"
		res.Body["user"] = u
		res.Send(http.StatusOK)
		return
	}

	if u.StripeToken == "" {
		res.Body["status"] = "expired"
		res.Body["user"] = u
		res.Send(http.StatusOK)
		return
	}

	// Charge them, update expiration, & respond with found.
	// Charge Stripe Customer
	chargeParams := stripe.ChargeParams{
		Desc:     "Crosby Annual License",
		Amount:   2500,
		Currency: "usd",
		Customer: u.StripeToken,
	}
	_, err = stripe.Charges.Create(&chargeParams)
	if err != nil {
		res.Body["status"] = "failed"
		res.Body["error"] = err.Error()
		res.Send(http.StatusBadRequest)
		return
	}
	u.Expiration = time.Now()
	if err := u.Save(); err != nil {
		res.Body["status"] = "failed"
		res.Body["error"] = err.Error()
		res.Send(http.StatusBadRequest)
		return
	}

	res.Body["status"] = "found"
	res.Body["user"] = u
	res.Send(http.StatusOK)
	return
}

// GET /signup, Renders signup find. Will also handle billing
func SignUpHandler(res http.ResponseWriter, req *http.Request) {
	http.ServeFile(res, req, "static/signup.html")
}
