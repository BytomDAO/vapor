CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build  -o vapord_ending cmd/vapord/main.go

# vapord  start
nohup /mnt/node/vapor/vapord_1.1.9.1 node --home /mnt/node/vapor/data --auth.disable &

nohup /mnt/node/vapor/vapord_ending node --home /mnt/node/vapor/data --auth.disable &