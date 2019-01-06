// Date         : 2018/12/4 5:24 PM
// Author       : zhangchaoyang
// Description  :
package test

import "ParameterServer/util"

func init()  {
	util.InitConfig()
	util.InitLogger(util.ConfPath+"/log4go.xml")
	util.InitMetric("parameter_server")
}
