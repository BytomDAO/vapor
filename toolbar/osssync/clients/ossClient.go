package clients

import (
	"bytes"
	"fmt"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/bytom/vapor/toolbar/osssync/config"
	"io/ioutil"
	"strings"
)


type OssClient struct {
	*oss.Client
}

func NewOssClient(Oss *config.Oss) (*OssClient, error) {
	// AccessKeyID和AccessKeySecret不提交到仓库
	client, err := oss.New(Oss.Endpoint, Oss.AccessKeyID, Oss.AccessKeySecret)
	// client, err := oss.New("oss-cn-hongkong.aliyuncs.com", "LTAI4G4eoPHBmxcU22vBgx2E", "nBK4MG3ygxoSb1lJ38aScXicSDsRJw")
	if err != nil {
		return nil, err
	}
	return &OssClient{client}, err
}

type OssBucket struct {
	*oss.Bucket
}

func (c *OssClient) AccessBucket(bucketName string) (*OssBucket, error) {
	bucket, err := c.Bucket(bucketName)
	if err != nil {
		return nil, err
	}
	return &OssBucket{bucket}, err
}

// Upload String object
func (b *OssBucket) PutObjString(objectName, objectValue string) error {
	// 指定存储类型为标准存储，缺省也为标准存储。
	storageType := oss.ObjectStorageClass(oss.StorageStandard)
	// 指定访问权限为公共读，缺省为继承bucket的权限。
	objectAcl := oss.ObjectACL(oss.ACLPublicRead)
	// 上传字符串。
	return b.PutObject(objectName, strings.NewReader(objectValue), storageType, objectAcl)
}

// Upload Byte Array object
func (b *OssBucket) PutObjByteArr(objectName string, objectValue []byte) error {
	// 指定访问权限为公共读，缺省为继承bucket的权限。
	objectAcl := oss.ObjectACL(oss.ACLPublicRead)
	// 上传Byte数组。
	return b.PutObject(objectName, bytes.NewReader(objectValue), objectAcl)
}

// Upload Local File object
func (b *OssBucket) PutObjLocal(objectName, localDir string) error {
	// 上传本地文件。
	return b.PutObjectFromFile(objectName, localDir + "/" + objectName)
}

// Delete Object
func (b *OssBucket) DelObj(objectName string) error {
	return b.DeleteObject(objectName)
}

// List Objects
func (b *OssBucket) ListObjs() error {
	// 列举所有文件。
	marker := ""
	for {
		lsRes, err := b.ListObjects(oss.Marker(marker))
		if err != nil {
			return err
		}

		// 打印列举结果。默认情况下，一次返回100条记录。
		for _, object := range lsRes.Objects {
			fmt.Println("File: ", object.Key)
		}

		if lsRes.IsTruncated {
			marker = lsRes.NextMarker
		} else {
			break
		}
	}
	return nil
}

// Download object to stream
func (b *OssBucket) GetObjToData(objectName string) ([]byte, error) {
	// 下载文件到流。
	body, err := b.GetObject(objectName)
	if err != nil {
		return nil, err
	}
	// 数据读取完成后，获取的流必须关闭，否则会造成连接泄漏，导致请求无连接可用，程序无法正常工作。
	defer body.Close()

	data, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, err
	}
	return data, err
}

// Download object to local
func (b *OssBucket) GetObjToLocal(objectName, localDir string) error {
	return b.GetObjectToFile(objectName, localDir + "/" + objectName)
}

// IsExist checks if the file exist
func (b *OssBucket) IsExist(objectName string) (bool, error) {
	return b.IsObjectExist(objectName)
}
