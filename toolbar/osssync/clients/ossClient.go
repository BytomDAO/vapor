package clients

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"

	"github.com/bytom/vapor/toolbar/osssync/config"
)

type OssClient struct {
	*oss.Client
}

func NewOssClient(Oss *config.Oss) (*OssClient, error) {
	client, err := oss.New(Oss.Endpoint, Oss.AccessKeyID, Oss.AccessKeySecret)
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

// PutObjString upload String object
func (b *OssBucket) PutObjString(objectName, objectValue string) error {
	storageType := oss.ObjectStorageClass(oss.StorageStandard)
	objectAcl := oss.ObjectACL(oss.ACLPublicRead)
	return b.PutObject(objectName, strings.NewReader(objectValue), storageType, objectAcl)
}

// PutObjByteArr upload Byte Array object
func (b *OssBucket) PutObjByteArr(objectName string, objectValue []byte) error {
	objectAcl := oss.ObjectACL(oss.ACLPublicRead)
	return b.PutObject(objectName, bytes.NewReader(objectValue), objectAcl)
}

// PutObjLocal upload Local File object
func (b *OssBucket) PutObjLocal(objectName, localDir string) error {
	return b.PutObjectFromFile(objectName, localDir+"/"+objectName)
}

// DelObj deletes Object
func (b *OssBucket) DelObj(objectName string) error {
	return b.DeleteObject(objectName)
}

// ListObjs list all objects
func (b *OssBucket) ListObjs() error {
	marker := ""
	for {
		lsRes, err := b.ListObjects(oss.Marker(marker))
		if err != nil {
			return err
		}

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

// GetObjToData download object to stream
func (b *OssBucket) GetObjToData(objectName string) ([]byte, error) {
	body, err := b.GetObject(objectName)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	data, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, err
	}
	return data, err
}

// GetObjToLocal download object to local
func (b *OssBucket) GetObjToLocal(objectName, localDir string) error {
	return b.GetObjectToFile(objectName, localDir+"/"+objectName)
}

// IsExist checks if the file exist
func (b *OssBucket) IsExist(objectName string) (bool, error) {
	return b.IsObjectExist(objectName)
}
