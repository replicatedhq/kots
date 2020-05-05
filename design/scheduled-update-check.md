# Scheduled Update Checks for Online Installs

KOTS (Admin Console) does not automatically check for new updates. This proposal adds the 
ability for KOTS (Admin Console) to check for updates every X period of time for online installations.

## Goals

- Pull new updates automatically for KOTS (Admin Console) online installations

## Non Goals

- Deployment of the new updates
- Scheduled Update Checks for Airgap Installations

## Background

Many customers assume that scheduled update checks are a given and are surprised when they do not receive 
updates automatically.

## High-Level Design

1. This proposal is only for online installations in KOTS Admin console.
2. The default period of time (X) is configurable.
3. The ability to disable this feature completely.

The update check will happen in a background process/thread which is triggered every X period of time.
On each trigger, the process will get all available applications and run update checks for each app.
The update check will automatically create new versions for each update available in an app.

## Detailed Design

3 new columns (`update_checker_enabled` & `update_checker_interval` & `update_checker_status`) will be 
added to the `app` table in the postgress database using a migration.

* The `update_checker_enabled` column is a bool and defaults to `true`. 
* The `update_checker_interval` column is an integer and defaults to `360` minutes.
* The `update_checker_status` column is a text and defaults to `NULL`

* The `update_checker_enabled` column indicates if the feature is enabled or disabled.
* The `update_checker_interval` column holds the value of X (the period of time between update checks).
* The `update_checker_status` column holds the status of the update checker as a json object with two
keys ("status" & "message"). "status" can be ("running", "stopped", "failed").
The "message" key will hold a value as to why the status is "failed".

By default, when the KOTS Admin Console's api starts:

1. The api will get the configuration values from the database.
2. If the feature is enabled, start a goroutine service "loop". 
3. The service sleeps for X period of time.
4. Use the same logic from the update check request to check for updates for each application.
5. The update check request logic already creates new versions automatically if there are updates available.
6. If the api fails to start the service, a "failed" status along with the reason will be saved in
the `update_checker_status` column in the database and displayed in the Admin Console's dashboard page.

To configure these options:

1. There will be an additional card in the Admin Console's dashboard page.
2. The card will display the values of these options along with the status of the update checker.
3. The card will have a "Update" button. 
4. Upon clicking "Update", a modal will show up which enables the user to edit those values.
5. The modal will have a "Submit" button. 
6. Once the user clicks "Submit", a request will be made with the new values to the Admin Console's api.
7. The request will update the values in the database and kill the current service (if running).
8. The request will the check if the feature is still enabled and start a new one with the new interval.
9. If any of this fails, the request will return a failure message and the new status which will be
displayed on the dashobard card.