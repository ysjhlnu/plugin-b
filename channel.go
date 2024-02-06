package b

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"golang.org/x/net/html/charset"
	"net/http"
	"strconv"
	"strings"
	"time"

	"sync/atomic"

	"github.com/ghettovoice/gosip/sip"
	"github.com/goccy/go-json"
	"go.uber.org/zap"
	. "m7s.live/engine/v4"
	"m7s.live/engine/v4/log"
	"m7s.live/plugin/b/utils"
	"m7s.live/plugin/ps/v4"
)

var QUERY_RECORD_TIMEOUT = time.Second * 5

type PullStream struct {
	opt       *InviteOptions
	channel   *Channel
	inviteRes sip.Response
}

func (p *PullStream) CreateRequest(method sip.RequestMethod) (req sip.Request) {
	res := p.inviteRes
	req = p.channel.CreateRequst(method)
	from, _ := res.From()
	to, _ := res.To()
	callId, _ := res.CallID()
	req.ReplaceHeaders(from.Name(), []sip.Header{from})
	req.ReplaceHeaders(to.Name(), []sip.Header{to})
	req.ReplaceHeaders(callId.Name(), []sip.Header{callId})
	return
}

func (p *PullStream) Bye() int {
	req := p.CreateRequest(sip.BYE)
	resp, err := p.channel.device.SipRequestForResponse(req)
	if p.opt.IsLive() {
		p.channel.status.Store(0)
	}
	if p.opt.recyclePort != nil {
		p.opt.recyclePort(p.opt.MediaPort)
	}
	if err != nil {
		return http.StatusInternalServerError
	}
	return int(resp.StatusCode())
}

func (p *PullStream) info(body string) int {
	d := p.channel.device
	req := p.CreateRequest(sip.INFO)
	contentType := sip.ContentType("Application/MANSRTSP")
	req.AppendHeader(&contentType)
	req.SetBody(body, true)

	resp, err := d.SipRequestForResponse(req)
	if err != nil {
		log.Warnf("Send info to stream error: %v, stream=%s, body=%s", err, p.opt.StreamPath, body)
		return getSipRespErrorCode(err)
	}
	return int(resp.StatusCode())
}

// 暂停播放
func (p *PullStream) Pause() int {
	body := fmt.Sprintf(`PAUSE RTSP/1.0
CSeq: %d
PauseTime: now
`, p.channel.device.sn)
	return p.info(body)
}

// 恢复播放
func (p *PullStream) Resume() int {
	d := p.channel.device
	body := fmt.Sprintf(`PLAY RTSP/1.0
CSeq: %d
Range: npt=now-
`, d.sn)
	return p.info(body)
}

// 跳转到播放时间
// second: 相对于起始点调整到第 sec 秒播放
func (p *PullStream) PlayAt(second uint) int {
	d := p.channel.device
	body := fmt.Sprintf(`PLAY RTSP/1.0
CSeq: %d
Range: npt=%d-
`, d.sn, second)
	return p.info(body)
}

// 快进/快退播放
// speed 取值： 0.25 0.5 1 2 4 或者其对应的负数表示倒放
func (p *PullStream) PlayForward(speed float32) int {
	d := p.channel.device
	body := fmt.Sprintf(`PLAY RTSP/1.0
CSeq: %d
Scale: %0.6f
`, d.sn, speed)
	return p.info(body)
}

type Channel struct {
	device      *Device      // 所属设备
	status      atomic.Int32 // 通道状态,0:空闲,1:正在invite,2:正在播放
	LiveSubSP   string       // 实时子码流，通过rtsp
	GpsTime     time.Time    // gps时间
	Longitude   string       // 经度
	Latitude    string       // 纬度
	*log.Logger `json:"-" yaml:"-"`
	ChannelInfo
}

