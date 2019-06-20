# cross-chain transaction

Related API:

+ `build-transaction` / `build-chain-transactions` 
+ `sign-transaction` / `sign-transactions`
+ `submit-transaction` / `submit-transactions`
+ `list-transactions`

## mainchain(bytom) to sidechain(vapor)

To build a mainchain-to-sidechain transaction, `build-transaction` is called by one or more federation members, using `cross_chain_in` action.

### Parameters

`Object`:

- `String` - *base_transaction*, base data for the transaction, default is null.
- `Arrary of Object` - *actions*:
  - `Object`:
    - `String` - *asset_id* | *asset_alias*, (type is cross_chain_in, control_program and control_address) alias or ID of asset.
    - `Integer` - *amount*, (type is cross_chain_in, control_program and control_address) the specified asset of the amount sent with this transaction.
    - `String`- *type*, type of transaction, valid types: 'cross_chain_in', 'control_address', 'control_program'.
    - `String` - *address*, (type is control_address) address of receiver, the style of address is P2PKH or P2SH.
    - `String` - *control_program*, (type is control_program) control program of receiver.
    - `Integer` - *vm_version*, (type is cross_chain_in) asset vm_version.
    - `String` - *issuance_program*, (type is cross_chain_in) asset issuance_program hexdecimal string.
    - `String` - *raw_definition_byte*, (type is cross_chain_in) asset raw_definition_byte hexdecimal string.
    - `String` - *source_id*, (type is cross_chain_in) mainchain output mux id.
    - `Integer` - *source_pos*, (type is cross_chain_in) mainchain output source position.

### Returns

- `Object of build-transaction` - *transaction*, builded transaction.

### Example

```js
// Request
curl -X POST build-transaction -d '{
    "base_transaction":null,
    "actions":[
        {
            "asset_id":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
            "amount":20000000,
            "source_id":"d5156f4477fcb694388e6aed7ca390e5bc81bb725ce7461caa241777c1f62236",
            "source_pos":3,
            "type":"cross_chain_in"
        },
        {
            "amount":20000000,
            "asset_id":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
            "address":"vp1qfk5rudmkfny6x5enzxzj0qks3ce9dvgmyg4d2h",
            "type":"control_address"
        }
    ]
}'
```

```js
// Result
{
  "status":"success",
  "data":{
    "raw_transaction":"07010001019001008d01d5156f4477fcb694388e6aed7ca390e5bc81bb725ce7461caa241777c1f62236ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80dac409030146ae2071dcacf8495651f205858a3a4b64c4d4bb24f382ff517b558c8e0acaac5819652039579ddd54e667057e175032683c1152578c9b05b5ecbc11b3f500b4263e41885152ad01000001013e003cffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80ade204011600144da83e37764cc9a3533311852782d08e3256b11b00",
    "signing_instructions":[
      {
        "position":0,
        "witness_components":[
          {
            "type":"raw_tx_signature",
            "quorum":1,
            "keys":[
              {
                "xpub":"71dcacf8495651f205858a3a4b64c4d4bb24f382ff517b558c8e0acaac5819650abcd618c10058f867fce07b0455cdb34c273b2bf8fc7ee9e3c70706d5bba8f4",
                "derivation_path":[

                ]
              },
              {
                "xpub":"39579ddd54e667057e175032683c1152578c9b05b5ecbc11b3f500b4263e418868da2376963dbbfb18a0d8aac64ea0d97e7c82061c58e98e22d4d1c7f5f5809d",
                "derivation_path":[

                ]
              }
            ],
            "signatures":null
          }
        ]
      }
    ],
    "fee":10000000,
    "allow_additional_actions":false
  }
}
```

### Following steps

Then the federation members sign the transaction and submit it to a vapord node, using `sign-transaction` and `submit-transaction` respectively. The usages are as same as those for bytomd, see bytomd RPC document's [`sign-transaction`](https://github.com/Bytom/bytom/wiki/API-Reference#sign-transaction) and [`submit-transaction`](https://github.com/Bytom/bytom/wiki/API-Reference#submit-transaction) for detail.

