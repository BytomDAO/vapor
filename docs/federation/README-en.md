# Federation

To run a federation node, you will need to:

1. init a MySQL database with this [schema](./federation.sql);
2. run a `bytomd` node;
3. run a `vapord` node and import the federation private key;
4. and last but not least, run a `fedd` node with a `fed_cfg.json`.

A `fed_cfg.json` would look like this:

```json
{
    "gin-gonic" : {
        "listening_port" : 3000,
        "is_release_mode": false
    },
    "mysql" : {
        "connection" : {
            "host": "127.0.0.1",
            "port": 3306,
            "username": "root",
            "password": "",
            "database": "federation"
        },
        "log_mode" : true
    },
    "collect_unsubimmited_minutes" : 5,
    "warders" : [
        {
            "position" : 1,
            "xpub" : "7f23aae65ee4307c38d342699e328f21834488e18191ebd66823d220b5a58303496c9d09731784372bade78d5e9a4a6249b2cfe2e3a85464e5a4017aa5611e47",
            "host_port" : "192.168.0.2:3000",
            "is_local" : false
        },
        {
            "position" : 1,
            "xpub" : "585e20143db413e45fbc82f03cb61f177e9916ef1df0012daa8cbf6dbb1025ce8f98e51ae319327b63505b64fdbbf6d36ef916d79e6dd67d51b0bfe76fe544c5",
            "host_port" : "127.0.0.1:3000",
            "is_local" : true
        },
        {
            "position" : 1,
            "xpub" : "b58170b51ca61604028ba1cb412377dfc2bc6567c0afc84c83aae1c0c297d0227ccf568561df70851f4144bbf069b525129f2434133c145e35949375b22a6c9d",
            "host_port" : "192.168.0.3:3000",
            "is_local" : false
        }
    ],
    "mainchain" : {
        "name" : "bytom",
        "confirmations" : 10,
        "upstream" : "http://127.0.0.1:9888",
        "sync_seconds" : 150
    },
    "sidechain" : {
        "name" : "vapor",
        "confirmations" : 100,
        "upstream" : "http://127.0.0.1:9889",
        "sync_seconds" : 5
    }
}
```
