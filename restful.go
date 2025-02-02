package b

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"m7s.live/engine/v4/util"
)

var (
	playScaleValues = map[float32]bool{0.25: true, 0.5: true, 1: true, 2: true, 4: true}
)

func (c *BConfig) API_list(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	if query.Get("interval") == "" {
		query.Set("interval", "5s")
	}
	util.ReturnFetchValue(func() (list []*Device) {
		list = make([]*Device, 0)
		Devices.Range(func(key, value interface{}) bool {
			list = append(list, value.(*Device))
			return true
		})
		return
	}, w, r)
}

// API_records 录像检索
func (c *BConfig) API_records(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	id := query.Get("id")
	channel := query.Get("channel")
	videoType := query.Get("type")
	startTime := query.Get("startTime")
	endTime := query.Get("endTime")
	trange := strings.Split(query.Get("range"), "-")
	if len(trange) == 2 {
		startTime = trange[0]
		endTime = trange[1]
	}
	if c := FindChannel(id, channel); c != nil {
		res, err := c.QueryRecord(videoType, startTime, endTime)
		if err == nil {
			util.ReturnValue(res, w, r)
		} else {
			util.ReturnError(util.APIErrorInternal, err.Error(), w, r)
		}
	} else {
		util.ReturnError(util.APIErrorNotFound, fmt.Sprintf("device %q channel %q not found", id, channel), w, r)
	}
}

//func (c *BConfig) API_control(w http.ResponseWriter, r *http.Request) {
//	id := r.URL.Query().Get("id")
//	channel := r.URL.Query().Get("channel")
//	ptzcmd := r.URL.Query().Get("ptzcmd")
//	l := r.URL.Query().Get("l") // 横向运动速度
//	p := r.URL.Query().Get("p") // 纵向运动速度
//	if c := FindChannel(id, channel); c != nil {
//		util.ReturnError(0, fmt.Sprintf("control code:%d", c.Control(ptzcmd, l, p)), w, r)
//	} else {
//		util.ReturnError(util.APIErrorNotFound, fmt.Sprintf("device %q channel %q not found", id, channel), w, r)
//	}
//}

func (c *BConfig) API_ptz(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	id := q.Get("id")
	channel := q.Get("channel")
	cmdStr := q.Get("cmd")      // 命令名称，见 ptz.go name2code 定义
	para1 := q.Get("cmd_para1") // 方向控制指令（上、下、左、右、左上、左下、右上、 右下等）代表横向运动速度，取值范围为[1，9]，1 为最低速度，9 为最高速度,预置位相关指令代表预置位编号，取值范围为[1，256]
	para2 := q.Get("cmd_para2") // 方向控制指令（上、下、左、右、左上、左下、右上、右下 等）代表纵向运动速度，取值范围为[1，9]，1 为最低速度，9 为最高速度；
	//zs := q.Get("cmd_para3") // 保留使用

	p1, err := strconv.ParseUint(para1, 10, 64)
	if err != nil {
		util.ReturnError(util.APIErrorQueryParse, "parameter1 is invalid", w, r)
		return
	}
	p2, err := strconv.ParseUint(para2, 10, 64)
	if err != nil {
		util.ReturnError(util.APIErrorQueryParse, "parameter2 is invalid", w, r)
		return
	}
	cmd, err := strconv.ParseUint(cmdStr, 16, 64)
	if err != nil {
		util.ReturnError(util.APIErrorQueryParse, "cmd parameter is invalid", w, r)
		return
	}

	//ptzcmd, err := toPtzStrByCmdName(cmd, uint8(hsN), uint8(vsN), uint8(zsN))
	//if err != nil {
	//	util.ReturnError(util.APIErrorQueryParse, err.Error(), w, r)
	//	return
	//}
	if c := FindChannel(id, channel); c != nil {
		code := c.Control(cmd, p1, p2)
		util.ReturnError(code, "device received", w, r)
	} else {
		util.ReturnError(util.APIErrorNotFound, fmt.Sprintf("device %q channel %q not found", id, channel), w, r)
	}
}

