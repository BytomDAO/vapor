json中的actions是一个数组，每个数组中都有一个type类型，account_id是钱包中账户的id

# vote

Example

```shell
curl -X POST http://ip:port/build-transaction -d '{"base_transaction":null,"actions":[{"account_id":"0BF63M2U00A04","amount":20000000,"asset_id":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff","type":"spend_account"},{"account_id":"0BF63M2U00A04","amount":99,"asset_id":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff","type":"spend_account"},{"amount":99,"asset_id":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff","address":"sm1qt3rl8gxa8c0fj4h7tv8cjurja5elnmaeu5e2su","vote":"af594006a40837d9f028daabb6d589df0b9138daefad5683e5233c2646279217294a8d532e60863bcf196625a35fb8ceeffa3c09610eb92dcfb655a947f13269","type":"vote_output"}],"ttl":0,"time_range": 43432}'
```

以上实例中：

第一个action的type:spend_account 是手续费

第二个action的type:spend_account 是要投多少票

第三个action的type:vote_output是投票的输出

1、vote_output的amout 与 第二个spend_account的amount相同

2、address指的是锁定到哪个地址

​    地址自己的：投票

​    地址别人的：转账+投票

3、vote

   被投票的公钥



# cancel vote

Example

```shell
curl -X POST http://ip:port/build-transaction -d '{"base_transaction":null,"actions":[{"account_id":"0BF63M2U00A04","amount":20000000,"asset_id":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff","type":"spend_account"},{"account_id":"0BF63M2U00A04","amount":99,"asset_id":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff","vote":"af594006a40837d9f028daabb6d589df0b9138daefad5683e5233c2646279217294a8d532e60863bcf196625a35fb8ceeffa3c09610eb92dcfb655a947f13269","type":"veto"},{"amount":99,"asset_id":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff","address":"bm1q50u3z8empm5ke0g3ngl2t3sqtr6sd7cepd3z68","type":"control_address"}],"ttl":0,"time_range": 43432}'
```

以上实例中：

第一个action的type:spend_account 是手续费

第二个action的type:veto  取消投票，vote 被取消投票的公钥

第三个action的type:control_address 还是以前的输出