# Precognitive

Keep monitoring (leader & candidate) consensus nodes status in vapor network.

## Init

### Database Schema
[precognitive_schema.sql](./sql_dump/precognitive_schema.sql)

### Config
run with [config_example.json](docs/precognitive/config_example.json)
```
go run cmd/precognitive/main.go docs/precognitive/config_example.json
```

## API

+ [/list-nodes](#list-nodes)

### /list-nodes

__method:__ POST

```
curl -X POST 127.0.0.1:3009/api/v1/list-nodes -d '{}'
```

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