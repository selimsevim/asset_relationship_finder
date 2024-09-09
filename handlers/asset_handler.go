package handlers

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "sync"
    "time"

    "github.com/patrickmn/go-cache"
    "asset_relationship_finder/services"
)

// Global cache variable to cache Data Extension form Responses
var deCache = cache.New(5*time.Minute, 10*time.Minute)

// ---- Request and Response Structs ----

// Request and Response Structs for Data Extensions
type DataExtensionRequest struct {
    Name         string            `json:"name"`
    CustomerKey  string            `json:"customerKey"`
    UserSelection map[string]bool  `json:"userselection"`
}

type DataExtensionResponse struct {
    Path                     string                         `json:"dePath"`
    Name                     string                         `json:"name"`  
    QueriesTargeting         []services.QueryDefinition     `json:"queriesTargeting"`  
    QueriesIncluding         []services.QueryDefinition     `json:"queriesIncluding"`  
    ImportsTargeting         []services.ImportDefinition    `json:"importsTargeting"`  
    FiltersTargeting         []services.FilterActivity      `json:"filtersTargeting"`  
    ContentEmailsIncluding   []services.Email               `json:"contentEmailsIncluding"`  
    InitiatedEmailsTargeting []services.EmailSendDefinition `json:"initiatedEmailsTargeting"` 
    JourneysUsingDE          []services.Journey             `json:"journeysUsingDE"`  
    ScriptsIncluding         []services.Script              `json:"scriptsIncluding"`  
    PagesIncluding           []services.CloudPage           `json:"pagesIncluding"`  
}

// Request and Response Structs for Automation Activities
type AutomationActivityRequest struct {
    Name string `json:"name"`
    Type string `json:"activityType"`
}

type AutomationActivityResponse struct {
    Automations []services.Automation `json:"automations"`
}

// Request and Response Structs for CloudPages
type CloudPageRequest struct {
    CloudPageID   string            `json:"cloudPageID"`
    UserSelection map[string]bool   `json:"userselection"`
}

type CloudPageResponse struct {
    EmailsUsingCloudPage     []services.Email      `json:"emailsUsingCloudPage"`
    CloudPagesUsingCloudPage []services.CloudPage  `json:"cloudPagesUsingCloudPage"`
}

// Request and Response Structs for CloudPages
type EmailRequest struct {
    ID      string                 `json:"ID"`
    Name    string                 `json:"Name"`
    UserSelection map[string]bool  `json:"userSelection"`
}

type EmailResponse struct {
    JourneysUsingEmail          []services.Journey                 `json:"journeysUsingEmail"`
    InitiatedEmailsUsing        []services.EmailSendDefinition     `json:"initiatedEmailsUsing"`
    TriggeredSends              []services.TriggeredSendDefinition `json:"triggeredSends"`
    Name                        string                             `json:"name"`
}

// ---- Task Channels Structs ----

// Task Channels for Data Extensions
type DataExtensionTaskChannels struct {
    PathChan                        chan string
    QueriesTargetingChan            chan []services.QueryDefinition
    QueriesIncludingChan            chan []services.QueryDefinition
    ImportsTargetingChan            chan []services.ImportDefinition
    FiltersTargetingChan            chan []services.FilterActivity
    ContentEmailsIncludingChan      chan []services.Email 
    InitiatedEmailsTargetingChan    chan []services.EmailSendDefinition
    JourneysUsingDEChan             chan []services.Journey
    ScriptsIncludingChan            chan []services.Script
    PagesIncludingChan              chan []services.CloudPage
    ErrorChan                       chan error
}

// CloudPage task channels struct
type CloudPageTaskChannels struct {
    EmailsUsingChan      chan []services.Email
    CloudPagesUsingChan  chan []services.CloudPage
    ErrorChan            chan error
}

// Task Channels for Emails
type EmailTaskChannels struct {
    JourneysUsingEmailChan           chan []services.Journey
    InitiatedEmailsUsingChan         chan []services.EmailSendDefinition
    TriggeredSendsChan               chan []services.TriggeredSendDefinition
    ErrorChan                        chan error
}

// ---- Setup and close channels ----

func setupDataExtensionChannels() DataExtensionTaskChannels {
    return DataExtensionTaskChannels{
        PathChan:                     make(chan string, 1),
        QueriesTargetingChan:         make(chan []services.QueryDefinition, 1),
        QueriesIncludingChan:         make(chan []services.QueryDefinition, 1),
        ImportsTargetingChan:         make(chan []services.ImportDefinition, 1),
        FiltersTargetingChan:         make(chan []services.FilterActivity, 1),
        ContentEmailsIncludingChan:   make(chan []services.Email, 1),
        InitiatedEmailsTargetingChan: make(chan []services.EmailSendDefinition, 1),
        JourneysUsingDEChan:          make(chan []services.Journey, 1),
        ScriptsIncludingChan:         make(chan []services.Script, 1),
        PagesIncludingChan:           make(chan []services.CloudPage, 1),
        ErrorChan:                    make(chan error, 1),
    }
}

func closeDataExtensionChannels(channels DataExtensionTaskChannels) {
    close(channels.PathChan)
    close(channels.QueriesTargetingChan)
    close(channels.QueriesIncludingChan)
    close(channels.ImportsTargetingChan)
    close(channels.FiltersTargetingChan)
    close(channels.ContentEmailsIncludingChan)
    close(channels.InitiatedEmailsTargetingChan)
    close(channels.JourneysUsingDEChan)
    close(channels.ScriptsIncludingChan)
    close(channels.PagesIncludingChan)
    close(channels.ErrorChan)
    log.Println("All Data Extension channels closed.")
}

