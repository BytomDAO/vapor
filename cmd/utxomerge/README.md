tool use

params

```shell
merge utxo.

Usage:
  utxomerge [flags]

Flags:
      --account_id string   The accountID of utxo needs to be merged
      --address string      The received address after merging utxo
      --amount uint         Total amount of merged utxo
  -h, --help                help for utxomerge
      --host_port string    The url for the node. Default:http://127.0.0.1:9889 (default "http://127.0.0.1:9889")
      --password string     Password of the account
```

example:

```shell
./votereward reward --reward_start_height 6000 --reward_end_height 7200
```