func (c *BConfig) API_invite(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	id := query.Get("id")
	channel := query.Get("channel")
	streamPath := query.Get("streamPath")
	port, _ := strconv.Atoi(query.Get("mediaPort"))
	opt := InviteOptions{
		dump:       query.Get("dump"),
		MediaPort:  uint16(port),
		StreamPath: streamPath,
	}
	startTime := query.Get("startTime")
	endTime := query.Get("endTime")
	trange := strings.Split(query.Get("range"), "-")
	if len(trange) == 2 {
		startTime = trange[0]
		endTime = trange[1]
	}
	opt.Validate(startTime, endTime)
	if c := FindChannel(id, channel); c == nil {
		BPlugin.Error(fmt.Sprintf("device %q channel %q not found", id, channel))
		util.ReturnError(util.APIErrorNotFound, fmt.Sprintf("device %q channel %q not found", id, channel), w, r)
	} else if opt.IsLive() && c.status.Load() > 0 {
		BPlugin.Warn("live stream already exists")
		util.ReturnError(util.APIErrorQueryParse, "live stream already exists", w, r)
	} else if code, err := c.Invite(&opt); err == nil {
		if code == 200 {
			util.ReturnOK(w, r)
		} else {
			BPlugin.Error(fmt.Sprintf("invite return code %d", code))
			util.ReturnError(util.APIErrorInternal, fmt.Sprintf("invite return code %d", code), w, r)
		}
	} else {
		BPlugin.Error(err.Error())
		util.ReturnError(util.APIErrorInternal, err.Error(), w, r)
	}
}

func (c *BConfig) API_bye(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	channel := r.URL.Query().Get("channel")
	streamPath := r.URL.Query().Get("streamPath")
	if c := FindChannel(id, channel); c != nil {
		util.ReturnError(0, fmt.Sprintf("bye code:%d", c.Bye(streamPath)), w, r)
	} else {
		util.ReturnError(util.APIErrorNotFound, fmt.Sprintf("device %q channel %q not found", id, channel), w, r)
	}
}

func (c *BConfig) API_play_pause(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	channel := r.URL.Query().Get("channel")
	streamPath := r.URL.Query().Get("streamPath")
	if c := FindChannel(id, channel); c != nil {
		util.ReturnError(0, fmt.Sprintf("pause code:%d", c.Pause(streamPath)), w, r)
	} else {
		util.ReturnError(util.APIErrorNotFound, fmt.Sprintf("device %q channel %q not found", id, channel), w, r)
	}
}

func (c *BConfig) API_play_resume(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	channel := r.URL.Query().Get("channel")
	streamPath := r.URL.Query().Get("streamPath")
	if c := FindChannel(id, channel); c != nil {
		util.ReturnError(0, fmt.Sprintf("resume code:%d", c.Resume(streamPath)), w, r)
	} else {
		util.ReturnError(util.APIErrorNotFound, fmt.Sprintf("device %q channel %q not found", id, channel), w, r)
	}
}

func (c *BConfig) API_play_seek(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	channel := r.URL.Query().Get("channel")
	streamPath := r.URL.Query().Get("streamPath")
	secStr := r.URL.Query().Get("second")
	sec, err := strconv.ParseUint(secStr, 10, 32)
	if err != nil {
		util.ReturnError(util.APIErrorQueryParse, "second parameter is invalid: "+err.Error(), w, r)
		return
	}
	if c := FindChannel(id, channel); c != nil {
		util.ReturnError(0, fmt.Sprintf("play code:%d", c.PlayAt(streamPath, uint(sec))), w, r)
	} else {
		util.ReturnError(util.APIErrorNotFound, fmt.Sprintf("device %q channel %q not found", id, channel), w, r)
	}
}

func (c *BConfig) API_play_forward(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	channel := r.URL.Query().Get("channel")
	streamPath := r.URL.Query().Get("streamPath")
	speedStr := r.URL.Query().Get("speed")
	speed, err := strconv.ParseFloat(speedStr, 32)
	secondErrMsg := "speed parameter is invalid, should be one of 0.25,0.5,1,2,4"
	if err != nil || !playScaleValues[float32(speed)] {
		util.ReturnError(util.APIErrorQueryParse, secondErrMsg, w, r)
		return
	}
	if c := FindChannel(id, channel); c != nil {
		util.ReturnError(0, fmt.Sprintf("playforward code:%d", c.PlayForward(streamPath, float32(speed))), w, r)
	} else {
		util.ReturnError(util.APIErrorNotFound, fmt.Sprintf("device %q channel %q not found", id, channel), w, r)
	}
}

