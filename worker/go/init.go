// Date         : 2018/12/10
// Author       : zhangchaoyang
// Description  : 
package main

import (
	"ParameterServer/util"
	"fmt"
)

func init() {
	util.InitConfig()
	util.InitLogger(util.ConfPath + "/log4go_client.xml")
	fmt.Printf("log conf file %s\n", util.ConfPath+"/log4go_client.xml")
	util.InitMetric("ps_client")
}
