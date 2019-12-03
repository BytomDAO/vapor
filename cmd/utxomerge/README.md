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
./utxomerge --host_port http://127.0.0.1:9889 --account_id 9e54300d-f81d-4c5f-bef3-4e771042d394 --password 123456 --address sp1q8u7xu3e389awrnct0x4flx0h3v7mrfnmpu858p --amount 200000000000
```