func (channel *Channel) MarshalJSON() ([]byte, error) {
	m := map[string]any{
		"DeviceID":     channel.DeviceID,
		"ParentID":     channel.ParentID,
		"Name":         channel.Name,
		"Manufacturer": channel.Manufacturer,
		"Model":        channel.Model,
		"Owner":        channel.Owner,
		"CivilCode":    channel.CivilCode,
		"Address":      channel.Address,
		"Port":         channel.Port,
		"Parental":     channel.Parental,
		"SafetyWay":    channel.SafetyWay,
		"RegisterWay":  channel.RegisterWay,
		"Secrecy":      channel.Secrecy,
		"Status":       channel.Status,
		"Longitude":    channel.Longitude,
		"Latitude":     channel.Latitude,
		"GpsTime":      channel.GpsTime,
		"LiveSubSP":    channel.LiveSubSP,
		"LiveStatus":   channel.status.Load(),
	}
	return json.Marshal(m)
}

// Channel 通道
type ChannelInfo struct {
	DeviceID     string // 通道ID
	ParentID     string
	Name         string
	Manufacturer string
	Model        string
	Owner        string
	CivilCode    string
	Address      string
	Longitude    string // 经度值
	Latitude     string // 纬度值
	Port         int
	Parental     int
	SafetyWay    int
	RegisterWay  int
	Secrecy      int
	Online       bool
	Status       ChannelStatus
}

type ChannelStatus string

const (
	ChannelOnStatus  = "ON"
	ChannelOffStatus = "OFF"
)

func (channel *Channel) CreateRequst(Method sip.RequestMethod) (req sip.Request) {
	d := channel.device
	d.sn++

	callId := sip.CallID(utils.RandNumString(10))
	userAgent := sip.UserAgentHeader("Monibuca")
	maxForwards := sip.MaxForwards(70) //增加max-forwards为默认值 70
	cseq := sip.CSeq{
		SeqNo:      uint32(d.sn),
		MethodName: Method,
	}
	port := sip.Port(conf.SipPort)
	serverAddr := sip.Address{
		//DisplayName: sip.String{Str: d.serverConfig.Serial},
		Uri: &sip.SipUri{
			FUser: sip.String{Str: conf.Serial},
			FHost: d.sipIP,
			FPort: &port,
		},
		Params: sip.NewParams().Add("tag", sip.String{Str: utils.RandNumString(9)}),
	}
	//非同一域的目标地址需要使用@host
	host := conf.Realm
	if channel.DeviceID[0:9] != host {
		if channel.Port != 0 {
			deviceIp := d.NetAddr
			deviceIp = deviceIp[0:strings.LastIndex(deviceIp, ":")]
			host = fmt.Sprintf("%s:%d", deviceIp, channel.Port)
		} else {
			host = d.NetAddr
		}
	}

	channelAddr := sip.Address{
		//DisplayName: sip.String{Str: d.serverConfig.Serial},
		Uri: &sip.SipUri{FUser: sip.String{Str: channel.DeviceID}, FHost: host},
	}
	req = sip.NewRequest(
		"",
		Method,
		channelAddr.Uri,
		"SIP/2.0",
		[]sip.Header{
			serverAddr.AsFromHeader(),
			channelAddr.AsToHeader(),
			&callId,
			&userAgent,
			&cseq,
			&maxForwards,
			serverAddr.AsContactHeader(),
		},
		"",
		nil,
	)

	req.SetTransport(conf.SipNetwork)
	req.SetDestination(d.NetAddr)
	return req
}

