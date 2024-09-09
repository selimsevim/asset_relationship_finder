package services

import (
    "bytes"
    "encoding/json"
    "encoding/xml"
    "fmt"
    "regexp"
    "io/ioutil"
    "log"
    "net/http"
    "os"
    "sort"
    "strings"
    "sync"
    "net/url"
    "time"

     "asset_relationship_finder/auth"
)


// Declaring structs for all assets
type Folder struct {
    ID           string `xml:"ID"`
    Name         string `xml:"Name"`
    ParentID     string `xml:"ParentFolder>ID"`
    ParentName   string `xml:"ParentFolder>Name"`
}

type DataExtension struct {
    CustomerKey  string `xml:"CustomerKey"`
    Name         string `xml:"Name"`
    CategoryID   string `xml:"CategoryID"`
    ObjectID     string `xml:"ObjectID"`
}

type DataExtensionTarget struct {
    Name string `xml:"Name"`
}

type QueryDefinition struct {
    Name         string `xml:"Name"`
    ObjectID     string `xml:"ObjectID"`
}

type ImportDefinition struct {
    Name         string `xml:"Name"`
    ObjectID     string `json:"ObjectID"`
}

type FilterActivity struct {
    Name                string `xml:"Name"`
    ObjectID            string `json:"ObjectID"`
}

type Email struct {
    Name string `json:"Name"`
    ID   json.Number `json:"ID"`
}

type CloudPage struct {
    Name string `json:"Name"`
    HTML string `json:"HTML"`
}

type EmailSendDefinition struct {
    Name           string
    ObjectID       string
    CustomObjectID string
    EmailID        string
}

type Script struct {
    Name       string `json:"Name"`
    ObjectID   string `json:"ssjsActivityId"`
}

type EventDefinition struct {
    DataExtensionName string    `json:"dataExtensionName"`
    CreatedDate       Time      `json:"createdDate"`
}

type Journey struct {
    Name               string `json:"Name"`
    ID                 string `json:"ID"`
    EventDefinitionKey string `json:"-"`

}

type Automation struct {
    ObjectID string `xml:"ObjectID"`
    Name     string `xml:"Name"`
}

type Activity struct {
    Name       string `xml:"Name"`
    Program struct {
        ObjectID string `xml:"ObjectID"`
    } `xml:"Program"`
}

type TriggeredSendDefinition struct {
    Name    string `xml:"Name"`
}

// PageResponse is a generic structure to hold paginated results.
type PageResponse struct {
    Count int         `json:"count"`
    Items interface{} `json:"items"`
}

// Custom Time type that implements json.Unmarshaler to handle the custom format
type Time struct {
    time.Time
}

// UnmarshalJSON method to handle custom time format
func (t *Time) UnmarshalJSON(b []byte) (err error) {
    str := string(b)
    str = str[1 : len(str)-1] // Remove the surrounding quotes

    // Define a slice of layouts to handle various formats
    layouts := []string{
        "2006-01-02T15:04:05.000", // Layout with fractional seconds
        "2006-01-02T15:04:05",     // Without fractional seconds
        "2006-01-02T15:04:05.00",  // Layout for two decimal points
    }

    // Try each layout until one succeeds
    for _, layout := range layouts {
        t.Time, err = time.Parse(layout, str)
        if err == nil {
            return nil // Parsing successful
        }
    }

    return fmt.Errorf("unable to parse time: %s", str)
}

// --- Asset Retrieval Functions ---

func GetDataExtensions(filter string) ([]DataExtension, error) {
    token, err := auth.GetAccessToken()
    if err != nil {
        return nil, err
    }

    requestBody := fmt.Sprintf(xmlTemplate, os.Getenv("SOAP_ENDPOINT"), token, "DataExtension", `
        <Properties>Name</Properties>
        <Properties>CustomerKey</Properties>
        <Properties>CategoryID</Properties>
        <Properties>ObjectID</Properties>
    `, filter)

    resp, err := soapRequest(requestBody)
    if err != nil {
        return nil, err
    }

    var response struct {
        Results []DataExtension `xml:"Body>RetrieveResponseMsg>Results"`
    }
    err = xml.Unmarshal(resp, &response)
    if err != nil {
        return nil, err
    }

    return response.Results, nil
}

func GetActivities(activityObjectID string) ([]Activity, error) {
    var activities []Activity

    token, err := auth.GetAccessToken() // Retrieve access token
    if err != nil {
        return nil, err
    }

    // Construct SOAP request body for retrieving activities with the filter for Definition.ObjectID
    requestBody := fmt.Sprintf(xmlTemplate, os.Getenv("SOAP_ENDPOINT"), token, "Activity", `
        <Properties>Name</Properties>
        <Properties>Program.ObjectID</Properties>
    `, fmt.Sprintf(`
        <Filter xsi:type="SimpleFilterPart">
            <Property>Definition.ObjectID</Property>
            <SimpleOperator>equals</SimpleOperator>
            <Value>%s</Value>
        </Filter>`, activityObjectID))

    // Call the soapRequest function
    responseBody, err := soapRequest(requestBody)
    if err != nil {
        return nil, err
    }

    // Unmarshal and process the response
    var response struct {
        Results []Activity `xml:"Body>RetrieveResponseMsg>Results"`
    }
    err = xml.Unmarshal(responseBody, &response)
    if err != nil {
        return nil, err
    }

    // Append the retrieved activities to the slice
    activities = append(activities, response.Results...)

    return activities, nil
}

func GetAutomations(automationObjectID string) ([]Automation, error) {
    var automations []Automation

    token, err := auth.GetAccessToken() // Retrieve access token
    if err != nil {
        return nil, err
    }

    // SOAP request to retrieve Automations (Programs) with the Definition.ObjectID filter
    requestBody := fmt.Sprintf(xmlTemplate, os.Getenv("SOAP_ENDPOINT"), token, "Program", `
        <Properties>Name</Properties>
        <Properties>ObjectID</Properties>
    `, fmt.Sprintf(`
        <Filter xsi:type="SimpleFilterPart">
            <Property>ObjectID</Property>
            <SimpleOperator>equals</SimpleOperator>
            <Value>%s</Value>
        </Filter>`, automationObjectID))

    // Call the soapRequest function
    responseBody, err := soapRequest(requestBody)
    if err != nil {
        return nil, err
    }

    // Unmarshal the response
    var response struct {
        Results []Automation `xml:"Body>RetrieveResponseMsg>Results"`
    }
    err = xml.Unmarshal(responseBody, &response)
    if err != nil {
        return nil, err
    }

    // Append the retrieved automations to the slice
    automations = append(automations, response.Results...)

    return automations, nil
}

