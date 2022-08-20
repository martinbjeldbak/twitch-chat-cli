# Twitch Chat CLI
A [Twitch.tv](https://twitch.tv) chat client in your terminal

Still in early development and missing many core features

<!-- DEMO CLI gif here -->

## Features

- Connect to multiple Twitch channels
- Chat in multiple Twitch channels (requires authentication)


Goal is to be simmiler to [Chatterino](https://chatterino.com/), but in the CLI

Built With
- Go
- bubbletea

## How to use

This CLI can be started without any arguments and will launch in anonymous mode, meaning you won't be able to chat.

If you wish to chat, you must auth first with the `auth` command and add the credentials to the `--accounts` flag or the config file.

### Example

Below is an example run, where all arguments are passed in via flags

```sh
$ twitch-chat-cli auth # get the arguments passed in to --accounts
$ twitch-chat-cli --channels "blastpremier,jakenbakelive" --accounts "username=qcx;user_id=1234;client_id=123;oauth_token=456"
```

This tool is completely self-sufficient and does not rely on any other services than the Twitch.tv APIs.

A help menu can be found by calling `--help`


### Help

```sh
$ twitch-chat-cli --help
A Twitch chat client in the terminal.

Allows chatting in multiple Twitch channels from the comfort of your terminal.
Supports connecting anonymously or as an authenticated user.

This application pairs nicely with Streamlink <https://streamlink.github.io/> for
a complete website-free viewing experience!

Usage:
  twitch-chat-cli [flags]
  twitch-chat-cli [command]

Available Commands:
  auth        Authenticate with Twitch
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  version     Print the version number of twitch-chat-cli

Flags:
      --accounts auth      accounts and their oauth tokens to use, for example see example.yaml. Fetched via the auth command
      --channels strings   channels to join (default [pokimane])
      --config string      config file (default is $HOME/.twitch-chat-cli.yaml)
  -h, --help               help for twitch-chat-cli
  -t, --toggle             Help message for toggle
```

### Command: auth

```sh
$ twitch-chat-cli auth
```

This command spins up a local server which will act as the callback URL for the Twitch [OIDC implicit grant flow](https://dev.twitch.tv/docs/authentication/getting-tokens-oidc#oidc-implicit-grant-flow) dance.

You will be asked to sign in to the Twitch account you wish to use this CLI with.

Once authenticated, you will be redirected to the local web server will display a web page with auth string that needs to be passed to the `--accounts` flag, or pasted into your config.

## Installation

TODO: add better installation instructions.

Currently the binary needs to be downloaded manually from the GitHub releases page at <https://github.com/martinbjeldbak/twitch-chat-cli/releases> for the right OS and placed into the PATH or manually run.


## Inspiration
- https://github.com/chatterino/chatterino2 - Chatterino
- https://github.com/atye/ttchat, similar principle, 1 channel only and need dev account
- https://github.com/dlvhdr/gh-dash - beautiful TUI

## Docs
- https://dev.twitch.tv/docs/cli
- https://dev.twitch.tv/docs/irc
- https://github.com/gempir/go-twitch-irc
- https://github.com/nicklaw5/helix


## TODO
- [ ] Style authentication page smilar to https://chatterino.com/client_login or https://twitchapps.com/tmi/ (starts site with url first, can do this)
- [ ] Open streamlink for current channel using bubbles keybind
- [ ] Add multiuser support
- [x] Tab for each chat/channel
- [x] Scrolling chat window
- [ ] Remove older messages (performance improvement. do if needed)
  - Consider via oauth2 pkg https://pkg.go.dev/golang.org/x/oauth2#AuthStyle, https://dev.twitch.tv/docs/authentication/getting-tokens-oauth/#implicit-grant-flow
- [x] Enter chat msg
- [ ] Vim-modes for navigation
- [ ] Idea: Start in lurker mode. No chat input, just focus on seeing
- [x] Goreleaser https://goreleaser.com/intro/
- [ ] Goreleaser: release docker image (have Dockerfile already, but first fix it up with `distroless`)
- [ ] Animated emotes (image/gif?)
- [ ] Add to lists
  - https://github.com/charmbracelet/bubbletea#bubble-tea-in-the-wild
  - https://github.com/rothgar/awesome-tuis
- [x] Add viper for config files https://github.com/spf13/viper
- [ ] BTTW emote support via https://github.com/pajlada/gobttv (see usage https://github.com/pajbot/pajbot2/search?q=gobttv)
- [ ] Add emote cache https://github.com/charmbracelet/charm/tree/main/kv
- [ ] User profile pages + mentions
- [ ] Nicer text wrapping (indentet length of user name)
- [ ] Add info about terminal emote support (kitty / iterm2 / sixel)
  - https://github.com/alacritty/alacritty (soon)
    - soon https://github.com/alacritty/alacritty/pull/4763, https://github.com/alacritty/alacritty/issues/910
  - Windows Terminal does not support emotes
- [ ] `PM` and `WhisperMessage` support. See https://pkg.go.dev/github.com/gempir/go-twitch-irc/v3@v3.0.0#readme-available-data


## License

This application is released under the MIT license. See [`LICENSE`](LICENSE)
