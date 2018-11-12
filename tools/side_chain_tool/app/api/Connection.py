import requests
import json


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
