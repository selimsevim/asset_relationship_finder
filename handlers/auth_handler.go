package handlers

import (
    "fmt"
    "log"
    "net/http"
    "os"
    "asset_relationship_finder/auth"  // Import the auth package for token management
)

// SalesforceLoginHandler handles Salesforce login by redirecting to the OAuth authorization page
func SalesforceLoginHandler(w http.ResponseWriter, r *http.Request) {
    authURL := fmt.Sprintf("%s/v2/authorize?response_type=code&client_id=%s&redirect_uri=%s",
        os.Getenv("AUTHORIZATION_URL"),
        os.Getenv("CLIENT_ID"),
        os.Getenv("REDIRECT_URI"))
    http.Redirect(w, r, authURL, http.StatusFound)
}

// SalesforceLogoutHandler handles Salesforce logout by invalidating local tokens and clearing the session
func SalesforceLogoutHandler(w http.ResponseWriter, r *http.Request) {
    auth.LogoutTokens()  // Call the function from auth package to clear tokens
    log.Println("User logged out and tokens invalidated.")
}

// HomeHandler acts as both the home page handler and OAuth callback handler
func HomeHandler(w http.ResponseWriter, r *http.Request) {
    code := r.URL.Query().Get("code")
    if code != "" {
        log.Printf("Authorization code received: %s", code)
        err := auth.ExchangeOrRefreshToken(code, false)  // Use the auth package to exchange tokens
        if err != nil {
            log.Printf("Error exchanging code for token: %v", err)
            http.Error(w, "Failed to authenticate with Salesforce", http.StatusInternalServerError)
            return
        }

        entID, err := auth.GetUserInfo()  // Get user info using the auth package
        if err != nil {
            http.Error(w, "Failed to get user info", http.StatusInternalServerError)
            return
        }

        // Store entID in cookies with SameSite and Secure attributes
        http.SetCookie(w, &http.Cookie{
            Name:     "entID",
            Value:    entID,
            Path:     "/",
            HttpOnly: true,                      // Ensures the cookie is only accessible through HTTP(S)
            Secure:   true,                      // Ensures the cookie is only sent over HTTPS
            SameSite: http.SameSiteNoneMode,     // Allows the cookie to be sent in cross-site requests (e.g., iframes)
        })

        // Redirect back to the homepage after authentication
        http.Redirect(w, r, "/", http.StatusFound)
        return
    }

    // Serve the home page (static files)
    http.ServeFile(w, r, "public/index.html")
}
