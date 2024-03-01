package gb28181

import (
	"encoding/xml"
	"fmt"
	"m7s.live/plugin/gb28181/v4/utils"
	"math/rand"
	"strconv"
	"testing"
	"time"
)

func TestDecodeXML(t *testing.T) {

	body := `<?xml version="1.0" encoding="utf-8"?>
<Notify>
<CmdType>UploadSnapShotFinished</CmdType>
<SN>4</SN>
<DeviceID>15010201031321000011</DeviceID>
<SessionID>123</SessionID>
<SnapShotList><SnapShotFileID>15010201031321000011022024022114142500003</SnapShotFileID></SnapShotList>
</Notify>
`

	//temp := &struct {
	//	XMLName      xml.Name
	//	CmdType      string // 命令类型
	//	SN           int    // 请求序列号，一般用于对应 request 和 response
	//	DeviceID     string // 下级设备 ID
	//	DeviceName   string
	//	Manufacturer string
	//	Model        string
	//	Channel      string
	//	RecordList   []*Record `xml:"RecordList>Item"`
	//	SumNum       int       // 录像结果的总数 SumNum，录像结果会按照多条消息返回，可用于判断是否全部返回
	//}{}
	//decoder := xml.NewDecoder(bytes.NewReader([]byte(body)))
	////decoder.Entity = map[string]string{
	////	"n": "(noun)",
	////}
	////decoder.Strict = false
	//
	//decoder.CharsetReader = charset.NewReaderLabel
	//err := decoder.Decode(temp)
	//if err != nil {
	//	t.Log(err)
	//	err = utils.DecodeGbk(temp, []byte(body))
	//	if err != nil {
	//		t.Log(err)
	//	}
	//}
	type UploadSnapShotFinishedNotify struct {
		XMLName      xml.Name `xml:"Notify"`
		Text         string   `xml:",chardata"`
		CmdType      string   `xml:"CmdType"`
		SN           string   `xml:"SN"`
		DeviceID     string   `xml:"DeviceID"`
		SessionID    string   `xml:"SessionID"`
		SnapShotList struct {
			Text           string   `xml:",chardata"`
			SnapShotFileID []string `xml:"SnapShotFileID"`
		} `xml:"SnapShotList"`
	}

	tu := UploadSnapShotFinishedNotify{}
	if err := utils.DecodeXML([]byte(body), &tu); err != nil {
		GB28181Plugin.Error(err.Error())
		return
	}
	fmt.Println(tu.SnapShotList.SnapShotFileID)
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

func TestTrunc(t *testing.T) {
	id := "15010201031321000011022024022111244200003"
	timestamp := id[22:39]
	ctime, err := time.Parse("20060102150405000", timestamp)
	if err != nil {
		t.Log(err)
		return
	}
	t.Log(ctime)
	t.Log(timestamp)
}
