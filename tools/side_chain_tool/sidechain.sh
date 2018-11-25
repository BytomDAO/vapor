#!/bin/bash

if [ ! -f "key_pair" ];then
./vapor init --chain_id solonet -r "side_chain"
nohup ./vapor node -r "side_chain" > /dev/null &
sleep 30
curl -s -X POST -d '{}' http://127.0.0.1:8888/create-key-pair > key_pair
ps -ef | grep vapor | grep -v grep | awk  '{print $2}' |xargs  kill -9
rm -rf side_chain
fi

xprv=$(cat key_pair | jq ".data.xprv" | sed "s/\"//g")
xpub=$(cat key_pair | jq ".data.xpub" | sed "s/\"//g")
ps -ef | grep vapor | grep -v grep | awk  '{print $2}' |xargs  kill -9
ps -ef | grep bytomd | grep -v grep | awk  '{print $2}' |xargs  kill -9

./bytomd init --chain_id solonet -r "main_chain"
nohup ./bytomd node -r "main_chain" --auth.disable > /dev/null &
sleep 50

./vapor init --chain_id solonet -r "side_chain"
nohup ./vapor node -r "side_chain" --auth.disable --side.fedpeg_xpubs $xpub  --side.sign_block_xpubs $xpub --signer $xprv --validate_pegin true --side.parent_genesis_block_hash "a97a7a59e0e313f9300a2d7296336303889930bfdf5a80d8a9b05db343c03380" > /dev/null &
sleep 30
virtualenv --no-site-packages venv
source venv/bin/activate
if [ ! -f "install" ];then
pip install -r requirements.txt
python manager.py db init
python manager.py db migrate -m "initial migration"
python manager.py db upgrade
touch install
fi

nohup python manager.py runserver -p 8080 -h 0.0.0.0 > /dev/null &
sleep 30
cd web
nohup python -m SimpleHTTPServer 80 > /dev/null &
