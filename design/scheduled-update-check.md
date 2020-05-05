# Scheduled Update Checks for Online Installs

KOTS (Admin Console) does not automatically check for new updates. This proposal adds the 
ability for KOTS (Admin Console) to check for updates every X period of time for online installations.

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
2. The default period of time (X) is configurable.
3. The ability to disable this feature completely. It will be enabled by default.

The update check will happen in a background process/thread which is triggered every X period of time.
On each trigger, the process will get all available applications and run update checks for each app.
The update check will automatically create new versions for each update available in an app.

## Detailed Design

3 new columns (`update_checker_enabled` & `update_checker_interval` & `update_checker_status`) will be 
added to the `app` table in the postgress database using a migration.

* The `update_checker_enabled` column is a bool and defaults to `true`. 
* The `update_checker_interval` column is a text and defaults to `6h`.
* The `update_checker_status` column is a text and defaults to `NULL`

* The `update_checker_enabled` column indicates if the feature is enabled or disabled.
* The `update_checker_interval` column holds the value of X (the period of time between update checks).
* The `update_checker_status` column holds the status of the update checker as a json object with two
keys ("status" & "message"). "status" can be ("running", "stopped", "failed").
The "message" key will hold a value as to why the status is "failed".

By default, when the KOTS Admin Console's api starts:

1. The api will get the configuration values from the database.
2. If the feature is enabled, start a cron job schedule which runs every X period of time. 
3. Use the same logic from the update check request to check for updates for each application.
4. The update check request logic already creates new versions automatically if there are updates available.
5. If the api fails to start the cron job, a "failed" status along with the reason will be saved in
the `update_checker_status` column in the database.
6. A custom troubleshoot analyzer will be built into kotsadm to detect those failures.

To configure these options:

1. There will be a "Configure update checker" link in the Admin Console's version history card & page.
2. Upon clicking the link, a modal will display the values of these options along with the status of the checker.
3. The modal will have a "Update" button. 
4. Once the user clicks "Update", a request will be made with the new values to the Admin Console's api.
5. The request will update the values in the database and kill the current cron job (if running).
6. The request will then check if the feature is still enabled and start a new cron job with the new interval.
7. If any of this fails, the request will return a failure message and the new status which will be displayed.
