package util

import (
	"compress/gzip"
	"os"
)

func (f *FileUtil) GzipCompress(filename string) error {
	filedirname := f.localDir + "/" + filename + ".json.gz"
	fw, err := os.Create(filedirname)   // 创建gzip包文件，返回*io.Writer
	if err != nil {
		return err
	}
	defer fw.Close()

	// 实例化心得gzip.Writer
	gw := gzip.NewWriter(fw)
	defer gw.Close()

	// 获取要打包的文件信息
	filedirname = f.localDir + "/" + filename + ".json"
	fr, err := os.Open(filedirname)
	if err != nil {
		return err
	}
	defer fr.Close()

	// 获取文件头信息
	fi, err := fr.Stat()
	if err != nil {
		return err
	}

	// 创建gzip.Header
	gw.Header.Name = fi.Name()

	// 读取文件数据
	buf := make([]byte, fi.Size())
	_, err = fr.Read(buf)
	if err != nil {
		return err
	}

	// 写入数据到zip包
	_, err = gw.Write(buf)
	if err != nil {
		return err
	}
	return err
}

func (f *FileUtil) GzipUncompress(filename string) error {
	// 打开gz文件
	filedirname := f.localDir + "/" + filename + ".json.gz"
	fr, err := os.Open(filedirname)
	if err != nil {
		return err
	}
	defer fr.Close()

	// 创建gzip.Reader
	gr, err := gzip.NewReader(fr)
	if err != nil {
		return err
	}
	defer gr.Close()

	// 读取文件内容
	buf := make([]byte, 1024 * 1024 * 10)// 如果单独使用，需自己决定要读多少内容，根据官方文档的说法，你读出的内容可能超出你的所需（当你压缩gz文件中有多个文件时，强烈建议直接和tar组合使用）
	n, err := gr.Read(buf)

	// 将包中的文件数据写入
	filedirname = f.localDir + "/" + gr.Header.Name
	fw, err := os.Create(filedirname)
	if err != nil {
		return err
	}
	_, err = fw.Write(buf[:n])
	if err != nil {
		return err
	}
	return err
}
