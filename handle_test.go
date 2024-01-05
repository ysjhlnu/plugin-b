package gb28181

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"golang.org/x/net/html/charset"
	"m7s.live/plugin/gb28181/v4/utils"
	"math/rand"
	"strconv"
	"testing"
)

func TestDecodeXML(t *testing.T) {

	body := `<?xml version=\"1.0\" encoding=\"utf-8\"?>
<Response>
<CmdType>RecordInfo</CmdType>
<SN>33</SN>\r\n
<DeviceID>34020000001310000001</DeviceID>\r\n
<Name>34020000001310000001</Name>\r\n
<SumNum>1</SumNum>\r\n
<RecordList Num="1">\r\n
<Item>
        <DeviceID>34020000001320500162</DeviceID>\r\n
        <Name>34020000001320500162</Name>\r\n
        <FilePath>99000845100866_1_20231220150852.mp4</FilePath>\r\n
        <Address>rtsp://192.168.1.251:9913?vod=99000845100866_1_20231220150852&abc=1</Address>
        <StartTime>2023-12-20T15:08:52</StartTime>\r\n
        <EndTime>2023-12-20T15:09:07</EndTime>\r\n
        <Secrecy>0</Secrecy>\r\n
        <Type>time</Type>\r\n
        <RecorderID>32040000002000000010</RecorderID>
        </Item>
</RecordList>
</Response>`

	temp := &struct {
		XMLName      xml.Name
		CmdType      string // 命令类型
		SN           int    // 请求序列号，一般用于对应 request 和 response
		DeviceID     string // 下级设备 ID
		DeviceName   string
		Manufacturer string
		Model        string
		Channel      string
		RecordList   []*Record `xml:"RecordList>Item"`
		SumNum       int       // 录像结果的总数 SumNum，录像结果会按照多条消息返回，可用于判断是否全部返回
	}{}
	decoder := xml.NewDecoder(bytes.NewReader([]byte(body)))
	//decoder.Entity = map[string]string{
	//	"n": "(noun)",
	//}
	//decoder.Strict = false

	decoder.CharsetReader = charset.NewReaderLabel
	err := decoder.Decode(temp)
	if err != nil {
		t.Log(err)
		err = utils.DecodeGbk(temp, []byte(body))
		if err != nil {
			t.Log(err)
		}
	}
	fmt.Println(temp.RecordList[0])
}

func TestGenSSRC(t *testing.T) {
	var IsLive bool = true
	serial := "34020000002000000001"
	ssrc := make([]byte, 10)
	if IsLive {
		ssrc[0] = '0'
	} else {
		ssrc[0] = '1'
	}
	copy(ssrc[1:6], serial[3:8])
	randNum := 1000 + rand.Intn(8999)
	copy(ssrc[6:], strconv.Itoa(randNum))
	ssrcInt := string(ssrc)
	_ssrc, _ := strconv.ParseInt(ssrcInt, 10, 0)
	SSRC := uint32(_ssrc)
	fmt.Printf("%010d\n", SSRC)
}
