import json

from .Account import Account


class Transaction(object):
    @staticmethod
    def build_transaction(connection, actions):
        # ttl: 15min=900000ms
        body_json = {"base_transaction": None, "actions": actions,
                     "ttl": 1, "time_range": 0}
        
        response = connection.request("/build-transaction", body_json)

        resp_json = json.loads(response.text)
        
        if resp_json['status'] == 'success':
            return resp_json['data'], True
        elif resp_json['status'] == 'fail':
            return resp_json['msg'], False
        else:
            return resp_json, False

    @staticmethod
    def sign_transaction(connection, password, transaction):
        body_json = {"password": password, "transaction": transaction}
        response = connection.request("/sign-transaction", body_json)

        resp_json = json.loads(response.text)
        
        if resp_json['status'] == 'success':
            return resp_json['data'], True
        elif resp_json['status'] == 'fail':
            return resp_json['msg'], False
        else:
            return resp_json, False

    @staticmethod
    def submit_transaction(connection, raw_transaction):
        body_json = {"raw_transaction": raw_transaction}
        response = connection.request("/submit-transaction", body_json)

        resp_json = json.loads(response.text)

        if resp_json['status'] == 'success':
            return resp_json['data']


class Action(object):

    @staticmethod
    def spend_account(amount, account_id, asset_id):
        return {
            'amount': amount,
            'account_id': account_id,
            'asset_id': asset_id,
            'type': 'spend_account'
        }
    
    @staticmethod
    def control_address(amount, asset_id, address):
        return {
            'amount': amount,
            'asset_id': asset_id,
            'address': address,
            'type': 'control_address'
        }

    @staticmethod
    def unspent_output(output_id):
        return {
            'type': 'spend_account_unspent_output',
            'output_id': output_id
        }