func (channel *Channel) QueryRecord(videoType, startTime, endTime string) ([]RespVideoRetrievalItem, error) {
	d := channel.device
	request := d.CreateRequest(sip.MESSAGE)
	contentType := sip.ContentType("application/xml")
	request.AppendHeader(&contentType)
	start, _ := strconv.ParseInt(startTime, 10, 0)
	end, _ := strconv.ParseInt(endTime, 10, 0)

	item := RequestHistoryVideoItem{
		Code:      d.ID,
		Type:      0xFFFFFFFF,
		UserCode:  channel.DeviceID,
		BeginTime: intTotime(start).Format("2006-01-02T15:04:05Z"),
		EndTime:   intTotime(end).Format("2006-01-02T15:04:05Z"),
		FromInx:   1,
		ToIndex:   10,
	}

	req := RequestHistoryVideoSIPXML{
		EventType: "Request_History_Video",
		Item:      item,
	}

	reqRaw, err := xml.MarshalIndent(req, " ", "  ")
	if err != nil {
		return nil, err
	}

	request.SetBody(xml.Header+string(reqRaw), true)
	BPlugin.Sugar().Debugf("Request_History_Video: \n%s", request.String())
	//resultCh := RecordQueryLink.WaitResult(d.ID, channel.DeviceID, d.sn, QUERY_RECORD_TIMEOUT)

	resp, err := d.SipRequestForResponse(request)
	if err != nil {
		return nil, fmt.Errorf("query error: %s", err)
	}

	BPlugin.Sugar().Debugf("Request_History_Video: res \n%s", resp.String())

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("query error, status=%d", resp.StatusCode())
	}

	resXml := RespVideoRetrieval{}
	decoder := xml.NewDecoder(bytes.NewReader([]byte(resp.Body())))
	decoder.Strict = false //关闭强制char值的严格模式  处理invalid character entity &trackID (no semicolon)问题 原因:在属性值和字符数据中，不处理未知或格式错误的字符实体(以&开头的序列)
	decoder.CharsetReader = charset.NewReaderLabel
	err = decoder.Decode(&resXml)
	if err != nil {
		err = utils.DecodeGbk(&resXml, []byte(resp.Body()))
		if err != nil {
			BPlugin.Error("decode catelog err", zap.Error(err))
			return nil, fmt.Errorf("query error, status=%d", resp.StatusCode())
		}
	}

	// RecordQueryLink 中加了超时机制，该结果一定会返回
	// 所以此处不用再增加超时等保护机制
	//r := <-resultCh
	//return r.list, r.err
	return resXml.SubList.Item, nil
}

type CtrlXML struct {
	XMLName   xml.Name `xml:"SIP_XML"`
	Text      string   `xml:",chardata"`
	EventType string   `xml:"EventType,attr"`
	Item      ctrlItem `xml:"Item"`
}

type ctrlItem struct {
	Text         string `xml:",chardata"`
	Code         string `xml:"Code,attr"`
	Command      int32  `xml:"Command,attr"`
	CommandPara1 int32  `xml:"CommandPara1,attr"`
	CommandPara2 int32  `xml:"CommandPara2,attr"`
	CommandPara3 int32  `xml:"CommandPara3,attr"`
}

func (channel *Channel) Control(PTZCmd, p1, p2 uint64) int {
	d := channel.device
	request := d.CreateRequest(sip.MESSAGE)
	contentType := sip.ContentType("application/xml")
	request.AppendHeader(&contentType)

	item := ctrlItem{
		Code:         channel.DeviceID,
		Command:      int32(PTZCmd),
		CommandPara1: int32(p1),
		CommandPara2: int32(p2),
	}

	s := CtrlXML{
		EventType: "Control_Camera",
		Item:      item,
	}

	//raw, err := xml.Marshal(s)
	raw, err := xml.MarshalIndent(s, " ", "  ")
	if err != nil {
		fmt.Println(err)
		return -1
	}

	dest := make([]byte, len(xml.Header)+len(raw))
	copy(dest, xml.Header)
	copy(dest[len(xml.Header):], raw)

	request.SetBody(string(dest), true)

	BPlugin.Sugar().Debugf("ptz control: \n%s", request.String())
	resp, err := d.SipRequestForResponse(request)
	if err != nil {
		return http.StatusRequestTimeout
	}
	BPlugin.Sugar().Debugf("ptz control res: \n%s", resp.String())

	return int(resp.StatusCode())
}

