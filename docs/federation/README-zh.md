# Federation

## 发起 主链转侧链 交易
federation 节点监测到 主链转侧链 跨链请求（即有资产打入 federation 多签地址）时，构建 主链转侧链 交易。

构建 主链转侧链 交易，和使用 bytomd rpc 的方法相同，也是调用 [build-transaction](https://github.com/Bytom/bytom/wiki/API-Reference#build-transaction) 接口。

构建 主链转侧链 交易需要使用的 action 为 `cross_chain_in`。相应的参数为：

| 参数 | 类型 | 含义 |
| - | - | - |
| asset_id | hash string | 资产ID |
| issuance_program | hexdecimal string | 发行该资产时定义的 issuance_program，用于校验 asset_id |
| raw_definition_byte | hexdecimal string | 发行该资产时定义的 raw_definition_byte，用于校验 asset_id |
| amount | int | 资产数量 |
| source_id | hash string | 主链上锁到 federation 多签地址的 output 的 mux_id，用于和 source_pos 一起确定一个 output 同时计算出 outputid |
| source_pos | int | 主链上锁到 federation 多签地址的 output 的 source_position，用于和 source_id 一起确定一个 output 同时计算出 outputid |

然后需要使用 `control_address` action 将 资产转入 用户侧链地址。相应的参数为：

| 参数 | 类型 | 含义 |
| - | - | - |
| asset_id | hash string | 资产ID |
| amount | int | 资产数量 |
| address | string | 用户侧链上的地址，由 federation 解析主链上用户 control program 而来 (通过解析跨链请求的 input 计算出对应侧链地址) |

注意，因为该类型不收取手续费，所以输出总额可以等于输入总额（但输出总额不应大于输入总额）。

样例请求如下：
```
{
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
            "asset_id":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
            "amount":20000000,
            "address":"vp1qfk5rudmkfny6x5enzxzj0qks3ce9dvgmyg4d2h",
            "type":"control_address"
        }
    ]
}
```

对交易进行签名的方法，和使用 bytomd rpc 的方法相同，参见 [sign-transaction](https://github.com/Bytom/bytom/wiki/API-Reference#sign-transaction)。

federation 节点之间相互通信，完成签名，然后将签完名的交易提交给 vapord 节点。

## 发起 侧链转主链 交易

侧链用户想将资产转回主链时，构建 侧链转主链 交易。

构建 侧链转主链 交易，和使用 bytomd rpc 的方法相同，也是调用 [build-transaction](https://github.com/Bytom/bytom/wiki/API-Reference#build-transaction) 接口。

构建 侧链转主链 交易需要使用的 action 为 `cross_chain_out`。相应的参数为：

| 参数 | 类型 | 含义 |
| - | - | - |
| asset_id | hash string | 资产ID |
| amount | int | 资产数量 |
| address | string | 资产转移回主链时的目标主链地址 |

样例请求如下：
```
{
    "base_transaction":null,
    "actions":[
        {
            "asset_id":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
            "amount":9000000,
            "account_id":"0TAOSK9J00A02",
            "type":"spend_account"
        },
        {
            "asset_id":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
            "amount":8000000,
            "address":"bm1q3yt265592czgh96r0uz63ta8fq40uzu5a8c2h0",
            "type":"cross_chain_out"
        }
    ]
}
```

用户对这笔交易进行签名，提交给 vapord 节点。

federation 节点监测到这笔交易时（通过判断 vapor 侧链上的交易是否含有 CrossChainOutput 这种类型的 output），构造主链解锁交易，将资产转入对应地址。federation 节点之间相互通信，完成签名，然后将签完名的交易提交给 bytomd 节点。
