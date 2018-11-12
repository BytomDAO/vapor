# -*- coding:utf-8 -*-
import json
from flask import request, jsonify, make_response, current_app, render_template
#from sqlalchemy.exc import SQLAlchemyError, IntegrityError

from . import api
from .Connection import Connection
from .. import db
from ..models import KeyPair, PeginAddress


@api.route('/create_key_pair', methods=['POST', 'GET', 'OPTIONS'])
def create_key_pair():
    keypairs = KeyPair.query.all()
    if len(keypairs) > 5 :
        return json_contents(jsonify(code=200, msg="There are already 5 private-public key pairs"))

    connection = Connection("http://127.0.0.1:8888")
    response = connection.request("/create-key-pair")
    resp_json = json.loads(response.text)
    if resp_json['status'] == 'success':
        xprv = resp_json['data']['xprv']
        xpub = resp_json['data']['xpub']
        key = KeyPair(xprv=xprv,xpub=xpub)
        db.session.add(key)
        db.session.commit()
    elif resp_json['status'] == 'fail':
        return json_contents(jsonify(code=-1, msg=resp_json['msg']))
    else:
        return json_contents(jsonify(code=-1, msg="create key pair fail"))

    return json_contents(jsonify(code=200, msg=""))

@api.route('/get_key_pair',methods=['POST', 'GET', 'OPTIONS'])
def get_key_pair():
    keypairs = KeyPair.query.all()
    json_data = ""
    index = 0
    num = len(keypairs)
    for keypair in keypairs:
        data = {
            "xprv":keypair.xprv.encode('utf-8'),
            "xpub": keypair.xpub.encode('utf-8')
        }
        json_data = json_data + json.dumps(data)
        index = index + 1
        if index < num:
            json_data = json_data + ","
    return json_contents(jsonify(code=200, msg="sucess", data="[" + json_data + "]"))


@api.route('/create_pegin_address', methods=['GET', 'POST', 'OPTIONS'])
def create_pegin_address():
    if not request.json or not 'account_id' in request.json:
        return json_contents(jsonify(code=-1, msg="The json format is incorrect"))
    accountID = request.json['account_id']
    connSide = Connection("http://127.0.0.1:8888")
    body_json = {"account_id": accountID}
    response = connSide.request("/get-pegin-address",body_json)
    resp_json = json.loads(response.text)
    if resp_json['status'] == 'success':
        mainchain_address = resp_json['data']['mainchain_address']
        claim_script = resp_json['data']['claim_script']
        pegin = PeginAddress(address=mainchain_address, claim_script=claim_script)
        db.session.add(pegin)
        db.session.commit()
    elif resp_json['status'] == 'fail':
        return json_contents(jsonify(code=-1, msg=resp_json['msg']))
    else:
        return json_contents(jsonify(code=-1, msg="get pegin address fail"))
    #return render_template('peginaddress.html', mainchain_address=mainchain_address,claim_script=claim_script)
    return json_contents(jsonify(code=200, msg=""))

@api.route('/get_pegin_address',methods=['POST', 'GET', 'OPTIONS'])
def get_pegin_address():
    peginAddrs = PeginAddress.query.all()
    json_data = ""
    index = 0
    num = len(peginAddrs)
    for peginAddr in peginAddrs:
        data = {
            "mainchain_address":peginAddr.address.encode('utf-8'),
            "claim_script": peginAddr.claim_script.encode('utf-8')
        }
        json_data = json_data + json.dumps(data)
        index = index + 1
        if index < num:
            json_data = json_data + ","
    return json_contents(jsonify(code=200, msg="sucess", data="[" + json_data + "]"))


