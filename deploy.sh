#!/bin/bash

if [ "$#" -ne 1 ]; then
    echo "Usage: $0 <s3-backups-bucket-name>"
    exit 1
fi

# Variables
S3_BUCKET_BACKUPS="$1"
GUESTS_DB="guests.db"
NAMES_CSV="names.csv"
GO_APP="wedding-rsvps" 

# Trap any errors and ensure exit
trap 'exit 1' ERR

# Check if guests.db exists
if [ ! -f "$GUESTS_DB" ]; then
    echo "$GUESTS_DB does not exist. Copying from S3..."
    
    aws s3 cp "s3://$S3_BUCKET_BACKUPS/$GUESTS_DB" "$GUESTS_DB"
    
    if [ ! -f "$GUESTS_DB" ]; then
        echo "$GUESTS_DB could not be copied from S3. Attempting to copy fallback file..."

        aws s3 cp "s3://$S3_BUCKET_BACKUPS/$NAMES_CSV" "$NAMES_CSV"
        
        if [ ! -f "$NAMES_CSV" ]; then
            echo "Error: Neither file could be copied from S3. Exiting."
            exit 1
        fi
    fi
else
    echo "$GUESTS_DB already exists."
fi

ps

# Find the PID of any currently running instance of the app
CURRENT_PID=$(pgrep -f "$GO_APP")

if [ -n "$CURRENT_PID" ]; then
  echo "Terminating the existing instance of $GO_APP with PID $CURRENT_PID..."
  kill -SIGTERM "$CURRENT_PID"

  # Wait for the process to terminate
  sleep 3

  if ps -p "$CURRENT_PID" > /dev/null; then
    echo "$GO_APP did not terminate gracefully; force-killing."
    kill -9 "$CURRENT_PID"
  else
    echo "$GO_APP terminated successfully."
  fi
else
  echo "No existing instance of $GO_APP found."
fi

mv ./"$GO_APP"-temp ./"$GO_APP"

echo "Running the Go application..."
nohup ./"$GO_APP" > /dev/null 2>&1 & # The & runs it in the background; remove it if you want it to run in the foreground
PID=$!                     # Capture PID

# Give the process some time to start
sleep 5

# Check if the process is still running
if ps -p $PID > /dev/null
then
  echo "$GO_APP is running."
else
  echo "$GO_APP failed to start."
  exit 1
fi

echo "Deployment complete."
exit 0
