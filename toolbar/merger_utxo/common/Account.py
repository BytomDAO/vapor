import json


class Account(object):
    def __init__(self, id, alias, key_index, quorum, xpubs=[], *args, **kwargs):
        self.id = id
        self.alias = alias
        self.key_index = key_index
        self.quorum = quorum
        self.xpubs = xpubs

    @staticmethod
    def list(connection):
        response = connection.request("/list-accounts")

        resp_json = json.loads(response.text)

        if resp_json['status'] == 'success':
            account_list = list(map(lambda x: Account(**x), resp_json['data']))
            return account_list
        elif resp_json['status'] == 'fail':
            return resp_json['msg']
        else:
            return resp_json

    @staticmethod
    def list_address(connection, account_alias, account_id):

        body_json = {"account_alias": account_alias, "account_id": account_id}

        response = connection.request("/list-addresses", body_json)

        resp_json = json.loads(response.text)

        if resp_json['status'] == 'success':
            return resp_json['data'], 1
        elif resp_json['status'] == 'fail':
            return resp_json['msg'], -1
        else:
            return resp_json, 0

    @staticmethod
    def find_by_alias(connection, alias):
        account_list = Account.list(connection)
        for account in account_list:
            if account.alias == alias:
                return account

    @staticmethod
    def find_address_by_alias(connection, account_alias):
        account_id = Account.find_by_alias(connection, account_alias).id
        address_list, ret = Account.list_address(connection, account_alias, account_id)
        if ret == 1:
            return address_list[0]['address']

    @staticmethod
    def create(connection, root_xpubs, alias, quorum):
        body_json = {"root_xpubs": root_xpubs, "alias": alias, "quorum": quorum}
        response = connection.request("/create-account", body_json)

        resp_json = json.loads(response.text)

        if resp_json['status'] == 'success':
            return Account(**resp_json['data'])
        elif resp_json['status'] == 'fail':
            return resp_json['msg']
        else:
            return resp_json

    @staticmethod
    def delete(connection, account_info):
        # String - account_info, alias or ID of account.
        body_json = {"account_info": account_info}

        response = connection.request("/delete-account", body_json)

        resp_json = json.loads(response.text)

        if resp_json['status'] == 'success':
            return "true"
        else:
            return "false"

    @staticmethod
    def create_address(connection, account_alias, account_id):
        body_json = {"account_alias": account_alias, "account_id": account_id}

        response = connection.request("/create-account-receiver", body_json)

        resp_json = json.loads(response.text)

        if resp_json['status'] == "success":
            return resp_json['data']
        else:
            return "false"