@api.route('/claim_tx', methods=['GET', 'POST', 'OPTIONS'])
def claim_tx():
    print request.json
    if not request.json or not 'claim_script' in request.json or not 'block_height' in request.json or not 'tx_id' in request.json or not 'password' in request.json:
        return json_contents(jsonify(code=-1, msg="The json format is incorrect"))
    block_height = int(request.json['block_height'])
    tx_id = request.json['tx_id'].encode('utf-8')
    password = request.json['password'].encode('utf-8')
    claim_script = request.json['claim_script'].encode('utf-8')
    connSide = Connection("http://127.0.0.1:8888")
    connMain = Connection("http://127.0.0.1:9888")
    raw_block = ""
    raw_transaction = ""
    block_hash = ""
    proof=""
    # 从主链获取raw_block
    body_json = {"block_height": block_height}
    response = connMain.request("/get-raw-block",body_json)
    resp_json = json.loads(response.text)
    if resp_json['status'] == 'success':
        raw_block = resp_json['data']['raw_block'].encode('utf-8')
    elif resp_json['status'] == 'fail':
        return json_contents(jsonify(code=-1, msg="get-raw-block: " + resp_json['msg']))
    else:
        return json_contents(jsonify(code=-1, msg="get raw block fail"))

    # 获取raw transaction
    body_json = {"tx_id": tx_id,"raw_block": raw_block}
    response = connSide.request("/get-raw-transaction",body_json)
    resp_json = json.loads(response.text.encode('utf-8'))
    if resp_json['status'] == 'success':
        raw_transaction = resp_json['data']['raw_transaction'].encode('utf-8')
        block_hash = resp_json['data']['block_hash'].encode('utf-8')
    elif resp_json['status'] == 'fail':
        return json_contents(jsonify(code=-1, msg="get-raw-transaction: " + resp_json['msg']))
    else:
        return json_contents(jsonify(code=-1, msg="get raw transaction fail"))

    # 主链获取proof
    body_json = {"tx_id": tx_id,"block_hash": block_hash}
    response = connMain.request("/get-merkle-proof",body_json)
    resp_json = json.loads(response.text)
    if resp_json['status'] == 'success':
        proof = json.dumps(resp_json['data']).strip('{}')
    elif resp_json['status'] == 'fail':
        return json_contents(jsonify(code=-1, msg="get-merkle-proof:" + resp_json['msg']))
    else:
        return json_contents(jsonify(code=-1, msg="get raw transaction fail"))

    # 调用claimtx
    body_json = '{"password": "%s","raw_transaction": "%s","claim_script":"%s",%s}' % (password,raw_transaction,claim_script,proof)
    print body_json
    response = connSide.request("/claim-pegin-transaction",json.loads(body_json))
    resp_json = json.loads(response.text)
    if resp_json['status'] == 'success':
        return json_contents(jsonify(code=200, msg=resp_json['data']))
    elif resp_json['status'] == 'fail':
        return json_contents(jsonify(code=-1, msg="claim-pegin-transaction: " + resp_json['msg']))
    else:
        return json_contents(jsonify(code=-1, msg="claim pegin transaction fail"))


