#!/bin/bash

# Check if the S3 bucket argument is provided
if [ "$#" -ne 1 ]; then
    echo "Usage: $0 <s3-backups-bucket-name>"
    exit 1
fi

# Variables
S3_BUCKET_BACKUPS="$1"
GUESTS_DB="guests.db"
NAMES_CSV="names.csv"
GO_APP="wedding-rsvps" # Replace with your Go application's binary name

# Check if guests.db exists
if [ ! -f "$GUESTS_DB" ]; then
    echo "$GUESTS_DB does not exist. Copying from S3..."
    
    # Copy the primary file from S3
    aws s3 cp "s3://$S3_BUCKET_BACKUPS/$GUESTS_DB" "$GUESTS_DB"
    
    # Check if the primary file copy was successful
    if [ ! -f "$GUESTS_DB" ]; then
        echo "$GUESTS_DB could not be copied from S3. Attempting to copy fallback file..."

        # Copy the fallback file from S3
        aws s3 cp "s3://$S3_BUCKET_BACKUPS/$NAMES_CSV" "$NAMES_CSV"
        
        # Check if the fallback file copy was successful
        if [ ! -f "$NAMES_CSV" ]; then
            echo "Error: Neither file could be copied from S3. Exiting."
            exit 1
        fi
    fi
else
    echo "$GUESTS_DB already exists."
fi

# Build the Go web server application
echo "Building the Go application..."
go build -o "$GO_APP"

# Run the Go application
echo "Running the Go application..."
"$GO_APP" & # The & runs it in the background; remove it if you want it to run in the foreground

echo "Deployment complete."
