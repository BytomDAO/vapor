from flask import Blueprint, jsonify
from functools import wraps
from flask import request
from uuid import uuid4
import json
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
    },
{
        "IP": "52.82.25.183",
        "level": "height",
        "is_connect": True
    }
]

PASSWORD = "123456"


def login_required(func):
    @wraps(func)
    def decorator(*args, **kwargs):
        data = json.loads(request.get_data(as_text=True))
        session_id = data.get('token')
        if session_id not in LOGINED_UUID:
            return jsonify({"code": 400, "msg": "no user", "data": ""})
        else:
            return func(data)

    return decorator


@tele.route('/login', methods=["POST"])
def login():
    data = json.loads(request.get_data(as_text=True))
    password = data["password"]
    if password != PASSWORD:
        return jsonify({"code": 400, "msg": "password error", "data": ""})
    session_id = uuid4()
    LOGINED_UUID.append(str(session_id))
    response = jsonify({"code": 200, "msg": "", "data": str(session_id)})

    return response


# @tele.route('/logout', methods=["POST"])
# @login_required
# def logout():
#     response = jsonify({"code": 200, "msg": "", "data": ""})
#     response.delete_cookie("tele")
#     return response


@tele.route('/get-all-node', methods=["GET","POST"])
@login_required
def get_all_node(data):
    return jsonify({"code": 200, "msg": "", "data": NODE_LIST})


@tele.route('/set-node-permission', methods=["POST"])
@login_required
def set_node_permission(data):
    ip = data.get('ip')
    level = request.form.get('level')
    for i in NODE_LIST:
        if i["IP"] == ip:
            i["level"] = level
            return jsonify({"code": 200, "msg": "", "data": ""})
    return jsonify({"code": 400, "msg": "no ip", "data": ""})


@tele.route('/set-connect', methods=["POST"])
@login_required
def set_connect(data):
    ip = data.get('ip')
    is_connect = data.get('is_connect')
    for i in NODE_LIST:
        if i["IP"] == ip:
            i["is_connect"] = is_connect
            return jsonify({"code": 200, "msg": "", "data": ""})
    return jsonify({"code": 400, "msg": "no ip", "data": ""})