@api.route('/send_to_mainchain', methods=['GET', 'POST', 'OPTIONS'])
def send_to_mainchain():
    #if not request.json or not 'claim_script' in request.json or not 'xprvs' in request.json or not 'tx_id' in request.json or not 'id' in request.json or not 'side_tx_id' in request.json or not 'block_height' in request.json or not 'side_block_height' in request.json or not 'alias' in request.json or not 'root_xpubs' in request.json or not 'address' in request.json or not 'control_program' in request.json:
    if not request.json or not 'claim_script' in request.json or not 'tx_id' in request.json or not 'id' in request.json or not 'side_tx_id' in request.json or not 'block_height' in request.json or not 'side_block_height' in request.json or not 'alias' in request.json or not 'address' in request.json or not 'control_program' in request.json:
        return json_contents(jsonify(code=-1, msg="The json format is incorrect"))

    connSide = Connection("http://127.0.0.1:8888")
    connMain = Connection("http://127.0.0.1:9888")
    tx_id = request.json['tx_id'].encode('utf-8')
    id = request.json['id'].encode('utf-8')
    block_height = int(request.json['block_height'])
    claim_script = request.json['claim_script'].encode('utf-8')

    utxo = ""
    # 从主链获取raw_block
    body_json = {"block_height": block_height}
    response = connMain.request("/get-raw-block",body_json)
    resp_json = json.loads(response.text)
    if resp_json['status'] == 'success':
        raw_block = resp_json['data']['raw_block'].encode('utf-8')
    elif resp_json['status'] == 'fail':
        return json_contents(jsonify(code=-1, msg="get-raw-block: " + resp_json['msg']))
    else:
        return json_contents(jsonify(code=-1, msg="get raw block fail"))

    # 获取主链上的utxo
    address = request.json['address'].encode('utf-8')
    body_json = {"tx_id": tx_id, "id": id, "raw_block": raw_block, "address": address}
    response = connSide.request("/get-utxo-from-transaction",body_json)
    resp_json = json.loads(response.text)
    if resp_json['status'] == 'success':
        utxo = json.dumps(resp_json['data']).strip('{}')+"}"
    elif resp_json['status'] == 'fail':
        return json_contents(jsonify(code=-1, msg="get-utxo-from-transaction: " + resp_json['msg']))
    else:
        return json_contents(jsonify(code=-1, msg="fail get utxo from transaction"))

    block_height = int(request.json['side_block_height'])
    tx_id  = request.json['side_tx_id'].encode('utf-8')
    alias = request.json['alias'].encode('utf-8')
    #root_xpubs = json.dumps(request.json['root_xpubs']).strip('{}')
    root_xpubs = "["
    control_program = request.json['control_program'].encode('utf-8')
    key_pair = {}
    f = open("key_pair","r") 
    lines = f.readlines()
    index = 0
    num = len(lines)
    for line in lines:
        tmp = json.loads(line)
        xprv = tmp['data']['xprv'].encode('utf-8')
        xpub = tmp['data']['xpub'].encode('utf-8')
        key_pair[xprv] = xpub
        root_xpubs += "\"" + xpub + "\""
        index = index + 1
        if index < num:
            root_xpubs = root_xpubs + ","

    root_xpubs = root_xpubs + "]"
    # 获取侧链raw transaction
    body_json = {"tx_id": tx_id,"block_height": block_height}
    response = connSide.request("/get-side-raw-transaction",body_json)
    resp_json = json.loads(response.text.encode('utf-8'))
    if resp_json['status'] == 'success':
        raw_transaction = resp_json['data']['raw_transaction'].encode('utf-8')
        block_hash = resp_json['data']['block_hash'].encode('utf-8')
    elif resp_json['status'] == 'fail':
        return json_contents(jsonify(code=-1, msg="get-raw-transaction: " + resp_json['msg']))
    else:
        return json_contents(jsonify(code=-1, msg="get raw transaction fail"))

    # 构建主链交易
    body_json = '{"claim_script":"%s","raw_transaction": "%s","alias": "%s","control_program":"%s","root_xpubs":%s,%s}' % (claim_script,raw_transaction,alias,control_program,root_xpubs,utxo)
    response = connSide.request("/build-mainchain-tx",json.loads(body_json))
    resp_json = json.loads(response.text.encode('utf-8'))
    tmpl = ""
    if resp_json['status'] == 'success':
        tmpl = json.dumps(resp_json['data'])
    elif resp_json['status'] == 'fail':
        return json_contents(jsonify(code=-1, msg="get-raw-transaction: " + resp_json['msg']))
    else:
        return json_contents(jsonify(code=-1, msg="get raw transaction fail"))

    # 签名
    #xpubs = request.json['root_xpubs']
    #xprvs = request.json['xprvs']
    #for index in range(len(xpubs)):
    for key,value in key_pair.items():
        #xprv = request.json['xprv'].encode('utf-8')
        body_json = '{"xprv": "%s","xpub":"%s","transaction":%s,"claim_script":"%s"}' % (key,value,tmpl,claim_script)
        response = connSide.request("/sign-with-key",json.loads(body_json))
        resp_json = json.loads(response.text.encode('utf-8'))
        if resp_json['status'] == 'success':
            raw_transaction = resp_json['data']['transaction']['raw_transaction'].encode('utf-8')
        elif resp_json['status'] == 'fail':
            return json_contents(jsonify(code=-1, msg="sign-with-key: " + resp_json['msg']))
        else:
            return json_contents(jsonify(code=-1, msg="sign-with-key fail"))
    
    # 提交到主链
    body_json = '{"raw_transaction": "%s"}' % (raw_transaction)
    response = connMain.request("/submit-transaction",json.loads(body_json))
    resp_json = json.loads(response.text.encode('utf-8'))
    if resp_json['status'] == 'success':
        return json_contents(jsonify(code=200, msg=resp_json['data']))
    elif resp_json['status'] == 'fail':
        return json_contents(jsonify(code=-1, msg="submit-transaction: " + resp_json['msg']))
    else:
        return json_contents(jsonify(code=-1, msg="submit-transaction fail"))


def json_contents(jsonify):
    response = make_response(jsonify)
    response.headers['Access-Control-Allow-Origin'] = '*'
    response.headers['Access-Control-Allow-Methods'] = '*'
    response.headers['Access-Control-Allow-Headers'] = '*'
    return response