// Setup channels for CloudPage tasks
func setupCloudPageChannels() CloudPageTaskChannels {
    return CloudPageTaskChannels{
        EmailsUsingChan:     make(chan []services.Email, 1),
        CloudPagesUsingChan: make(chan []services.CloudPage, 1),
        ErrorChan:           make(chan error, 1),
    }
}

// Close channels for CloudPage tasks
func closeCloudPageChannels(channels CloudPageTaskChannels) {
    close(channels.EmailsUsingChan)
    close(channels.CloudPagesUsingChan)
    close(channels.ErrorChan)
    log.Println("All CloudPage channels closed.")
}

// Setup channels for email tasks.
func setupEmailTaskChannels() EmailTaskChannels {
    return EmailTaskChannels{
        JourneysUsingEmailChan:   make(chan []services.Journey, 1),
        InitiatedEmailsUsingChan: make(chan []services.EmailSendDefinition, 1),
        TriggeredSendsChan:       make(chan []services.TriggeredSendDefinition, 1),
        ErrorChan:                make(chan error, 1),
    }
}

// Close channels once all tasks are complete.
func closeEmailTaskChannels(channels EmailTaskChannels) {
    close(channels.JourneysUsingEmailChan)
    close(channels.InitiatedEmailsUsingChan)
    close(channels.TriggeredSendsChan)
    close(channels.ErrorChan)
    log.Println("All Email channels closed.")
}


// ---- Utility and Helper Functions ----

func handleError(w http.ResponseWriter, message string, statusCode int) {
    http.Error(w, message, statusCode)
}

func sendJSONResponse(w http.ResponseWriter, response interface{}) {
    log.Println("All tasks completed successfully.")
    w.Header().Set("Content-Type", "application/json")
    if err := json.NewEncoder(w).Encode(response); err != nil {
        http.Error(w, "Failed to encode response", http.StatusInternalServerError)
        log.Printf("Error encoding response: %v", err)
    }
}

// Helper function to get cookie values
func getCookieValue(r *http.Request, name string) (string, error) {
    cookie, err := r.Cookie(name)
    if err != nil {
        return "", err
    }
    return cookie.Value, nil
}

// ---- Data Extension Related Functions and Handlers ----

func DataExtensionDetail(w http.ResponseWriter, r *http.Request) {
    // Set up context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer func() {
        log.Println("Context canceled")
        cancel()
    }()


    // 1. Get entID from cookies
    entID, err := getCookieValue(r, "entID")
    if err != nil {
        handleError(w, "entID not found", http.StatusUnauthorized)
        return
    }

    // 2. Parse incoming request to get Data Extension Name and User Selections
    var req DataExtensionRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        handleError(w, "invalid request payload1", http.StatusBadRequest)
        return
    }

    // 3. Build filter based on the request
    filter := buildFilterFromRequest(req)
    dataExtensions, isShared, err := fetchDataExtensions(filter, entID, req)
    if err != nil || len(dataExtensions) == 0 {
        handleError(w, "No Data Extension found with this CustomerKey or Name", http.StatusInternalServerError)
        return
    }

    // 4. Retrieve details from the found Data Extension
    dataExtension := dataExtensions[0]
    deCustomerKey := dataExtension.CustomerKey
    deCategoryID := dataExtension.CategoryID
    deName := dataExtension.Name
    deObjectID := dataExtension.ObjectID // We'll need this for ImportDefinition filter

    // 5. Setup channels and WaitGroup
    var wg sync.WaitGroup
    channels := setupDataExtensionChannels()

    // 6. Start tasks based on user selection
    startTasks(ctx, &wg, req.UserSelection, deCategoryID, deName, deCustomerKey, deObjectID, isShared, channels)

    // 7. Wait for all tasks to finish
    go func() {
        wg.Wait()
        closeDataExtensionChannels(channels)
    }()

    // 8. Collect response
    response := collectResponse(ctx, channels, deObjectID, deName, w)

    // 9. Send the final response
    sendJSONResponse(w, response)
}

// Based on the input user enters, build the filter to retrieve the Data Extension
func buildFilterFromRequest(req DataExtensionRequest) string {
    if req.Name != "" {
        return fmt.Sprintf(`
            <Filter xsi:type="SimpleFilterPart">
                <Property>Name</Property>
                <SimpleOperator>equals</SimpleOperator>
                <Value>%s</Value>
            </Filter>`, req.Name)
    } else if req.CustomerKey != "" {
        return fmt.Sprintf(`
            <Filter xsi:type="SimpleFilterPart">
                <Property>CustomerKey</Property>
                <SimpleOperator>equals</SimpleOperator>
                <Value>%s</Value>
            </Filter>`, req.CustomerKey)
    }
    return ""
}

