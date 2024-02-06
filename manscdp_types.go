package b

import "encoding/xml"

type RequestHistoryVideoSIPXML struct {
	XMLName   xml.Name                `xml:"SIP_XML"`
	Text      string                  `xml:",chardata"`
	EventType string                  `xml:"EventType,attr"`
	Item      RequestHistoryVideoItem `xml:"Item"`
}

type RequestHistoryVideoItem struct {
	Text      string `xml:",chardata"`
	Code      string `xml:"Code,attr"`
	Type      int    `xml:"Type,attr"`
	UserCode  string `xml:"UserCode,attr"`
	BeginTime string `xml:"BeginTime,attr"`
	EndTime   string `xml:"EndTime,attr"`
	FromInx   int32  `xml:"FromInx,attr"`
	ToIndex   int32  `xml:"ToIndex,attr"`
}

type ReqSnapshot struct {
	XMLName   xml.Name     `xml:"SIP_XML"`
	Text      string       `xml:",chardata"`
	EventType string       `xml:"EventType,attr"`
	Item      snapshotItem `xml:"Item"`
}

type snapshotItem struct {
	Text      string `xml:",chardata"`
	Code      string `xml:"Code,attr"`
	PicServer string `xml:"PicServer,attr"`
	Range     string `xml:"Range,attr"`
	SnapType  int32  `xml:"SnapType,attr"`
	Interval  int32  `xml:"Interval,attr"`
}

type RespVideoRetrieval struct {
	XMLName   xml.Name `xml:"SIP_XML"`
	Text      string   `xml:",chardata"`
	EventType string   `xml:"EventType,attr"`
	SubList   struct {
		Text      string                   `xml:",chardata"`
		RealNum   string                   `xml:"RealNum,attr"`
		SubNum    string                   `xml:"SubNum,attr"`
		FromIndex string                   `xml:"FromIndex,attr"`
		ToIndex   string                   `xml:"ToIndex,attr"`
		Item      []RespVideoRetrievalItem `xml:"Item"`
	} `xml:"SubList"`
}

type RespVideoRetrievalItem struct {
	Text       string `xml:",chardata"`
	FileName   string `xml:"FileName,attr"`
	FileUrl    string `xml:"FileUrl,attr"`
	BeginTime  string `xml:"BeginTime,attr"`
	EndTime    string `xml:"EndTime,attr"`
	Size       string `xml:"Size,attr"`
	DecoderTag string `xml:"DecoderTag,attr"`
	Type       string `xml:"Type,attr"`
}