func GetEmails(deName string, cloudPageID string) ([]Email, error) {
    token, err := auth.GetAccessToken()
    if err != nil {
        return nil, err
    }

    client := &http.Client{}
    pageSize := 50
    var totalPages int
    var allEmails []Email
    var mu sync.Mutex

     // Construct request body based on whether cloudPageID or deName is provided
    var requestBody map[string]interface{}

    if cloudPageID != "" && deName == "" {
        // Case 1: cloudPageID is provided, deName is empty
        requestBody = map[string]interface{}{
            "page": map[string]interface{}{
                "page":     1,
                "pageSize": pageSize,
            },
            "query": map[string]interface{}{
                "leftOperand": map[string]interface{}{
                    "property":      "assetType.name",
                    "simpleOperator": "equal",
                    "value":         "templatebasedemail",
                },
                "logicalOperator": "OR",
                "rightOperand": map[string]interface{}{
                    "property":      "assetType.name",
                    "simpleOperator": "equal",
                    "value":         "htmlemail",
                },
            },
            "sort": []map[string]interface{}{
                {"property": "id", "direction": "ASC"},
            },
            "fields": []string{"name", "id", "views"},  // Include views field
        }

    } else if deName != "" && cloudPageID == "" {
        // Case 2: deName is provided, cloudPageID is empty
        requestBody = map[string]interface{}{
            "page": map[string]interface{}{
                "page":     1,
                "pageSize": pageSize,
            },
            "query": map[string]interface{}{
                "leftOperand": map[string]interface{}{
                    "leftOperand": map[string]interface{}{
                        "property":      "assetType.name",
                        "simpleOperator": "equal",
                        "value":         "templatebasedemail",
                    },
                    "logicalOperator": "OR",
                    "rightOperand": map[string]interface{}{
                        "property":      "assetType.name",
                        "simpleOperator": "equal",
                        "value":         "htmlemail",
                    },
                },
                "logicalOperator": "AND",
                "rightOperand": map[string]interface{}{
                    "property":      "content",
                    "simpleOperator": "mustContain",
                    "value":         fmt.Sprintf(`%%%q%%`, deName),
                },
            },
            "sort": []map[string]interface{}{
                {"property": "id", "direction": "ASC"},
            },
            "fields": []string{"name", "id"},
        }
    } else {
        return nil, fmt.Errorf("either deName or cloudPageID must be provided")
    }

    // Marshal and send the initial request
    jsonBody, err := json.Marshal(requestBody)
    if err != nil {
        return nil, err
    }

    req, err := http.NewRequest("POST", fmt.Sprintf("%s/asset/v1/content/assets/query", os.Getenv("REST_ENDPOINT")), bytes.NewBuffer(jsonBody))
    if err != nil {
        return nil, err
    }
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        bodyBytes, _ := ioutil.ReadAll(resp.Body)
        return nil, fmt.Errorf("non-200 response code: %d, body: %s", resp.StatusCode, string(bodyBytes))
    }

    var pageResponse PageResponse
    bodyBytes, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    err = json.Unmarshal(bodyBytes, &pageResponse)
    if err != nil {
        return nil, err
    }

    items, ok := pageResponse.Items.([]interface{})
    if !ok {
        return nil, fmt.Errorf("failed to assert pageResponse.Items as []interface{}")
    }

    mu.Lock()
    processEmailsWrapper(items, &allEmails, deName, cloudPageID)
    mu.Unlock()

    totalItems := pageResponse.Count
    totalPages = (totalItems + pageSize - 1) / pageSize

    if totalPages <= 1 {
        return allEmails, nil
    }

    var wg sync.WaitGroup
    for p := 2; p <= totalPages; p++ {
        wg.Add(1)
        go func(page int) {
            defer wg.Done()

            requestBody["page"].(map[string]interface{})["page"] = page
            jsonBody, err := json.Marshal(requestBody)
            if err != nil {
                log.Printf("Error generating request body for page %d: %v", page, err)
                return
            }

            req, err := http.NewRequest("POST", fmt.Sprintf("%s/asset/v1/content/assets/query", os.Getenv("REST_ENDPOINT")), bytes.NewBuffer(jsonBody))
            if err != nil {
                log.Printf("Error creating request for page %d: %v", page, err)
                return
            }
            req.Header.Set("Content-Type", "application/json")
            req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

            resp, err := client.Do(req)
            if err != nil {
                log.Printf("Error making request for page %d: %v", page, err)
                return
            }
            defer resp.Body.Close()

            if resp.StatusCode != http.StatusOK {
                bodyBytes, _ := ioutil.ReadAll(resp.Body)
                log.Printf("Non-200 response code for page %d: %d, body: %s", page, resp.StatusCode, string(bodyBytes))
                return
            }

            bodyBytes, err := ioutil.ReadAll(resp.Body)
            if err != nil {
                log.Printf("Error reading response body for page %d: %v", page, err)
                return
            }

            err = json.Unmarshal(bodyBytes, &pageResponse)
            if err != nil {
                log.Printf("Error decoding response for page %d: %v", page, err)
                return
            }

            items, ok := pageResponse.Items.([]interface{})
            if !ok {
                log.Printf("Failed to assert pageResponse.Items as []interface{} for page %d", page)
                return
            }

            mu.Lock()
            processEmailsWrapper(items, &allEmails, deName, cloudPageID)
            mu.Unlock()

        }(p)
    }

    wg.Wait()
    return allEmails, nil
}


// Main function to process emails based on whether cloudPageID is present
func processEmailsWrapper(items []interface{}, allEmails *[]Email, deName, cloudPageID string) {
    if cloudPageID != "" {
        // More complex processing if cloudPageID is provided
        processEmailsWithCloudPageID(items, allEmails, cloudPageID)
    } else {
        // Simpler processing if no cloudPageID
        processSimpleEmails(items, allEmails)
    }
}

// Main function to process emails based on whether cloudPageID is present
func processEmailsWithCloudPageID(items []interface{}, allEmails *[]Email, cloudPageID string) {
    for _, item := range items {
        itemMap, ok := item.(map[string]interface{})
        if !ok {
            continue
        }

        // Safely handle the "name" field to check for "EL Content Builder Test"
        emailName, ok := itemMap["name"].(string)
        if !ok {
            continue
        }

        // Recursively combine all "content" fields in the itemMap
        combinedHTML := combineAllContent(itemMap)

        // Search for CloudPageID in the combined content
        if cloudPageID != "" && strings.Contains(combinedHTML, cloudPageID) {

            // Safely handle the "id" field which will likely be a float64
            id, ok := itemMap["id"].(float64)
            if !ok {
                log.Printf("Skipping invalid ID type for item: %v", itemMap["id"])
                continue
            }

            // Convert the ID to a string (since json.Number is just a string representation of the number)
            emailID := json.Number(fmt.Sprintf("%.0f", id)) // No decimal places for whole numbers

            // Append the valid Email to the slice
            *allEmails = append(*allEmails, Email{Name: emailName, ID: emailID})
        }
    }
}

