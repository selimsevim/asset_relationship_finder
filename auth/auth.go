package auth

import (
    "fmt"
    "log"
    "net/http"
    "os"
    "sync"
    "time"
    "bytes"
    "io/ioutil"
    "encoding/json"
)

// Declaring variables for token management
var (
    accessToken  string
    refreshToken string
    tokenExpiry  time.Time
    tokenMutex   sync.Mutex
)

// TokenResponse struct for tokens
type TokenResponse struct {
    AccessToken  string `json:"access_token"`
    RefreshToken string `json:"refresh_token"`
    ExpiresIn    int    `json:"expires_in"`
}

// --- Public Token Management Functions ---

// GetAccessToken retrieves the token, refreshes it if necessary
func GetAccessToken() (string, error) {
    tokenMutex.Lock()
    defer tokenMutex.Unlock()

    var err error

    // Check if the token is valid
    if time.Now().Before(tokenExpiry) && accessToken != "" {
        // Token is still valid
        return accessToken, nil
    }

    // Token expired or missing, refresh it
    if refreshToken == "" {
        err = fmt.Errorf("no refresh token available")
    } else {
        err = ExchangeOrRefreshToken("", true)
        if err != nil {
            log.Printf("Failed to refresh token: %v", err)
        }
    }

    return accessToken, err
}

// ExchangeOrRefreshToken handles the exchange/refresh of tokens
func ExchangeOrRefreshToken(code string, isRefresh bool) error {
    tokenMutex.Lock()
    defer tokenMutex.Unlock()

    // Determine grant type
    reqBody := map[string]string{
        "client_id":     os.Getenv("CLIENT_ID"),
        "client_secret": os.Getenv("CLIENT_SECRET"),
        "redirect_uri":  os.Getenv("REDIRECT_URI"),
    }
    if isRefresh {
        reqBody["grant_type"] = "refresh_token"
        reqBody["refresh_token"] = refreshToken
    } else {
        reqBody["grant_type"] = "authorization_code"
        reqBody["code"] = code
    }

    jsonReqBody, _ := json.Marshal(reqBody)
    authURL := os.Getenv("AUTHORIZATION_URL")
    tokenEndpoint := fmt.Sprintf("%s/v2/token", authURL)

    log.Printf("AUTHORIZATION_URL: %s", authURL)

    req, err := http.NewRequest("POST", tokenEndpoint, bytes.NewBuffer(jsonReqBody))
    if err != nil {
        return fmt.Errorf("failed to create token request: %v", err)
    }

    req.Header.Set("Content-Type", "application/json")
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return fmt.Errorf("failed to get tokens: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("token endpoint returned non-200 status: %d", resp.StatusCode)
    }

    var tokenRes TokenResponse
    err = json.NewDecoder(resp.Body).Decode(&tokenRes)
    if err != nil {
        return fmt.Errorf("error decoding token response: %v", err)
    }

    accessToken = tokenRes.AccessToken
    refreshToken = tokenRes.RefreshToken
    tokenExpiry = time.Now().Add(time.Duration(tokenRes.ExpiresIn) * time.Second)

    log.Printf("Tokens received: Access Token: %s, Refresh Token: %s", accessToken, refreshToken)
    return nil
}

// --- User Info Retrieval Function ---

// getUserInfo retrieves user information for Enterprise ID via the REST API
func GetUserInfo() (string, error) {
    token, err := GetAccessToken() // Get the access token
    if err != nil {
        return "", err
    }

    // Construct the REST API URL
    url := fmt.Sprintf("%s/v2/userinfo", os.Getenv("AUTHORIZATION_URL"))

    // Create a new GET request
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return "", err
    }

    // Set the authorization header with the access token
    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
    log.Printf("Authorization Header: Bearer %s", token)

    // Send the request
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        log.Printf("Error sending request: %v", err)
        return "", err
    }
    defer resp.Body.Close()

    // Check if the response status is OK
    log.Printf("Response status: %d", resp.StatusCode)
    if resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("failed to fetch user information, status: %d", resp.StatusCode)
    }

    // Read the response body
    bodyBytes, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        log.Printf("Error reading response body: %v", err)
        return "", err
    }

    // Parse the response body
    var userInfo struct {
        Organization struct {
            EnterpriseID float64 `json:"enterprise_id"` // Expect Enterprise ID as a number
        } `json:"organization"`
    }

    // Decode the response JSON
    err = json.Unmarshal(bodyBytes, &userInfo)
    if err != nil {
        log.Printf("Error decoding JSON response: %v", err)
        return "", err
    }

    // Convert the EnterpriseID to string
    enterpriseID := fmt.Sprintf("%.0f", userInfo.Organization.EnterpriseID)

    // Return only the Enterprise ID
    return enterpriseID, nil
}

// --- Handle Logout from auth_handler ---

// LogoutTokens clears the stored tokens and expiry
func LogoutTokens() {
    tokenMutex.Lock()
    defer tokenMutex.Unlock()
    accessToken = ""
    refreshToken = ""
    tokenExpiry = time.Now()
}
