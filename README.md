Vapor
======

[![Build Status](https://travis-ci.org/Bytom/vapor.svg)](https://travis-ci.org/Bytom/vapor) [![AGPL v3](https://img.shields.io/badge/license-AGPL%20v3-brightgreen.svg)](./LICENSE)

**Golang implemented sidechain for Bytom.**

## Requirements

- [Go](https://golang.org/doc/install) version 1.11 or higher, with `$GOPATH` set to your preferred directory

## Get source code

```
$ git clone https://github.com/Bytom/vapor.git $GOPATH/src/github.com/vapor
```

Then, you have two ways to get vapor executable file:

1. compile source code
2. build it using Docker

## Installation

```
$ cd $GOPATH/src/github.com/vapor
$ make install
```

## Run Directly

Firstly, you need initialize node:

```
$ vapord init --chain_id=vapor --home <vapor-data-path>
```

For example, you can store vapor data in `$HOME/bytom/vapor`:

```
$ vapord init --chain_id=vapor --home $HOME/bytom/vapor
```

Then, start your node:

```
$ vapord node --home <vapor-data-path>
```

## Running in Docker

### Build the image

```
$ cd $GOPATH/src/github.com/vapor
$ docker build -t vapor .
```

### Enter the iterative mode

```
$ docker run -it --net=host -v <vapor/data/directory/on/host/machine>:/root/.vapor vapor:latest
```

vapor data directory has three config files:

- `config.toml`
- `federation.json`
- `node_key.txt`

Use `exit` to exit Docker's iterative mode.

### Daemon mode

For example,

```bash
$ docker run -d --net=host -v <vapor/data/directory/on/host/machine>:/root/.vapor vapor:latest vapord node --web.closed --auth.disable
```

To list the running containners and check their container id, image, corresponding command, created time, status, name and ports being used:

```
$ docker container ls
```

or

```
$ docker ps
```

To execute a command inside a containner, for example:

```
$ docker exec -it <containerId> vaporcli create-access-token <tokenId>
```

To stop a running containner:

```
$ docker stop <containerId>
```

To remove a containner:

```
$ docker rm <containerId>
```

## License

[AGPL v3](./LICENSE)