## list cross-chain transactions

To list cross-chain transactions, `list-transactions` needs to be called by a vapord node.

If a transaction contains a `cross_chain_in` input or a `cross_chain_out` output, it is recognized as a cross-chain transaction.

### Parameters

`Object`:

optional:

- `String` - *id*, transaction id, hash of transaction.
- `String` - *account_id*, id of account.
- `Boolean` - *detail* , flag of detail transactions, default false (only return transaction summary)
- `Boolean` - *unconfirmed*, flag of unconfirmed transactions(query result include all confirmed and unconfirmed transactions), default false.
- `Integer` - *from*, the start position of first transaction
- `Integer` - *count*, the number of returned

### Returns

`Array of Object`, transaction array.

optional:

  - `Object`:(summary transaction)
    - `String` - *tx_id*, transaction id, hash of the transaction.
    - `Integer` - *block_time*, the unix timestamp for when the requst was responsed.
    - `Array of Object` - *inputs*, object of summary inputs for the transaction.
      - `String` - *type*, the type of input action, available option include: 'spend', 'issue', 'coinbase'.
      - `String` - *asset_id*, asset id.
      - `String` - *asset_alias*, name of asset.
      - `Integer` - *amount*, amount of asset.
      - `String` - *account_id*, account id.
      - `String` - *account_alias*, name of account.
      - `Object` - *arbitrary*, arbitrary infomation can be set by miner, it only exist when type is 'coinbase'.
    - `Array of Object` - *outputs*, object of summary outputs for the transaction.
      - `String` - *type*, the type of output action, available option include: 'retire', 'control'.
      - `String` - *asset_id*, asset id.
      - `String` - *asset_alias*, name of asset.
      - `Integer` - *amount*, amount of asset.
      - `String` - *account_id*, account id.
      - `String` - *account_alias*, name of account.
      - `Object` - *arbitrary*, arbitrary infomation can be set by miner, it only exist when type is input 'coinbase'(this place is empty).

  - `Object`:(detail transaction)
    - `String` - *tx_id*, transaction id, hash of the transaction.
    - `Integer` - *block_time*, the unix timestamp for when the requst was responsed.
    - `String` - *block_hash*, hash of the block where this transaction was in.
    - `Integer` - *block_height*, block height where this transaction was in.
    - `Integer` - *block_index*, position of the transaction in the block.
    - `Integer` - *block_transactions_count*, transactions count where this transaction was in the block.
    - `Boolean` - *status_fail*, whether the state of the transaction request has failed.
    - `Integer` - *size*, size of transaction.
    - `Array of Object` - *inputs*, object of inputs for the transaction.
      - `String` - *type*, the type of input action, available option include: 'spend', 'issue', 'coinbase'.
      - `String` - *asset_id*, asset id.
      - `String` - *asset_alias*, name of asset.
      - `Object` - *asset_definition*, definition of asset(json object).
      - `Integer` - *amount*, amount of asset.
      - `Object` - *issuance_program*, issuance program, it only exist when type is 'issue'.
      - `Object` - *control_program*, control program of account, it only exist when type is 'spend'.
      - `String` - *address*, address of account, it only exist when type is 'spend'.
      - `String` - *spent_output_id*, the front of outputID to be spent in this input, it only exist when type is 'spend'.
      - `String` - *account_id*, account id.
      - `String` - *account_alias*, name of account.
      - `Object` - *arbitrary*, arbitrary infomation can be set by miner, it only exist when type is 'coinbase'.
      - `String` - *input_id*, hash of input action.
      - `Array of String` - *witness_arguments*, witness arguments.
    - `Array of Object` - *outputs*, object of outputs for the transaction.
      - `String` - *type*, the type of output action, available option include: 'retire', 'control'.
      - `String` - *id*, outputid related to utxo.
      - `Integer` - *position*, position of outputs.
      - `String` - *asset_id*, asset id.
      - `String` - *asset_alias*, name of asset.
      - `Object` - *asset_definition*, definition of asset(json object).
      - `Integer` - *amount*, amount of asset.
      - `String` - *account_id*, account id.
      - `String` - *account_alias*, name of account.
      - `Object` - *control_program*, control program of account.
      - `String` - *address*, address of account.

