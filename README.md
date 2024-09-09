# SFMC Asset Relationship Finder

SFMC Asset Relationship Finder is a powerful application designed to help Salesforce Marketing Cloud (SFMC) users easily explore and understand the relationships between various assets such as Data Extensions, Automations, Emails, and Cloud Pages. With a focus on API-driven exploration and relationship mapping, this tool simplifies the complex task of discovering dependencies across your assets within SFMC.

## Key Features

- **Asset Relationship Discovery**: Quickly identifies the relationships between different SFMC assets, including Data Extensions, Automations, Emails, and Cloud Pages.
- **Concurrent API Calls**: Ensures rapid, simultaneous API interactions, making the application faster and more responsive even when managing multiple asset relationships in real-time.
- **Multi-Asset Type Support**: Supports searching and relationship mapping for multiple asset types (Data Extensions, Cloud Pages, Emails, Automations).
- **Smart Asset Filtering**: Filter assets by name or key, and view detailed relationships to other assets within SFMC.
- **Efficient Search & Display**: Presents asset information in a clear and concise format, including which automations use specific Data Extensions or which Emails reference Cloud Pages and etc.
- **API-Based Retrieval**: Uses efficient API consumption strategies to retrieve data with minimal overhead, optimizing the interaction with SFMC’s REST and SOAP APIs.
- **Enhanced User Interaction**: Includes a “View More” feature for long lists of relationships, allowing users to expand or collapse results as needed without overwhelming the dashboard.

![Screenshot](/screenshots/2.png)

![Screenshot](/screenshots/3.png)

![Screenshot](/screenshots/4.png)

![Screenshot](/screenshots/5.png)

![Screenshot](/screenshots/8.png)

## Getting Started

This application is developed for Heroku-hosted deployments, complementing Salesforce Marketing Cloud (SFMC) integration.

### Salesforce Marketing Cloud Setup

1. Create a Web App package with the necessary permissions below:
   - **Email**: Read
   - **Documents and Images**: Read
   - **Saved Content**: Read
   - **Automations**: Read
   - **Journeys**: Read
   - **Data Extensions**: Read
   - **File Locations**: Read
2. Set the redirect URI to https://yourherokudomain.herokuapp.com.

For SFMC UI integration:

- Configure the "Login Endpoint" to https://yourherokudomain.herokuapp.com/auth/login and "Logout Endpoint" can point to https://yourherokudomain.herokuapp.com/auth/logout.

### Heroku Configuration

After deploying the app via a GitHub repository:

Define the necessary configuration variables within Heroku's settings to match the SFMC package details:
  - `AUTHORIZATION_URL`
  - `CLIENT_ID`
  - `CLIENT_SECRET`
  - `REDIRECT_URI`
  - `REST_ENDPOINT`
  - `SOAP_ENDPOINT`

With these configurations, the app is ready for use.

## License

SFMC Asset Relationship Finder is open-sourced under the MIT License. See the LICENSE file for more details.

---
