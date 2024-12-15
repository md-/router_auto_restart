package main

import (
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"os"
	"time"
)

func main() {
	// Define URLs to check
	sites := []string{
		"https://www.google.com",
		"https://www.cloudflare.com",
	}

	username := os.Getenv("ROUTER_USERNAME")
	password := os.Getenv("ROUTER_PASSWORD")

	if username == "" || password == "" {
		fmt.Println("Error: ROUTER_USERNAME and ROUTER_PASSWORD must be set.")
		os.Exit(1)
	}

	// Check router login at startup
	if !canLoginToRouter(username, password) {
		fmt.Println("Cannot log in to the router. Exiting.")
		os.Exit(1)
	}
	fmt.Println("Router login check passed.")

	// Monitoring variables
	checkInterval := 5 * time.Second
	unreachableDuration := 30 * time.Second
	start := time.Now()

	for {

		if !isSiteAccessible("http://192.168.0.1") {
			fmt.Println("Router is not accessible. Skipping site checks.")
			time.Sleep(checkInterval)
			start = time.Now() // reset timer while router is down
			continue
		}

		allUnavailable := true

		// Check site accessibility
		for _, site := range sites {
			if isSiteAccessible(site) {
				allUnavailable = false
				break // At least one site is accessible, no need to check further
			}
			fmt.Printf("Site %s is not accessible\n", site)
		}

		if !allUnavailable {
			start = time.Now() // Reset the timer
		} else {
			elapsed := time.Since(start)
			if elapsed >= unreachableDuration {
				fmt.Println("All sites have been inaccessible for 30 seconds. Restarting router...")
				if err := RestartRouterProcess(username, password); err != nil {
					fmt.Println("Error restarting router:", err)
				} else {
					fmt.Println("Router restarted successfully.")
				}
				fmt.Println("Waiting for 2 minutes.")
				time.Sleep(2 * time.Minute)
				start = time.Now() // Reset the timer after a restart
				fmt.Println("Proceeding")
			}
		}

		// Wait before the next check
		time.Sleep(checkInterval)
	}
}

func isSiteAccessible(url string) bool {
	client := http.Client{
		Timeout: 2 * time.Second,
	}
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func canLoginToRouter(username, password string) bool {
	routerURL := "http://192.168.0.1/goform/logon"

	jar, err := cookiejar.New(nil)
	if err != nil {
		return false
	}
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// Attempt to log in to the router
	success, err := loginToRouter(client, routerURL, username, password)
	if err != nil {
		fmt.Printf("Router login check failed: %v\n", err)
		return false
	}
	return success
}