// Simple processing function when no cloudPageID is provided
func processSimpleEmails(items []interface{}, allEmails *[]Email) {
    for _, item := range items {
        emailData, err := json.Marshal(item)
        if err != nil {
            log.Printf("Error marshalling item: %v", err)
            continue
        }

        var email Email
        if err := json.Unmarshal(emailData, &email); err != nil {
            log.Printf("Error unmarshalling to Email: %v", err)
            continue
        }

        *allEmails = append(*allEmails, email)
    }
}

// Recursive function to combine all "content" fields in a map or slice
func combineAllContent(data interface{}) string {
    combinedContent := ""

    switch v := data.(type) {
    case map[string]interface{}:
        for key, value := range v {
            if key == "content" {
                if contentStr, ok := value.(string); ok {
                    combinedContent += contentStr
                }
            } else {
                // Recursively handle nested maps
                combinedContent += combineAllContent(value)
            }
        }
    case []interface{}:
        for _, item := range v {
            // Recursively handle slices
            combinedContent += combineAllContent(item)
        }
    }

    return combinedContent
}

// The function that retrieves the email when email name or ID submitted
func GetEmailByIDOrName(emailID string, emailName string) (*Email, error) {
    token, err := auth.GetAccessToken()
    if err != nil {
        return nil, err
    }

    client := &http.Client{}

    // Construct request body based on whether EmailID or EmailName is provided
    var requestBody map[string]interface{}
    if emailID != "" {
        // Case 1: EmailID is provided, fetch by ID
        requestBody = map[string]interface{}{
            "page": map[string]interface{}{
                "page":     1,
                "pageSize": 1, // We're only expecting one result
            },
            "query": map[string]interface{}{
                "property":      "data.email.legacy.legacyId",
                "simpleOperator": "equal",
                "value":         emailID,
            },
            "fields": []string{"name", "id"},
        }
    } else if emailName != "" {
        // Case 2: EmailName is provided, fetch by name
        requestBody = map[string]interface{}{
            "page": map[string]interface{}{
                "page":     1,
                "pageSize": 1, // We're only expecting one result
            },
            "query": map[string]interface{}{
                "property":      "name",
                "simpleOperator": "equal",
                "value":         emailName,
            },
            "fields": []string{"name", "id"},
        }
    } else {
        return nil, fmt.Errorf("either EmailID or EmailName must be provided")
    }

    // Marshal and send the request
    jsonBody, err := json.Marshal(requestBody)
    if err != nil {
        return nil, err
    }

    log.Printf("Request Body: %s", string(jsonBody))

    req, err := http.NewRequest("POST", fmt.Sprintf("%s/asset/v1/content/assets/query", os.Getenv("REST_ENDPOINT")), bytes.NewBuffer(jsonBody))
    if err != nil {
        return nil, err
    }
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        bodyBytes, _ := ioutil.ReadAll(resp.Body)
        return nil, fmt.Errorf("non-200 response code: %d, body: %s", resp.StatusCode, string(bodyBytes))
    }

    var pageResponse PageResponse
    bodyBytes, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    err = json.Unmarshal(bodyBytes, &pageResponse)
    if err != nil {
        return nil, err
    }

    // Process the response and return the email
    items, ok := pageResponse.Items.([]interface{})
    if !ok || len(items) == 0 {
        return nil, fmt.Errorf("no email found with the provided ID or Name")
    }

    // Extract data.email.legacy.legacyId manually
    itemMap, ok := items[0].(map[string]interface{})
    if !ok {
        return nil, fmt.Errorf("failed to parse item")
    }

    // Retrieve the legacyId from the nested structure
    legacyId := ""
    if data, ok := itemMap["data"].(map[string]interface{}); ok {
        if emailData, ok := data["email"].(map[string]interface{}); ok {
            if legacy, ok := emailData["legacy"].(map[string]interface{}); ok {
                if id, ok := legacy["legacyId"].(float64); ok {
                    legacyId = fmt.Sprintf("%.0f", id) // Convert to string
                }
            }
        }
    }

    if legacyId == "" {
        return nil, fmt.Errorf("failed to retrieve legacyId")
    }

    // Create and return the Email struct
    email := Email{
        Name: itemMap["name"].(string),
        ID:   json.Number(legacyId),
    }

    return &email, nil
}

func GetJourneys(deName string, emailID string) ([]Journey, error) {
    token, err := auth.GetAccessToken()
    if err != nil {
        return nil, err
    }

    if deName != "" {
        // Use structured processing with Journey structs when deName is provided
        return getJourneysByDeName(token, deName)
    } else if emailID != "" {
        // Use dynamic processing with emailID filtering
        return getJourneysByEmailID(token, emailID)
    }

    return nil, fmt.Errorf("both deName and emailID cannot be empty")
}

// This function handles the case when deName is provided
func getJourneysByDeName(token, deName string) ([]Journey, error) {
    var allJourneys []Journey
    var mu sync.Mutex

    // Fetch the first page to determine the total count
    firstPageJourneys, totalItems, err := fetchJourneyPageStructured(token, 1, 50)
    if err != nil {
        return nil, err
    }

    allJourneys = append(allJourneys, firstPageJourneys...)

    // Calculate total number of pages
    totalPages := (totalItems + 50 - 1) / 50

    results := make(chan []Journey, totalPages-1)

    // Fetch remaining pages concurrently
    for page := 2; page <= totalPages; page++ {
        go func(page int) {
            journeys, _, err := fetchJourneyPageStructured(token, page, 50)
            if err != nil {
                log.Printf("Error fetching page %d: %v", page, err)
                results <- []Journey{}
                return
            }
            results <- journeys
        }(page)
    }

    // Collect remaining pages' results
    for page := 2; page <= totalPages; page++ {
        journeys := <-results
        mu.Lock()
        allJourneys = append(allJourneys, journeys...)
        mu.Unlock()
    }

    // Process journeys and filter by deName
    return processJourneysAndFetchEventDefinitions(allJourneys, token, deName), nil
}

// This function handles the case when emailID is provided
func getJourneysByEmailID(token, emailID string) ([]Journey, error) {
    var dynamicJourneys []map[string]interface{}
    var mu sync.Mutex

    // Fetch the first page to determine the total count
    firstPageJourneys, totalItems, err := fetchJourneyPageDynamic(token, 1, 50, true)
    if err != nil {
        return nil, err
    }

    dynamicJourneys = append(dynamicJourneys, firstPageJourneys...)

    // Calculate total number of pages
    totalPages := (totalItems + 50 - 1) / 50

    results := make(chan []map[string]interface{}, totalPages-1)

    // Fetch remaining pages concurrently
    for page := 2; page <= totalPages; page++ {
        go func(page int) {
            journeys, _, err := fetchJourneyPageDynamic(token, page, 50, true)
            if err != nil {
                log.Printf("Error fetching page %d: %v", page, err)
                results <- []map[string]interface{}{}
                return
            }
            results <- journeys
        }(page)
    }

    // Collect remaining pages' results
    for page := 2; page <= totalPages; page++ {
        journeys := <-results
        mu.Lock()
        dynamicJourneys = append(dynamicJourneys, journeys...)
        mu.Unlock()
    }

    // Filter dynamic journeys by emailID
    filteredJourneys := filterJourneysByEmailID(dynamicJourneys, emailID)
    var result []Journey
    for _, journeyData := range filteredJourneys {
        var journey Journey
        journeyBytes, _ := json.Marshal(journeyData)
        json.Unmarshal(journeyBytes, &journey)
        result = append(result, journey)
    }

    return result, nil
}

