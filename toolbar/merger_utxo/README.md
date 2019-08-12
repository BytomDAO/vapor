# UTXO merger


> **One last disclaimer:**

**the code we are about to go over is in no way intended to be used as an example of a robust solution.**

**We wouldn't be responsible for the consequences of using this tool.**

**please check this python code carefully and use it later.**


Requirements: Python 3.x, with requests package

Dependencies:
   ```
    pip install requests
   ```

Options:
  ```
  $ python merger_utxo.py  -h
usage: merger_utxo.py [-h] [-o URL] [-a ACCOUNT_ALIAS] [-p PASSWORD]
                      [-x MAX_AMOUNT] [-s MIN_AMOUNT] [-l] [-m MERGE_LIST]
                      [-f FOR_LOOP] [-y]

Vapor merge utxo tool

optional arguments:
  -h, --help            show this help message and exit
  -o URL, --url URL     API url to connect
  -a ACCOUNT_ALIAS, --account ACCOUNT_ALIAS
                        account alias
  -p PASSWORD, --pass PASSWORD
                        account password
  -x MAX_AMOUNT, --max MAX_AMOUNT
                        range lower than max_amount
  -s MIN_AMOUNT, --min MIN_AMOUNT
                        range higher than min_amount
  -l, --list            Show UTXO list without merge
  -m MERGE_LIST, --merge MERGE_LIST
                        UTXO to merge
  -f FOR_LOOP, --forloop FOR_LOOP
                        size for loop of UTXO to merge
  -y, --yes             confirm transfer
  
  ```

Example:
   ```
$ python btmspanner.py utxomerger -o http://127.0.0.1:9888 -a your_account_alias -p your_password -x 41250000000 -s 0 -m 20 -f 3 -y
   ```

Result:
```
$ python ./merger_utxo.py -o http://127.0.0.1:9889 -a test -p 123456 -x 2000000000 -s 0 -f 1
   0.   11.00000000 BTM d996363b3443407fe3828517f53551b1dda19b9e7503974892874318d03b9ed3 (mature)
   1.   11.00000000 BTM d9308dae432c592e7f32ba39ddf6c7882350bbd4b0269c5b96842f1df14b31cf (mature)
   2.   11.00000000 BTM c6fd223ffc2475ab0029bc5233be47c289970c141505effe20fe49f06cb575d6 (mature)
   3.   11.00000000 BTM 74851bebb8e94375bd346e36718ce1a4c273ecaff97501ab1985097ff85d7eaf (mature)
   4.   11.00000000 BTM 6610f18f0ecf4238568dea5a3dd1e360c5d3bff74c1bf0fefc19848a0852aabf (mature)
   5.   11.00000000 BTM 4e9eb10481f1c7306ab833f632bc04134979d39605f82b40c4c107015ec15322 (mature)
   6.   11.00000000 BTM 343a7d72cc830b992f8b777d32a78074a6338f642c060905997663a5c4549a8a (mature)
   7.   11.00000000 BTM 338f38f99211c1da076114db3951cfccf57c6612f1ad87bafb4043d161fc9572 (mature)
   8.   11.00000000 BTM 23c3d1210e636a1a30d0769c41ad2fcd25459c5f557d9fbd5b5a414d31f66ca2 (mature)
   9.   11.00000000 BTM 12198e59879b139768b92bd1272fce1a6274afe43829312105b8e36ec76f8c4a (mature)
  10.   11.00000000 BTM 10a420ddb64b34f7d1b3f56230d6c07e0e5c299f98c1ad5fbfe20100736128ab (mature)
  11.   11.00000000 BTM 00055e2354b20a5b88c6c6914ede710b6e19575d24937325461374909446821e (mature)
total size of available utxos is 12
To merge 12 UTXOs with  132.00000000 BTM totally.

One last disclaimer: the code we are about to go over is in no way intended to be used as an example of a robust solution. 
You will transfer BTM to an address, please check this python code and DO IT later.

this is the 1 times to merge utxos. -----begin
   0.   11.00000000 BTM d996363b3443407fe3828517f53551b1dda19b9e7503974892874318d03b9ed3 (mature)
   1.   11.00000000 BTM d9308dae432c592e7f32ba39ddf6c7882350bbd4b0269c5b96842f1df14b31cf (mature)
   2.   11.00000000 BTM c6fd223ffc2475ab0029bc5233be47c289970c141505effe20fe49f06cb575d6 (mature)
   3.   11.00000000 BTM 74851bebb8e94375bd346e36718ce1a4c273ecaff97501ab1985097ff85d7eaf (mature)
   4.   11.00000000 BTM 6610f18f0ecf4238568dea5a3dd1e360c5d3bff74c1bf0fefc19848a0852aabf (mature)
   5.   11.00000000 BTM 4e9eb10481f1c7306ab833f632bc04134979d39605f82b40c4c107015ec15322 (mature)
   6.   11.00000000 BTM 343a7d72cc830b992f8b777d32a78074a6338f642c060905997663a5c4549a8a (mature)
   7.   11.00000000 BTM 338f38f99211c1da076114db3951cfccf57c6612f1ad87bafb4043d161fc9572 (mature)
   8.   11.00000000 BTM 23c3d1210e636a1a30d0769c41ad2fcd25459c5f557d9fbd5b5a414d31f66ca2 (mature)
   9.   11.00000000 BTM 12198e59879b139768b92bd1272fce1a6274afe43829312105b8e36ec76f8c4a (mature)
  10.   11.00000000 BTM 10a420ddb64b34f7d1b3f56230d6c07e0e5c299f98c1ad5fbfe20100736128ab (mature)
  11.   11.00000000 BTM 00055e2354b20a5b88c6c6914ede710b6e19575d24937325461374909446821e (mature)
total size of available utxos is 12
To merge 12 UTXOs with  132.00000000 BTM
Confirm [y/N] y
tx_id: 38b38a6715ef223643a4c961b0f0553edcf6eb67b82546f741e4c391400cffa0
this is the 1 times to merge utxos. -----end
```