// Fetch the Data Extension checking regular data extensions first and then shared ones
func fetchDataExtensions(filter string, entID string, req DataExtensionRequest) ([]services.DataExtension, bool, error) {
    
    // Fetch non-shared data extensions first
    dataExtensions, err := services.GetDataExtensions(filter)
    if err != nil {
        return nil, false, err
    }

    // If we found data extensions in the non-shared context, return them
    if len(dataExtensions) > 0 {
        return dataExtensions, false, nil // Not shared
    }

    // Try fetching shared data extensions
    sharedFilter := fmt.Sprintf(`
        <QueryAllAccounts>true</QueryAllAccounts>
        <Filter xsi:type="ComplexFilterPart">
            <LeftOperand xsi:type="SimpleFilterPart">
                <Property>Client.ID</Property>
                <SimpleOperator>equals</SimpleOperator>
                <Value>%s</Value>
            </LeftOperand>
            <RightOperand xsi:type="SimpleFilterPart">
                <Property>%s</Property>
                <SimpleOperator>equals</SimpleOperator>
                <Value>%s</Value>
            </RightOperand>
            <LogicalOperator>AND</LogicalOperator>
        </Filter>`, entID, 
        func() string {
            if req.Name != "" {
                return "Name"
            }
            return "CustomerKey"
        }(),
        func() string {
            if req.Name != "" {
                return req.Name
            }
            return req.CustomerKey
        }())

    sharedDataExtensions, err := services.GetDataExtensions(sharedFilter)
    if err != nil {
        return nil, false, err
    }

    // If we found shared data extensions, return them
    if len(sharedDataExtensions) > 0 {
        return sharedDataExtensions, true, nil // Shared
    }

    // No data extensions found
    return nil, false, nil
}

// startTasks iterates through the user's selected fields (e.g., queries, imports) for a Data Extension
func startTasks(ctx context.Context, wg *sync.WaitGroup, userSelection map[string]bool, categoryID, deName, deCustomerKey, deObjectID string, isShared bool, channels DataExtensionTaskChannels) {
    cachedResponse, found := deCache.Get(deObjectID)
    var cachedData DataExtensionResponse
    if found {
        cachedData = cachedResponse.(DataExtensionResponse)
        log.Printf("Cache hit for objectID: %s", deObjectID)
    } else {
        log.Printf("Cache miss for objectID: %s", deObjectID)
    }

    // Define a map between selection keys and their respective tasks
    taskMap := map[string]struct {
        cachedDataField interface{}
        isCached        bool
        fetchFunc       func() (interface{}, error)
        channel         interface{}
    }{
        "dePath":                     {cachedData.Path, cachedData.Path != "", func() (interface{}, error) { return fetchPath(categoryID, isShared) }, channels.PathChan},
        "queriesTargeting":           {cachedData.QueriesTargeting, len(cachedData.QueriesTargeting) > 0, func() (interface{}, error) { return fetchQueriesTargeting(deName) }, channels.QueriesTargetingChan},
        "queriesIncluding":           {cachedData.QueriesIncluding, len(cachedData.QueriesIncluding) > 0, func() (interface{}, error) { return fetchQueriesIncluding(deName) }, channels.QueriesIncludingChan},
        "importsTargeting":           {cachedData.ImportsTargeting, len(cachedData.ImportsTargeting) > 0, func() (interface{}, error) { return fetchImportsForDE(deObjectID) }, channels.ImportsTargetingChan},
        "filtersTargeting":           {cachedData.FiltersTargeting, len(cachedData.FiltersTargeting) > 0, func() (interface{}, error) { return fetchFilters(deObjectID) }, channels.FiltersTargetingChan},
        "contentEmailsIncluding":     {cachedData.ContentEmailsIncluding, len(cachedData.ContentEmailsIncluding) > 0, func() (interface{}, error) { return services.GetEmails(deName, "") }, channels.ContentEmailsIncludingChan},
        "initiatedEmailsTargeting":   {cachedData.InitiatedEmailsTargeting, len(cachedData.InitiatedEmailsTargeting) > 0, func() (interface{}, error) { return services.GetInitiatedEmails(deObjectID, "") }, channels.InitiatedEmailsTargetingChan},
        "journeysUsingDE":            {cachedData.JourneysUsingDE, len(cachedData.JourneysUsingDE) > 0, func() (interface{}, error) { return services.GetJourneys(deName, "") }, channels.JourneysUsingDEChan},
        "scriptsIncluding":           {cachedData.ScriptsIncluding, len(cachedData.ScriptsIncluding) > 0, func() (interface{}, error) { return services.GetScripts(deName, deCustomerKey, "") }, channels.ScriptsIncludingChan},
        "pagesIncluding":             {cachedData.PagesIncluding, len(cachedData.PagesIncluding) > 0, func() (interface{}, error) { return services.GetCloudPages(deName, deCustomerKey, "") }, channels.PagesIncludingChan},
    }

    // Iterate through userSelection and start tasks for fields that are true
    for key, selected := range userSelection {
        if selected {
            task, exists := taskMap[key]
            if exists {
                wg.Add(1)  // Add to WaitGroup for each task
                go processTask(ctx, wg, task.isCached, task.cachedDataField, task.fetchFunc, task.channel, key)  // Process each task concurrently
            }
        }
    }
}


// Function to handle task processing, including both cached data and dynamic fetching
func processTask(ctx context.Context, wg *sync.WaitGroup, isCached bool, cachedData interface{}, fetchFunc func() (interface{}, error), channel interface{}, taskName string) {
    defer wg.Done() // Mark this goroutine as done once the task finishes

    if isCached {
        log.Printf("Using cached data for task: %s", taskName) // Log only when the cached data for this task is used
        sendDataToChannel(channel, cachedData)
    } else {
        log.Printf("Starting task to fetch data for task: %s", taskName) // Log when starting task
        runTaskWithChannel(ctx, fetchFunc, channel)
    }
}

