package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	// "errors"

	// "golang.org/x/net/html"
	"github.com/PuerkitoBio/goquery"
)

func RestartRouterProcess(username, password string) error {
	// Your existing router restart logic
	fmt.Println("Restarting router...")

	routerURL := "http://192.168.0.1/goform/logon"
	restartPageURL := "http://192.168.0.1/ad_restart_gateway.html"
	restartActionURL := "http://192.168.0.1/goform/ad_restart_gateway"

	// Create an HTTP client with a cookie jar
	jar, err := cookiejar.New(nil)
	if err != nil {
		return err
	}
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// Login to the router
	success, err := loginToRouter(client, routerURL, username, password)
	if err != nil {
		return fmt.Errorf("login error: %w", err)
	}

	if !success {
		return fmt.Errorf("login failed: invalid credentials")
	}

	fmt.Println("Login successful!")

	// Fetch the restart page to extract the CSRF token
	csrfToken, err := fetchCSRFToken(client, restartPageURL)
	if err != nil {
		return fmt.Errorf("fetch CSRF token error: %w", err)
	}

	fmt.Println("CSRF Token:", csrfToken)

	// Submit the restart form
	err = restartRouter(client, restartActionURL, csrfToken)
	if err != nil {
		fmt.Println("Error restarting router:", err)
		os.Exit(1)
	}

	fmt.Println("Router restart request submitted successfully!")
	// Call login and restart methods here
	return nil
}

func loginToRouter(client *http.Client, loginURL, username, password string) (bool, error) {
	// Prepare the form data
	data := url.Values{}
	data.Set("username_login", username)
	data.Set("password_login", password)

	// Send the POST request
	req, err := http.NewRequest("POST", loginURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	// Check for 302 redirect and log the Location header
	if resp.StatusCode == http.StatusFound {
		fmt.Println("Redirect detected to:", resp.Header.Get("Location"))
	}

	// Print cookies received
	fmt.Println("Cookies from router:")
	for _, cookie := range resp.Cookies() {
		fmt.Printf("- Name: %s, Value: %s, Domain: %s, Path: %s\n", cookie.Name, cookie.Value, cookie.Domain, cookie.Path)
	}

	// Check for the "sec" cookie
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "sec" {
			fmt.Println("Session cookie found!")
			return true, nil
		}
	}

	return false, nil
}

func fetchCSRFToken(client *http.Client, url string) (string, error) {
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Parse the HTML to extract the CSRF token
	token := ""
	token, err = extractHiddenInputValue(resp.Body, "csrftoken")
	if err != nil {
		return "", err
	}

	return token, nil
}

func extractHiddenInputValue(body io.Reader, inputName string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return "", err
	}

	// Find the input by name
	value, exists := doc.Find("input[name='" + inputName + "']").Attr("value")
	if !exists {
		return "", fmt.Errorf("hidden input with name %s not found", inputName)
	}

	return value, nil
}

func restartRouter(client *http.Client, actionURL, csrfToken string) error {
	// Prepare the form data
	data := url.Values{}
	data.Set("csrftoken", csrfToken)
	data.Set("tch_devicerestart", "0x00")

	// Send the POST request
	req, err := http.NewRequest("POST", actionURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36")

	// UNCOMMENT
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
