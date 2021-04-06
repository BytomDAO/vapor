Osssync is a tool for synchronizing blocks data to OSS, and get blocks data from OSS before start the Vapor node. 

[Usage of Vapor and Docker](https://github.com/Bytom/vapor/blob/master/README.md)

# Sample usage
## Upload
Upload blocks to OSS. 

### Build the image

```bash
$ docker build -t osssync -f toolbar/osssync/Dockerfile .
```

### Running in Docker
```bash
$ docker run -d --name osssync -v <config.json-path-on-host>:/config.json osssync:latest osssync /config.json
```

config.json file: 
```json
{
  "oss_config": {
    "login": {
      "endpoint": "",
      "access_key_id": "",
      "access_key_secret": ""
    },
    "bucket": "",
    "directory": "vapor/"
  },
  "vapor_url": "http://localhost:9889"
}
```

## Download
Download blocks from OSS before starting a node:  
Run vapor with flag `oss.url`
```bash
$ vapord node --home <vapor-data-path> --oss.url <oss-url>
```