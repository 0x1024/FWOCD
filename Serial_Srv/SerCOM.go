package Serial_Srv

import (
	"FW_OCD/util"
	"RMS_Srv/Protocol"
	"RMS_Srv/Public"
	"fmt"
	"github.com/juju/errors"
	log "github.com/sirupsen/logrus"
	"github.com/tarm/serial"
	"os"
	"syscall"
	"time"
)

var ComPortDown_ch chan int
var ComTrans_Ch chan []byte
var ctr uint

//// Open the first serial port detected at 9600bps N81
//var mode = &serial.Mode{
//	BaudRate: 115200,
//	Parity:   serial.NoParity,
//	DataBits: 8,
//	StopBits: serial.OneStopBit,
//}

//Call定义
type Call struct {
	Id      int
	To      *time.Timer
	Request interface{}
	Reply   interface{}
	Flag    uint
	Done    chan *Call //用于结果返回时,消息通知,使用者必须依靠这个来获取真正的结果。
}

func (call *Call) TO() {
	fmt.Println("ooops")
	call.Flag = 48
	call.Reply = []byte("com time out")
	call.done()
}

func (call *Call) Del() {
	call.To.Stop()
	delete(affair, call.Id)
}

// 非常重要的异步调用结果返回，供框架内部使用。
func (call *Call) done() {
	select {
	case call.Done <- call:
		if call.Reply == nil {
			call.Reply = []byte("done")
		}
	default:
		// 阻塞情况处理,这里忽略
	}
}

//用于检查是否存在事务请求
var affair = make(map[int]*Call)

var port *serial.Port

func SerialPortDaemon() {
	var c serial.Config

	portlist, err := GetPortsList()
	if err != nil {

		log.Panic("no port founded ", err)
	}
	fmt.Println("found ports", portlist)

	c.Baud = 115200
	c.Size = 8
	c.Parity = serial.ParityNone
	c.StopBits = serial.Stop1

	//port, _ = serial.OpenPort(&c)

	ComTrans_Ch = make(chan []byte, 1)
out_search_dev:
	for _, portx := range portlist {
		c.Name = portx
		fmt.Println("try search device on port ", c.Name)
		port, err = serial.OpenPort(&c)
		if err != nil {
			log.Debug("open port ", err)
			c.Name = ""
			continue
		}
		defer port.Close()
		port.Flush()

		affair[1] = new(Call)
		affair[1].Id = 1
		affair[1].Done = make(chan *Call, 1)
		affair[1].Request = SUB_PROC_VER
		affair[1].To = time.AfterFunc(1e9, affair[1].TO)

		go EchoWaiter(*port)

		echo(*port)
		//read uid

		ret := <-affair[1].Done
		//fmt.Println("reply", ret.Reply)
		switch ret.Flag {

		case 10:
			var cc Public.TcpTrucker
			cc.Cmd = Protocol.Fc_HB
			cc.Dat = ret.Reply
			cc.Ip = Public.LocalNode.NodeIPP
			Public.TcpSender_Ch <- cc
			ret.Del()
			break out_search_dev
		default:
			c.Name = ""
			ret.Del()

		}

	}

	if c.Name != "" {
		fmt.Println("found dev,start download with port ", c.Name)
		input := "iRobot1_HGD.bin"
		Sendfile(input)
	} else {
		fmt.Println("can't find device or port,exit ")

	}

}

func EchoWaiter(port serial.Port) { // Read and print the response

	buff := make([]byte, 4096)
	var res []byte
	var cnt int = 0

	//Out:
	for true {
		for {
			// Reads up to 100 bytes
			n, err := port.Read(buff)

			if err != nil {
				if err == error(syscall.ERROR_OPERATION_ABORTED) {
					return
				}
				log.Debug(err)
				break
			}
			if n == 0 {

				log.Println("\nEOF")
				break
			}

			cnt++
			//for _, as := range buff[:n] {
			//	fmt.Printf("%02X ", as)
			//}
			//fmt.Printf("\r\nrec %d     \r\n", cnt, res)
			res = append(res, buff[:n]...)

			if len(res) > 7 {
				err := CheckFrame(res)
				if err.Error() == "0" {

					switch res[5] {
					case MAIN_RMS:
						ctr++
						//fmt.Println("\ngot pack", ctr, res)
						//fmt.Printf("\ngot pack %d, %s \n", ctr, res[:int(9+res[4])])
						ComTrans_Ch <- res[:int(9+res[4])]
						res = res[int(9+res[4]):]
					case MAIN_STRING:

					case MAIN_PROC:
						fmt.Printf("\nproc %s \n", res[:int(9+res[4])])

						//affair
						for _, v := range affair {

							switch v.Request {
							case SUB_PROC_VER:
								v.Reply = res[7:int(7+res[4])]
								v.Flag = 10
								v.done()
							case SUB_PROC_UID:
								v.Flag = 16
								v.Reply = res[7:int(7+res[4])]
								v.done()
							default:

							}

						}
					default:
					}

					//break Out
				} else if err.Error() == "1" {
					//fmt.Println("\nshort pack", ctr, res)
					//fmt.Printf("\nshort pack %d, %s", ctr, res)
				} else if err.Error() == "2" {
					switch res[5] {
					case MAIN_RMS:
						ctr++
						//fmt.Println("\ngot pack", ctr, res)
						//fmt.Printf("\ngot pack %d, %s \n", ctr, res[:int(9+res[4])])
						ComTrans_Ch <- res[:int(9+res[4])]
					case MAIN_STRING:
						fmt.Printf("\ngot msg %s \n", res[:int(9+res[4])])

					default:

					}
					res = res[int(9+res[4]):]

				} else if err.Error() == "3" {

				} else if err.Error() == "4" {
					res = res[:0]
				} else {
					res = res[:0]
					//fmt.Println(err)
					//log.Debug(err)
				}

			}

		}

	}
}

