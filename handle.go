package b

import (
	"bytes"
	"crypto/md5"
	"encoding/xml"
	"fmt"
	"strconv"

	"go.uber.org/zap"
	"m7s.live/plugin/gb28181/v4/utils"

	"github.com/ghettovoice/gosip/sip"

	"net/http"
	"time"

	"golang.org/x/net/html/charset"
)

type Authorization struct {
	*sip.Authorization
}

func (a *Authorization) Verify(username, passwd, realm, nonce string) bool {

	//1、将 username,realm,password 依次组合获取 1 个字符串，并用算法加密的到密文 r1
	s1 := fmt.Sprintf("%s:%s:%s", username, realm, passwd)
	r1 := a.getDigest(s1)
	//2、将 method，即REGISTER ,uri 依次组合获取 1 个字符串，并对这个字符串使用算法 加密得到密文 r2
	s2 := fmt.Sprintf("REGISTER:%s", a.Uri())
	r2 := a.getDigest(s2)

	if r1 == "" || r2 == "" {
		BPlugin.Error("Authorization algorithm wrong")
		return false
	}
	//3、将密文 1，nonce 和密文 2 依次组合获取 1 个字符串，并对这个字符串使用算法加密，获得密文 r3，即Response
	s3 := fmt.Sprintf("%s:%s:%s", r1, nonce, r2)
	r3 := a.getDigest(s3)

	//4、计算服务端和客户端上报的是否相等
	return r3 == a.Response()
}

func (a *Authorization) getDigest(raw string) string {
	switch a.Algorithm() {
	case "MD5":
		return fmt.Sprintf("%x", md5.Sum([]byte(raw)))
	default: //如果没有算法，默认使用MD5
		return fmt.Sprintf("%x", md5.Sum([]byte(raw)))
	}
}

