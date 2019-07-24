A `reward.json` would look like this:

```json
{
  "node_ip": "http://127.0.0.1:9889",
  "chain_id": "solonet",
  "mysql": {
    "connection": {
      "host": "192.168.30.186",
      "port": 3306,
      "username": "root",
      "password": "123456",
      "database": "reward"
    },
    "log_mode": false
  },
  "reward_config": {
    "xpub": "9742a39a0bcfb5b7ac8f56f1894fbb694b53ebf58f9a032c36cc22d57a06e49e94ff7199063fb7a78190624fa3530f611404b56fc9af91dcaf4639614512cb64",
    "account_id": "bd775113-49e0-4678-94bf-2b853f1afe80",
    "password": "123456",
    "reward_ratio": 20,
    "mining_address": "sp1qfpgjve27gx0r9t7vud8vypplkzytgrvqr74rwz"
  }
}
```



tool use

params

```shell
distribution of reward.

Usage:
  reward [flags]

Flags:
      --config_file string         config file. default: reward.json (default "reward.json")
  -h, --help                       help for reward
      --reward_end_height uint     The end height of the distributive income reward interval (default 2400)
      --reward_start_height uint   The starting height of the distributive income reward interval (default 1200)

```

example:

```shell
./votereward reward --reward_start_height 6000 --reward_end_height 7200 --chain_id solonet
```