// Structured journey fetch for deName case
func fetchJourneyPageStructured(token string, page, pageSize int) ([]Journey, int, error) {
    client := &http.Client{}
    req, err := http.NewRequest("GET", fmt.Sprintf("%s/interaction/v1/interactions?$page=%d&$pageSize=%d", os.Getenv("REST_ENDPOINT"), page, pageSize), nil)
    if err != nil {
        return nil, 0, err
    }
    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

    resp, err := client.Do(req)
    if err != nil {
        return nil, 0, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, 0, fmt.Errorf("non-200 response: %d", resp.StatusCode)
    }

    var pageResponse PageResponse
    bodyBytes, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return nil, 0, err
    }

    err = json.Unmarshal(bodyBytes, &pageResponse)
    if err != nil {
        return nil, 0, err
    }

    journeys := make([]Journey, len(pageResponse.Items.([]interface{})))
    for i, item := range pageResponse.Items.([]interface{}) {
        journeyData, _ := json.Marshal(item)
        json.Unmarshal(journeyData, &journeys[i])

        journeyMap := item.(map[string]interface{})
        
        // Extract EventDefinitionKey from nested structure
        if defaults, ok := journeyMap["defaults"].(map[string]interface{}); ok {
            if emails, ok := defaults["email"].([]interface{}); ok && len(emails) > 0 {
                if email, ok := emails[0].(string); ok {
                    journeys[i].EventDefinitionKey = extractEventDefinitionKey(email)
                }
            }
        }
    }

    return journeys, pageResponse.Count, nil
}

// Dynamic journey fetch for emailID case
func fetchJourneyPageDynamic(token string, page, pageSize int, includeActivities bool) ([]map[string]interface{}, int, error) {
    client := &http.Client{}
    url := fmt.Sprintf("%s/interaction/v1/interactions?$page=%d&$pageSize=%d", os.Getenv("REST_ENDPOINT"), page, pageSize)
    if includeActivities {
        url += "&extras=activities"
    }

    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil, 0, err
    }
    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

    resp, err := client.Do(req)
    if err != nil {
        return nil, 0, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, 0, fmt.Errorf("non-200 response: %d", resp.StatusCode)
    }

    var pageResponse map[string]interface{}
    bodyBytes, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return nil, 0, err
    }

    err = json.Unmarshal(bodyBytes, &pageResponse)
    if err != nil {
        return nil, 0, err
    }

    totalItems := int(pageResponse["count"].(float64))
    journeys := pageResponse["items"].([]interface{})

    mappedJourneys := make([]map[string]interface{}, len(journeys))
    for i, journey := range journeys {
        mappedJourneys[i] = journey.(map[string]interface{})
    }

    return mappedJourneys, totalItems, nil
}

// processJourneysAndFetchEventDefinitions for deName logic
func processJourneysAndFetchEventDefinitions(journeys []Journey, token, deName string) []Journey {
    var mu sync.Mutex
    var wg sync.WaitGroup
    filteredJourneys := make([]Journey, 0, len(journeys))

    // Semaphore to limit concurrent event definition fetching to 50
    semaphore := make(chan struct{}, 50)

    log.Printf("Starting to process %d journeys", len(journeys))

    totalJourneys := len(journeys)
    if totalJourneys == 0 {
        return filteredJourneys
    }

    // Iterate over all the journeys and add a delay after processing every 50 journeys
    for i, journey := range journeys {
        // Add to the wait group for concurrent execution
        wg.Add(1)
        go func(journey Journey) {
            defer wg.Done()

            // Semaphore: Wait if there are already 50 concurrent tasks
            semaphore <- struct{}{}
            defer func() { <-semaphore }() // Release semaphore after task completion

            // Fetch event definition for each journey
            eventDef, err := fetchEventDefinition(token, journey.EventDefinitionKey, journey.Name)
            if err != nil {
                log.Printf("Error fetching event definition for journey %s: %v", journey.Name, err)
                return
            }

            // Check if the event definition matches the provided Data Extension name
            if eventDef != nil && eventDef.DataExtensionName == deName {
                mu.Lock()
                filteredJourneys = append(filteredJourneys, journey)
                mu.Unlock()
            }
        }(journey)

        // Introduce a delay after processing every 50 journeys to control the flow
        if (i+1)%50 == 0 {
            log.Printf("Processed %d journeys, introducing delay...", i+1)
            time.Sleep(500 * time.Millisecond)
        }
    }

    // Wait for all goroutines to finish
    wg.Wait()

    log.Printf("Finished processing journeys. Total filtered journeys: %d", len(filteredJourneys))
    return filteredJourneys
}


// Filter journeys based on the emailID in the activities
func filterJourneysByEmailID(journeys []map[string]interface{}, emailID string) []map[string]interface{} {
    var filteredJourneys []map[string]interface{}

    for _, journey := range journeys {
        // Check if "activities" exist in the journey map
        if activities, ok := journey["activities"].([]interface{}); ok {
            log.Printf("Found %d activities in journey: %s\n", len(activities), journey["name"])

            for _, activity := range activities {
                activityMap, ok := activity.(map[string]interface{})
                if !ok {
                    log.Println("Error: Activity is not in expected map format.")
                    continue
                }

                log.Printf("Checking activity type: %v\n", activityMap["type"])

                // Look for "EMAILV2" activity type and match the email ID
                if activityMap["type"] == "EMAILV2" {
                    configArgs, ok := activityMap["configurationArguments"].(map[string]interface{})
                    if !ok {
                        log.Println("Error: configurationArguments not found or not in expected format.")
                        continue
                    }

                    triggeredSend, ok := configArgs["triggeredSend"].(map[string]interface{})
                    if !ok {
                        log.Println("Error: triggeredSend not found or not in expected format.")
                        continue
                    }

                    // Check if the emailID matches
                    if emailIDFromActivity, ok := triggeredSend["emailId"].(float64); ok {
                        log.Printf("Found emailId in activity: %.0f", emailIDFromActivity)

                        if fmt.Sprintf("%.0f", emailIDFromActivity) == emailID {
                            log.Printf("Match found for emailID: %s in journey: %s", emailID, journey["name"])
                            filteredJourneys = append(filteredJourneys, journey)
                            break // Stop checking other activities if we found a match
                        }
                    } else {
                        log.Println("Error: emailId not found or not in expected format.")
                    }
                }
            }
        } else {
            log.Printf("No activities found in journey: %s\n", journey["name"])
        }
    }

    return filteredJourneys
}

