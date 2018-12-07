import requests
import json
from websocket import create_connection


class Connection(object):
    def __init__(self, base_url, token=''):
        self.baseUrl = base_url
        self.token = token

    def request(self, path, body={}):
        url = self.baseUrl + path
        headers = {}
        resp = requests.post(url, data=json.dumps(body), headers=headers)
        return resp

    @staticmethod
    def generate():
        return Connection("http://127.0.0.1:9888")


class WSClient(object):
    def __init__(self, url):
        self.ws = create_connection(url)

    def send(self, data):
        self.ws.send(data)
    
    def recv(self):
        return self.ws.recv()
    
    def close(self):
        self.ws.close()
