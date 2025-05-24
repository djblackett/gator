# 🐊 Gator CLI – Getting Started

## 🛠 Prerequisites

To run the gator CLI, you’ll need:

    Go installed (version 1.21+ recommended)

    PostgreSQL running locally or accessible remotely

Ensure both go and psql are available in your terminal:

go version
psql --version

## 📦 Installing the Gator CLI

You can install the CLI using the go install command:

go install github.com/YOUR-GITHUB-USERNAME/gator@latest

    Replace YOUR-GITHUB-USERNAME with your actual GitHub username.

This will install the binary to your $GOPATH/bin directory (commonly ~/go/bin). Make sure that directory is in your $PATH:

export PATH=$PATH:$(go env GOPATH)/bin

## ⚙️ Configuring Gator

Before running Gator, you need a config file so it knows how to connect to your PostgreSQL database.
Create a .gatorconfig file in your home directory:

touch ~/.gatorconfig

Example contents:

`{"db_url":"postgres://postgres:postgres@localhost:5432/gator?sslmode=disable"}`

🚀 Running Gator

Once installed and configured, simply run:

    ```bash
    gator
    ```

You’ll be dropped into an interactive CLI where you can run various commands.
Example Commands:

    feeds – List all feeds in the database.

    users – Display users who have subscribed to feeds.

    help – Get a list of all available commands.

    exit – Exit the CLI.
