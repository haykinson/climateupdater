# Climate Records Tracker - Deployment Guide

This guide provides instructions for deploying the Climate Records Tracker application onto a fresh Ubuntu 24.04 server with Nginx as a reverse proxy.

## Overview
The deployment involves:
1. Compiling the Go backend (`main.go`, `fetcher.go`, `calculator.go`) into a binary.
2. Setting up a dedicated system user (`climateupdater`) to securely run the service.
3. Placing the binary and static assets (`static/`) into `/opt/climateupdater`.
4. Creating a `systemd` service to keep the backend running and to manage restarts automatically.
5. Configuring `nginx` to proxy HTTP traffic on port 80 to the internal Go server on port 8081.

---

## 1. Prerequisites

Run the following commands on your Ubuntu 24.04 server to install Go 1.22+ and Nginx:

```bash
sudo apt update
sudo apt install -y golang-go nginx git
```

## 2. Installation using `install.sh`

We provided an installation script to handle building the code, creating the user, moving the files, and enabling the systemd service.

1. Clone or copy your project files to the server.
2. Inside the project directory, ensure the script is executable and run it:
   ```bash
   chmod +x install.sh
   ./install.sh
   ```

*The `install.sh` script will prompt you for your `sudo` password to create directories and install the systemd service.*

## 3. Configure NGINX (Reverse Proxy)

After the application is running via systemd (on local port 8081), configure Nginx to expose it to the public:

1. Create a new Nginx server block configuration.
   If your server uses the traditional Debian layout:
   ```bash
   sudo nano /etc/nginx/sites-available/climateupdater
   ```
   *(If your Nginx installation doesn't have a `sites-available` folder, you can simply create the file at `/etc/nginx/conf.d/climateupdater.conf` instead).*

2. Add the following Nginx configuration (replace `your_domain_or_ip` with your server's IP or domain name):
   **Note**: This block initially listens on port 80 (HTTP). When you run Certbot later, Certbot will automatically rewrite this file to configure port 443 (HTTPS) and set up redirects.

   ```nginx
   server {
       listen 80;
       server_name your_domain_or_ip;

       location / {
           proxy_pass http://localhost:8081;
           proxy_http_version 1.1;
           proxy_set_header Upgrade $http_upgrade;
           proxy_set_header Connection 'upgrade';
           proxy_set_header Host $host;
           proxy_set_header X-Real-IP $remote_addr;
           proxy_cache_bypass $http_upgrade;
       }
   }
   ```

3. If you used `sites-available`, enable the site by symlinking it to `sites-enabled`:
   ```bash
   sudo ln -s /etc/nginx/sites-available/climateupdater /etc/nginx/sites-enabled/
   ```
   *(If you used `conf.d`, you can skip this step).*

4. Test your Nginx configuration to ensure there are no syntax errors:
   ```bash
   sudo nginx -t
   ```

5. Reload Nginx to apply the changes:
   ```bash
   sudo systemctl reload nginx
   ```

## 4. Managing the Service

You can monitor and manage the Go backend using standard `systemctl` and `journalctl` commands:

- **Check Service Status:**
  ```bash
  sudo systemctl status climateupdater
  ```
- **View Live Application Logs:**
  ```bash
  sudo journalctl -u climateupdater -f
  ```
- **Stop the Application:**
  ```bash
  sudo systemctl stop climateupdater
  ```

## 5. Updating the Code

When you make changes to the Go source code or HTML files, you simply need to pull the new code and rerun the install script. The script safely overwrites the old binary and static files and restarts the server automatically!

1. Navigate to your project directory on the server:
   ```bash
   cd ~/climateupdater
   ```

2. Pull the latest code (e.g., via git):
   ```bash
   git pull
   ```

3. Rerun the installation script:
   ```bash
   ./install.sh
   ```

The script will recompile the Go binary, copy the updated files to `/opt/climateupdater`, and run `systemctl restart climateupdater` so your new changes go live instantly.

Your Climate Records Tracker should now be accessible via your server's IP address or domain name!
