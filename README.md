# Cubic RCE Bot

[![Build](https://github.com/database64128/cubic-rce-bot/actions/workflows/build.yml/badge.svg)](https://github.com/database64128/cubic-rce-bot/actions/workflows/build.yml)
[![Release](https://github.com/database64128/cubic-rce-bot/actions/workflows/release.yml/badge.svg)](https://github.com/database64128/cubic-rce-bot/actions/workflows/release.yml)
[![AUR version](https://img.shields.io/aur/version/cubic-rce-bot-git?label=cubic-rce-bot-git)](https://aur.archlinux.org/packages/cubic-rce-bot-git)

Execute commands on a remote host via a Telegram bot.

## Overview

Configuration examples and systemd unit files can be found in the [docs](docs) directory.

- Only authorized users can execute allowed commands.
- Configuration can be reloaded by sending a `SIGUSR1` signal to the process.

## License

[AGPLv3](LICENSE)
