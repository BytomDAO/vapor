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

__method:__ POST

__example response:__
```
{
    "best_height": 1024,
    "policy": {
        "confirmations": 150,
        "required_lantency_ms": 500
    } 
}
```

### /list-nodes

__method:__ POST

__example response:__
```
{
    [
        {
            "alias": "cobo",
            "pubkey": "...",
            "host": "vapornode.cobo.com",
            "port": 123,
            "best_height": 1023,
            "lantency_ms": 300,
            "active_minutes": 4096,
            "status": "healthy"
        },
        {
            "alias": "matpool",
            "pubkey": "...",
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


