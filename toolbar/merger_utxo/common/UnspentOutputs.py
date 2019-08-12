import json


class UnspentOutputs(object):

    @staticmethod
    def get_block_height(connection):
        response = connection.request("/get-block-count")

        resp_json = json.loads(response.text)

        if resp_json['status'] == 'success':
            return resp_json['data']['block_count'], 1
        elif resp_json['status'] == 'fail':
            return resp_json['msg'], -1
        else:
            return resp_json, 0

    @staticmethod
    def list_UTXO(connection):
        response = connection.request("/list-unspent-outputs")

        resp_json = json.loads(response.text)
        if resp_json['status'] == 'success':
            return resp_json['data'], 1
        elif resp_json['status'] == 'fail':
            return resp_json['msg'], -1
        else:
            return resp_json, 0
