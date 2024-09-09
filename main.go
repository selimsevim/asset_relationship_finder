package main

import (
    "fmt"
    "log"
    "net/http"
    "os"

    "asset_relationship_finder/handlers"
)

func main() {
    // Serve static files from the public folder
    fs := http.FileServer(http.Dir("public"))
    http.Handle("/static/", http.StripPrefix("/static/", fs))

    // Handle asset-related requests
    http.HandleFunc("/data-extension-detail", handlers.DataExtensionDetail)
    http.HandleFunc("/automation-activity-detail", handlers.AutomationActivityDetail)
    http.HandleFunc("/cloud-page-detail", handlers.CloudPageDetail)
    http.HandleFunc("/email-detail", handlers.EmailDetail)

    // Handle OAuth login and logout
    http.HandleFunc("/auth/login", handlers.SalesforceLoginHandler)
    http.HandleFunc("/auth/logout", handlers.SalesforceLogoutHandler)


    // Handle home page 
    http.HandleFunc("/", handlers.HomeHandler)

    // Set the port, default to 8080 if not specified
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
        fmt.Println("No PORT environment variable detected, defaulting to", port)
    }

    fmt.Printf("Server started at :%s\n", port)
    if err := http.ListenAndServe(":"+port, nil); err != nil {
        log.Fatalf("Server failed: %s", err)
    }
}
