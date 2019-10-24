Vapor version 1.0.4 is now available from:

  https://github.com/Bytom/vapor/releases/tag/v1.0.4


Please report bugs using the issue tracker at github:

  https://github.com/Bytom/vapor/issues




1.0.4 changelog
================
__Vapor Node__

+ `PR #401`
    - Define the levelDB database structure of MOV.
      - Including utxo, order, transaction pair and database status of MOV
+ `PR #404`
    - Database leveldb realizes the function of MOV data paging query and storage.
      - Including order storage and sorting of MOV, paging query function of order and transaction pairs
+ `PR #407`
    - Solve the problem of node discovery service name consistency.
+ `PR #409`
    - Update the gas cost by the chain transactions.
+ `PR #423`
    - Solve the problem cross chain asset name alias can't be found  that cost by the format of upper case and lower case.


__Vapor Dashboard__

- Update the dashboard with a switcher for either the chain transactions or the normal transactions. The transaction contains common voting cross-chain. This feature supports the official BTM asset only.

Credits
--------

Thanks to everyone who directly contributed to this release:

- Agouri
- Colt-Z
- HAOYUatHZ
- langyu
- Paladz
- shenao78
- shengling2008
- yahtoo
- zcc0721
- ZhitingLin

And everyone who helped test.
