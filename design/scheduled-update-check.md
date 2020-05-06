# Scheduled Update Checks for Online Installs

KOTS (Admin Console) does not automatically check for new updates. This proposal adds the 
ability for KOTS (Admin Console) to check for updates on specific times for online installations.

## Goals

- Pull new updates automatically for KOTS (Admin Console) online installations

## Non Goals

- Deployment of the new updates
- Scheduled Update Checks for Airgap Installations

## Background

Many customers assume that scheduled update checks are a given and are surprised when they do
not receive updates automatically.

## High-Level Design

1. This proposal is only for online installations in KOTS Admin console.
2. The times on which the update check will happen is configurable by a cron spec.
3. The ability to disable this feature completely. It will be enabled by default.

The update check will happen in a background process/thread which is triggered on specific times
defined by a cron job spec. On each trigger, the process will get all available applications and run
update checks for each app. The update check will automatically create new versions for each 
update available in an app.

## Detailed Design

A new column `update_checker_spec` will be added to the `app` table in the postgress database 
using a migration. The column is a text and defaults to `@daily`, which will indicate that the cron job
will run daily at 1 AM.

By default, when the KOTS Admin Console's api starts:

1. The api will read the cron spec value from the database.
2. If the feature is enabled (`!= @never`), start a cron job schedule with the provided spec.
3. Use the same logic from the update check request to check for updates for each application.
4. The update check request logic already creates new versions automatically if there are updates available,
and handles the scenario if multiple requests have been made to check for updates.

To configure the update checker spec:

1. There will be a "Configure update checker" link in the Admin Console's version history card & page.
2. Upon clicking the link, a small modal will display the value of the cron job spec.
3. The modal will have a "Update" button. 
4. Once the user clicks "Update", a request will be made with the new spec to the Admin Console's api.
5. The request will update the spec in the database and kill the current cron job (if running).
6. The request will then check if the feature is still enabled and start a new cron job with the new spec.
7. If any of this fails, the request will return a failure message which will be displayed in the modal.