func (c *BConfig) OnRegister(req sip.Request, tx sip.ServerTransaction) {
	from, ok := req.From()
	if !ok || from.Address == nil {
		BPlugin.Error("OnRegister", zap.String("error", "no from"))
		return
	}
	id := from.Address.User().String()

	BPlugin.Debug("SIP<-OnMessage", zap.String("id", id), zap.String("source", req.Source()), zap.String("req", req.String()))

	isUnregister := false
	if exps := req.GetHeaders("Expires"); len(exps) > 0 {
		exp := exps[0]
		expSec, err := strconv.ParseInt(exp.Value(), 10, 32)
		if err != nil {
			BPlugin.Info("OnRegister",
				zap.String("error", fmt.Sprintf("wrong expire header value %q", exp)),
				zap.String("id", id),
				zap.String("source", req.Source()),
				zap.String("destination", req.Destination()))
			return
		}
		if expSec == 0 {
			isUnregister = true
		}
	} else {
		BPlugin.Info("OnRegister",
			zap.String("error", "has no expire header"),
			zap.String("id", id),
			zap.String("source", req.Source()),
			zap.String("destination", req.Destination()))
		return
	}

	BPlugin.Info("OnRegister",
		zap.Bool("isUnregister", isUnregister),
		zap.String("id", id),
		zap.String("source", req.Source()),
		zap.String("destination", req.Destination()))

	if len(id) != 18 {
		BPlugin.Info("Wrong B-Interface 编码长度错误", zap.String("id", id))
		return
	}
	passAuth := false
	// 不需要密码情况
	if c.Username == "" && c.Password == "" {
		passAuth = true
	} else {
		// 需要密码情况 设备第一次上报，返回401和加密算法
		if hdrs := req.GetHeaders("Authorization"); len(hdrs) > 0 {
			authenticateHeader := hdrs[0].(*sip.GenericHeader)
			auth := &Authorization{sip.AuthFromValue(authenticateHeader.Contents)}

			// 有些摄像头没有配置用户名的地方，用户名就是摄像头自己的国标id
			var username string
			if auth.Username() == id {
				username = id
			} else {
				username = c.Username
			}

			if dc, ok := DeviceRegisterCount.LoadOrStore(id, 1); ok && dc.(int) > MaxRegisterCount {
				response := sip.NewResponseFromRequest("", req, http.StatusForbidden, "Forbidden", "")
				tx.Respond(response)
				return
			} else {
				// 设备第二次上报，校验
				_nonce, loaded := DeviceNonce.Load(id)
				if loaded && auth.Verify(username, c.Password, c.Realm, _nonce.(string)) {
					passAuth = true
				} else {
					DeviceRegisterCount.Store(id, dc.(int)+1)
				}
			}
		}
	}
	if passAuth {
		// 通过校验
		BPlugin.Info("密码校验通过")
		var d *Device
		if isUnregister {
			tmpd, ok := Devices.LoadAndDelete(id)
			if ok {
				BPlugin.Info("Unregister Device", zap.String("id", id))
				d = tmpd.(*Device)
			} else {
				return
			}
		} else {
			if v, ok := Devices.Load(id); ok {
				d = v.(*Device)
				c.RecoverDevice(d, req)
			} else {
				d = c.StoreDevice(id, req)
			}
		}

		// 删除nonce,在刷新注册的时候会提示401然后返回nonce再注册成功
		BPlugin.Sugar().Infof("%s--删除nonce", id)
		BPlugin.Sugar().Infof("%s--删除注册次数", id)
		DeviceNonce.Delete(id)
		DeviceRegisterCount.Delete(id)

		resp := sip.NewResponseFromRequest("", req, http.StatusOK, "OK", "")
		to, _ := resp.To()
		resp.ReplaceHeaders("To", []sip.Header{&sip.ToHeader{Address: to.Address, Params: sip.NewParams().Add("tag", sip.String{Str: utils.RandNumString(9)})}})
		resp.RemoveHeader("Allow")
		expires := sip.Expires(3600)
		resp.AppendHeader(&expires)
		resp.AppendHeader(&sip.GenericHeader{
			HeaderName: "Date",
			Contents:   time.Now().Format(TIME_LAYOUT),
		})
		_ = tx.Respond(resp)

		BPlugin.Debug("获取设备信息")
		if !isUnregister {
			go d.QueryDeviceInfo()
			go d.DeviceWorkInfo(0xFFFFFFFF)

			go d.GetFrontAbility() // 获取前端支持能力集

			go d.syncChannels()
		}
	} else {
		// 未通过校验
		BPlugin.Info("密码未通过校验")
		BPlugin.Info("OnRegister unauthorized", zap.String("id", id), zap.String("source", req.Source()),
			zap.String("destination", req.Destination()))
		response := sip.NewResponseFromRequest("", req, http.StatusUnauthorized, "Unauthorized", "")
		_nonce, _ := DeviceNonce.LoadOrStore(id, utils.RandNumString(32))
		auth := fmt.Sprintf(
			`Digest realm="%s",algorithm=%s,nonce="%s"`,
			c.Realm,
			"MD5",
			_nonce.(string),
		)
		response.AppendHeader(&sip.GenericHeader{
			HeaderName: "WWW-Authenticate",
			Contents:   auth,
		})
		_ = tx.Respond(response)
	}
}

// syncChannels
// 同步设备信息、下属通道信息，包括主动查询通道信息，订阅通道变化情况
func (d *Device) syncChannels() {
	if time.Since(d.lastSyncTime) > 2*conf.HeartbeatInterval {
		d.lastSyncTime = time.Now()
		//d.Catalog()
		//d.Subscribe()
		//d.QueryDeviceInfo()
	}
}

