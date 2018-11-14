# 主链与侧链之间交易工具
- 1.安装所有requirements.txt中的模块,`pip install -r requirements.txt`。
- 2.安装数据库迁移。输入以下命令
  * `python manager.py db init` (使用init命令创建迁移仓库)
  * `python manager.py db migrate -m "initial migration"`(migrate命令用来自动创建迁移脚本)
  * `python manager.py db upgrade`(更新数据库，第一次使用该命令会新建一个数据库，可以利用pycharm右侧的Database查看该数据库)
- 3.在本地运行程序,`python manager.py runserver -p 8000 -h 0.0.0.0`打开http://127.0.0.1:8000端口查看, 按Ctrl+C退出程序。
- 5.在web下，执行`python -m SimpleHTTPServer 8080` 打开http://127.0.0.1:8080端口查看, 按Ctrl+C退出程序。

# 脚本启动测试
bytomd、vapor放到本目录下，通过脚本一次启动
安装virtualenv、jq
```bash
./sidechain.sh
```
