package log

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitLogFile(t *testing.T) {
	logPath:="/Users/linshi/vapor/data/log"
	files,err:=ioutil.ReadDir(logPath)
	if err!=nil{
		fmt.Println(err)
	}
	for _,file:=range files{
		if ok:=strings.HasSuffix(file.Name(),"_lock");ok{
			err:=os.Remove(filepath.Join(logPath,file.Name()))
			if err!=nil{
				fmt.Println(err)
			}
		}
	}
}