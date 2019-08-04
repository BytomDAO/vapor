# Precog

Keep monitoring (leader & candidate) consensus nodes status in vapor network.

## Init

### Database Scheme
[federation_shema.sql](./sql_dump/federation_shema.sql)

### Config

## API

+ [/chain-status](#chain-status)
+ [/list-nodes](#list-nodes)

### /chain-status

### /list-nodes

__method:__ POST

__example response:__
```
{
    [
        {
            "alias": "cobo",
            "public_key": "b928e46bb01e834fdf167185e31b15de7cc257af8bbdf17f9c7fefd5bb97b306d048b6bc0da2097152c1c2ff38333c756a543adbba7030a447dcc776b8ac64ef",
            "host": "vapornode.cobo.com",
            "port": 123,
            "best_height": 1023,
            "lantency_ms": 300,
            "active_minutes": 4096,
            "status": "healthy"
        },
        {
            "alias": "matpool",
            "public_key": "0f8669abbd3cc0a167156188e428f940088d5b2f36bb3449df71d2bdc5e077814ea3f68628eef279ed435f51ee26cff00f8bd28fabfd500bedb2a9e369f5c825",
            "host": "vapornode.matpool.io",
            "port": 321,
            "best_height": 1024,
            "lantency_ms": 299,
            "active_minutes": 4097,
            "status": "healthy"
        }
    ] 
}
```


### /get-node-statistics