// fetchEventDefinition first tries to fetch event definition by key, then falls back to name if needed
func fetchEventDefinition(token, eventDefinitionKey, journeyName string) (*EventDefinition, error) {
    // Try fetching by eventDefinitionKey first
    eventDef, err := fetchEventDefinitionByKey(token, eventDefinitionKey)
    if err == nil && eventDef != nil {
        log.Printf("Found event definition by key: %s", eventDefinitionKey)
        return eventDef, nil
    }

    log.Printf("No match found by key: %s, trying by journey name: %s", eventDefinitionKey, journeyName)

    // Remove anything after the first occurrence of '[' or '{' from the journey name
    sanitizedJourneyName := url.QueryEscape(sanitizeJourneyName(journeyName))

    // Try searching by the journey name
    req, err := http.NewRequest("GET", fmt.Sprintf("%s/interaction/v1/eventDefinitions?name=%s", os.Getenv("REST_ENDPOINT"), sanitizedJourneyName), nil)
    if err != nil {
        return nil, err
    }

    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

    log.Printf("Request URL: %s", req.URL)

    // Perform the request
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("non-200 status code returned: %d", resp.StatusCode)
    }

    // Decode response body using PageResponse
    var pageResponse PageResponse
    bodyBytes, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    err = json.Unmarshal(bodyBytes, &pageResponse)
    if err != nil {
        return nil, err
    }

    // Convert Items to []interface{} and then map it to []EventDefinition
    items, ok := pageResponse.Items.([]interface{})
    if !ok {
        return nil, fmt.Errorf("unable to parse items into []interface{}")
    }

    var eventDefs []EventDefinition
    for _, item := range items {
        itemBytes, err := json.Marshal(item)
        if err != nil {
            return nil, err
        }

        var eventDef EventDefinition
        err = json.Unmarshal(itemBytes, &eventDef)
        if err != nil {
            return nil, err
        }
        eventDefs = append(eventDefs, eventDef)
    }

    // If we found results, order them by createdDate and return the newest one
    if len(eventDefs) > 0 {
        log.Printf("Found %d event definitions by name. Sorting by createdDate...", len(eventDefs))

        // Sort by createdDate (descending to get the newest one first)
        sort.Slice(eventDefs, func(i, j int) bool {
            return eventDefs[i].CreatedDate.Time.After(eventDefs[j].CreatedDate.Time)
        })

        return &eventDefs[0], nil
    }

    // If no results are found, return an error
    return nil, fmt.Errorf("no event definition found by either key or name for journey: %s", journeyName)
}

// fetchEventDefinitionByKey fetches the event definition by event key
func fetchEventDefinitionByKey(token, eventDefinitionKey string) (*EventDefinition, error) {
    client := &http.Client{}

    req, err := http.NewRequest("GET", fmt.Sprintf("%s/interaction/v1/eventDefinitions/key:%s", os.Getenv("REST_ENDPOINT"), eventDefinitionKey), nil)
    if err != nil {
        return nil, err
    }

    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("non-200 status code returned: %d", resp.StatusCode)
    }

    var eventDef EventDefinition
    bodyBytes, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    err = json.Unmarshal(bodyBytes, &eventDef)
    if err != nil {
        return nil, err
    }

    return &eventDef, nil
}

// sanitizeJourneyName removes characters starting from '[' or '{'
func sanitizeJourneyName(name string) string {
    specialChars := []string{"[", "{"}

    for _, char := range specialChars {
        if idx := strings.Index(name, char); idx != -1 {
            return name[:idx] // Return substring before the special character
        }
    }

    return name // Return the full name if no special character is found
}

func GetScripts(deName, deCustomerKey, scriptName string) ([]Script, error) {
    token, err := auth.GetAccessToken()
    if err != nil {
        return nil, err
    }

    var allScripts []Script
    var mu sync.Mutex

    // If scriptName is provided, fetch the scripts with a name filter
    if scriptName != "" {
        filteredScripts, err := fetchScriptsByName(token, scriptName)
        if err != nil {
            return nil, err
        }
        return filteredScripts, nil
    }

    // Fetch the first page to determine the total count (for deName and deCustomerKey case)
    firstPageScripts, totalItems, rawItems, err := fetchScriptPage(token, 1, 50)
    if err != nil {
        return nil, err
    }

    // Use the scripts from the first page directly
    allPages := [][]Script{firstPageScripts}
    allRawItems := [][]interface{}{rawItems}

    // Calculate total number of pages
    totalPages := (totalItems + 50 - 1) / 50

    results := make(chan []Script, totalPages-1)
    rawResults := make(chan []interface{}, totalPages-1)

    // Fetch remaining pages concurrently
    for page := 2; page <= totalPages; page++ {
        go func(page int) {
            scripts, _, rawItems, err := fetchScriptPage(token, page, 50)
            if err != nil {
                log.Printf("Error fetching page %d: %v", page, err)
                results <- []Script{}
                rawResults <- []interface{}{}
                return
            }
            results <- scripts
            rawResults <- rawItems
        }(page)
    }

    // Collect remaining pages' results
    for page := 2; page <= totalPages; page++ {
        scripts := <-results
        rawItems := <-rawResults
        allPages = append(allPages, scripts)
        allRawItems = append(allRawItems, rawItems)
    }

    // Process all pages' scripts and filter them using deName and deCustomerKey
    for i, scripts := range allPages {
        rawItems := allRawItems[i]
        filteredScripts := processScripts(scripts, deName, deCustomerKey, rawItems)
        mu.Lock()
        allScripts = append(allScripts, filteredScripts...)
        mu.Unlock()
    }

    return allScripts, nil
}

// Function to fetch scripts by script name
func fetchScriptsByName(token, scriptName string) ([]Script, error) {
    client := &http.Client{}
    // Fetch scripts filtered by name
    req, err := http.NewRequest("GET", fmt.Sprintf("%s/automation/v1/scripts?$filter=name%%20eq%%20%v", os.Getenv("REST_ENDPOINT"), scriptName), nil)
    if err != nil {
        return nil, err
    }
    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("non-200 response: %d", resp.StatusCode)
    }

    var pageResponse PageResponse
    bodyBytes, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    err = json.Unmarshal(bodyBytes, &pageResponse)
    if err != nil {
        return nil, err
    }

    scripts := make([]Script, len(pageResponse.Items.([]interface{})))
    for i, item := range pageResponse.Items.([]interface{}) {
        scriptData, _ := json.Marshal(item)
        json.Unmarshal(scriptData, &scripts[i])
    }

    return scripts, nil
}