// Function to run the task and send results to the appropriate channel
func runTaskWithChannel(ctx context.Context, fetchFunc func() (interface{}, error), channel interface{}) {
    result, err := fetchFunc()  // Execute the task
    if err != nil {
        log.Printf("Error fetching data: %v", err)
        return
    }
    sendDataToChannel(channel, result)
}

// Function to send data to the appropriate channel based on its type
func sendDataToChannel(channel interface{}, data interface{}) {
    switch ch := channel.(type) {
    case chan string:
        if strData, ok := data.(string); ok {
            ch <- strData
        }
    case chan []services.QueryDefinition:
        if queryData, ok := data.([]services.QueryDefinition); ok {
            ch <- queryData
        }
    case chan []services.ImportDefinition:
        if importData, ok := data.([]services.ImportDefinition); ok {
            ch <- importData
        }
    case chan []services.FilterActivity:
        if filterData, ok := data.([]services.FilterActivity); ok {
            ch <- filterData
        }
    case chan []services.Email:
        if emailData, ok := data.([]services.Email); ok {
            ch <- emailData
        }
    case chan []services.EmailSendDefinition:
        if emailSendData, ok := data.([]services.EmailSendDefinition); ok {
            ch <- emailSendData
        }
    case chan []services.Journey:
        if journeyData, ok := data.([]services.Journey); ok {
            ch <- journeyData
        }
    case chan []services.Script:
        if scriptData, ok := data.([]services.Script); ok {
            ch <- scriptData
        }
    case chan []services.CloudPage:
        if pageData, ok := data.([]services.CloudPage); ok {
            ch <- pageData
        }
    default:
        log.Println("Unsupported channel type")
    }
}

// Build the response for all selected options
func collectResponse(ctx context.Context, channels DataExtensionTaskChannels, deObjectID, deName string, w http.ResponseWriter) DataExtensionResponse {
    var response DataExtensionResponse
    response.Name = deName
    var pathClosed, queriesTargetingClosed, queriesIncludingClosed, importsTargetingClosed, filtersTargetingClosed, contentEmailsClosed, initiatedEmailsClosed, journeysUsingDEClosed, scriptsIncludingClosed, pagesIncludingClosed, errorClosed bool

    for !(pathClosed && queriesTargetingClosed && queriesIncludingClosed && importsTargetingClosed && filtersTargetingClosed && contentEmailsClosed && initiatedEmailsClosed && journeysUsingDEClosed && scriptsIncludingClosed && pagesIncludingClosed && errorClosed) {
        select {
        case <-ctx.Done():
            log.Println("Context canceled, stopping response collection.")
            return response

        case err, ok := <-channels.ErrorChan:
            if !errorClosed {
                if !ok {
                    log.Println("Error channel closed or no errors received")
                    errorClosed = true
                } else if err != nil {
                    log.Println("Error received from error channel:", err)
                    errorClosed = true
                    http.Error(w, err.Error(), http.StatusInternalServerError)
                    return response
                }
            }

        case path, ok := <-channels.PathChan:
            if !pathClosed {
                if !ok {
                    log.Println("Path channel closed or no path received")
                    pathClosed = true
                } else {
                    log.Println("Path received:", path)
                    response.Path = path
                }
            }

        case queriesTargeting, ok := <-channels.QueriesTargetingChan:
            if !queriesTargetingClosed {
                if !ok {
                    log.Println("Queries targeting channel closed or no queries targeting received")
                    queriesTargetingClosed = true
                } else {
                    log.Println("Queries targeting received:", len(queriesTargeting))
                    response.QueriesTargeting = queriesTargeting
                }
            }

        case queriesIncluding, ok := <-channels.QueriesIncludingChan:
            if !queriesIncludingClosed {
                if !ok {
                    log.Println("Queries including channel closed or no queries including received")
                    queriesIncludingClosed = true
                } else {
                    log.Println("Queries including received:", len(queriesIncluding))
                    response.QueriesIncluding = queriesIncluding
                }
            }

        case importsTargeting, ok := <-channels.ImportsTargetingChan:
            if !importsTargetingClosed {
                if !ok {
                    log.Println("Imports targeting channel closed or no imports targeting received")
                    importsTargetingClosed = true
                } else {
                    log.Println("Imports targeting received:", len(importsTargeting))
                    response.ImportsTargeting = importsTargeting
                }
            }

        case filtersTargeting, ok := <-channels.FiltersTargetingChan:
            if !filtersTargetingClosed {
                if !ok {
                    log.Println("Filters targeting channel closed or no filters targeting received")
                    filtersTargetingClosed = true
                } else {
                    log.Println("Filters targeting received:", len(filtersTargeting))
                    response.FiltersTargeting = filtersTargeting
                }
            }

        case contentEmailsIncluding, ok := <-channels.ContentEmailsIncludingChan:
            if !contentEmailsClosed {
                if !ok {
                    log.Println("Content Emails channel closed or no content emails received")
                    contentEmailsClosed = true
                } else {
                    log.Println("Content Emails received:", len(contentEmailsIncluding))
                    response.ContentEmailsIncluding = contentEmailsIncluding
                }
            }

        case initiatedEmailsTargeting, ok := <-channels.InitiatedEmailsTargetingChan:
            if !initiatedEmailsClosed {
                if !ok {
                    log.Println("Initiated Emails channel closed or no initiated emails received")
                    initiatedEmailsClosed = true
                } else {
                    log.Println("Initiated Emails received:", len(initiatedEmailsTargeting))
                    response.InitiatedEmailsTargeting = initiatedEmailsTargeting
                }
            }

        case journeysUsingDE, ok := <-channels.JourneysUsingDEChan:
            if !journeysUsingDEClosed {
                if !ok {
                    log.Println("Journeys channel closed or no journeys received")
                    journeysUsingDEClosed = true
                } else {
                    log.Println("Journeys received:", len(journeysUsingDE))
                    response.JourneysUsingDE = journeysUsingDE
                }
            }

        case scriptsIncluding, ok := <-channels.ScriptsIncludingChan:
            if !scriptsIncludingClosed {
                if !ok {
                    log.Println("Scripts Including channel closed or no scripts received")
                    scriptsIncludingClosed = true
                } else {
                    log.Println("Scripts Including received:", len(scriptsIncluding))
                    response.ScriptsIncluding = scriptsIncluding
                }
            }

        case pagesIncluding, ok := <-channels.PagesIncludingChan:
            if !pagesIncludingClosed {
                if !ok {
                    log.Println("Pages Including channel closed or no pages received")
                    pagesIncludingClosed = true
                } else {
                    log.Println("Pages Including received:", len(pagesIncluding))
                    response.PagesIncluding = pagesIncluding
                }
            }
        }
    }

    // Update cache before returning the response
    cachedResponse, found := deCache.Get(deObjectID)
    if found {
        updateCache(deObjectID, cachedResponse.(DataExtensionResponse), response)
    } else {
        deCache.Set(deObjectID, response, cache.DefaultExpiration)
    }

    return response
}

