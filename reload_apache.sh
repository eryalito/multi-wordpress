#!/bin/bash

set -e
DEBUG=1

# Start Apache in the background
echo "[DEBUG] Starting Apache in foreground mode..."
apache2ctl -D FOREGROUND &

# Get the PID of the Apache process
APACHE_PID=$!
echo "[DEBUG] Apache started with PID $APACHE_PID"

# Function to gracefully reload Apache
reload_apache() {
  echo "[DEBUG] Detected changes in Apache configuration. Reloading Apache..."
  apache2ctl graceful
}

echo "[DEBUG] Monitoring /etc/apache2/sites-enabled for changes..."
while true; do
  EVENT=$(inotifywait -e create -e delete -e modify -e move --format '%e %w%f' /etc/apache2/sites-enabled/)
  echo "[DEBUG] Event detected: $EVENT"
  reload_apache
done &

# Wait for the Apache process to exit
wait $APACHE_PID