// Function to fetch pages for script API responses
func fetchScriptPage(token string, page, pageSize int) ([]Script, int, []interface{}, error) {
    client := &http.Client{}
    req, err := http.NewRequest("GET", fmt.Sprintf("%s/automation/v1/scripts?$page=%d&$pageSize=%d", os.Getenv("REST_ENDPOINT"), page, pageSize), nil)
    if err != nil {
        return nil, 0, nil, err
    }
    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

    resp, err := client.Do(req)
    if err != nil {
        return nil, 0, nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, 0, nil, fmt.Errorf("non-200 response: %d", resp.StatusCode)
    }

    var pageResponse PageResponse
    bodyBytes, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return nil, 0, nil, err
    }

    err = json.Unmarshal(bodyBytes, &pageResponse)
    if err != nil {
        return nil, 0, nil, err
    }

    scripts := make([]Script, len(pageResponse.Items.([]interface{})))
    for i, item := range pageResponse.Items.([]interface{}) {
        scriptData, _ := json.Marshal(item)
        json.Unmarshal(scriptData, &scripts[i])
    }

    return scripts, pageResponse.Count, pageResponse.Items.([]interface{}), nil
}

// Function to fetch pages for script API responses
func processScripts(scripts []Script, deName, deCustomerKey string, rawItems []interface{}) []Script {
    var filteredScripts []Script

    for i, script := range scripts {
        // Access the raw item corresponding to this script
        itemMap, ok := rawItems[i].(map[string]interface{})
        if !ok {
            continue
        }

        // Check if the "script" field exists and is a string
        scriptContent, ok := itemMap["script"].(string)
        if !ok {
            continue
        }

        // Check if the "script" field contains the deName and deCustomerKey
        if strings.Contains(scriptContent, deName) || strings.Contains(scriptContent, deCustomerKey) {
            filteredScripts = append(filteredScripts, script)
        }
    }

    return filteredScripts
}

func GetCloudPages(deName, deCustomerKey, cloudPageID string) ([]CloudPage, error) {
    token, err := auth.GetAccessToken()
    if err != nil {
        return nil, err
    }

    var allCloudPages []CloudPage
    var mu sync.Mutex

    // Fetch the first page to determine the total count
    firstPageCloudPages, totalItems, rawItems, err := fetchCloudPage(token, 1, 50)
    if err != nil {
        return nil, err
    }

    // Use the CloudPages from the first page directly
    allPages := [][]CloudPage{firstPageCloudPages}
    allRawItems := [][]interface{}{rawItems}

    // Calculate total number of pages
    totalPages := (totalItems + 50 - 1) / 50

    results := make(chan []CloudPage, totalPages-1)
    rawResults := make(chan []interface{}, totalPages-1)

    // Fetch remaining pages concurrently
    for page := 2; page <= totalPages; page++ {
        go func(page int) {
            cloudPages, _, rawItems, err := fetchCloudPage(token, page, 50)
            if err != nil {
                log.Printf("Error fetching page %d: %v", page, err)
                results <- []CloudPage{}
                rawResults <- []interface{}{}
                return
            }
            results <- cloudPages
            rawResults <- rawItems
        }(page)
    }

    // Collect remaining pages' results
    for page := 2; page <= totalPages; page++ {
        cloudPages := <-results
        rawItems := <-rawResults
        allPages = append(allPages, cloudPages)
        allRawItems = append(allRawItems, rawItems)
    }

    // Process all pages' CloudPages and filter them
    for i, cloudPages := range allPages {
        rawItems := allRawItems[i]
        filteredCloudPages := processCloudPages(cloudPages, deName, deCustomerKey, cloudPageID, rawItems)
        mu.Lock()
        allCloudPages = append(allCloudPages, filteredCloudPages...)
        mu.Unlock()
    }

    return allCloudPages, nil
}

func fetchCloudPage(token string, page, pageSize int) ([]CloudPage, int, []interface{}, error) {
    client := &http.Client{}

    // Generate the request body for the POST request
    requestBody, err := generateCloudPageRequestBody(page, pageSize)
    if err != nil {
        return nil, 0, nil, err
    }

    req, err := http.NewRequest(
        "POST",
        fmt.Sprintf("%s/asset/v1/content/assets/query", os.Getenv("REST_ENDPOINT")),
        bytes.NewBuffer(requestBody),
    )
    if err != nil {
        return nil, 0, nil, err
    }
    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
    req.Header.Set("Content-Type", "application/json")

    resp, err := client.Do(req)
    if err != nil {
        return nil, 0, nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, 0, nil, fmt.Errorf("non-200 response: %d", resp.StatusCode)
    }

    var pageResponse PageResponse
    bodyBytes, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return nil, 0, nil, err
    }

    err = json.Unmarshal(bodyBytes, &pageResponse)
    if err != nil {
        return nil, 0, nil, err
    }

    cloudPages := make([]CloudPage, len(pageResponse.Items.([]interface{})))
    for i, item := range pageResponse.Items.([]interface{}) {
        cloudPageData, _ := json.Marshal(item)
        json.Unmarshal(cloudPageData, &cloudPages[i])
    }

    return cloudPages, pageResponse.Count, pageResponse.Items.([]interface{}), nil
}

func generateCloudPageRequestBody(page int, pageSize int) ([]byte, error) {
    requestBody := map[string]interface{}{
        "page": map[string]interface{}{
            "page":     page,
            "pageSize": pageSize,
        },
        "query": map[string]interface{}{
            "leftOperand": map[string]interface{}{
                "property":      "content",
                "simpleOperator": "isNotNull",
            },
            "logicalOperator": "AND",
            "rightOperand": map[string]interface{}{
                "property":      "assetType.name",
                "simpleOperator": "equal",
                "value":         "webpage",
            },
        },
        "sort": []map[string]interface{}{
            {"property": "id", "direction": "ASC"},
        },
        "fields": []string{"name", "views"},
    }

    return json.Marshal(requestBody)
}

// Function to process CloudPages with the "combine all content" logic
func processCloudPages(cloudPages []CloudPage, deName, deCustomerKey, cloudPageID string, rawItems []interface{}) []CloudPage {
    var filteredCloudPages []CloudPage

    for i, cloudPage := range cloudPages {
        // Access the raw item corresponding to this CloudPage
        itemMap, ok := rawItems[i].(map[string]interface{})
        if !ok {
            continue
        }

        // Combine all "content" fields in the itemMap
        combinedHTML := combineAllContent(itemMap)

        // If CloudPageID is provided, search for "CloudPagesURL(CloudPageID)"
        if cloudPageID != "" {
            if strings.Contains(combinedHTML, cloudPageID) {
                filteredCloudPages = append(filteredCloudPages, cloudPage)
            }
        } else {
            // If CloudPageID is not provided, search using deName or deCustomerKey
            if strings.Contains(combinedHTML, deName) || strings.Contains(combinedHTML, deCustomerKey) {
                filteredCloudPages = append(filteredCloudPages, cloudPage)
            }
        }
    }

    return filteredCloudPages
}