// Updating the cache when new options are selected for same data extension
func updateCache(deObjectID string, cachedResponse DataExtensionResponse, newResponse DataExtensionResponse) {
    // Check and update the Path field
    if cachedResponse.Path == "" && newResponse.Path != "" {
        cachedResponse.Path = newResponse.Path
    }

    // Check and update the QueriesTargeting field
    if len(cachedResponse.QueriesTargeting) == 0 && len(newResponse.QueriesTargeting) > 0 {
        cachedResponse.QueriesTargeting = newResponse.QueriesTargeting
    }

    // Check and update the QueriesIncluding field
    if len(cachedResponse.QueriesIncluding) == 0 && len(newResponse.QueriesIncluding) > 0 {
        cachedResponse.QueriesIncluding = newResponse.QueriesIncluding
    }

    // Check and update the ImportsTargeting field
    if len(cachedResponse.ImportsTargeting) == 0 && len(newResponse.ImportsTargeting) > 0 {
        cachedResponse.ImportsTargeting = newResponse.ImportsTargeting
    }

    // Check and update the FiltersTargeting field
    if len(cachedResponse.FiltersTargeting) == 0 && len(newResponse.FiltersTargeting) > 0 {
        cachedResponse.FiltersTargeting = newResponse.FiltersTargeting
    }

    // Check and update the ContentEmailsIncluding field
    if len(cachedResponse.ContentEmailsIncluding) == 0 && len(newResponse.ContentEmailsIncluding) > 0 {
        cachedResponse.ContentEmailsIncluding = newResponse.ContentEmailsIncluding
    }

    // Check and update the InitiatedEmailsTargeting field
    if len(cachedResponse.InitiatedEmailsTargeting) == 0 && len(newResponse.InitiatedEmailsTargeting) > 0 {
        cachedResponse.InitiatedEmailsTargeting = newResponse.InitiatedEmailsTargeting
    }

    // Check and update the JourneysUsing field
    if len(cachedResponse.JourneysUsingDE) == 0 && len(newResponse.JourneysUsingDE) > 0 {
        cachedResponse.JourneysUsingDE = newResponse.JourneysUsingDE
    }

    // Check and update the ScriptsIncluding field
    if len(cachedResponse.ScriptsIncluding) == 0 && len(newResponse.ScriptsIncluding) > 0 {
        cachedResponse.ScriptsIncluding = newResponse.ScriptsIncluding
    }

    // Check and update the PagesIncluding field
    if len(cachedResponse.PagesIncluding) == 0 && len(newResponse.PagesIncluding) > 0 {
        cachedResponse.PagesIncluding = newResponse.PagesIncluding
    }

    // Update the cache with the new or updated data
    deCache.Set(deObjectID, cachedResponse, cache.DefaultExpiration)
}

// runTask is a helper function that executes a task concurrently within a goroutine.
func runTask(ctx context.Context, wg *sync.WaitGroup, task func() (interface{}, error), onComplete func(interface{})) {
    wg.Add(1)
    go func() {
        defer wg.Done()
        result, err := task()
        if err != nil {
            log.Printf("Error in task: %v", err)
            return
        }
        onComplete(result)
    }()
}


// Fetch path for Data Extension
func fetchPath(categoryID string, shared bool) (string, error) {
    return services.GetDataExtensionPath(categoryID, shared)
}