// Invite 发送Invite报文 invites a channel to play
// 注意里面的锁保证不同时发送invite报文，该锁由channel持有
/***
f字段： f = v/编码格式/分辨率/帧率/码率类型/码率大小a/编码格式/码率大小/采样率
各项具体含义：
    v：后续参数为视频的参数；各参数间以 “/”分割；
编码格式：十进制整数字符串表示
1 –MPEG-4 2 –H.264 3 – SVAC 4 –3GP
    分辨率：十进制整数字符串表示
1 – QCIF 2 – CIF 3 – 4CIF 4 – D1 5 –720P 6 –1080P/I
帧率：十进制整数字符串表示 0～99
码率类型：十进制整数字符串表示
1 – 固定码率（CBR）     2 – 可变码率（VBR）
码率大小：十进制整数字符串表示 0～100000（如 1表示1kbps）
    a：后续参数为音频的参数；各参数间以 “/”分割；
编码格式：十进制整数字符串表示
1 – G.711    2 – G.723.1     3 – G.729      4 – G.722.1
码率大小：十进制整数字符串
音频编码码率： 1 — 5.3 kbps （注：G.723.1中使用）
   2 — 6.3 kbps （注：G.723.1中使用）
   3 — 8 kbps （注：G.729中使用）
   4 — 16 kbps （注：G.722.1中使用）
   5 — 24 kbps （注：G.722.1中使用）
   6 — 32 kbps （注：G.722.1中使用）
   7 — 48 kbps （注：G.722.1中使用）
   8 — 64 kbps（注：G.711中使用）
采样率：十进制整数字符串表示
	1 — 8 kHz（注：G.711/ G.723.1/ G.729中使用）
	2—14 kHz（注：G.722.1中使用）
	3—16 kHz（注：G.722.1中使用）
	4—32 kHz（注：G.722.1中使用）
	注1：字符串说明
本节中使用的“十进制整数字符串”的含义为“0”～“4294967296” 之间的十进制数字字符串。
注2：参数分割标识
各参数间以“/”分割，参数间的分割符“/”不能省略；
若两个分割符 “/”间的某参数为空时（即两个分割符 “/”直接将相连时）表示无该参数值；
注3：f字段说明
使用f字段时，应保证视频和音频参数的结构完整性，即在任何时候，f字段的结构都应是完整的结构：
f = v/编码格式/分辨率/帧率/码率类型/码率大小a/编码格式/码率大小/采样率
若只有视频时，音频中的各参数项可以不填写，但应保持 “a///”的结构:
f = v/编码格式/分辨率/帧率/码率类型/码率大小a///
若只有音频时也类似处理，视频中的各参数项可以不填写，但应保持 “v/”的结构：
f = v/a/编码格式/码率大小/采样率
f字段中视、音频参数段之间不需空格分割。
可使用f字段中的分辨率参数标识同一设备不同分辨率的码流。
*/

