package util

import (
	"path"
	"os"
	"fmt"
)

const (
	TIME_FORMAT_HUMAN = "2006-01-02 15:04:05"
	TIME_FORMAT_SQL   = "20060102150405"
	DATE_FORMAT_HUMAN = "2006-01-02"
	DATE_FORMAT_SQL   = "20060102"
	RFC3339           = "2006-01-02T15:04:05Z07:00" //形如：2018-07-23T11:38:41+08:00，从mysql中读出来就是这种格式
	MaxInt            = int(^uint(0) >> 1)
)

var (
	RootPath string
	ConfPath string
	DataPath string
)

func InitConfig() {
	GOPATH := os.Getenv("GOPATH")
	RootPath = GOPATH + "/src/ParameterServer"
	ConfPath = path.Join(RootPath, "conf")
	DataPath = path.Join(RootPath, "data")

	fmt.Println("init system config finish, root path is " + RootPath)
}
