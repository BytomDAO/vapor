Vapor
======

[![Build Status](https://travis-ci.org/Bytom/vapor.svg)](https://travis-ci.org/Bytom/vapor) [![AGPL v3](https://img.shields.io/badge/license-AGPL%20v3-brightgreen.svg)](./LICENSE)

**Golang implemented sidechain for Bytom.**

## Requirements

- [Go](https://golang.org/doc/install) version 1.11 or higher, with `$GOPATH` set to your preferred directory

## Get source code

```
$ cd $GOPATH/src/github.com/bytom
$ git clone https://github.com/Bytom/vapor.git
```

Then, you have two ways to get vapor executable file:

1. compile source code
2. build it using Docker

## Installation

```
$ cd $GOPATH/src/github.com/bytom/vapor
$ make install
```

## Run Directly

Firstly, you need initialize node:

```
$ vapord init --chain_id=mainnet --home <vapor-data-path>
```

The default vapor data path (on the host) is:
+ Mac: `~/Library/Application Support/Vapor`
+ Linux: `~/.vapor`
+ Windows: `%APPDATA%\Vapor`

Then, start your node:

```
$ vapord node --home <vapor-data-path>
```

## Running in Docker

### Build the image

```
$ cd $GOPATH/src/github.com/bytom/vapor
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

### Reward distribution tool

After the supernode and alternative node receive the reward from the node, they will allocate the reward

according to the interest rate. 

The reward calculation rules: 

 calculate the reward (consensus reward * interest rate * voting weight) according to the weight of votes

cast in consensus around, and choose how many rounds of consensus to allocate the reward flexibly.

[Tool usage details](./cmd/votereward/README.md)


### Merger utxo
UTXO has been merged to solve the problem that too much UTXO input causes a failed send transaction to fail. 
[details](./cmd/utxomerge/README.md)

## License

[AGPL v3](./LICENSE)
