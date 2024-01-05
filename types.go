package b

import "encoding/xml"

type SubList struct {
	XMLName xml.Name           `xml:"SubList"`
	SubNum  string             `xml:"SubNum,attr"`
	Item    []resourceInfoItem `xml:"Item"`
}

// 上报前端系统的资源
type resourceInfoItem struct {
	Text       string `xml:",chardata"`
	Code       string `xml:"Code,attr"`       // 设备地址编码
	Name       string `xml:"Name,attr"`       // 名 称
	Status     string `xml:"Status,attr"`     // 节点状态值  0：不可用，1：可用
	DecoderTag string `xml:"DecoderTag,attr"` // 解 码 插 件 标 签
	Longitude  string `xml:"Longitude,attr"`  // 经度值
	Latitude   string `xml:"Latitude,attr"`   // 纬度值
	SubNum     string `xml:"SubNum,attr"`     // 包含的字节点数目
}