func (c *BConfig) OnMessage(req sip.Request, tx sip.ServerTransaction) {
	from, ok := req.From()
	if !ok || from.Address == nil {
		BPlugin.Error("OnMessage", zap.String("error", "no from"))
		return
	}
	id := from.Address.User().String()
	BPlugin.Debug("SIP<-OnMessage", zap.String("id", id), zap.String("source", req.Source()), zap.String("req", req.String()))
	if v, ok := Devices.Load(id); ok {
		d := v.(*Device)
		switch d.Status {
		case DeviceOfflineStatus, DeviceRecoverStatus:
			c.RecoverDevice(d, req)
			go d.syncChannels()
		case DeviceRegisterStatus:
			d.Status = DeviceOnlineStatus
		}
		d.UpdateTime = time.Now()

		gen := &struct {
			XMLName   xml.Name `xml:"SIP_XML"`
			Text      string   `xml:",chardata"`
			EventType string   `xml:"EventType,attr"`
		}{}

		//temp := &struct {
		//	XMLName      xml.Name
		//	CmdType      string
		//	SN           int // 请求序列号，一般用于对应 request 和 response
		//	DeviceID     string
		//	DeviceName   string
		//	Manufacturer string
		//	Model        string
		//	Channel      string
		//	DeviceList   []ChannelInfo `xml:"DeviceList>Item"`
		//	RecordList   []*Record     `xml:"RecordList>Item"`
		//	SumNum       int           // 录像结果的总数 SumNum，录像结果会按照多条消息返回，可用于判断是否全部返回
		//}{}

		decoder := xml.NewDecoder(bytes.NewReader([]byte(req.Body())))
		decoder.CharsetReader = charset.NewReaderLabel
		err := decoder.Decode(gen)
		if err != nil {
			err = utils.DecodeGbk(gen, []byte(req.Body()))
			if err != nil {
				BPlugin.Error("decode catelog err", zap.Error(err))
			}
		}

		var body string
		switch gen.EventType {
		case "Keepalive":
			d.LastKeepaliveAt = time.Now()
			//callID !="" 说明是订阅的事件类型信息
			if d.lastSyncTime.IsZero() {
				go d.syncChannels()
			} else {
				d.channelMap.Range(func(key, value interface{}) bool {
					if conf.InviteMode == INVIDE_MODE_AUTO {
						value.(*Channel).TryAutoInvite(&InviteOptions{})
					}
					return true
				})
			}
			//在KeepLive 进行位置订阅的处理，如果开启了自动订阅位置，则去订阅位置
			if c.Position.AutosubPosition && time.Since(d.GpsTime) > c.Position.Interval*2 {
				d.MobilePositionSubscribe(d.ID, c.Position.Expires, c.Position.Interval)
				BPlugin.Debug("Mobile Position Subscribe", zap.String("deviceID", d.ID))
			}
		case "Catalog":
			//d.UpdateChannels(temp.DeviceList...)
		case "RecordInfo":
			//RecordQueryLink.Put(d.ID, temp.DeviceID, temp.SN, temp.SumNum, temp.RecordList)
		case "Alarm":
			d.Status = DeviceAlarmedStatus
			body = BuildAlarmResponseXML(d.ID)
		case "Station_Response_GetSystemInfo":
			d.Debug("设备基本信息")

			devInfo := &struct {
				XMLName   xml.Name `xml:"SIP_XML"`
				Text      string   `xml:",chardata"`
				EventType string   `xml:"EventType,attr"`
				Item      struct {
					Text    string `xml:",chardata"`
					Code    string `xml:"Code,attr"`
					Valid   string `xml:"Valid,attr"`
					Version struct {
						Text     string `xml:",chardata"`
						Software string `xml:"Software,attr"`
						Hardware string `xml:"Hardware,attr"`
					} `xml:"Version"`
					Device struct {
						Text         string `xml:",chardata"`
						Manufacturer string `xml:"Manufacturer,attr"`
						Model        string `xml:"Model,attr"`
						CameraNum    string `xml:"CameraNum,attr"`
					} `xml:"Device"`
				} `xml:"Item"`
			}{}

			decoder := xml.NewDecoder(bytes.NewReader([]byte(req.Body())))
			decoder.CharsetReader = charset.NewReaderLabel
			err := decoder.Decode(devInfo)
			if err != nil {
				err = utils.DecodeGbk(devInfo, []byte(req.Body()))
				if err != nil {
					BPlugin.Error("decode Station_Response_GetSystemInfo err", zap.Error(err))
				}
			}

			BPlugin.Sugar().Debugf("%#v", devInfo)

			// 主设备信息
			d.Name = devInfo.Item.Code
			d.Code = devInfo.Item.Code
			d.Manufacturer = devInfo.Item.Device.Manufacturer
			d.Model = devInfo.Item.Device.Model
			d.Software = devInfo.Item.Version.Software
			d.Hardware = devInfo.Item.Version.Hardware
			d.CameraNum = devInfo.Item.Device.CameraNum

		case "Station_Request_SetVideoParm":
			d.Debug("设备工作状态获取")

			workState := &struct {
				XMLName   xml.Name `xml:"SIP_XML"`
				Text      string   `xml:",chardata"`
				EventType string   `xml:"EventType,attr"`
				SubList   struct {
					Text   string `xml:",chardata"`
					Code   string `xml:"Code,attr"`
					SubNum string `xml:"SubNum,attr"`
					Item   []struct {
						Text         string `xml:",chardata"`
						Code         string `xml:"Code,attr"`
						InfoType     string `xml:"InfoType,attr"`
						DeviceStatus string `xml:"DeviceStatus,attr"`
						DiskStatus   string `xml:"DiskStatus,attr"`
						ChannelsNum  string `xml:"ChannelsNum,attr"`
						VideoChannel []struct {
							Text               string `xml:",chardata"`
							Code               string `xml:"Code,attr"`
							InfoType           string `xml:"InfoType,attr"`
							ChannelRecord      string `xml:"ChannelRecord,attr"`
							Status             string `xml:"Status,attr"`
							ChannelVideoStatus string `xml:"ChannelVideoStatus,attr"`
							ChannelClientNum   string `xml:"ChannelClientNum,attr"`
						} `xml:"VideoChannel"`
					} `xml:"Item"`
				} `xml:"SubList"`
			}{}

			decoder := xml.NewDecoder(bytes.NewReader([]byte(req.Body())))
			decoder.CharsetReader = charset.NewReaderLabel
			err := decoder.Decode(workState)
			if err != nil {
				err = utils.DecodeGbk(workState, []byte(req.Body()))
				if err != nil {
					BPlugin.Error("decode Station_Response_GetSystemInfo err", zap.Error(err))
				}
			}

			BPlugin.Sugar().Debugf("%#v", workState)

		case "Station_Response_GetCapability":
			d.Debug("响应获取前端能力集")

			ability := &struct {
				XMLName   xml.Name `xml:"SIP_XML"`
				Text      string   `xml:",chardata"`
				EventType string   `xml:"EventType,attr"`
				Public    struct {
					Text  string `xml:",chardata"`
					Code  string `xml:"Code,attr"`
					Valid string `xml:"Valid,attr"`
					Item  []struct {
						Text  string `xml:",chardata"`
						Name  string `xml:"Name,attr"`
						Value string `xml:"Value,attr"`
					} `xml:"Item"`
				} `xml:"Public"`
			}{}

			decoder := xml.NewDecoder(bytes.NewReader([]byte(req.Body())))
			decoder.CharsetReader = charset.NewReaderLabel
			err := decoder.Decode(ability)
			if err != nil {
				err = utils.DecodeGbk(ability, []byte(req.Body()))
				if err != nil {
					BPlugin.Error("decode Station_Response_GetSystemInfo err", zap.Error(err))
				}
			}

			BPlugin.Sugar().Debugf("%#v", ability)

		default:
			d.Warn("Not supported EventType", zap.String("EventType", gen.EventType), zap.String("body", req.Body()))
			response := sip.NewResponseFromRequest("", req, http.StatusBadRequest, "", "")
			tx.Respond(response)
			return
		}

		tx.Respond(sip.NewResponseFromRequest("", req, http.StatusOK, "OK", body))
	} else {
		BPlugin.Debug("Unauthorized message, device not found", zap.String("id", id))
	}
}
func (c *BConfig) OnBye(req sip.Request, tx sip.ServerTransaction) {
	tx.Respond(sip.NewResponseFromRequest("", req, http.StatusOK, "OK", ""))
}