func (c *BConfig) API_position(w http.ResponseWriter, r *http.Request) {
	//CORS(w, r)
	query := r.URL.Query()
	//设备id
	id := query.Get("id")
	//订阅周期(单位：秒)
	expires := query.Get("expires")
	//订阅间隔（单位：秒）
	interval := query.Get("interval")

	expiresInt, err := time.ParseDuration(expires)
	if expires == "" || err != nil {
		expiresInt = c.Position.Expires
	}
	intervalInt, err := time.ParseDuration(interval)
	if interval == "" || err != nil {
		intervalInt = c.Position.Interval
	}

	if v, ok := Devices.Load(id); ok {
		d := v.(*Device)
		util.ReturnError(0, fmt.Sprintf("mobileposition code:%d", d.MobilePositionSubscribe(id, expiresInt, intervalInt)), w, r)
	} else {
		util.ReturnError(util.APIErrorNotFound, fmt.Sprintf("device %q  not found", id), w, r)
	}
}

type DevicePosition struct {
	ID        string
	GpsTime   time.Time //gps时间
	Longitude string    //经度
	Latitude  string    //纬度
}

func (c *BConfig) API_get_position(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	//设备id
	id := query.Get("id")
	if query.Get("interval") == "" {
		query.Set("interval", fmt.Sprintf("%fs", c.Position.Interval.Seconds()))
	}
	util.ReturnFetchValue(func() (list []*DevicePosition) {
		if id == "" {
			Devices.Range(func(key, value interface{}) bool {
				d := value.(*Device)
				if time.Since(d.GpsTime) <= c.Position.Interval {
					list = append(list, &DevicePosition{ID: d.ID, GpsTime: d.GpsTime, Longitude: d.Longitude, Latitude: d.Latitude})
				}
				return true
			})
		} else if v, ok := Devices.Load(id); ok {
			d := v.(*Device)
			list = append(list, &DevicePosition{ID: d.ID, GpsTime: d.GpsTime, Longitude: d.Longitude, Latitude: d.Latitude})
		}
		return
	}, w, r)
}

// API_capture 抓图
func (c *BConfig) API_capture(w http.ResponseWriter, r *http.Request) {

	id := r.URL.Query().Get("id")
	snapTypeStr := r.URL.Query().Get("snapType")
	channel := r.URL.Query().Get("channel")
	intervalStr := r.URL.Query().Get("interval")
	timeRange := r.URL.Query().Get("timeRange")

	snapType, err := strconv.Atoi(snapTypeStr)
	if err != nil {
		BPlugin.Sugar().Errorf("解析抓图类型错误,%v", err)
		util.ReturnError(1000, "解析抓图类型错误", w, r)
		return
	}

	interval, err := strconv.Atoi(intervalStr)
	if err != nil {
		BPlugin.Sugar().Errorf("解析抓图间隔错误,%v", err)
		util.ReturnError(1000, "解析抓图间隔错误", w, r)
		return
	}

	if c := FindChannel(id, channel); c != nil {
		util.ReturnError(c.Capture("http://192.168.1.166:8080/gb28181/api/imgUpload", timeRange, snapType, interval), "device received", w, r)
	} else {
		util.ReturnError(404, "设备通道未找到", w, r)
	}
}

// API_ImgUpload 抓图上传http图片文件
func (c *BConfig) API_ImgUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Retrieve the uploaded file
	file, header, err := r.FormFile("file")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Error retrieving the file: %v", err)
		return
	}
	defer file.Close()

	// Create a new file in the server to store the uploaded image
	newFile, err := os.Create(header.Filename)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error creating the file: %v", err)
		return
	}
	defer newFile.Close()

	// Copy the uploaded file to the new file on the server
	_, err = io.Copy(newFile, file)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error copying the file: %v", err)
		return
	}

	fmt.Fprintf(w, "File uploaded successfully!")
}

// API_device_resourceInfo 资源信息获取(测试未成功)
func (c *BConfig) API_device_resourceInfo(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		util.ReturnError(404, "设备ID为空", w, r)
		return
	}
	if v, ok := Devices.Load(id); ok {
		d := v.(*Device)
		util.ReturnError(d.ResourceInfo(), "", w, r)
		return
	}

	util.ReturnError(404, "设备未找到", w, r)
}
