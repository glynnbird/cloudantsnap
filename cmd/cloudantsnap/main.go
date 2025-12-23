package main

import (
	snap "github.com/glynnbird/cloudantsnap/internal/app"
)

func main() {

	// create cloudant snap
	cloudantSnap, err := snap.New()
	if err != nil {
		panic(err)
	}

	// run it
	err = cloudantSnap.Run()
	if err != nil {
		panic(err)
	}
}
