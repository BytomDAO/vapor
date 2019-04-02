# dpos共识

​	在共识协议上，vapor采用了委托权益证明（DPOS）的机制。DPoS 是基于 POW 及 POS 的基础上，出现的一种新型的保障数字货币网络安全的共识算法。它既能解决 POW 在挖矿过程中产生的大量能源过耗的问题，也能避免 POS 权益分配下可能产生的“信任天平”偏颇的问题。

## DPoS 共识机制

​	其原理是让每一个持币者进行投票，选出一定数量的持币者代表,或理解为一定数量的代表节点，并由这些代表节点来完成交易验证和区块生产的工作。持币者可以随时通过投票更换这些代表，以维系链上系统的“长久纯洁性”，保证该协议有充分的去中心化程度。

​	在目前区块链的实现中DPoS共识只用于账户模型， UTXO模型与DPos的结合也会有许多额外的优势，UTXO 模型是存放记录的一种方式，用于交易存储、组织及验证；DPoS 是一种共识算法，用于保证在分布式网络中参与者也可以对交易数据取得一致认识。

## 时间戳

​	UTXO 和 DPoS 结合的一大难点在于时间戳，DPoS 共识基于时间，会严格检查区块时间。全节点系统时间必须设置为和标准时间一样，否则共识一致性会出现问题。而 UTXO 本身也记录了时间戳的功能，但时间戳并不基于标准时间。在 LBTC 里将时间戳统一成标准时间协议，以保证区块的正常运行。当存在作恶节点或者时间不同步的区块时，出块被作为异常块处理，出块节点被作为异常节点处理

## 数据快照

​	在 UTXO 模型中，并不支持查询地址余额的功能，是通过全局遍历 UTXO 数据，实时计算地址余额。实时计算的工作量相当巨大，现实中不具备可行性。为了 DPoS 算法的需要，vapor中新增地址余额计算、节点注册、节点投票新功能。考虑到共识算法的高性能要求、注册节点数目的有限性，把地址余额、节点注册及投票信息保存在内存中，并把数据回写到db。通过数据库和地址余额、投票信息来链接 UTXO 记账信息和 DPoS 共识机制：

- 注册、投票的信息由vapor底层协议负责传输。
- 把注册、投票信息保存在内存以及db中。
- DPoS 共识模块查看注册、投票信息，完成共识。



## 业务流程

1、vapor侧链启动，由创始块中超级出块人出块

2、用户从主链转移资产到侧链，并注册为候选出块人

3、用户投票给候选出块人，赛选出块人

## dpos逻辑

1、交易格式

~~~json

        ```
        {
        "base_transaction":null,
        "actions":[
            {
                "address":"vsm1qndq3w79kwtk9acnuswxlwxjqweglwhg8yrzp2c",
                "amount":100000000,
                "asset_id":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
                "name":"test-node1",
                "dpos_type":1,
                "type":"dpos"
            }
        ],
        "ttl":0,
        "time_range":43432
        }
        ```
~~~

​	dpos_type: 1代表注册为候选出块人，2代表投票，3代表取消投票

​        type: dpos表示跟共识有关系的交易

​	amount: 表示注册交易的交易费，目前是1btm

2、逻辑说明

​	(1)、检查交易费在用户地址是否够用，序列化的注册、注册类型(用op表示)序列化后放入tx的referenceData

​	(2)、在内核chain做saveBlock之前做dpos验证

- 验证块的时间-当前时间的>出块时间间隔
- 通过上一个块与当前出的块的时间获取出块轮数以及当前轮的出块索引，判断当前出块的顺序是否正确

​	(3)、验证不可逆的区块时候有效(针对同步)

​	(4)、验证block中的候选人列表时候正确，并且生成不可逆区块

- 如果当前block与上一个不是同一个出块的轮，验证出块人列表，并确认不可逆的区块
- 如果在同一个出块轮，验证出块顺序以及出块人列表。
- 验证当前的出块人是否正确

​	(5)、验证交易完成后，先计算用户的余额，在处理与dpos相关的交易(注册、投票、取消投票)

### 计算余额

统计交易的输入输出计算每个地址的余额

### 注册逻辑

1、判断注册交易的交易费是否大于RegisrerForgerFee

2、判断注册的名字是否已经注册，存在则注册失败

3、判断注册的地址是否已经注册，存在则注册失败

4、否则，添加到name、address的注册列表

### 投票逻辑

1、判断投票人的投票的出块人是否大于MaxNumberOfVotes

2、判断被投票人地址是否已经注册

3、判断被投票人是否被投票人投过

4、判断通过写入到投票人、被投票人的投票列表

### 取消投票逻辑

1、判断被投票人是否已经投票，投过就从被投票人列表中删除

2、从投票人列表删除被投票人信息



### dpos注册、投票、取消投票的数据结构

1、交易结构ReferenceData的数据

type DposMsg struct {

​    Type vm.Op  `json:"type"`

​    Data []byte `json:"data"`

}

Data：是以下的数据结构的序列化

Type：   

​    OP_DELEGATE Op = 0xd0

​    OP_REGISTE  Op = 0xd1

​    OP_VOTE     Op = 0xd2

​    OP_REVOKE   Op = 0xd3

2、coinbase交易中的出块人列表

// DELEGATE_IDS PUBKEY SIG(block.time)

type DelegateInfoList struct {

​    Delegate DelegateInfo       `json:"delegate"`

​    Xpub     chainkd.XPub       `json:"xpub"`

​    SigTime  chainjson.HexBytes `json:"sig_time"`

}

type Delegate struct {

​    DelegateAddress string `json:"delegate_address"`

​    Votes           uint64 `json:"votes"`

}

type Delegate struct {

​    DelegateAddress string `json:"delegate_address"`

​    Votes           uint64 `json:"votes"`

}



3、注册出块人的信息

type RegisterForgerData struct {

​    Name string `json:"name"`

}

4、投票的信息

type VoteForgerData struct {

​    Forgers []string `json:"forgers"`

}

5、取消投票的信息

type CancelVoteForgerData struct {

​    Forgers []string `json:"forgers"`

}

