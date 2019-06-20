Vapor
======

[![Build Status](https://travis-ci.org/Bytom/vapor.svg)](https://travis-ci.org/Bytom/vapor) [![AGPL v3](https://img.shields.io/badge/license-AGPL%20v3-brightgreen.svg)](./LICENSE)

**Golang implemented sidechain for Bytom.**

## Requirements

- [Go](https://golang.org/doc/install) version 1.8 or higher, with `$GOPATH` set to your preferred directory

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

## Run

Firstly, you need initialize node:

```
$ bytomd init --chain_id=vapor --home <vapor-data-path>
```

For example, you can store vapor data in `$HOME/bytom/vapor`:

```
$ bytomd init --chain_id=vapor --home $HOME/bytom/vapor
```

And some files in `<vapor-data-path>`:

```
$ cd <vapor-data-path>
$ tree -L 1
.
├── LOCK
├── config.toml
├── data
├── federation.json
├── keystore
└── node_key.txt
```

`config.toml` save vapor network info, like:

```
$ cat config.toml
# This is a TOML config file.
# For more information, see https://github.com/toml-lang/toml
fast_sync = true
db_backend = "leveldb"
api_addr = "0.0.0.0:9889"
moniker = ""
chain_id = "vapor"
[p2p]
laddr = "tcp://0.0.0.0:56659"
seeds = "52.83.133.152:56659"
```

`federation.json` save relayed node xpub, like:

```
$ cat federation.json
{
  "xpubs": [
    "50ef22b3a3fca7bc08916187cc9ec2f4005c9c6b1353aa1decbd4be3f3bb0fbe1967589f0d9dec13a388c0412002d2c267bdf3b920864e1ddc50581be5604ce1"
  ],
  "quorum": 1
}
```

Then, start your node:

```
$ bytomd node --home <vapor-data-path>
```

Solonet mode:

```
$ bytomd init --chain_id=solonet --home <vapor-data-path>
$ bytomd node --home <vapor-data-path>
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

Then you can use bytomd and bytomcli following [Bytom Wiki](https://github.com/Bytom/bytom/wiki/Command-Line-Options).

Use `exit` to exit Docker's iterative mode.

### Daemon mode

For example,

```bash
$ docker run -d --net=host -v <vapor/data/directory/on/host/machine>:/root/.vapor vapor:latest bytomd node --web.closed --auth.disable
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
$ docker exec -it <containerId> bytomcli create-access-token <tokenId>
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
