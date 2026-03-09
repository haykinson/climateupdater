#!/bin/bash
set -e

# Configuration
APP_NAME="climateupdater"
INSTALL_DIR="/opt/$APP_NAME"
SYSTEM_USER="climateupdater"

echo "Starting installation of $APP_NAME for Ubuntu 24.04..."

# 1. Check for Go
if ! command -v go >/dev/null 2>&1; then
    echo "Error: Go compiler is not installed."
    echo "Please install Go 1.22+ first (e.g., sudo apt install golang-go)"
    exit 1
fi

# 2. Build the application binary
echo "Compiling the Go application..."
# Force standard linux build
GOOS=linux GOARCH=amd64 go build -o $APP_NAME

# 3. Create a dedicated system user
echo "Creating dedicated system user ($SYSTEM_USER)..."
if ! id -u $SYSTEM_USER >/dev/null 2>&1; then
    sudo useradd -r -M -s /bin/false $SYSTEM_USER
else
    echo "User $SYSTEM_USER already exists, skipping."
fi

# 4. Setup directories and copy files
echo "Setting up installation directory at $INSTALL_DIR..."
sudo mkdir -p $INSTALL_DIR
sudo mkdir -p $INSTALL_DIR/static

echo "Copying binary and static assets..."
sudo cp $APP_NAME $INSTALL_DIR/
sudo cp -r static/* $INSTALL_DIR/static/

# Set ownership
sudo chown -R $SYSTEM_USER:$SYSTEM_USER $INSTALL_DIR

# 5. Install and enable systemd service
echo "Installing systemd service..."
if [ ! -f "climateupdater.service" ]; then
    echo "Error: climateupdater.service not found in the current directory."
    exit 1
fi

sudo cp climateupdater.service /etc/systemd/system/
sudo systemctl daemon-reload

echo "Enabling and starting the service..."
sudo systemctl enable $APP_NAME.service
sudo systemctl restart $APP_NAME.service

echo "============================================="
echo "Installation Complete!"
echo "The application is now running as a background service."
echo "You can check its status with: sudo systemctl status $APP_NAME"
echo "You can view logs with: sudo journalctl -u $APP_NAME -f"
echo "By default, it is serving on port 8081."
