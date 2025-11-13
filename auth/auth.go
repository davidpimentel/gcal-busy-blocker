package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	credentialsFile = "credentials.json"
)

func getOauthConfig(scope []string) *oauth2.Config {
	b, err := os.ReadFile(credentialsFile)
	if err != nil {
		log.Fatalf("Unable to read credentials.json: %v", err)
	}

	config, err := google.ConfigFromJSON(b, scope...)
	if err != nil {
		log.Fatalf("unable to parse client secret file to config: %v", err)
	}
	return config
}

func GetClient(tokenFile string, scope []string) (*http.Client, error) {
	config := getOauthConfig(scope)

	tok, err := tokenFromFile(tokenFile)
	if err != nil {
		return nil, fmt.Errorf("token not found, please run 'login' command first: %v", err)
	}

	return config.Client(context.Background(), tok), nil
}

func GetTokenFromWeb(tokenFile string, scope []string) {
	config := getOauthConfig(scope)

	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser:\n%v\n", authURL)
	fmt.Println("Enter the authorization code:")

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	saveToken(tokenFile, tok)
	fmt.Printf("Authentication successful! Token saved to %s\n", tokenFile)
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}
