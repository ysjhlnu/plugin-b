package gb28181

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"golang.org/x/net/html/charset"
	"testing"
)

func TestDecode(t *testing.T) {

	g := struct {
		XMLName   xml.Name `xml:"SIP_XML"`
		Text      string   `xml:",chardata"`
		EventType string   `xml:"EventType,attr"`
	}{}

	body := `<?xml version="1.0" encoding="UTF-8"?>
<SIP_XML EventType=" Snapshot_Notify">
<Item Code="1" Type=" 文件类型 ,0: 图片, 其他值预留 " Time="抓拍时 间(时间 如20051110T132050Z)" FileUrl="抓拍图片的下载地址" FileSize="文件大小,单位:字节" Verfiy ="SHA256sum"/>
<Item Code=" 前端设备地址 编码 " Type=" 文件类型 ,0: 图片, 其他值预留 " Time="抓拍时 间(时间 如20051110T132050Z)" FileUrl="抓拍图片的下载地址" FileSize="文件大小,单位:字节" erfiy ="SHA256sum"/>
</SIP_XML>`

	decoder := xml.NewDecoder(bytes.NewReader([]byte(body)))
	decoder.CharsetReader = charset.NewReaderLabel
	err := decoder.Decode(g)
	if err != nil {
		t.Log(err)
		return
		//err = utils.DecodeGbk(g, []byte(req.Body()))
		//if err != nil {
		//	GB28181Plugin.Error("decode catelog err", zap.Error(err))
		//}
	}

	fmt.Printf("%#v\n", g)
}
