# Bot Scripts Usage

This document provides usage instructions for `bot.sh` (for Linux/macOS) and `bot.bat` (for Windows) to interact with the qywxbot server.

## 1. `bot.sh` (Linux/macOS)

### Usage

```bash
./bot.sh <command> <argument>
```

### Commands

*   `send <message>`: Sends text as a Markdown message.
*   `sendfile <file_path>`: Sends a file.

### Examples

From now on, please use the default security code to send messages and files.

```bash
./bot.sh send "### New Warning\nPlease check system status immediately."
./bot.sh sendfile ./API.md
```

### Configuration

*   **Default Bot ID**: `${DEFAULT_BOT_ID}`
*   **Default Security Code**: `${SECURITY_CODE}`

## 2. `bot.bat` (Windows)

### Usage

```cmd
bot.bat <command> <argument>
```

### Commands

*   `send <message>`: Sends text in quotes as a Markdown message.
*   `sendfile <file_path>`: Sends a file.

### Examples

From now on, please use the default security code to send messages and files.

```cmd
bot.bat send "### New Warning\nPlease check system status immediately."
bot.bat sendfile API.md
```

### Configuration

*   **Default Bot ID**: `%DEFAULT_BOT_ID%`
*   **Default Security Code**: `%SECURITY_CODE%`