// Fetch queries targeting the Data Extension
func fetchQueriesTargeting(deName string) ([]services.QueryDefinition, error) {
    filter := fmt.Sprintf(`
        <Filter xsi:type="SimpleFilterPart">
            <Property>DataExtensionTarget.Name</Property>
            <SimpleOperator>equals</SimpleOperator>
            <Value>%s</Value>
        </Filter>`, deName)
    return services.GetQueries(filter)
}

// Fetch queries including the Data Extension
func fetchQueriesIncluding(deName string) ([]services.QueryDefinition, error) {
    filter := fmt.Sprintf(`
        <Filter xsi:type="SimpleFilterPart">
            <Property>QueryText</Property>
            <SimpleOperator>like</SimpleOperator>
            <Value>%s</Value>
        </Filter>`, deName)
    return services.GetQueries(filter)
}

// Fetch import activities targeting the Data Extension
func fetchImportsForDE(deObjectID string) ([]services.ImportDefinition, error) {
    filter := fmt.Sprintf(`
        <Filter xsi:type="SimpleFilterPart">
            <Property>DestinationObject.ObjectID</Property>
            <SimpleOperator>equals</SimpleOperator>
            <Value>%s</Value>
        </Filter>`, deObjectID)
    return services.GetImports(filter)
}

// Fetch filters using the complex filter logic
func fetchFilters(deObjectID string) ([]services.FilterActivity, error) {
    filter := fmt.Sprintf(`
        <Filter xsi:type="ComplexFilterPart">
            <LeftOperand xsi:type="SimpleFilterPart">
                <Property>DestinationTypeID</Property>
                <SimpleOperator>equals</SimpleOperator>
                <Value>2</Value>
            </LeftOperand>
            <RightOperand xsi:type="SimpleFilterPart">
                <Property>DestinationObjectID</Property>
                <SimpleOperator>equals</SimpleOperator>
                <Value>%s</Value>
            </RightOperand>
            <LogicalOperator>AND</LogicalOperator>
        </Filter>`, deObjectID)

    return services.GetFilters(filter)  // Call GetFilters with the constructed filter
}

// ---- Automation Activity Related Functions and Handlers ----

func AutomationActivityDetail(w http.ResponseWriter, r *http.Request) {
    var req AutomationActivityRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        handleError(w, "invalid request payload2", http.StatusBadRequest)
        return
    }

    // Call the relevant fetch function based on req.Type before concurrent tasks
    var activityObjectID string
    var err error

    // Determine the type and fetch the correct asset (activity) first
    switch req.Type {
    case "Queries":
        var queries []services.QueryDefinition
        queries, err = fetchQueriesForAutomation(req.Name) // FetchQueries returns []QueryDefinition
        if len(queries) > 0 {
            activityObjectID = queries[0].ObjectID
        }
    case "Import Activities":
        var imports []services.ImportDefinition
        imports, err = fetchImportsForAutomation(req.Name) // FetchImports returns []ImportDefinition
        if len(imports) > 0 {
            activityObjectID = imports[0].ObjectID
        }
    case "Scripts":
        var scripts []services.Script
        scripts, err = services.GetScripts("","",req.Name) // FetchQueries returns []QueryDefinition
        if len(scripts) > 0 {
            activityObjectID = scripts[0].ObjectID
        }
    case "Filter Activities":
        var filters []services.FilterActivity
        filters, err = fetchFiltersForAutomation(req.Name) // FetchImports returns []ImportDefinition
        if len(filters) > 0 {
            activityObjectID = filters[0].ObjectID
        }
    default:
        handleError(w, "unsupported activity type", http.StatusBadRequest)
        return
    }

    if err != nil {
        handleError(w, fmt.Sprintf("Error fetching activities: %v", err), http.StatusInternalServerError)
        return
    }

    if activityObjectID == "" {
        handleError(w, "No activity found with the provided name", http.StatusNotFound)
        return
    }

    // Call GetActivities based on activityObjectID
    activities, err := services.GetActivities(activityObjectID)
    if err != nil {
        handleError(w, fmt.Sprintf("Error fetching activities: %v", err), http.StatusInternalServerError)
        return
    }

    // Ensure there's at least one activity in the slice
    if len(activities) == 0 {
        handleError(w, "No automations found for this activity.", http.StatusNotFound)
        return
    }

    // Call GetAutomations based on the Definition.ObjectID of the first activity
    automations, err := services.GetAutomations(activities[0].Program.ObjectID)
    if err != nil {
        handleError(w, fmt.Sprintf("Error fetching automations: %v", err), http.StatusInternalServerError)
        return
    }

    // Prepare the response
    var response AutomationActivityResponse
    response.Automations = automations

    // Send the final response
    sendJSONResponse(w, response)
}

// Fetch queries 
func fetchQueriesForAutomation(queryName string) ([]services.QueryDefinition, error) {
    filter := fmt.Sprintf(`
        <Filter xsi:type="SimpleFilterPart">
            <Property>Name</Property>
            <SimpleOperator>equals</SimpleOperator>
            <Value>%s</Value>
        </Filter>`, queryName)
    return services.GetQueries(filter)
}

func fetchImportsForAutomation(importName string) ([]services.ImportDefinition, error) {
    filter := fmt.Sprintf(`
        <Filter xsi:type="SimpleFilterPart">
            <Property>Name</Property>
            <SimpleOperator>equals</SimpleOperator>
            <Value>%s</Value>
        </Filter>`, importName)
    return services.GetImports(filter)
}

