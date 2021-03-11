package sync

/*
import (
	"fmt"
	"github.com/Outer-God/osssync/clients"
	"github.com/Outer-God/osssync/util"
	"github.com/bytom/vapor/protocol"
	"os"
)

//func Download(localChain *protocol.Chain) {
func Download() {
	vaporClient := clients.NewVaporClient()
	blocks := vaporClient.GetBlockArrayByBlockHeight(43, 10)

	// 获取本地当前最新区块高度 Get the latest block height
	//latestDown := GetLatestDownloadBlockHeight(localChain)
	latestDown := uint64(0)
	bucket.GetObjToLocal(string(latestDown), LOCALDIR)








	filename := "43"


	blocks, err := vaporClient.GetRawBlockArrayByBlockHeight(43, 10)
	if err != nil {
		return err
	}
	_, err = fileUtil.SaveBlockFile(filename, blocks)
	if err != nil {
		return err
	}
	_, err = fileUtil.GetJson(filename)
	if err != nil {
		return err
	}
	err = fileUtil.GzipCompress(filename)
	if err != nil {
		return err
	}






	//fmt.Println(util.Json2Struct(data, blocks))

	//fmt.Println("hash: ", block.Hash)
	//fmt.Println("input: ", block.Transactions[0].Inputs[0].Amount)
	//fmt.Println("output: ", block.Transactions[0].Outputs[0].Amount)



	//fmt.Println("OSS Go SDK Version: ", oss.Version)
	ossClient, err = clients.NewOssClient()
	if err != nil {
		return err
	}
	bucket, err := ossClient.AccessBucket("bytom-seed")
	if err != nil {
		return err
	}


	objectName := "1"

	err = bucket.PutObjString(objectName, "hgjjgj")
	if err != nil {
		return err
	}
	fmt.Println("----------------")
	//bucket.GetObjToData("aaa")
	//bucket.AddObjLocal(objectName, LOCALDIR)


	//bucket.GetObjToLocal(objectName, LOCALDIR)
	//fileUtil.GzipUncompress(objectName)
	//bucket.DelObj("objectName")






}

 */