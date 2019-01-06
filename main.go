// Date         : 2018/12/4 5:47 PM
// Author       : zhangchaoyang
// Description  :
package main

import (
	"flag"
	"fmt"
	server2 "ParameterServer/server"
	"ParameterServer/util"
)

const (
	MANAGER = "manager"
	SERVER  = "server"
)

func init() {
	util.InitConfig()
	util.InitLogger(util.ConfPath + "/log4go.xml")
	util.InitMetric("parameter_server")
}

func main() {
	group := flag.String("group", "", "group name")
	role := flag.String("role", "", "role")
	port := flag.Int("port", 0, "port")
	manager := flag.String("manager", "", "manager host")
	parameterCount := flag.Int("parameter_count", 0, "total parameter count")
	flag.Parse()
	fmt.Printf("group name: %s\n", *group)
	fmt.Printf("role: %s\n", *role)
	fmt.Printf("port: %d\n", *port)
	fmt.Printf("manager: %s\n", *manager)
	fmt.Printf("parameter count: %d\n", *parameterCount)
	if *port <= 10000 {
		fmt.Printf("could not bind to port less than 10000")
		return
	}
	switch *role {
	case MANAGER:
		pc := int32(*parameterCount)
		if pc <= 0 {
			fmt.Printf("parameter count must more than 0")
			return
		}
		server := new(server2.ServerManager)
		server.Init(*group, pc, *port)
		server.Work()
	case SERVER:
		if len(*manager) == 0 {
			fmt.Println("please input server manager")
			return
		}
		server := new(server2.Server)
		server.Init(*manager, *port)
		server.Work()
	default:
		fmt.Printf("please input valid role: %s %s", MANAGER, SERVER)
	}
}
