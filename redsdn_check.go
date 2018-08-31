package main

import (
	"bufio"
	"encoding/binary"
	"log"
	"os"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/ootw/redsdn-check/redsdn"
	"io"
	"time"
)

const (
	equalSign            = "="
	logPathPrefix        = "logpath="
	checkFileName        = "checkFN="
	bit           uint64 = 1
	unit          uint32 = 64
	superSend     uint32 = 121
)

func main() {
	args := os.Args
	programName := args[0]
	logPath := "."
	if strings.HasPrefix(args[1], logPathPrefix) {
		logPath = strings.Split(args[1], equalSign)[1]
	}
	fileName := ""
	if strings.HasPrefix(args[2], checkFileName) {
		fileName = strings.Split(args[2], equalSign)[1]
	}

	logFile, _ := os.Create(logPath + "/" + fileName + ".log")
	logger := log.New(logFile, "// DEBUG: ", log.LstdFlags|log.Lshortfile)
	logger.Printf("工具【%s】开始启动", programName)
	defer logFile.Close()
	defer logger.Printf("工具【%s】结束", programName)
	logger.Printf("开始校验【%s】文件...", fileName)
	file, err := os.OpenFile(fileName, os.O_RDONLY, os.ModePerm)
	if err != nil {
		logger.Fatalf("打开文件【%s】异常:%v", fileName, err)
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	size := reader.Size()
	i := 1
	for {
		var len uint16
		err := binary.Read(reader, binary.LittleEndian, &len)
		if err != nil {
			if io.EOF == err {
				break
			}
			logger.Fatalf("读取【len】异常:%v", err)
		}
		//logger.Printf("第【%d】次读取【%d】长度的数据", i, len)
		i++
		buf := make([]byte, len)
		readLen, bufErr := reader.Read(buf)
		if bufErr != nil {
			logger.Fatalf("读取【data】异常:%v", bufErr)
		}
		if uint16(readLen) != len {
			//logger.Printf("读取【data】:设定数据长度与读取数据长度不相等:【%d】!=【%d】", len, readLen)
			//logger.Println("重置reader")
			reader.Reset(file)
			size += reader.Size()
			newBuf := make([]byte, len-uint16(readLen))
			newReadLen, _ := reader.Read(newBuf)
			copy(buf[readLen:len-1], newBuf[0:newReadLen-1])
			if uint16(readLen+newReadLen) != len {
				logger.Fatalf("读取【data】异常:设定数据长度与读取数据长度不相等:【%d】!=【%d】", len, readLen+newReadLen)
			}
		}
		ParseProtoMessage(buf, logger)
	}
	logger.Printf("文件【%s】大小【%d】bytes", fileName, size)
}

func ParseProtoMessage(buf []byte, logger *log.Logger) {
	if len(buf) == 0 {
		return
	}
	message := new(redsdn.ProtoMessage)
	proto.Unmarshal(buf, message)
	if message.GetUserInfo() != nil {
		logger.Println(message.GetUserInfo().String())
	} else if message.GetRelayInfo() != nil {
		logger.Println(message.GetRelayInfo().String())
	} else if message.GetMulticastInfo() != nil {
		logger.Println(message.GetMulticastInfo().String())
	} else if message.GetMulticastEventInfo() != nil {
		logger.Println(message.GetMulticastEventInfo().String())
	} else if message.GetNicList() != nil {
		logger.Println(message.GetNicList().String())
	} else if message.GetLinkEvent() != nil {
		logger.Println(message.GetLinkEvent().String())
	} else if message.GetStatData() != nil {
		statData := message.GetStatData()
		itemValue := statData.GetItemValue()
		dataValues := itemValue.GetDataValues()
		objKey := itemValue.GetObjKey()
		objId := objKey.GetSelf().GetObjId()
		objType := objKey.GetSelf().GetObjType()
		statType := objKey.GetStatType()
		timestamp := statData.GetTimestamp()
		//var packets []uint64
		for _, dataValue := range dataValues {
			bitmapValues := dataValue.GetBitmapValues()
			sum := dataValue.GetSum()
			var packetIds []uint64
			for _, bitmapValue := range bitmapValues {
				idx := bitmapValue.GetIndex()
				bitmap := bitmapValue.GetBitMap()
				var i uint32
				for i = 0; i < unit; i++ {
					if (bitmap & (bit << i)) != 0 {
						packetId := uint64(idx*unit + i)
						packetIds = append(packetIds, packetId)
					}
				}
			}
			sec := timestamp / 1000
			nsec := timestamp % 1000 * 1000000
			time := time.Unix(int64(sec), int64(nsec)).Format("2006-01-02 15:04:05.000")
			//logger.Printf("%s timestamp=[%d] objId=[%s], objType=[%d], statType=[%d], index=[%d] sum=[%d] size=[%d]KByte packetIds=%v",
			//	time, timestamp, objId, objType, statType, dataValue.GetIndex(), sum, dataValue.GetSize(), packetIds)

			if int(sum) != len(packetIds) && objType == 121 && !strings.EqualFold(objId, "0") {
				logger.Printf("%s timestamp=[%d] objId=[%s], objType=[%d], statType=[%d], index=[%d] sum=[%d] size=[%d]KByte packetIds=%v",
					time, timestamp, objId, objType, statType, dataValue.GetIndex(), sum, dataValue.GetSize(), packetIds)
			}

			//if  statType == superSend && objType == 121 && !strings.EqualFold(objId, "0") && len(packets) != int(sum) {
			//	logger.Printf("%s objId=[%s], objType=[%d], statType=[%d], index=[%d] sum=[%d] size=[%d]KByte packetIds=%v",
			//		time, objId, objType, statType, dataValue.GetIndex(), sum, dataValue.GetSize(), packetIds)
			//	packets = append(packets, packetIds...)
			//}
			//if statType == superSend && objType == 121 && !strings.EqualFold(objId, "0") {
			//	logger.Printf("%s objId=[%s], objType=[%d], statType=[%d] packets=%v",
			//		time, objId, objType, statType, packets)
			//}
		}
		//logger.Println(message.GetStatData().String())
	}
}
