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

The default JSON-RPC endpoint is: [http://host:port/api/v1/federation/](http://host:port/api/v1/federation/)

The response contains some meta data of:

+ success/error status, which can be told from `code` and `msg`;
+ pagination info, which can be told from `start`, `limit` and `_links` (`_links` is used to look up the preceding and the succeeding items);

and looks like:
```
{
  "code":200,
  "msg":"",
  "result":{
    "_links":{
    },
    "data":...,
    "limit":10,
    "start":0
  }
}
```

If a request succeed, `data` field contains the detailed result as an object or as an array of objects.

### Pagination

Append `?start=<integer>&limit=<integer>` to the url in order to use pagination.

### Methods

#### `/list-crosschain-txs`

To list cross-chain transactions and filter the transactions.

##### Parameters

<!--  -->

Optional:

- `Object` - *filter*, transactions filter.
    + Optional
        * `String` - *status*, transactions status, which can be `pending` or `completed`.
        * `String` - *from*, transactions source chain, which can be `bytom` or `vapor`.
        * `String` - *source_tx_hash*, souce transaction hash string.
        * `String` - *dest_tx_hash*, destination transaction hash string.
- `Object` - *sort*, transactions sorter.
    + Optional
        * `String` - *order*, transactions order sorter, which can be `asc` or `desc`.


##### Returns


`Object`:

- `Integer` - *source_block_height*, block height of the cross-chain transaction on the source chain.
- `String` - *source_block_hash*, block hash of the cross-chain transaction on the source chain.
- `Integer` - *source_tx_index*, transaction index in the source block.
- `String` - *source_tx_hash*, source transaction hash.
- `Oject` - *dest_block_height*, block height of the cross-chain transaction on the destination chain.
    + `Integer` - *Int64*, destination block height if cross-chain transaction completed. 
    + `Boolean` - *Valid*, `true` if cross-chain transaction completed.
- `Oject` - *dest_block_hash*, destination block hash of the cross-chain transaction on the destination chain.
    + `String` - *String*, destination block hash if cross-chain transaction completed. 
    + `Boolean` - *Valid*, `true` if cross-chain transaction completed.
- `Oject` - *dest_tx_index*, transaction index in the destination block.
    + `Integer` - *Int64*, destination transaction index if cross-chain transaction completed. 
    + `Boolean` - *Valid*, `true` if cross-chain transaction completed.
- `Oject` - *dest_tx_hash*, destination transaction hash.
    + `Integer` - *String*, destination transaction hash if cross-chain transaction completed. 
    + `Boolean` - *Valid*, `true` if cross-chain transaction completed.
- `Integer` - *status*, cross-chain transaction status, `1` for `pending` and `2` for `completed`.
- `Array of objects` - *crosschain_requests*, asset transfer details per request included in the cross-chain transaction.
    + `Integer` - *amount*, asset transfer amount.
    + `Object` - *asset*, asset detail.
        * `String` - *asset_id*, asset id string.

##### Example

```js
// Request
curl -X POST 127.0.0.1:3000/api/v1/federation/list-crosschain-txs -d '{}'

// Result
{
  "code":200,
  "msg":"",
  "result":{
    "_links":{

    },
    "data":[
      {
        "source_block_height":174,
        "source_block_hash":"569a3a5a43910ea634a947fd092bb3085359db451235ae59c20daab4e4b0d274",
        "source_tx_index":1,
        "source_tx_hash":"584d1dcc4dfe741bb3ae5b193896b08db469169e6fd76098eac132af628a3183",
        "dest_block_height":{
          "Int64":0,
          "Valid":false
        },
        "dest_block_hash":{
          "String":"",
          "Valid":false
        },
        "dest_tx_index":{
          "Int64":0,
          "Valid":false
        },
        "dest_tx_hash":{
          "String":"",
          "Valid":false
        },
        "status":1,
        "crosschain_requests":[
          {
            "amount":1000000,
            "asset":{
              "asset_id":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
            }
          }
        ]
      }
    ],
    "limit":10,
    "start":0
  }
}
```
