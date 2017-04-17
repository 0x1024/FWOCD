package Serial_Srv

import (
	"RMS_Node/Common"
	"RMS_Node/util"
	"fmt"
	"github.com/juju/errors"
	log "github.com/sirupsen/logrus"
	//"go.bug.st/serial.v1"
	"github.com/tarm/serial"
)

var ComPortDown_ch chan int
var ctr uint

//// Open the first serial port detected at 9600bps N81
//var mode = &serial.Mode{
//	BaudRate: 115200,
//	Parity:   serial.NoParity,
//	DataBits: 8,
//	StopBits: serial.OneStopBit,
//}

var port *serial.Port

func PF(){
	port.Flush()
}

func SerialPortDaemon() {
	var c serial.Config
	c.Baud = 115200
	c.Name = "COM5"
	c.Size = 8
	c.Parity = serial.ParityNone
	c.StopBits = serial.Stop1
	port, _ = serial.OpenPort(&c)

	go EchoWaiter(*port)
	//echo(*port)

	SendCMD(*port, []byte{0xF0, 0x60}, []byte{}) //trig file trans task
	Common.SpecialComStat = true
	Common.SpecialComTast = Common.File_Trans

}

//var port serial.Port
//func SerialPortDaemon() {
//
//	for port == nil {
//		//enum ports
//		ports, err := serial.GetPortsList()
//		if err != nil {
//			log.Fatal(err)
//		}
//		if len(ports) == 0 {
//			log.Fatal("No serial ports found!")
//		}
//		// Print the list of detected ports
//		for _, port := range ports {
//			fmt.Printf("Found port: %v\n", port)
//		}
//
//		for _, portx := range ports {
//
//			port, err = serial.Open(portx, mode)
//			if err != nil {
//				log.Fatal(err)
//			}
//			go EchoWaiter(port)
//			echo(port)
//			//}
//			time.Sleep(1e9)
//
//			////write id name info
//			//var mix =make([]byte,32)
//			//mix[0]= 0
//			//mix[1]=0
//			//for i,k := range[]byte("YB0001"){
//			//	mix[2+i]=k
//			//}
//			//
//			//SendCMD(port ,[]byte{0xa1,01},mix)
//
//			time.Sleep(1e9)
//			//SendCMD(port ,[]byte{0xa1,02},[]byte{00,00}) //read eeprom data,addr 0
//			SendCMD(port, []byte{0xF0, 0x60}, []byte{}) //trig file trans task
//			Common.SpecialComStat = true
//			Common.SpecialComTast = Common.File_Trans
//			//go Xmodem.XmodemTransmit()
//			//EchoWaiter(port)
//
//		}
//
//		fmt.Println("\nserv end ")
//		for {
//			time.Sleep(10e9)
//		}
//		<-ComPortDown_ch
//	}
//
//}

func EchoWaiter(port serial.Port) { // Read and print the response

	buff := make([]byte, 4096)
	var res []byte
	var cnt int = 0
	var mark bool = true

	//Out:
	for true {
		for {
			// Reads up to 100 bytes
			n, err := port.Read(buff)

			if err != nil {
				log.Fatal(err)
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
			if Common.SpecialComStat {

				res = res[len(res):]
				switch Common.SpecialComTast {
				case Common.File_Trans:
					if mark {
						Common.Ch_Ft_start <- 1
						mark = false
					}

					fmt.Printf("File_Trans %s", buff[:n])

					fmt.Println("File_Trans", buff[:n])
					if buff[0] == 6 {
						fmt.Println("ack")
						fmt.Println(Common.Ch_ComStreamData)
					}
					Common.Ch_ComStreamData <- buff[:n]

				default:
					break

				}

			} else if len(res) > 7 {
				err := DeFrame(res)
				if err == nil {
					//break Out
				} else if err.Error() == "1" {

				} else if err.Error() == "2" {
					res = res[int(9+res[4]):]
				} else {
					fmt.Println(err)
					panic(err)
				}

			}

		}

	}
}

func DeFrame(res []byte) error {
	if res[0] == 0xAA && res[1] == 0x55 && len(res) == int(9+res[4]) {
		ctr++
		fmt.Println("\ngot pack", ctr, res)
		fmt.Printf("\nshort pack %d, %s", ctr, res[:int(9+res[4])])
		return nil
	} else if res[0] == 0xAA && res[1] == 0x55 && len(res) < int(9+res[4]) {
		fmt.Println("\nshort pack", ctr, res)
		fmt.Printf("\nshort pack %d, %s", ctr, res[:int(9+res[4])])
		return errors.New("1")
	} else if res[0] == 0xAA && res[1] == 0x55 && len(res) > int(9+res[4]) {
		fmt.Println("\nshort pack", ctr, res[:int(9+res[4])])
		fmt.Printf("\nshort pack %d, %s", ctr, res[:int(9+res[4])])

		return errors.New("2")
	}
	fmt.Println("\nwrong pack", res)
	fmt.Printf("\nwrong pack %x\n\n", res)
	return errors.New("wrong Frame~!")
}

func echo(port serial.Port) {

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
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Handshake echo, %d bytes:%x \n", n, send[:9])

}

func SendCMD(port serial.Port, cmd []byte, dat []byte) {

	send := make([]byte, 64)
	dl := len(dat)

	send[0] = 0xaa
	send[1] = 0x55
	send[2] = 0x01
	//len 3h 4l
	send[3] = 0x00
	send[4] = byte(dl)
	//cmd
	send[5] = cmd[0]
	send[6] = cmd[1]
	var i int = 0
	var k byte
	for i, k = range dat {
		send[7+i] = k
	}
	ret := util.CRC16(send, 7+dl)
	send[7+dl] = byte((ret >> 8) & 0xff)
	send[8+dl] = byte(ret & 0xff)

	n, err := port.Write(send[:9+dl])
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\n\nHandshake echo, %d bytes:%x \n", n, send[:9+dl])

}

func SendByte(c []byte) {
	n, err := port.Write(c )
	if err != nil {
		log.Fatal(n, err)
	}

}
