Osssync is a tool for synchronizing blocks data to OSS, and get blocks data from OSS before start the Vapor node. 

# Sample usage
## Upload
Run *Dockerfile* with config json file directory  
e.g.:   
```bash
$ docker build -t osssync -f toolbar/osssync/Dockerfile .
$ docker run -d -v config.json:config.json osssync:latest osssync config.json
```

config json file: 
```json
{
  "oss_config": {
    "login": {
      "endpoint": "",
      "access_key_id": "",
      "access_key_secret": ""
    },
    "bucket": "bycoin",
    "directory": "bytom-seed/"
  },
  "vapor_url": "http://localhost:9889"
}
```

## Download
Run vapor with keyword *oss.url*  
e.g.: 
```bash
$ vapord node --home "/Users/admin/Desktop/work/VaporTest" --oss.url "http://bycoin.oss-cn-shanghai.aliyuncs.com/bytom-seed"
```