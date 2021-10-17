# Lyncser

Lyncser is a program for syncing files between machines. It backups the specified files to Google Drive. When a newer version of the file is detected in Google Drive, that file is downloaded to the current machine. When a modification is made to the file locally, that file is uploaded to Google Drive.

The reason why this tool was built as opposed to simply using Google Drive Backup and Sync is because there is not a Linux version of that tool.

## Install

Currently, installation is only supported on Linux.

### Linux

Download the latest release from https://github.com/chrismcgehee/lyncser/releases, and extract using `tar xvf`. If using systemd, lyncser can be installed by running:

```sh
lyncser/install.sh
```

## Usage

Lyncser requires generating an OAuth 2.0 Client ID within Google Cloud Platform and downloading the credentials to `~/.config/lyncser/credentials.json`. Then running `lyncser` will prompt you to authorize the application via a browser. This first run will also generate the config files. You can then modify `~/.config/lyncser/globalConfig.yaml` to list the files you want to sync. This file will be automatically synced across all machines. You can use tags to limit files to be synced only on certain machines, for example:

```yaml
paths:
  all:
    - "~/.gitconfig"
  work_machines:
    - "~/.bashrc"
    - "~/code/"
  personal_machines:
    - "~/Documents/"
```

The `~/.config/lyncser/localConfig.yaml` file is for applying tags to the current machine. This file is not synced (unless explicitly listed in `globalConfig.yaml`). For example:

```yaml
tags:
  - all
  - personal_machines
```

If the install script was executed, `lyncser` will run every 5 minutes and perform syncing. You may also run `lyncser` (no arguments necessary) at any time to perform a sync.

## Future Plans
- Use something like OAuth PKCE so the program does not need to access the client secret.
- Add tests, especially for the logic around when to upload vs. when to download a file