func (channel *Channel) Invite(opt *InviteOptions) (code int, err error) {
	if opt.IsLive() {
		if !channel.status.CompareAndSwap(0, 1) {
			return 304, nil
		}
		defer func() {
			if err != nil {
				BPlugin.Error("Invite", zap.Error(err))
				channel.status.Store(0)
				if conf.InviteMode == 1 {
					// 5秒后重试
					time.AfterFunc(time.Second*5, func() {
						channel.Invite(opt)
					})
				}
			} else {
				channel.status.Store(2)
			}
		}()
	}
	d := channel.device
	streamPath := fmt.Sprintf("%s/%s", d.ID, channel.DeviceID)
	s := "Play"
	opt.CreateSSRC()
	if opt.Record() {
		s = "Playback"
		streamPath = fmt.Sprintf("%s/%s/%d-%d", d.ID, channel.DeviceID, opt.Start, opt.End)
	}
	if opt.StreamPath != "" {
		streamPath = opt.StreamPath
	} else {
		opt.StreamPath = streamPath
	}
	if opt.dump == "" {
		opt.dump = conf.DumpPath
	}
	protocol := ""
	networkType := "udp"
	reusePort := true
	if conf.IsMediaNetworkTCP() {
		networkType = "tcp"
		protocol = "TCP/"
		if conf.tcpPorts.Valid {
			opt.MediaPort, err = conf.tcpPorts.GetPort()
			opt.recyclePort = conf.tcpPorts.Recycle
			reusePort = false
		}
	} else {
		if conf.udpPorts.Valid {
			opt.MediaPort, err = conf.udpPorts.GetPort()
			opt.recyclePort = conf.udpPorts.Recycle
			reusePort = false
		}
	}
	if err != nil {
		return http.StatusInternalServerError, err
	}
	if opt.MediaPort == 0 {
		opt.MediaPort = conf.MediaPort
	}

	sdpInfo := []string{
		"v=0",
		fmt.Sprintf("o=- 0 0 IN IP4 %s", d.mediaIP),
		"s=" + s,
		"c=IN IP4 " + d.mediaIP,
		//"u=" + channel.DeviceID + ":0",

		//opt.String(),
		fmt.Sprintf("m=video %d %sRTP/AVP 100", opt.MediaPort, protocol),
		"y=" + opt.ssrc,
		"a=rtpmap:100 H264/90000",
		"a=fmtp:100 CIF=1;4CIF=1;F=1;K=1",
		"a=rate:main",
		"a=recvonly",
	}
	if conf.IsMediaNetworkTCP() {
		sdpInfo = append(sdpInfo, "a=setup:passive", "a=connection:new")
	}
	invite := channel.CreateRequst(sip.INVITE)
	contentType := sip.ContentType("application/sdp")
	invite.AppendHeader(&contentType)

	invite.SetBody(strings.Join(sdpInfo, "\r\n")+"\r\n", true)

	subject := sip.GenericHeader{
		HeaderName: "Subject", Contents: fmt.Sprintf("%s:%s,%s:0", channel.DeviceID, opt.ssrc, conf.Serial),
	}
	invite.AppendHeader(&subject)
	BPlugin.Sugar().Debugf("invite: \n%s", invite.String())
	inviteRes, err := d.SipRequestForResponse(invite)
	if err != nil {
		channel.Error("invite", zap.Error(err), zap.String("msg", invite.String()))
		return http.StatusInternalServerError, err
	}
	code = int(inviteRes.StatusCode())
	channel.Info("invite response", zap.Int("status code", code))

	if code == http.StatusOK {
		ds := strings.Split(inviteRes.Body(), "\r\n")
		for _, l := range ds {
			if ls := strings.Split(l, "="); len(ls) > 1 {
				if ls[0] == "y" && len(ls[1]) > 0 {
					if _ssrc, err := strconv.ParseInt(ls[1], 10, 0); err == nil {
						opt.SSRC = uint32(_ssrc)
					} else {
						channel.Error("read invite response y ", zap.Error(err))
					}
					//	break
				}
				if ls[0] == "m" && len(ls[1]) > 0 {
					netinfo := strings.Split(ls[1], " ")
					if strings.ToUpper(netinfo[2]) == "TCP/RTP/AVP" {
						channel.Debug("Device support tcp")
					} else {
						channel.Debug("Device not support tcp")
						networkType = "udp"
					}
				}
			}
		}
		err = ps.Receive(streamPath, opt.dump, fmt.Sprintf("%s:%d", networkType, opt.MediaPort), opt.SSRC, reusePort)
		if err == nil {
			PullStreams.Store(streamPath, &PullStream{
				opt:       opt,
				channel:   channel,
				inviteRes: inviteRes,
			})
			err = srv.Send(sip.NewAckRequest("", invite, inviteRes, "", nil))
		}
	}
	return
}

func (channel *Channel) Bye(streamPath string) int {
	d := channel.device
	if streamPath == "" {
		streamPath = fmt.Sprintf("%s/%s", d.ID, channel.DeviceID)
	}
	if s, loaded := PullStreams.LoadAndDelete(streamPath); loaded {
		s.(*PullStream).Bye()
		if s := Streams.Get(streamPath); s != nil {
			s.Close()
		}
		return http.StatusOK
	}
	return http.StatusNotFound
}

