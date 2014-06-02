// Copyright 2013-2014 Bowery, Inc.
package main

import (
	"code.google.com/p/go-uuid/uuid"
	"github.com/orchestrate-io/gorc"
	"time"
)

var orchestrate *gorc.Client

var UserCollection = "users"

func init() {
	orchestrate = gorc.NewClient("d10728b1-fb0d-4f02-b778-c0d6f3c725ff")
}

type User struct {
	ID          string    `json:"id"`
	Name        string    `json:"name,omitempty"`
	Email       string    `json:"email,omitempty"`
	Password    string    `json:"password,omitempty"`
	Salt        string    `json:"salt,omitempty"`
	StripeToken string    `json:"stripeToken,omitempty"`
	Expiration  time.Time `json:"expiration,omitempty"`
}

func GetUser(id string) (*User, error) {
	rawResult, err := orchestrate.Get(UserCollection, id)
	if err != nil {
		return nil, err
	}
	u := &User{}
	if err := rawResult.Value(u); err != nil {
		return nil, err
	}
	return u, nil
}

func (u *User) Save() error {
	if u.ID == "" {
		u.ID = uuid.New()
	}
	_, err := orchestrate.Put(UserCollection, u.ID, u)
	return err
}

func (u *User) AddEvent(name string, data interface{}) error {
	// if there's no ID then create the user/ID
	if u.ID == "" {
		err := u.Save()
		if err != nil {
			return err
		}
	}
	return orchestrate.PutEvent(UserCollection, u.ID, name, data)
}

// Abstracting this into its own function because it will probably change at some point
func (u *User) HasExpired() bool {
	// null time or before now
	return u.Expiration.Equal(time.Time{}) || u.Expiration.Before(time.Now())
}