func extractEventDefinitionKey(defaultEmail string) string {
    start := strings.Index(defaultEmail, "{{Event.")
    if start == -1 {
        return ""
    }
    start += len("{{Event.")

    end := strings.Index(defaultEmail[start:], ".")
    if end == -1 {
        return ""
    }

    return defaultEmail[start : start+end]
}

func GetTriggeredSends(emailID string) ([]TriggeredSendDefinition, error) {
    token, err := auth.GetAccessToken()
    if err != nil {
        return nil, err
    }

    // Define the filter for TriggeredSendStatus and Email.ID
    filter := fmt.Sprintf(`
        <Filter xsi:type="par:ComplexFilterPart" xmlns:par="http://exacttarget.com/wsdl/partnerAPI">
            <LeftOperand xsi:type="par:SimpleFilterPart">
                <Property>TriggeredSendStatus</Property>
                <SimpleOperator>notEquals</SimpleOperator>
                <Value>Deleted</Value>
            </LeftOperand>
            <LogicalOperator>AND</LogicalOperator>
            <RightOperand xsi:type="par:SimpleFilterPart">
                <Property>Email.ID</Property>
                <SimpleOperator>equals</SimpleOperator>
                <Value>%s</Value>
            </RightOperand>
        </Filter>`, emailID)

    // Prepare the SOAP request body for TriggeredSendDefinition
    requestBody := fmt.Sprintf(xmlTemplate, os.Getenv("SOAP_ENDPOINT"), token, "TriggeredSendDefinition", `
        <Properties>Name</Properties>
    `, filter)

    // Perform the SOAP request
    resp, err := soapRequest(requestBody)
    if err != nil {
        return nil, err
    }

    // Parse the response into the appropriate structure
    var response struct {
        Results []TriggeredSendDefinition `xml:"Body>RetrieveResponseMsg>Results"`
    }

    err = xml.Unmarshal(resp, &response)
    if err != nil {
        return nil, err
    }

    // Regular expression to filter out names with hash values
    hashRegex := regexp.MustCompile(`-\s*[a-f0-9]{32}$`)

    // Filter out TriggeredSendDefinitions with a hash in their name
    var filteredResults []TriggeredSendDefinition
    for _, result := range response.Results {
        if !hashRegex.MatchString(result.Name) {
            filteredResults = append(filteredResults, result)
        }
    }

    log.Printf("Filtered out results with hash: %d remaining", len(filteredResults))
    return filteredResults, nil
}

func GetInitiatedEmails(deObjectID, emailID string) ([]EmailSendDefinition, error) {
    var emailSendDefinitions []EmailSendDefinition

    token, err := auth.GetAccessToken()
    if err != nil {
        return nil, err
    }

    // SOAP request for retrieving EmailSendDefinition objects
    requestBody := fmt.Sprintf(xmlTemplate, os.Getenv("SOAP_ENDPOINT"), token, "EmailSendDefinition", `
        <Properties>Name</Properties>
        <Properties>ObjectID</Properties>
        <Properties>SendDefinitionList</Properties>
        <Properties>Email.ID</Properties>
    `, "")

    resp, err := soapRequest(requestBody)
    if err != nil {
        return nil, err
    }

    var response struct {
        Results []struct {
            Name                string `xml:"Name"`
            ObjectID            string `xml:"ObjectID"`
            SendDefinitionList  struct {
                CustomObjectID string `xml:"CustomObjectID"`
                List           struct {
                    ID string `xml:"ID"`
                } `xml:"List"`
            } `xml:"SendDefinitionList"`
            Email struct {
                ID string `xml:"ID"`
            } `xml:"Email"`
        } `xml:"Body>RetrieveResponseMsg>Results"`
    }

    err = xml.Unmarshal(resp, &response)
    if err != nil {
        return nil, err
    }

    // Regular expression to match a pattern like "_1234567890", i.e., at least 10 digits after an underscore
    re := regexp.MustCompile(`_([0-9]{10,})$`)

    // Filter and collect valid EmailSendDefinitions
    for _, result := range response.Results {
        // Exclude the result if CustomObjectID and Email.ID are both empty
        if result.SendDefinitionList.CustomObjectID == "" && result.Email.ID == "" {
            continue // Skip this result if CustomObjectID and Email.ID are empty
        }

        // Check if the result.Name contains an underscore followed by at least 10 digits
        if re.MatchString(result.Name) {
            continue // Skip if the name ends with at least 10 digits after an underscore
        }

        // Process results based on whether deObjectID or emailID is provided
        if deObjectID != "" {
            if result.SendDefinitionList.CustomObjectID == deObjectID {
                // Append valid EmailSendDefinition to the slice if deObjectID matches
                emailSendDefinitions = append(emailSendDefinitions, EmailSendDefinition{
                    Name:           result.Name,
                    ObjectID:       result.ObjectID,
                    CustomObjectID: result.SendDefinitionList.CustomObjectID,
                    EmailID:        result.Email.ID,
                })
            }
        } else if emailID != "" {
            if result.Email.ID == emailID {
                // Append valid EmailSendDefinition to the slice if emailID matches
                emailSendDefinitions = append(emailSendDefinitions, EmailSendDefinition{
                    Name:           result.Name,
                    ObjectID:       result.ObjectID,
                    CustomObjectID: result.SendDefinitionList.CustomObjectID,
                    EmailID:        result.Email.ID,
                })
            }
        }
    }

    return emailSendDefinitions, nil
}

func GetQueries(filter string) ([]QueryDefinition, error) {
    token, err := auth.GetAccessToken()
    if err != nil {
        return nil, err
    }

    requestBody := fmt.Sprintf(xmlTemplate, os.Getenv("SOAP_ENDPOINT"), token, "QueryDefinition", `
        <Properties>Name</Properties>
        <Properties>ObjectID</Properties>
    `, filter)

    resp, err := soapRequest(requestBody)
    if err != nil {
        return nil, err
    }

    var response struct {
        Results []QueryDefinition `xml:"Body>RetrieveResponseMsg>Results"`
    }
    err = xml.Unmarshal(resp, &response)
    if err != nil {
        return nil, err
    }

    return response.Results, nil
}