### Example

list all the available transactions:

```js
// Request
curl -X POST list-transactions -d {}

// Result
{
  "status":"success",
  "data":[
    {
      "tx_id":"0dcd7312b120a1d4e3e441d7154710dd20093434a2184eda523ca003faba3039",
      "block_time":1559638198209,
      "inputs":[
        {
          "type":"spend",
          "asset_id":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
          "asset_alias":"BTM",
          "amount":10000000,
          "account_id":"0TAOSK9J00A02",
          "account_alias":"11111"
        }
      ],
      "outputs":[
        {
          "type":"control",
          "asset_id":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
          "asset_alias":"BTM",
          "amount":1000000,
          "account_id":"0TAOSK9J00A02",
          "account_alias":"11111"
        },
        {
          "type":"cross_chain_out",
          "asset_id":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
          "asset_alias":"BTM",
          "amount":4000000
        }
      ]
    },
    {
      "tx_id":"137ea7ab674f41720f731e9e0e94aeef16027a993c4fb78b0b79b7ce61584789",
      "block_time":1559638147539,
      "inputs":[
        {
          "type":"cross_chain_in",
          "asset_id":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
          "asset_alias":"BTM",
          "amount":20000000
        }
      ],
      "outputs":[
        {
          "type":"control",
          "asset_id":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
          "asset_alias":"BTM",
          "amount":10000000,
          "account_id":"0TAOSK9J00A02",
          "account_alias":"11111"
        }
      ]
    },
    {
      "tx_id":"0c0b0ed9458fd66e1ed004cfb513cef5982dc4112db0afdf106bdf7abf88c8c1",
      "block_time":1559636817158,
      "inputs":[
        {
          "type":"spend",
          "asset_id":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
          "asset_alias":"BTM",
          "amount":10000000,
          "account_id":"0TAOSK9J00A02",
          "account_alias":"11111"
        }
      ],
      "outputs":[
        {
          "type":"control",
          "asset_id":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
          "asset_alias":"BTM",
          "amount":1000000,
          "account_id":"0TAOSK9J00A02",
          "account_alias":"11111"
        },
        {
          "type":"cross_chain_out",
          "asset_id":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
          "asset_alias":"BTM",
          "amount":4000000
        }
      ]
    },
    {
      "tx_id":"75271e40894c198e7b2a0574227a056945fbaa28b1a9d5efde6268f4a8e30a3a",
      "block_time":1559636756076,
      "inputs":[
        {
          "type":"cross_chain_in",
          "asset_id":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
          "asset_alias":"BTM",
          "amount":20000000
        }
      ],
      "outputs":[
        {
          "type":"control",
          "asset_id":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
          "asset_alias":"BTM",
          "amount":10000000,
          "account_id":"0TAOSK9J00A02",
          "account_alias":"11111"
        }
      ]
    },
    {
      "tx_id":"e008984f63c2dd36a7488ba82f95e2a9caa2ed20d6fd46faa53c0b3eb51283ea",
      "block_time":1559635831740,
      "inputs":[
        {
          "type":"cross_chain_in",
          "asset_id":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
          "asset_alias":"BTM",
          "amount":20000000
        }
      ],
      "outputs":[
        {
          "type":"control",
          "asset_id":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
          "asset_alias":"BTM",
          "amount":10000000,
          "account_id":"0TAOSK9J00A02",
          "account_alias":"11111"
        }
      ]
    }
  ]
}
```


