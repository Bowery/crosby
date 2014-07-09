// Copyright 2013-2014 Bowery, Inc.
package main

import (
	"time"

	"github.com/Bowery/gopackages/schemas"
)

func GetUser(id string) (*schemas.Developer, error) {
	return nil, nil
}

func Save(dev *schemas.Developer) error {
	return nil
}

// not used o// Abstracting this into its own function because it will probably change at some point
func HasExpired(dev *schemas.Developer) bool {
	// null time or before now
	return dev.Expiration.Equal(time.Time{}) || dev.Expiration.Before(time.Now())
}
