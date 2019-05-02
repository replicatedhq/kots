# Setting Up A GitHub App

Log in to GitHub and create a new app by clicking: https://github.com/settings/apps/new

Use the following information to configure this app:

App Name: Ship Cluster Dev

Homepage URL: http://localhost:8000

User authorization callback URL: http://localhost:8000/auth/github/callback

Setup URL: http://localhost:8000

Redirect on Update: ✓

If using ngrok to receive github pull request events, you can add this:
Webhook URL: https://your-ngrok.ngrok.io/api/v1/hooks/github

**Permissions:**

Repository Contents: Read & Write

Pull Requests: Read & Write

**Subscribe To Events:**

Pull request ✓

