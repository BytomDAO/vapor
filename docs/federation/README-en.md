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
## API
A federation node can function as an api server for querying cross-chain transactions.

Default JSON-RPC endpoints:
http://host:port/api/v1/federation

### Pagination

### Methods

#### `/list-crosschain-txs`

To list cross-chain transactions and filter the transactions.

##### Parameters

`Object`:

- `Object` - *filter*, transactions filter.
- `Object` - *sort*, transactions sorter.
<!-- 
Optional:

- `String` - *mnemonic*, mnemonic of the key, create key by specified mnemonic.
 -->


i##### Returns

`Object`:

- `String` - *alias*, name of the key.
- `String` - *xpub*, root pubkey of the key.
- `String` - *file*, path to the file of key.

Optional:

- `String` - *mnemonic*, mnemonic of the key, exist when the request mnemonic is null.

##### Example

create key by random pattern:

```js
// Request
curl -X POST create-key -d '{"alias": "alice", "password": "123456", "language": "en"}'

// Result
{
    "status": "success",
    "data": {
        "alias": "alice",
        "xpub": "c4afb96f600dc7da388b77107ceb471f604aadf49e6d1ec745abf9ae797e69a2a1f113e2cb2541166609ba725dea4072e54376ed90bcbdd0200853191a2f560a",
        "file": "/home/ec2-user/vapor_test/keystore/UTC--2019-06-18T11-04-34.512032731Z--66942e00-1466-45ea-b9f1-32ea30000017",
        "mnemonic": "attend build fog oak awful make diesel episode glove mind fire sleep"
    }
}
```

create key by specified mnemonic:

```js
// Request
curl -X POST create-key -d '{"alias":"jack", "password":"123456", "mnemonic":"please observe raw beauty blue sea believe then boat float beyond position", "language":"en"}'

// Result
{
    "status": "success",
    "data": {
        "alias": "jack",
        "xpub": "c7bcb65febd31c6d900bc84c386d95c3d5b047090628d9bf5c51a848945b6986e99ff70388018a7681fa37a240dbd8df39a994c86f9314a61e75feb33563ca72",
        "file": "/home/ec2-user/vapor_test/keystore/UTC--2019-06-18T11-10-02.390062724Z--3e5a16a3-93bf-4c81-aec1-c4279458a605"
    }
}
```

----
