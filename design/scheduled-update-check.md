# Scheduled Update Check

KOTS does not automatically check for new updates.
This proposal adds the ability for KOTS to check for updates every X period of time.

## Goals

- Pull new updates automatically

## Non Goals

- Deployment of new updates

## Background

Many customers assume that scheduled update checks are a given and are surprised when they do not receive updates automatically.

## High-Level Design

A background process/thread that is triggered every X period of time which makes a "check for updates" request to the KOTS server

## Detailed Design

The KOTS operator will have a "UpdateChecker" goroutine loop which starts once the client connects to the api.
The "UpdateChecker" sleeps for X period of time and then issues a "check for updates" request to the KOTS server.
The "check for updates" request handler on the KOTS server already creates new versions automatically if there are updates available.