## sidechain(vapor) to mainchain(bytom)

To build a sidechain-to-mainchain transaction, `build-transaction` is called by a vapor user, using `cross_chain_out` action.

### Parameters

`Object`:

- `String` - *base_transaction*, base data for the transaction, default is null.
- `Integer` - *ttl*, integer of the time to live in milliseconds, it means utxo will be reserved(locked) for builded transaction in this time range, if the transaction will not to be submitted into block, it will be auto unlocked for build transaction again after this ttl time. it will be set to 5 minutes(300 seconds) defaultly when ttl is 0.
- `Integer` - *time_range*, the block height at which this transaction will be allowed to be included in a block. If the block height of the main chain exceeds this value, the transaction will expire and no longer be valid.
- `Arrary of Object` - *actions*:
  - `Object`:
    - `String` - *account_id* | *account_alias*, (type is spend_account) alias or ID of account.
    - `String` - *asset_id* | *asset_alias*, (type is spend_account, cross_chain_out) alias or ID of asset.
    - `Integer` - *amount*, (type is spend_account, cross_chain_out) the specified asset of the amount sent with this transaction.
    - `String`- *type*, type of transaction, valid types: 'spend_account', 'spend_account_unspent_output', 'cross_chain_out', 'control_program'.
    - `String` - *address*, (type is cross_chain_out) address of receiver, the style of address is P2PKH or P2SH.
    - `String` - *use_unconfirmed*, (type is spend_account and spend_account_unspent_output) flag of use unconfirmed UTXO, default is false.

### Returns

- `Object of build-transaction` - *transaction*, builded transaction.

### Example

```js
// Request
curl -X POST build-transaction -d '{
    "base_transaction":null,
    "actions":[
        {
            "amount":9000000,
            "asset_id":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
            "account_id":"0TAOSK9J00A02",
            "type":"spend_account",
            "use_unconfirmed":true
        },
        {
            "amount":4000000,
            "asset_id":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
            "address":"bm1q3yt265592czgh96r0uz63ta8fq40uzu5a8c2h0",
            "type":"cross_chain_out"
        }
    ]
}'
```

```js
// Result
{
  "status":"success",
  "data":{
    "raw_transaction":"07010001015f015dcca6be7f2b17d0695ed2ae5497f4711788e355ff1c3a93401c8e035a1f84a7b8ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80ade20400011600144da83e37764cc9a3533311852782d08e3256b11b010002013d003bffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc0843d011600146e544784f94bc9b6b608b561075c03804319ed4f00013e013cffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8092f401011600148916ad528556048b97437f05a8afa7482afe0b9400",
    "signing_instructions":[
      {
        "position":0,
        "witness_components":[
          {
            "type":"raw_tx_signature",
            "quorum":1,
            "keys":[
              {
                "xpub":"71dcacf8495651f205858a3a4b64c4d4bb24f382ff517b558c8e0acaac5819650abcd618c10058f867fce07b0455cdb34c273b2bf8fc7ee9e3c70706d5bba8f4",
                "derivation_path":[
                  "2c000000",
                  "99000000",
                  "01000000",
                  "00000000",
                  "01000000"
                ]
              }
            ],
            "signatures":null
          },
          {
            "type":"data",
            "value":"5b4c19722110ea7c1b5ace5d63393d401ef0797a605568a88a8fc4cdfcf293b6"
          }
        ]
      }
    ],
    "fee":5000000,
    "allow_additional_actions":false
  }
}
```

### Following steps

Then the vapor user sign the transaction and submit it to a vapord node, using `sign-transaction` and `submit-transaction` respectively. The usages are as same as those for bytomd, see bytomd RPC document's [`sign-transaction`](https://github.com/Bytom/bytom/wiki/API-Reference#sign-transaction) and [`submit-transaction`](https://github.com/Bytom/bytom/wiki/API-Reference#submit-transaction) for detail.