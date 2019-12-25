from flask import Blueprint, jsonify
from functools import wraps
from flask import request
from uuid import uuid4

tele = Blueprint('tele', __name__)

# 全局变量
LOGINED_UUID = []
NODE_LIST = [
    {
        "IP": "52.82.39.131",
        "level": "height",
        "is_connect": True
    },
    {
        "IP": "52.82.39.21",
        "level": "height",
        "is_connect": True
    }, {
        "IP": "52.82.123.129",
        "level": "height",
        "is_connect": True
    }
]

PASSWORD = "123456"


def login_required(func):
    @wraps(func)
    def decorator(*args, **kwargs):
        session_id = request.form.get('token')
        if not session_id:
            session_id = request.args.get('token')
        if session_id not in LOGINED_UUID:
            return jsonify({"code": 400, "msg": "no user", "data": ""})
        else:
            return func(*args, **kwargs)

    return decorator


@tele.route('/login', methods=["POST"])
def login():
    password = request.form.get('password')
    if password != PASSWORD:
        return jsonify({"code": 400, "msg": "password error", "data": ""})
    session_id = uuid4()
    LOGINED_UUID.append(session_id)
    response = jsonify({"code": 200, "msg": "", "data": ""})
    response.set_cookie("tele", str(session_id), max_age=86400)

    return response


@tele.route('/logout', methods=["POST"])
@login_required
def logout():
    response = jsonify({"code": 200, "msg": "", "data": ""})
    response.delete_cookie("tele")
    return response


@tele.route('/get-all-node', methods=["GET"])
@login_required
def get_all_node():
    return jsonify({"code": 200, "msg": "", "data": NODE_LIST})


@tele.route('/set-node-permission', methods=["POST"])
@login_required
def set_node_permission():
    ip = request.form.get('ip')
    level = request.form.get('level')
    for i in NODE_LIST:
        if i["IP"] == ip:
            i["level"] = level
            return jsonify({"code": 200, "msg": "", "data": ""})
    return jsonify({"code": 400, "msg": "no ip", "data": ""})


@tele.route('/set-connect', methods=["POST"])
@login_required
def set_connect():
    ip = request.form.get('ip')
    is_connect = request.form.get('is_connect')
    for i in NODE_LIST:
        if i["IP"] == ip:
            i["is_connect"] = is_connect
            return jsonify({"code": 200, "msg": "", "data": ""})
    return jsonify({"code": 400, "msg": "no ip", "data": ""})
