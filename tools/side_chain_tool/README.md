# 主链与侧链之间交易工具
- 1.安装所有requirements.txt中的模块,`pip install -r requirements.txt`。
- 2.安装数据库迁移。输入以下命令
  * `python manager.py db init` (使用init命令创建迁移仓库)
  * `python manager.py db migrate -m "initial migration"`(migrate命令用来自动创建迁移脚本)
  * `python manager.py db upgrade`(更新数据库，第一次使用该命令会新建一个数据库，可以利用pycharm右侧的Database查看该数据库)
- 3.在本地运行程序,`python manager.py runserver -p 8000 -h 0.0.0.0`打开http://127.0.0.1:8000端口查看, 按Ctrl+C退出程序。
- 5.在web下，执行`python -m SimpleHTTPServer 8080` 打开http://127.0.0.1:8080端口查看, 按Ctrl+C退出程序。

# 操作流程
编译生成bytom-mainchain、bytomd-sidechain，放在当前目录

# 1、启动节点
```bash
./bytomd-sidechain init --chain_id solonet -r "side_chain"
./bytomd-sidechain node -r "side_chain"
```

# 2、启动主链与侧链互转的工具
安装上面python工程的启动

# 3、获取公私钥
## 3.1 在界面上获取公私钥

## 3.2 调用节点的api获取公私钥
```bash
curl -s -X POST -d '{}' http://127.0.0.1:8888/create-key-pair
```

# 4、停止节点删除side_chan目录
```bash
    rm -rf side_chain
```

# 5、运行主链、侧链
## 主链启动
```bash
./bytomd-mainchain init --chain_id solonet -r "main_chain"
./bytomd-mainchain node -r "main_chain"
```

## 侧链启动 改变fedpeg_xpubs、sign_block_xpubs、signer为上面获取的公私钥
```bash
./bytomd-sidechain init --chain_id solonet -r "side_chain"
./bytomd-sidechain node -r "side_chain" --side.fedpeg_xpubs "b52c1b65a5dd1faa5ca031051a79404709b88514a4dffb09b326a4afd5206de77f7e46daede7317af9b076663f1dba89ae1148dbc517b131a4e4a85c34dd050b"  --side.sign_block_xpubs "b52c1b65a5dd1faa5ca031051a79404709b88514a4dffb09b326a4afd5206de77f7e46daede7317af9b076663f1dba89ae1148dbc517b131a4e4a85c34dd050b" --signer "a8fc4c959b6b3e437f4a0d58fe51519bd03a0c19f254959666d776647baea9507f7e46daede7317af9b076663f1dba89ae1148dbc517b131a4e4a85c34dd050b" --validate_pegin true --side.parent_genesis_block_hash "a97a7a59e0e313f9300a2d7296336303889930bfdf5a80d8a9b05db343c03380"
```

# 6、主链转侧链
- 6.1 工具页面获取mainchain_address(主链锁定地址)、claim_script(赎回脚本)

- 6.2 在主链的dashboard，发送btm到mainchain_address

- 6.3 工具页面赎回交易填入参数，发送交易

# 7、侧链到主链

- 7.1 在侧链的dashboard，发送btm到提供的主链地址

- 7.2 工具网页的侧链发送回主链的页面填入参数，发送交易

# 脚本启动测试
```bash
./sidechain.sh
```