func fetchFiltersForAutomation(filterName string) ([]services.FilterActivity, error) {
    filter := fmt.Sprintf(`
        <Filter xsi:type="SimpleFilterPart">
            <Property>Name</Property>
            <SimpleOperator>equals</SimpleOperator>
            <Value>%s</Value>
        </Filter>`, filterName)

    return services.GetFilters(filter)  // Call GetFilters with the constructed filter
}

// ---- CloudPages Related Functions and Handlers ----

func CloudPageDetail(w http.ResponseWriter, r *http.Request) {
    // Set up context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer func() {
        log.Println("Context canceled")
        cancel()
    }()

    // Parse incoming request to get CloudPageID and User Selections
    var req CloudPageRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        handleError(w, "Invalid request payload", http.StatusBadRequest)
        return
    }

    // Setup channels and WaitGroup
    var wg sync.WaitGroup
    channels := setupCloudPageChannels()

    // Start tasks based on user selection
    startCloudPageTasks(ctx, &wg, req.UserSelection, req.CloudPageID, channels)

    // Wait for all tasks to finish
    go func() {
        wg.Wait()
        closeCloudPageChannels(channels)
    }()

    // Collect response
    response := collectCloudPageResponse(ctx, channels, w)

    // Send the final response
    sendJSONResponse(w, response)
}

// Start tasks based on user selection
func startCloudPageTasks(ctx context.Context, wg *sync.WaitGroup, userSelection map[string]bool, cloudPageID string, channels CloudPageTaskChannels) {
    // Define task map for user selection
    taskMap := map[string]struct {
        fetchFunc func() (interface{}, error)
        channel   interface{}
    }{
        "emailsUsingCloudPage": {func() (interface{}, error) { return fetchEmailsUsingCloudPage(cloudPageID) }, channels.EmailsUsingChan},
        "cloudPagesUsingCloudPage": {func() (interface{}, error) { return fetchCloudPagesUsingCloudPage(cloudPageID) }, channels.CloudPagesUsingChan},
    }

    // Iterate over user selections and start concurrent tasks
    for key, selected := range userSelection {
        if selected {
            task, exists := taskMap[key]
            if exists {
                wg.Add(1)
                go processCloudPageTask(ctx, wg, task.fetchFunc, task.channel, key)
            }
        }
    }
}

// Process a CloudPage task and send the result to the appropriate channel
func processCloudPageTask(ctx context.Context, wg *sync.WaitGroup, fetchFunc func() (interface{}, error), channel interface{}, taskName string) {
    defer wg.Done()
    log.Printf("Starting task: %s", taskName)

    result, err := fetchFunc()
    if err != nil {
        log.Printf("Error fetching %s: %v", taskName, err)
        return
    }
    sendDataToCloudPageChannel(channel, result)
}

// Send data to the appropriate CloudPage channel
func sendDataToCloudPageChannel(channel interface{}, data interface{}) {
    switch ch := channel.(type) {
    case chan []services.Email:
        if emails, ok := data.([]services.Email); ok {
            ch <- emails
        }
    case chan []services.CloudPage:
        if cloudPages, ok := data.([]services.CloudPage); ok {
            ch <- cloudPages
        }
    default:
        log.Println("Unsupported CloudPage channel type")
    }
}

// Collect response for CloudPage based on the channels
func collectCloudPageResponse(ctx context.Context, channels CloudPageTaskChannels, w http.ResponseWriter) CloudPageResponse {
    var response CloudPageResponse
    var emailsUsingClosed, cloudPagesUsingClosed, errorClosed bool

    for !(emailsUsingClosed && cloudPagesUsingClosed && errorClosed) {
        select {
        case <-ctx.Done():
            log.Println("Context canceled, stopping CloudPage response collection.")
            return response

        case err, ok := <-channels.ErrorChan:
            if !errorClosed {
                if !ok {
                    log.Println("Error channel closed or no errors received")
                    errorClosed = true
                } else if err != nil {
                    log.Println("Error received from error channel:", err)
                    errorClosed = true
                    http.Error(w, err.Error(), http.StatusInternalServerError)
                    return response
                }
            }

        case emailsUsing, ok := <-channels.EmailsUsingChan:
            if !emailsUsingClosed {
                if !ok {
                    log.Println("Emails Using channel closed or no emails received")
                    emailsUsingClosed = true
                } else {
                    log.Println("Emails Using received:", len(emailsUsing))
                    response.EmailsUsingCloudPage = emailsUsing
                }
            }

        case cloudPagesUsing, ok := <-channels.CloudPagesUsingChan:
            if !cloudPagesUsingClosed {
                if !ok {
                    log.Println("CloudPages Using channel closed or no CloudPages received")
                    cloudPagesUsingClosed = true
                } else {
                    log.Println("CloudPages Using received:", len(cloudPagesUsing))
                    response.CloudPagesUsingCloudPage = cloudPagesUsing
                }
            }
        }
    }

    return response
}

// Fetch emails using the CloudPage
func fetchEmailsUsingCloudPage(cloudPageID string) ([]services.Email, error) {
    return services.GetEmails("", cloudPageID)
}

// Fetch CloudPages using the CloudPage
func fetchCloudPagesUsingCloudPage(cloudPageID string) ([]services.CloudPage, error) {
    return services.GetCloudPages("","", cloudPageID)
}

