This is small program that syncs local folders with the `Google Photos/Albums/`
folder from Google Drive.

This program assumes that:

- the "Create a Google Photos folder" from "Settings" from drive.google.com
  is checked
- the photos from the local folders are already uploaded to Google Photos
- the `Google Photos/Albums/` was created.

## How to get/build this program

In a Go workspace run:

    go get github.com/razvanm/google-album-sync