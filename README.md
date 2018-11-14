Vapor
====

[![Build Status](https://travis-ci.org/Bytom/bytom.svg)](https://travis-ci.org/Bytom/vapor) [![AGPL v3](https://img.shields.io/badge/license-AGPL%20v3-brightgreen.svg)](./LICENSE)

**Golang implemented sidechain for Bytom.**

## What is Vapor?

Vapor is software designed to extensions to the Bytom protocol, which allows partipicants to define, issue and transfer digitial assets on a multi-asset shared ledger or networks pegged to Bytom as a sidechain or run as a standalone blockchain.


## Building from source

### Requirements

- [Go](https://golang.org/doc/install) version 1.8 or higher, with `$GOPATH` set to your preferred directory

### Installation

Ensure Go with the supported version is installed properly:

```bash
$ go version
$ go env GOROOT GOPATH
```

- Get the source code

``` bash
$ git clone https://github.com/bytom/vapor.git $GOPATH/src/github.com/vapor
```

- Build source code

``` bash
$ cd $GOPATH/src/github.com/vapor
$ make vapor    # build vapor
$ make vaporcli  # build vaporcli
```

When successfully building the project, the `vapor` and `vaporcli` binary should be present in `cmd/vapor` and `cmd/vaporcli` directory, respectively.

### Executables

The Vapor project comes with several executables found in the `cmd` directory.

| Command      | Description                                                  |
| ------------ | ------------------------------------------------------------ |
| **vapor**   | vapor command can help to initialize and launch vapor domain by custom parameters. `vapor --help` for command line options. |
| **vaporcli** | Our main Vapor CLI client. It is the entry point into the Vapor network (main-, test- or private net), capable of running as a full node archive node (retaining all historical state). It can be used by other processes as a gateway into the Vapor network via JSON RPC endpoints exposed on top of HTTP, WebSocket and/or IPC transports. `vaporcli --help`. |

## Running vapor

Currently, vapor is still in active development and a ton of work needs to be done, but we also provide the following content for these eager to do something with `vapor`. This section won't cover all the commands of `vapor` and `vaporcli` at length, for more information, please the help of every command, e.g., `vaporcli help`.

### Initialize

First of all, initialize the node:

```bash
$ cd ./cmd/vapor
$ ./vapor init --chain_id mainnet
```

There are three options for the flag `--chain_id`:

- `mainnet`: connect to the mainnet.
- `testnet`: connect to the testnet wisdom.
- `solonet`: standalone mode.

After that, you'll see `config.toml` generated, then launch the node.

### launch

``` bash
$ ./vapor node
```

available flags for `vapor node`:

```
      --auth.disable                            Disable rpc access authenticate
      --chain_id string                         Select network type
  -h, --help                                    help for node
      --log_file string                         Log output file
      --log_level string                        Select log level(debug, info, warn, error or fatal
      --mainchain.mainchain_rpc_host string     The address which the daemon will try to connect to validate peg-ins, if enabled. (default "127.0.0.1")
      --mainchain.mainchain_rpc_port string     The port which the daemon will try to connect to validate peg-ins, if enabled. (default "9888")
      --mainchain.mainchain_token string        The rpc token that the daemon will use to connect to validate peg-ins, if enabled.
      --mining                                  Enable mining
      --p2p.dial_timeout int                    Set dial timeout (default 3)
      --p2p.handshake_timeout int               Set handshake timeout (default 30)
      --p2p.laddr string                        Node listen address. (0.0.0.0:0 means any interface, any port) (default "tcp://0.0.0.0:46656")
      --p2p.max_num_peers int                   Set max num peers (default 50)
      --p2p.pex                                 Enable Peer-Exchange  (default true)
      --p2p.seeds string                        Comma delimited host:port seed nodes
      --p2p.skip_upnp                           Skip UPNP configuration
      --prof_laddr string                       Use http to profile bytomd programs
      --side.fedpeg_xpubs string                Change federated peg to use a different xpub.
      --side.parent_genesis_block_hash string    (default "a75483474799ea1aa6bb910a1a5025b4372bf20bef20f246a2c2dc5e12e8a053")
      --side.pegin_confirmation_depth uint      Pegin claims must be this deep to be considered valid. (default: 6) (default 6)
      --side.sign_block_xpubs string            Change federated peg to use a different xpub.
      --signer string                           The signer corresponds to xpub of signblock
      --validate_pegin                          Validate pegin claims. All functionaries must run this.
      --vault_mode                              Run in the offline enviroment
      --wallet.disable                          Disable wallet
      --wallet.rescan                           Rescan wallet
      --web.closed                              Lanch web browser or not

```

Given the `vapor` node is running, the general workflow is as follows:

- create key, then you can create account.
- send transaction, i.e., build, sign and submit transaction.
- query all kinds of information, let's say, avaliable key, account, key, balances, transactions, etc.

### Dashboard

Access the dashboard:

```
$ open http://localhost:8888/
```

### Sidechain

* [Sidechain deployment](docs/vapor-deployment.md)

### In Docker

Ensure your [Docker](https://www.docker.com/) version is 17.05 or higher.


## Contributing

Thank you for considering helping out with the source code! Any contributions are highly appreciated, and we are grateful for even the smallest of fixes!

If you run into an issue, feel free to [vapor issues](https://github.com/bytom/vapor/issues/) in this repository. We are glad to help!

## License

[AGPL v3](./LICENSE)