func (channel *Channel) Pause(streamPath string) int {
	if s, loaded := PullStreams.Load(streamPath); loaded {
		r := s.(*PullStream).Pause()
		if s := Streams.Get(streamPath); s != nil {
			s.Pause()
		}
		return r
	}
	return http.StatusNotFound
}

func (channel *Channel) Resume(streamPath string) int {
	if s, loaded := PullStreams.Load(streamPath); loaded {
		r := s.(*PullStream).Resume()
		if s := Streams.Get(streamPath); s != nil {
			s.Resume()
		}
		return r
	}
	return http.StatusNotFound
}

func (channel *Channel) PlayAt(streamPath string, second uint) int {
	if s, loaded := PullStreams.Load(streamPath); loaded {
		r := s.(*PullStream).PlayAt(second)
		if s := Streams.Get(streamPath); s != nil {
			s.Resume()
		}
		return r
	}
	return http.StatusNotFound
}

func (channel *Channel) PlayForward(streamPath string, speed float32) int {
	if s, loaded := PullStreams.Load(streamPath); loaded {
		return s.(*PullStream).PlayForward(speed)
	}
	if s := Streams.Get(streamPath); s != nil {
		s.Resume()
	}
	return http.StatusNotFound
}

func (channel *Channel) TryAutoInvite(opt *InviteOptions) {
	condition := !opt.IsLive() || channel.CanInvite()
	channel.Debug("TryAutoInvite", zap.Any("opt", opt), zap.Bool("condition", condition))
	if condition {
		go channel.Invite(opt)
	}
}

func (channel *Channel) CanInvite() bool {
	if channel.status.Load() != 0 || len(channel.DeviceID) != 20 || channel.Status == ChannelOffStatus {
		return false
	}

	if conf.InviteIDs == "" {
		return true
	}

	// 11～13位是设备类型编码
	typeID := channel.DeviceID[10:13]

	// format: start-end,type1,type2
	tokens := strings.Split(conf.InviteIDs, ",")
	for _, tok := range tokens {
		if first, second, ok := strings.Cut(tok, "-"); ok {
			if typeID >= first && typeID <= second {
				return true
			}
		} else {
			if typeID == first {
				return true
			}
		}
	}

	return false
}

func getSipRespErrorCode(err error) int {
	if re, ok := err.(*sip.RequestError); ok {
		return int(re.Code)
	} else {
		return http.StatusInternalServerError
	}
}

// Capture 图片抓拍
func (channel *Channel) Capture(imgSrv, timeRange string, snapType, interval int) int {
	d := channel.device
	request := d.CreateRequest(sip.MESSAGE)
	contentType := sip.ContentType("application/xml")
	request.AppendHeader(&contentType)

	item := snapshotItem{
		Code:      channel.DeviceID,
		PicServer: imgSrv,
		Range:     timeRange,
		SnapType:  int32(snapType),
		Interval:  int32(interval),
	}

	reqXml := ReqSnapshot{
		EventType: "Camera_Snap",
		Item:      item,
	}
	raw, err := xml.MarshalIndent(reqXml, " ", "  ")
	if err != nil {
		BPlugin.Error(err.Error())
		return -1
	}

	//body := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
	//<SIP_XML EventType="Camera_Snap">
	//	<Item Code="%s" PicServer="%s" SnapType=%s Range="%s" Interval=%s>
	//</SIP_XML>`,
	//	channel.DeviceID, imgSrv, snapType, timeRange, interval)

	request.SetBody(xml.Header+string(raw), true)
	BPlugin.Sugar().Debugf("SIP->Camera_Snap:\n%s", request)

	resp, err := d.SipRequestForResponse(request)
	if err != nil {
		return http.StatusRequestTimeout
	}

	BPlugin.Sugar().Debugf("Camera_Snap resp:\n%s", resp)

	return int(resp.StatusCode())
}
