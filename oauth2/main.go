// a tiny script to interactively check if your oauth-client-credentials or oauth-authorization-code flow works

package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func main() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Select OAuth2 flow to test:")
	fmt.Println("1) Client Credentials Flow")
	fmt.Println("2) Authorization Code Flow")
	fmt.Print("Enter choice (1 or 2): ")
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	if choice == "1" {
		testClientCredentialsFlow(reader)
	} else if choice == "2" {
		testAuthorizationCodeFlow(reader)
	} else {
		fmt.Println("Invalid choice.")
	}
}

func testClientCredentialsFlow(reader *bufio.Reader) {
	fmt.Println("\nTesting OAuth2 Client Credentials Flow")

	fmt.Print("Enter Token Endpoint URL: ")
	tokenEndpoint, _ := reader.ReadString('\n')
	tokenEndpoint = strings.TrimSpace(tokenEndpoint)

	fmt.Print("Enter Client ID: ")
	clientID, _ := reader.ReadString('\n')
	clientID = strings.TrimSpace(clientID)

	fmt.Print("Enter Client Secret: ")
	clientSecret, _ := reader.ReadString('\n')
	clientSecret = strings.TrimSpace(clientSecret)

	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)

	resp, err := http.PostForm(tokenEndpoint, data)
	if err != nil {
		fmt.Println("Error making request:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("Response Status:", resp.Status)
	body, _ := io.ReadAll(resp.Body)
	fmt.Println("Response Body:", string(body))
}

func testAuthorizationCodeFlow(reader *bufio.Reader) {
	fmt.Println("\nTesting OAuth2 Authorization Code Flow")

	fmt.Print("Enter Authorization Endpoint URL: ")
	authEndpoint, _ := reader.ReadString('\n')
	authEndpoint = strings.TrimSpace(authEndpoint)

	fmt.Print("Enter Token Endpoint URL: ")
	tokenEndpoint, _ := reader.ReadString('\n')
	tokenEndpoint = strings.TrimSpace(tokenEndpoint)

	fmt.Print("Enter Client ID: ")
	clientID, _ := reader.ReadString('\n')
	clientID = strings.TrimSpace(clientID)

	fmt.Print("Enter Client Secret: ")
	clientSecret, _ := reader.ReadString('\n')
	clientSecret = strings.TrimSpace(clientSecret)

	fmt.Print("Enter Redirect URI: ")
	redirectURI, _ := reader.ReadString('\n')
	redirectURI = strings.TrimSpace(redirectURI)

	fmt.Print("Enter Scope (space-separated): ")
	scope, _ := reader.ReadString('\n')
	scope = strings.TrimSpace(scope)

	state := "state123" // You can generate a random state value here

	fmt.Println("\nOpen the following URL in your browser to authorize:")
	authURL := fmt.Sprintf("%s?response_type=code&client_id=%s&redirect_uri=%s&scope=%s&state=%s",
		authEndpoint,
		url.QueryEscape(clientID),
		url.QueryEscape(redirectURI),
		url.QueryEscape(scope),
		url.QueryEscape(state))
	fmt.Println(authURL)

	fmt.Print("\nAfter authorization, enter the Authorization Code: ")
	code, _ := reader.ReadString('\n')
	code = strings.TrimSpace(code)

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)

	resp, err := http.PostForm(tokenEndpoint, data)
	if err != nil {
		fmt.Println("Error making request:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("Response Status:", resp.Status)
	body, _ := io.ReadAll(resp.Body)
	fmt.Println("Response Body:", string(body))
}
