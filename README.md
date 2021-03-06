# Upterm

[Upterm](https://github.com/jingweno/upterm) is an open-source solution for sharing terminal sessions instantly with the public internet over secure tunnels.

## What it's good for

* Remote pair programming
* Access remote computers behind NATs and firewalls
* Remote debugging
* \<insert your creative use cases\>

## Installation

### Mac

```
brew tap jingweno/upterm
brew install upterm
```

### Standalone

`upterm` can be easily installed as an executable. Download the latest [compiled binaries](https://github.com/jingweno/upterm/releases) and put it in your executable path.

### From source

```
go get -u github.com/jingweno/upterm/cmd/upterm
```

## Quick Start

```bash
# Host a terminal session by running $SHELL
# The client's input/output is attached to the host's
$ upterm host

# Display the ssh connection string
$ upterm session current
=== BO6NOSSTP9LL08DOQ0RG
Command:                /bin/bash
Force Command:          n/a
Host:                   uptermd.upterm.dev:22
SSH Session:            ssh bo6nosstp9ll08doq0rg:MTAuMC4xNzAuMTY0OjIy@uptermd.upterm.dev

# Open a new terminal and connect to the session
$ ssh bo6nosstp9ll08doq0rg:MTAuMC4xNzAuMTY0OjIy@uptermd.upterm.dev

# Host a terminal session by running $SHELL
# The client's input/output is attached to the host's.
$ upterm host

# Display the ssh connection string
$ upterm session current
=== BO6NOSSTP9LL08DOQ0RG
Command:                /bin/bash
Force Command:          n/a
Host:                   uptermd.upterm.dev:22
SSH Session:            ssh bo6nosstp9ll08doq0rg:MTAuMC4xNzAuMTY0OjIy@uptermd.upterm.dev

# Open a new terminal and connect to the session
$ ssh bo6nosstp9ll08doq0rg:MTAuMC4xNzAuMTY0OjIy@uptermd.upterm.dev

# Host a session with a custom command.
# The client's input/output is attached to the host's.
$ upterm host -- docker run --rm -ti ubuntu bash

# Host a session by running 'tmux new -t pair-programming'.
# The host runs 'tmux attach -t pair-programming' after the client joins the session.
# The client's input/output is attached to this command's.
$ upterm host --force-command 'tmux attach -t pair-programming' -- tmux new -t pair-programming`,
```

More advanced usage is [here](https://github.com/jingweno/upterm/blob/master/docs/upterm.md).

## Demo

[![asciicast](https://asciinema.org/a/AnXTj0pOOtvSWALjUIQ63OKDm.svg)](https://asciinema.org/a/AnXTj0pOOtvSWALjUIQ63OKDm)

## How it works

You run the `upterm` program and specify the command for your terminal session.
Upterm starts an SSH server (a.k.a. `sshd`) in the host machine and sets up a reverse SSH tunnel to a [Upterm server](https://github.com/jingweno/upterm/tree/master/cmd/uptermd) (a.k.a. `uptermd`).
Clients connect to your terminal session over the public internet via `uptermd` using `ssh`.
A community Upterm server is running at `uptermd.upterm.dev` and `upterm` points to this server by default.

![upterm flowchart](https://raw.githubusercontent.com/jingweno/upterm/gh-pages/upterm-flowchart.png)

## License

[Apache 2.0](https://github.com/jingweno/upterm/blob/master/LICENSE)
