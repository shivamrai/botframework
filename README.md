# BotFramework Project

## Dev Container Setup

This project is configured to use a Dev Container with isolated SSH credentials.

### Setup Instructions

1.  **SSH Keys**: Before opening the Dev Container, ensure your GitHub SSH keys are placed in the `localssh` directory.
    *   The `localssh` directory is git-ignored to prevent accidental commits of credentials.
    *   Required files: `id_rsa` (private key) and `id_rsa.pub` (public key), or your specific key names.
2.  **Open in Container**: Use the "Dev Containers: Reopen in Container" command in VS Code.

### Configuration

The `.devcontainer/devcontainer.json` mounts the `localssh` folder to `/home/vscode/.ssh` inside the container.
