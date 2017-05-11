package main

import (
	"FW_OCD/Common"
	"FW_OCD/Serial_Srv"
	"FW_OCD/util"
	"RMS_Srv/Public"
	"bufio"
	"fmt"
	"io"
	"os"
	"runtime"
)

var RMSNode_EXIT chan int
var RMSNode_EXIT1 chan int

func main() {
	runtime.GOMAXPROCS(2)
	Common.Init()
	Public.Init()


	fmt.Println("\nHRG固件下载工具 v001\n")
	fmt.Println("下载文件需要放在同目录下，命名 iRobot1_HGD.bin")
	fmt.Println("按 回车键 开始，按ctrl-c取消")

	reader := bufio.NewReader(os.Stdin)
	_, err := reader.ReadBytes('\n')
	if err != io.EOF {
		fmt.Println("任务开始")
		util.HRBserive(false)
		Serial_Srv.SerialPortDaemon()
		util.HRBserive(true)
		fmt.Println("任务结束")

	} else {
		fmt.Println("任务取消")
	}
}