func CheckFrame(res []byte) error {
	if res[0] == 0xAA && res[1] == 0x55 && len(res) == int(9+res[4]) {
		return errors.New("0")
	} else if res[0] == 0xAA && res[1] == 0x55 && len(res) < int(9+res[4]) {
		return errors.New("1")
	} else if res[0] == 0xAA && res[1] == 0x55 && len(res) > int(9+res[4]) {
		//fmt.Printf("\nmore pack L:%d,C:%d % X %s", len(res), int(9+res[4]), res,res[:int(9+res[4])])

		return errors.New("2")
	}
	//fmt.Printf("\nwrong pack %s", res)
	fmt.Printf("\nwrong pack %s % X\n\n", res, res)
	return errors.New("5")
}

//send a echo ,return with send err
func echo(port serial.Port) bool {

	send := make([]byte, 32)
	send[0] = 0xaa
	send[1] = 0x55
	send[2] = 0x01
	send[3] = 0x00
	send[4] = 0x00
	send[5] = 0xF0
	send[6] = 0x80
	ret := util.CRC16(send, 7)
	send[7] = byte((ret >> 8) & 0xff)
	send[8] = byte(ret & 0xff)

	n, err := port.Write(send[:9])
	if err != nil || n != 9 {
		log.Info(err)
		return false
	}
	return true
	//fmt.Printf("Handshake echo, % d bytes:% X \n", n, send[:9])
}

func SendCMD(port serial.Port, cmd []byte, dat []byte) {

	send := make([]byte, 300)
	dl := len(dat)

	send[0] = 0xaa
	send[1] = 0x55
	send[2] = 0x01
	//len 3h 4l
	send[3] = byte(dl / 256)
	send[4] = byte(dl)
	//cmd
	send[5] = cmd[0]
	send[6] = cmd[1]

	copy(send[7:], dat)
	ret := util.CRC16(send, 7+dl)
	send[7+dl] = byte((ret >> 8) & 0xff)
	send[8+dl] = byte(ret & 0xff)

	_, err := port.Write(send[:9+dl])
	if err != nil {
		log.Fatal(err)
	}

	//fmt.Printf("\n\n [%s] send cmd, % d bytes:% X \n",time.Now().UnixNano(), n, send[:9+dl])
	//fmt.Printf("\n\n [%s] send cmd, % d  \n", time.Now().UnixNano(), n)

}

func SendByte(c []byte) {
	n, err := port.Write(c)
	if err != nil {
		log.Fatal(n, err)
	}

}

//2017 0418 new add rms segment
const (
	MAIN_RMS    = 0xC0
	MAIN_STRING = 0xFE
	MAIN_PROC   = 0xF0
)

//MAIN_RMS
const (
	SUB_RMS_FILEHEAD = 0x10
	SUB_RMS_FILEDATA = 0x11
	SUB_RMS_FILE_END = 0x12
)

//MAIN_STRING
const (
	SUB_STRING_S = 0x10
	SUB_STRING_E = 0x11
)

//MAIN_PROC
const (
	SUB_PROC_VER = 0x80
	SUB_PROC_UID = 0x83
)

func Sendfile(input string) {
	var dat []byte = make([]byte, 300)
	var offsert int64 = 0

	//open file
	//input := "e:/iRobot1_HGD.bin"
	fi, err := os.Open(string(input))
	if err != nil {
		panic(err)
	}
	defer fi.Close()
	fiinfo, err := fi.Stat()
	s := fiinfo.Size()
	fmt.Printf("\nsize of file is %d \n", s) //fiinfo.Size() return int64 type

	//send file head ,max file len 4GB(0xFFFFFFFF)
	dat[0] = byte(s >> 24)
	dat[1] = byte(s >> 16)
	dat[2] = byte(s >> 8)
	dat[3] = byte(s >> 0)
	dat[4] = 0
	dat[5] = 0
	ss := copy(dat[6:], fiinfo.Name())

	SendCMD(*port, []byte{MAIN_RMS, SUB_RMS_FILEHEAD}, dat[:6+ss])
	c := <-ComTrans_Ch
	if c[6] != SUB_RMS_FILEHEAD {
		fmt.Println("send hf err", c)
	}

	timecost := time.Now()

	//	fmt.Printf("\n [%s]file str \n", timecost.Format(time.RFC3339Nano))

	//file body
	for {
		// 0~3 block no
		dat[0] = byte(offsert >> 24)
		dat[1] = byte(offsert >> 16)
		dat[2] = byte(offsert >> 8)
		dat[3] = byte(offsert >> 0)

		//4~256+4 data block
		ss, err = fi.ReadAt(dat[4:256+4], offsert*256)
		if ss == 0 { // end of file
			SendCMD(*port, []byte{MAIN_RMS, SUB_RMS_FILE_END}, dat[:0])
			//fmt.Printf("\n [%s]file end [%s] \n", time.Now().Format(time.RFC3339Nano), time.Now().Sub(timecost))

			break
		}
		i := 100 * int64(offsert) / (s / 256)
		if offsert%10 == 0 || i == 100 {
			fmt.Printf("\rprocess  %d%%", i)
		}
		SendCMD(*port, []byte{MAIN_RMS, SUB_RMS_FILEDATA}, dat[:4+ss])
		c = <-ComTrans_Ch
		if c[6] != SUB_RMS_FILEDATA || c[7+3] != dat[3] {
			fmt.Println("send hf err", c)
		}

		offsert++
	}

	fmt.Printf("\nFile send complete ,time cost %q \n", time.Now().Sub(timecost))

}
