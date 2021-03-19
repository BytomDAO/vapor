package sync

import (
	"bytes"
	"io/ioutil"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

// PutObjByteArr upload Byte Array object
func (b *Sync) PutObjByteArr(objectName string, objectValue []byte) error {
	objectAcl := oss.ObjectACL(oss.ACLPublicRead)
	return b.OssBucket.PutObject(objectName, bytes.NewReader(objectValue), objectAcl)
}

// GetObjToData download object to stream
func (b *Sync) GetObjToData(objectName string) ([]byte, error) {
	body, err := b.OssBucket.GetObject(objectName)
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
