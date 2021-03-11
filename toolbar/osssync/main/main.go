package main

import (
	"fmt"
	"github.com/bytom/vapor/toolbar/osssync/clients"
	"github.com/bytom/vapor/toolbar/osssync/config"
	"github.com/bytom/vapor/toolbar/osssync/sync"
	"github.com/bytom/vapor/toolbar/osssync/util"
	"os"
)

const LOCALDIR = "./blocks"

func mkDir() error {
	err := os.Mkdir(LOCALDIR,0755)
	return err
}


func testInfoJson(cfg *config.Config) error {
	vaporClient := clients.NewVaporClient()
	ossClient, _ := clients.NewOssClient(&cfg.Oss)
	ossBucket, _ := ossClient.AccessBucket("bytom-seed")
	fileUtil := util.NewFileUtil()
	sync.CreateInfoJson(cfg)
	//util.AddInterval(ossBucket, 199999, 200000)
	//return sync.UploadFiles(vaporClient, ossBucket, fileUtil, LOCALDIR, 10000000, 19999999, 100000)
	return sync.UploadFiles(vaporClient, ossBucket, fileUtil, LOCALDIR, 0, 59999999, 100000)
}


func main() {
	//mkDir()
	//1823 5216




	cfg := &config.Config{}
	err := config.LoadConfig(&cfg)

	if err != nil {
		fmt.Println(err)
	}

	//sync.CreateInfoJson(cfg)

	//err = testInfoJson(cfg)
	err = sync.Upload(cfg)


	if err != nil {
		fmt.Println(err)
	}

	vaporClient := clients.NewVaporClient()
	fmt.Println(vaporClient.GetBlockCount())

	/*
	vaporClient := clients.NewVaporClient()
	//blockHeight, _ := vaporClient.GetBlockCount()
	//fmt.Println(blockHeight)
	//block, _ := vaporClient.GetBlockByBlockHeight(44)
	//fmt.Println(block.Hash)

	rawBlock, err := vaporClient.GetRawBlockByHeight(42)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(rawBlock.Height)
	// 直接把 types.Block - *rawBlock 写入到json 指针传入后写入时加*
	fmt.Println(*rawBlock)



	/*
	var text []byte
	err = rawBlock.UnmarshalText(text)
	fmt.Println("err = ", err)
	fmt.Println(string(text))
	 */


	//blocks, err := vaporClient.GetBlockArrayByBlockHeight(43, 10)
	//filename := "aaa"
	//fileUtil := util.NewFileUtil()
	//fileUtil.SaveBlockFile(filename, blocks)

	//fileUtil.GetJson(filename)
	//fileUtil.GzipCompress(filename)


	//fmt.Println(util.Json2Struct(data, blocks))

	//fmt.Println("hash: ", block.Hash)
	//fmt.Println("input: ", block.Transactions[0].Inputs[0].Amount)
	//fmt.Println("output: ", block.Transactions[0].Outputs[0].Amount)



	//fmt.Println("OSS Go SDK Version: ", oss.Version)
	/*
	ossClient, err := clients.NewOssClient()
	if err != nil {
		return
	}
	bucket, err := ossClient.AccessBucket("bytom-seed")
	if err != nil {
		return
	}


	objectName := "1"

	//bucket.PutObjString(objectName, "nbbb")
	//bucket.PutObjString(objectName, "ijnm")
	//bucket.PutObjLocal(objectName, LOCALDIR)

	fmt.Println("----------------")
	//bucket.GetObjToData(objectName)
	//bucket.GetObjToLocal(objectName, LOCALDIR)


	err = bucket.ListObjs()
	fmt.Println(bucket.IsExist(objectName))
	fmt.Println(bucket.IsExist("objectName"))
	//fileUtil.GzipUncompress(objectName)
	//bucket.DelObj("objectName")

	 */
}
