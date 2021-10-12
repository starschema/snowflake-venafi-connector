package main

import (
	"log"

	"github.com/Venafi/vcert/v4/pkg/endpoint"
	"github.com/Venafi/vcert/v4/pkg/venafi/tpp"
)

func GetVenafiCredentials(tppUrl string, refreshToken string, username string, password string) (string, string, int, error) {
	if tppUrl == "" {
		log.Fatal("\nPlease provide tpp url in -tppurl=<url> format\n")
	}
	if username == "" && password == "" && refreshToken == "" {
		log.Fatal("Please provide username and password or refreshToken to get credentials for Venafi platform")
	}
	c, err := tpp.NewConnector(tppUrl, "", false, nil)
	if err != nil {
		log.Fatal(err)
	}
	if username != "" && password != "" {
		auth := endpoint.Authentication{User: username, Password: password}
		new_creds, err := c.GetRefreshToken(&auth)
		if err != nil {
			log.Fatal(err)
		}
		return new_creds.Access_token, new_creds.Refresh_token, new_creds.Expires, nil
	}

	auth := endpoint.Authentication{RefreshToken: refreshToken}
	new_creds, err := c.RefreshAccessToken(&auth)
	if err != nil {
		log.Fatal(err)
	}
	return new_creds.Access_token, new_creds.Refresh_token, new_creds.Expires, nil

}
