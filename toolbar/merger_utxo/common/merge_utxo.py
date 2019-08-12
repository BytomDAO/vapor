import argparse
import getpass
import os
import time

from .Account import Account
from .Transaction import Action, Transaction
from .UnspentOutputs import UnspentOutputs
from .connection import Connection

parser = argparse.ArgumentParser(description='Vapor merge utxo tool')
parser.add_argument('-o', '--url', default='http://127.0.0.1:9889', dest='url', help='API url to connect')
parser.add_argument('-a', '--account', default=None, dest='account_alias', help='account alias')
parser.add_argument('-p', '--pass', default=None, dest='password', help='account password')
parser.add_argument('-x', '--max', default=41250000000, type=int, dest='max_amount', help='range lower than max_amount')
parser.add_argument('-s', '--min', default=1, type=int, dest='min_amount', help='range higher than min_amount')
parser.add_argument('-l', '--list', action='store_true', dest='only_list', help='Show UTXO list without merge')
parser.add_argument('-m', '--merge', default=90, type=int, dest='merge_list', help='UTXO to merge')
parser.add_argument('-f', '--forloop', default=1, type=int, dest='for_loop', help='size for loop of UTXO to merge')
parser.add_argument('-y', '--yes', action='store_true', default=None, dest='confirm', help='confirm transfer')


class VaporException(Exception):
    pass


class JSONRPCException(Exception):
    pass


def list_utxo(connection, account_alias, min_amount, max_amount):
    mature_utxos = []
    data, ret = UnspentOutputs.list_UTXO(connection=Connection(connection))
    block_height, ret_code = UnspentOutputs.get_block_height(connection=Connection(connection))
    if ret == 1 and ret_code == 1:
        for utxo in data:
            # append mature utxo to set
            if utxo['valid_height'] < block_height and utxo['account_alias'] == account_alias and utxo['asset_alias'] == 'BTM':
                mature_utxos.append(utxo)
    elif ret == -1:
        raise VaporException(data)

    result = []
    for utxo in mature_utxos:
        if utxo['amount'] <= max_amount and utxo['amount'] >= min_amount:
            result.append(utxo)

    return result


def send_tx(connection, utxo_list, to_address, password):
    actions = []
    amount = 0
    asset_id = 'ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff'

    for utxo in utxo_list:
        actions.append(Action.unspent_output(output_id=utxo['id']))
        amount += utxo['amount']

    actions.append(Action.control_address(amount=amount, asset_id=asset_id, address=to_address))

    time.sleep(1)
    transaction,f = Transaction.build_transaction(connection, actions)
    if f == False:
        return ""

    signed_transaction,f = Transaction.sign_transaction(connection, password, transaction)
    if signed_transaction['sign_complete']:
        raw_transaction = signed_transaction['transaction']['raw_transaction']
        result = Transaction.submit_transaction(connection, raw_transaction)
        return result['tx_id']
    else:
        raise VaporException('Sign not complete')


def main():
    options = parser.parse_args()
    utxo_total = []
    utxolist = list_utxo(options.url, options.account_alias, options.min_amount, options.max_amount)

    for i, utxo in enumerate(utxolist):
        print('{:4}. {:13.8f} BTM {}{}'.format(i, utxo['amount'] / 1e8, utxo['id'], ' (mature)'))
        if i >= options.merge_list * options.for_loop:
            break
        utxo_total.append(utxo)

    print("total size of available utxos is {}".format(len(utxolist)))

    if options.only_list:
        return

    print('To merge {} UTXOs with {:13.8f} BTM totally.\n'.format(len(utxo_total),
                                                                  sum(utxo['amount'] for utxo in utxo_total) / 1e8))

    merge_size = options.merge_list or input('Merge size of UTXOs (5, 13 or 20): ')
    for_loop = options.for_loop or input('for loop size (1, 10 or 50): ')
    
    print(
        'One last disclaimer: the code we are about to go over is in no way intended to be used as an example of a robust solution. ')
    print('You will transfer BTM to an address, please check this python code and DO IT later.\n')

    for loops in range(for_loop):

        utxo_mergelist = []

        # for i in range(merge_size if merge_size <= len(utxolist) else len(utxolist)):
        # utxo_mergelist.append(utxolist[i])
        for i in range(loops * merge_size,
                       ((loops + 1) * merge_size) if ((loops + 1) * merge_size) < len(utxolist) else len(utxolist)):
            utxo_mergelist.append(utxolist[i])

        # print(loops*merge_size, ", ", ((loops+1)*merge_size) if (loops*merge_size) < len(utxolist) else len(utxolist))
        print('this is the {} times to merge utxos. -----begin'.format(loops + 1))

        for i, utxo in enumerate(utxo_mergelist):
            print(
                '{:4}. {:13.8f} BTM {}{}'.format(loops * merge_size + i, utxo['amount'] / 1e8, utxo['id'], ' (mature)'))

        print("total size of available utxos is {}".format(len(utxo_mergelist)))

        if len(utxo_mergelist) < 2:
            print('Not Merge UTXOs, Exit...')
            return

        print('To merge {} UTXOs with {:13.8f} BTM'.format(len(utxo_mergelist),
                                                           sum(utxo['amount'] for utxo in utxo_mergelist) / 1e8))

        if not options.account_alias:
            options.account_alias = input('Transfer account alias: ')

        if not options.password:
            options.password = getpass.getpass('Vapor Account Password: ')

        if not (options.confirm or input('Confirm [y/N] ').lower() == 'y'):
            print('Not Merge UTXOs, Exit...')
            return

        to_address = Account.find_address_by_alias(Connection(options.url), options.account_alias)
        if not to_address:
            to_address = input('Transfer address: ')

        print('tx_id:', send_tx(Connection(options.url), utxo_mergelist, to_address, options.password))
        print('this is the {} times to merge utxos. -----end\n'.format(loops + 1))


if __name__ == '__main__':
    main()
