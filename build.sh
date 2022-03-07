CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build  -o vapord_ending_158634284 cmd/vapord/main.go

# vapord  start
#nohup /mnt/node/vapor/vapord_1.1.9.1 node --home /mnt/node/vapor/data --auth.disable &

#nohup /mnt/node/vapor/vapord_ending_158634284 node --home /mnt/node/vapor/data --auth.disable &

# 设定的vapord 停止块高： vapord_ending_158634284
# 当前vapor块高： 158598969  当前时间：2022-03-07 16:40:00