// OnNotify 订阅通知处理
func (c *BConfig) OnNotify(req sip.Request, tx sip.ServerTransaction) {
	from, ok := req.From()
	if !ok || from.Address == nil {
		BPlugin.Error("OnNotify", zap.String("error", "no from"))
		return
	}
	id := from.Address.User().String()
	if v, ok := Devices.Load(id); ok {
		d := v.(*Device)
		d.UpdateTime = time.Now()

		g := &struct {
			XMLName   xml.Name `xml:"SIP_XML"`
			Text      string   `xml:",chardata"`
			EventType string   `xml:"EventType,attr"`
		}{}

		decoder := xml.NewDecoder(bytes.NewReader([]byte(req.Body())))
		decoder.CharsetReader = charset.NewReaderLabel
		err := decoder.Decode(g)
		if err != nil {
			err = utils.DecodeGbk(g, []byte(req.Body()))
			if err != nil {
				BPlugin.Error("decode catelog err", zap.Error(err))
			}
		}

		BPlugin.Sugar().Debugf("通用解析： %#v", g)

		temp := &struct {
			XMLName   xml.Name `xml:"SIP_XML"`
			Text      string   `xml:",chardata"`
			EventType string   `xml:"EventType,attr"`
			Code      struct {
				Text string `xml:",chardata"` // 父节点（平台、场所、前端设备）地址编码
			} `xml:"Code"`
			SubList struct { // 场地、前端设备、摄像机的地址编码
				Text   string `xml:",chardata"`
				SubNum string `xml:"SubNum,attr"`
				Item   []struct {
					Text       string `xml:",chardata"`
					Code       string `xml:"Code,attr"`       // 设备地址编码
					Name       string `xml:"Name,attr"`       // 名 称
					Status     string `xml:"Status,attr"`     // 节点状态值  0：不可用，1：可用
					DecoderTag string `xml:"DecoderTag,attr"` // 解 码 插 件 标 签
					Longitude  string `xml:"Longitude,attr"`  // 经度值
					Latitude   string `xml:"Latitude,attr"`   // 纬度值
					SubNum     string `xml:"SubNum,attr"`     // 包含的字节点数目
				} `xml:"Item"`
			} `xml:"SubList"`
		}{}

		decoder = xml.NewDecoder(bytes.NewReader([]byte(req.Body())))
		decoder.CharsetReader = charset.NewReaderLabel
		err = decoder.Decode(temp)
		if err != nil {
			err = utils.DecodeGbk(temp, []byte(req.Body()))
			if err != nil {
				BPlugin.Error("decode catelog err", zap.Error(err))
			}
		}

		var body string
		switch temp.EventType {

		case "Push_Resourse": // 资源上报
			// 资源上报属于数据接口。前端系统加电启动并初次注册成功后，应向平台上报前端系统的设备资源信息
			BPlugin.Debug("资源上报")
			deviceList := make([]*notifyMessage, 0, len(temp.SubList.Item))
			for _, t := range temp.SubList.Item {

				d.UpdateChannelPosition(t.Code, t.Longitude, t.Latitude)

				deviceList = append(deviceList, &notifyMessage{
					DeviceID:     t.Code,
					ParentID:     temp.Code.Text,
					Name:         t.Name,
					Manufacturer: "",
					Model:        "",
					Owner:        "",
					CivilCode:    "",
					Address:      "",
					Port:         0,
					Parental:     0,
					SafetyWay:    0,
					RegisterWay:  0,
					Secrecy:      0,
					Status:       t.Status,
					Event:        t.Status,
					Latitude:     t.Latitude,
					Longitude:    t.Longitude,
				})
			}
			BPlugin.Sugar().Debugf("%#v", deviceList)
			d.UpdateChannelStatus(deviceList)
		case "Snapshot_Notify": // 图像数据上报通知

			sn := struct {
				XMLName   xml.Name `xml:"SIP_XML"`
				Text      string   `xml:",chardata"`
				EventType string   `xml:"EventType,attr"`
				Item      []struct {
					Text     string `xml:",chardata"`
					Code     string `xml:"Code,attr"`
					Type     string `xml:"Type,attr"`
					Time     string `xml:"Time,attr"`
					FileUrl  string `xml:"FileUrl,attr"`
					FileSize string `xml:"FileSize,attr"`
					Verfiy   string `xml:"Verfiy,attr"`
				} `xml:"Item"`
			}{}

			decoder = xml.NewDecoder(bytes.NewReader([]byte(req.Body())))
			decoder.CharsetReader = charset.NewReaderLabel
			err = decoder.Decode(sn)
			if err != nil {
				err = utils.DecodeGbk(sn, []byte(req.Body()))
				if err != nil {
					BPlugin.Error("decode catelog err", zap.Error(err))
				}
			}

		case "Catalog":
			//目录状态
			//d.UpdateChannelStatus(temp.DeviceList)
		case "MobilePosition":
			//更新channel的坐标
			//d.UpdateChannelPosition(temp.DeviceID, temp.Time, temp.Longitude, temp.Latitude)
		// case "Alarm":
		// 	//报警事件通知 TODO
		default:
			d.Warn("Not supported CmdType", zap.String("EventType", temp.EventType), zap.String("body", req.Body()))
			response := sip.NewResponseFromRequest("", req, http.StatusBadRequest, "", "")
			tx.Respond(response)
			return
		}

		tx.Respond(sip.NewResponseFromRequest("", req, http.StatusOK, "OK", body))
	}
}

type notifyMessage struct {
	DeviceID     string
	ParentID     string
	Name         string
	Manufacturer string
	Model        string
	Owner        string
	CivilCode    string
	Address      string
	Port         int
	Parental     int
	SafetyWay    int
	RegisterWay  int
	Secrecy      int
	Status       string
	//状态改变事件 ON:上线,OFF:离线,VLOST:视频丢失,DEFECT:故障,ADD:增加,DEL:删除,UPDATE:更新(必选)
	Event string

	Longitude string // 经度值
	Latitude  string // 纬度值
}
