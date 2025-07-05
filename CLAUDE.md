# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

This is a simple web service written in Go that manages WeCom (企业微信) bot webhooks. It allows users to register bots via a web interface and send messages to them using a JSON API. The service uses a local SQLite database (`bots.db`) to store bot information.

## Architecture

- **`main.go`**: This is the single source file containing all the application logic.
  - It uses the standard `net/http` library for routing.
  - **`/`**: The root endpoint serves a web page (`templates/index.html`) for registering new bots. On POST, it saves the bot's webhook URL to the database.
  - **`/send`**: A JSON API endpoint that accepts a POST request to send a message to a registered bot by its ID. It supports `text` and `markdown` message types.
  - **Database**: Uses `go-sqlite3` to interact with the `bots.db` SQLite file.
  - **Structs**:
    - `WeCom...` structs are used to serialize JSON payloads for the WeCom webhook API.
    - `SendMessageRequest` is used to deserialize incoming requests to the `/send` API.

- **`templates/index.html`**: A simple HTML file with inline CSS that provides the user interface for registering bots.

- **`API.md`**: Contains the documentation for the `/send` endpoint.

## Common Commands

- **Build the application:**
  ```bash
  go build -o qywxbot_server
  ```

- **Run the application:**
  ```bash
  ./qywxbot_server
  ```
  The server will start on port 8080.

- **Manage dependencies:**
  ```bash
  go mod tidy
  ```
