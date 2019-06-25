# Federation

To run a federation node, you will need to:

1. init a MySQL database with this [schema](./federation.sql);
2. run a `bytomd` node;
3. run a `vapord` node and import the federation private key;
4. and last but not least, run a `fedd` node with a `fed_cfg.json`.

A `fed_cfg.json` would look like this:

```json
{
    "api" : {
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
    "warders" : [
        {
            "position" : 1,
            "xpub" : "50ef22b3a3fca7bc08916187cc9ec2f4005c9c6b1353aa1decbd4be3f3bb0fbe1967589f0d9dec13a388c0412002d2c267bdf3b920864e1ddc50581be5604ce1"
        }
    ],
    "quorum": 1,
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
