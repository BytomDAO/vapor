# Vapor 侧链solonet部署说明

## 启动 vapor

consensus.json内容如下:

```json
    {
        "consensus":{
            "consensus_type": "dpos" ,
            "period": 3,
            "max_signers_count": 7,
            "min_boter_balance": 1000000000,
            "genesis_timestamp": 1524549600,
            "coinbase": "vsm1qkm743xmgnvh84pmjchq2s4tnfpgu9ae2f9slep",
            "xprv": "a8e281b615809046698fb0b0f2804a36d824d48fa443350f10f1b80649d39e5f1e85cf9855548915e36137345910606cbc8e7dd8497c831dce899ee6ac112445",
            "signers": [
                "vsm1qkm743xmgnvh84pmjchq2s4tnfpgu9ae2f9slep"
            ]
        }
    }
```

```shell
./vapor init --chain_id solonet -r "side_chain"
./vapor node -r "side_chain" --consensus_config_file consensus.json
```

## 获取公私钥

```shell
curl -s -X POST -d '{}' http://127.0.0.1:8888/create-key-pair  > key_pair

注: 公私钥用来生成主链上锁定资产以及解锁资产的合约地址
```

## 停止vapor并删除数据目录

```shell
rm -rf side_chain
```



## 启动 bytomd、vapor

- bytomd
```shell
./bytomd init --chain_id solonet -r "main_chain"
./bytomd node -r "main_chain"
```

- vapor
  fedpeg_xpubs、sign_block_xpubs、signer为上面获取的公私钥
```shell
xprv=$(cat key_pair | jq ".data.xprv" | sed "s/\"//g")
xpub=$(cat key_pair | jq ".data.xpub" | sed "s/\"//g")

./vapor init --chain_id solonet -r "side_chain"

./vapor node -r "side_chain" --auth.disable --side.fedpeg_xpubs $xpub  --consensus_config_file consensus.json --validate_pegin true --side.parent_genesis_block_hash "a97a7a59e0e313f9300a2d7296336303889930bfdf5a80d8a9b05db343c03380"
```

## 启动侧链工具

体验的主链与侧链交互的工具的使用如下：

拷贝上面生成key_pair文件到目录tools/side_chain_tool/

* [参考侧链工具README](../../tools/side_chain_tool/README.md)

## Bytom----->Vapor
- 工具页面输入侧链账户ID，获取mainchain_address(主链锁定地址)、claim_script(赎回脚本)

  ![pegin-address](pegin-address.png)

- 在主链的dashboard，发送btm到mainchain_address 或者启动monitor_tx自动处理claim tx

- 工具页面赎回交易填入参数，发送交易

  ![tosidechain](tosidechain.png)

Vapor----->Bytom

- 在主链的dashboard，新建一个主链地址，并备份

- 在侧链的dashboard，导入主链的备份，找到与主链新建地址相同的ctrlProgram的地址，并发送交易到这个地址

- 在侧链的dashboard上retire上面地址的资产

- 工具网页的侧链发送回主链的页面填入参数，发送交易

  ![tomain](tomain.png)



## 注册出块候选人

```shell
`curl -s -X POST -d '{"base_transaction":null,"actions":[{"address":"vsm1qndq3w79kwtk9acnuswxlwxjqweglwhg8yrzp2c","amount":100000000, "asset_id":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff","name":"test-node1","dpos_type":1,"type":"dpos"}],"ttl":0,"time_range":43432}' http://127.0.0.1:8888/build-transaction`
```



## 用户投票给候选人

```shell
`curl -s -X POST -d '{"base_transaction":null,"actions":[{"address":"vsm1qkm743xmgnvh84pmjchq2s4tnfpgu9ae2f9slep","amount":100000000, "asset_id":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff","forgers":["vsm1qndq3w79kwtk9acnuswxlwxjqweglwhg8yrzp2c", "vsm1q93jcjhwe62n5mdtym6m7utle95erd6s3jsn4tn","vsm1qtu926tcsky876hflm93getsv27w7pccv4jg2fs"],"dpos_type":2,"type":"dpos"}],"ttl":0,"time_range":43432}' http://127.0.0.1:8888/build-transaction`
```



## 用户取消投票

```shell
`curl -s -X POST -d '{"base_transaction":null,"actions":[{"address":"vsm1qkm743xmgnvh84pmjchq2s4tnfpgu9ae2f9slep","amount":100000000, "asset_id":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff","forgers":["vsm1qndq3w79kwtk9acnuswxlwxjqweglwhg8yrzp2c", "vsm1q93jcjhwe62n5mdtym6m7utle95erd6s3jsn4tn","vsm1qtu926tcsky876hflm93getsv27w7pccv4jg2fs"],"dpos_type":3,"type":"dpos"}],"ttl":0,"time_range":43432}' http://127.0.0.1:8888/build-transaction`
```