// ---- Email Related Functions and Handlers ----

func EmailDetail(w http.ResponseWriter, r *http.Request) {
    // Set up context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer func() {
        log.Println("Context canceled")
        cancel() // Cancel the context at the end
    }()

    // Parse the incoming request to get EmailID or EmailName and User Selections
    var req EmailRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        handleError(w, "Invalid request payload", http.StatusBadRequest)
        return
    }

    // Retrieve the email either by ID or by Name
    email, err := services.GetEmailByIDOrName(req.ID, req.Name)
    if err != nil {
        handleError(w, "No Email found with this ID or Name", http.StatusNotFound)
        return
    }

    // Setup channels and WaitGroup for concurrent tasks
    var wg sync.WaitGroup
    channels := setupEmailTaskChannels()

    emailID := email.ID.String()
    emailName := email.Name

    // Start tasks based on user selection
    startEmailTasks(ctx, &wg, req.UserSelection, emailID, channels)

    // Wait for all tasks to finish
    go func() {
        wg.Wait()
        closeEmailTaskChannels(channels)
    }()

    // Collect response
    response := collectEmailResponse(ctx, channels, emailName, w)

    // Send the final response
    sendJSONResponse(w, response)
}

func startEmailTasks(ctx context.Context, wg *sync.WaitGroup, userSelection map[string]bool, emailID string, channels EmailTaskChannels) {
    taskMap := map[string]struct {
        fetchFunc func() (interface{}, error)
        channel   interface{}
    }{
        "journeysUsingEmail":     {func() (interface{}, error) { return services.GetJourneys("", emailID) }, channels.JourneysUsingEmailChan},
        "initiatedEmailsUsing":   {func() (interface{}, error) { return services.GetInitiatedEmails("", emailID) }, channels.InitiatedEmailsUsingChan},
        "triggeredSends":         {func() (interface{}, error) { return services.GetTriggeredSends(emailID) }, channels.TriggeredSendsChan},
    }

    for key, selected := range userSelection {
        if selected {
            task, exists := taskMap[key]
            if exists {
                wg.Add(1)
                go processEmailTask(ctx, wg, task.fetchFunc, task.channel, key)
            }
        }
    }
}

// Process a task and send the result to the appropriate channel
func processEmailTask(ctx context.Context, wg *sync.WaitGroup, fetchFunc func() (interface{}, error), channel interface{}, taskName string) {
    defer wg.Done()
    log.Printf("Starting task: %s", taskName)

    result, err := fetchFunc()
    if err != nil {
        log.Printf("Error fetching %s: %v", taskName, err)
        return
    }
    sendDataToEmailChannel(channel, result)
}

func collectEmailResponse(ctx context.Context, channels EmailTaskChannels, emailName string, w http.ResponseWriter) EmailResponse {
    var response EmailResponse
    response.Name = emailName
    var journeysClosed, initiatedEmailsClosed, triggeredSendsClosed, errorClosed bool

    for !(journeysClosed && initiatedEmailsClosed && triggeredSendsClosed && errorClosed) {
        select {
        case <-ctx.Done():
            log.Println("Context canceled, stopping response collection.")
            return response

        case err, ok := <-channels.ErrorChan:
            if !errorClosed {
                if !ok {
                    log.Println("Error channel closed or no errors received")
                    errorClosed = true
                } else if err != nil {
                    log.Println("Error received from error channel:", err)
                    errorClosed = true
                    http.Error(w, err.Error(), http.StatusInternalServerError)
                    return response
                }
            }

        case journeys, ok := <-channels.JourneysUsingEmailChan:
            if !journeysClosed {
                if !ok {
                    log.Println("Journeys channel closed or no journeys received")
                    journeysClosed = true
                } else {
                    log.Println("Journeys received:", len(journeys))
                    response.JourneysUsingEmail = journeys
                }
            }

        case initiatedEmails, ok := <-channels.InitiatedEmailsUsingChan:
            if !initiatedEmailsClosed {
                if !ok {
                    log.Println("Initiated Emails channel closed or no initiated emails received")
                    initiatedEmailsClosed = true
                } else {
                    log.Println("Initiated Emails received:", len(initiatedEmails))
                    response.InitiatedEmailsUsing = initiatedEmails
                }
            }

        case triggeredSends, ok := <-channels.TriggeredSendsChan:
            if !triggeredSendsClosed {
                if !ok {
                    log.Println("Triggered Sends channel closed or no triggered sends received")
                    triggeredSendsClosed = true
                } else {
                    log.Println("Triggered Sends received:", len(triggeredSends))
                    response.TriggeredSends = triggeredSends
                }
            }
        }
    }

    return response
}

// Function to send data to the appropriate email task channel
func sendDataToEmailChannel(channel interface{}, data interface{}) {
    switch ch := channel.(type) {
    case chan []services.Journey:
        if journeys, ok := data.([]services.Journey); ok {
            ch <- journeys
        }
    case chan []services.EmailSendDefinition:
        if initiatedEmailsUsing, ok := data.([]services.EmailSendDefinition); ok {
            ch <- initiatedEmailsUsing
        }
    case chan []services.TriggeredSendDefinition:
        if triggeredSends, ok := data.([]services.TriggeredSendDefinition); ok {
            ch <- triggeredSends
        }
    default:
        log.Println("Unsupported email task channel type")
    }
}