func GetImports(filter string) ([]ImportDefinition, error) {
    token, err := auth.GetAccessToken() // Get the OAuth token or session token
    if err != nil {
        return nil, err
    }

    // Create the SOAP request for ImportDefinition
    requestBody := fmt.Sprintf(xmlTemplate, os.Getenv("SOAP_ENDPOINT"), token, "ImportDefinition", `
        <Properties>Name</Properties>
        <Properties>ObjectID</Properties>
    `, filter)

    // Send the SOAP request
    resp, err := soapRequest(requestBody)
    if err != nil {
        return nil, err
    }

    // Parse the response
    var response struct {
        Results []ImportDefinition `xml:"Body>RetrieveResponseMsg>Results"`
    }
    err = xml.Unmarshal(resp, &response)
    if err != nil {
        return nil, err
    }

    // Regular expression to match UUID-like patterns
    uuidPattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

    // Filter out results where Name matches the UUID pattern
    var validResults []ImportDefinition
    for _, result := range response.Results {
        if !uuidPattern.MatchString(result.Name) {
            validResults = append(validResults, result)
        }
    }

    // Return the filtered list of ImportDefinition objects
    return validResults, nil
}

func GetFilters(filter string) ([]FilterActivity, error) {
    token, err := auth.GetAccessToken() // Get the OAuth token or session token
    if err != nil {
        return nil, err
    }

    // Create the SOAP request for FilterActivity
    requestBody := fmt.Sprintf(xmlTemplate, os.Getenv("SOAP_ENDPOINT"), token, "FilterActivity", `
        <Properties>Name</Properties>
        <Properties>ObjectID</Properties>
    `, filter)

    // Send the SOAP request
    resp, err := soapRequest(requestBody)
    if err != nil {
        return nil, err
    }

    // Parse the response
    var response struct {
        Results []FilterActivity `xml:"Body>RetrieveResponseMsg>Results"`
    }
    err = xml.Unmarshal(resp, &response)
    if err != nil {
        return nil, err
    }

    // Regular expression to match UUID-like patterns
    uuidPattern := regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)

    // Filter out results where Name contains UUID-like patterns or starts with "Activity for result group"
    var validResults []FilterActivity
    for _, result := range response.Results {
        if !uuidPattern.MatchString(result.Name) && !strings.HasPrefix(result.Name, "Activity for result group") {
            validResults = append(validResults, result)
        }
    }

    // Return the filtered list of FilterActivity objects
    return validResults, nil
}

// GetDataExtensionPath retrieves the folder path for a Data Extension by recursively finding parent folders
func GetDataExtensionPath(categoryID string, shared bool) (string, error) {
    var pathElements []string
    currentID := categoryID

    for {
        // Retrieve the folder information for the current folder ID
        folder, err := getFolderByID(currentID, shared) // Pass shared flag
        if err != nil {
            return "", fmt.Errorf("failed to retrieve folder with ID %s: %v", currentID, err)
        }

        // Check if the current folder is either "Shared Data Extensions" or "Data Extensions"
        if folder.Name == "Shared Data Extensions" || folder.Name == "Data Extensions" {
            return folder.Name, nil
        }

        // Add the current folder name to the path
        pathElements = append([]string{folder.Name}, pathElements...)

        // Now check if the parent folder is "Shared Data Extensions" or "Data Extensions"
        if folder.ParentName == "Shared Data Extensions" || folder.ParentName == "Data Extensions" {
            // Prepend the parent folder name and return the path
            path := folder.ParentName + " > " + strings.Join(pathElements, " > ")
            return path, nil
        }

        // Move to the parent folder for the next iteration
        currentID = folder.ParentID
    }

    // This point should never be reached as the loop will either return or continue
}

// Helper function to retrieve folder information by ID using a SOAP request
func getFolderByID(folderID string, shared bool) (*Folder, error) {
    token, err := auth.GetAccessToken()
    if err != nil {
        return nil, err
    }

    var filter string
    if shared {
        // Use shared data extension filter logic
        filter = fmt.Sprintf(`
            <QueryAllAccounts>true</QueryAllAccounts>
            <Filter xsi:type="ComplexFilterPart">
                <LeftOperand xsi:type="SimpleFilterPart">
                    <Property>ContentType</Property>
                    <SimpleOperator>equals</SimpleOperator>
                    <Value>shared_dataextension</Value>
                </LeftOperand>
                <LogicalOperator>AND</LogicalOperator>
                <RightOperand xsi:type="SimpleFilterPart">
                    <Property>ID</Property>
                    <SimpleOperator>equals</SimpleOperator>
                    <Value>%s</Value>
                </RightOperand>
            </Filter>`, folderID)
    } else {
        // Use non-shared data extension filter logic
        filter = fmt.Sprintf(`
            <Filter xsi:type="SimpleFilterPart">
                <Property>ID</Property>
                <SimpleOperator>equals</SimpleOperator>
                <Value>%s</Value>
            </Filter>`, folderID)
    }

    requestBody := fmt.Sprintf(xmlTemplate, os.Getenv("SOAP_ENDPOINT"), token, "DataFolder", `
        <Properties>ID</Properties>
        <Properties>Name</Properties>
        <Properties>ParentFolder.ID</Properties>
        <Properties>ParentFolder.Name</Properties>
    `, filter)

    resp, err := soapRequest(requestBody)
    if err != nil {
        return nil, err
    }

    // Parse the SOAP response to extract the folder information
    var response struct {
        Results []Folder `xml:"Body>RetrieveResponseMsg>Results"`
    }

    err = xml.Unmarshal(resp, &response)
    if err != nil {
        return nil, err
    }

    // Ensure that at least one result was returned
    if len(response.Results) == 0 {
        return nil, fmt.Errorf("folder with ID %s not found", folderID)
    }

    return &response.Results[0], nil
}

// --- Utility Functions ---

// XML Template to use for all SOAP requests
var xmlTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope" xmlns:a="http://schemas.xmlsoap.org/ws/2004/08/addressing" xmlns:u="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd">
    <s:Header>
        <a:Action s:mustUnderstand="1">Retrieve</a:Action>
        <a:To s:mustUnderstand="1">%s</a:To>
        <fueloauth xmlns="http://exacttarget.com">%s</fueloauth>
    </s:Header>
    <s:Body xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema">
        <RetrieveRequestMsg xmlns="http://exacttarget.com/wsdl/partnerAPI">
            <RetrieveRequest>
                <ObjectType>%s</ObjectType>
                %s
                %s <!-- Filter placeholder -->
            </RetrieveRequest>
        </RetrieveRequestMsg>
    </s:Body>
</s:Envelope>`

// soapRequest function to do SOAP calls
func soapRequest(body string) ([]byte, error) {

    // Create and send the SOAP request
    req, err := http.NewRequest("POST", os.Getenv("SOAP_ENDPOINT"), strings.NewReader(body))
    if err != nil {
        return nil, err
    }

    req.Header.Set("Content-Type", "text/xml")
    client := &http.Client{Timeout: 30 * time.Second}

    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    // Check for successful status code
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("error: received non-200 response code: %d", resp.StatusCode)
    }

    // Read and return the response body
    responseBody, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    return responseBody, nil
}

