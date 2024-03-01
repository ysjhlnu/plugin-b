package gb28181

import (
	"fmt"
	"io"
	"m7s.live/plugin/gb28181/v4/model"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"m7s.live/engine/v4/util"
)

var (
	playScaleValues = map[float32]bool{0.25: true, 0.5: true, 1: true, 2: true, 4: true}
)

func (c *GB28181Config) API_list(w http.ResponseWriter, r *http.Request) {
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

// API_records 查询录像
func (c *GB28181Config) API_records(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	id := query.Get("id")
	channel := query.Get("channel")
	startTime := query.Get("startTime")
	endTime := query.Get("endTime")
	trange := strings.Split(query.Get("range"), "-")
	if len(trange) == 2 {
		startTime = trange[0]
		endTime = trange[1]
	}
	if c := FindChannel(id, channel); c != nil {
		res, err := c.QueryRecord(startTime, endTime)
		if err == nil {
			util.ReturnValue(res, w, r)
		} else {
			util.ReturnError(util.APIErrorInternal, err.Error(), w, r)
		}
	} else {
		util.ReturnError(util.APIErrorNotFound, fmt.Sprintf("device %q channel %q not found", id, channel), w, r)
	}
}

// API_control 设备控制
func (c *GB28181Config) API_control(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	channel := r.URL.Query().Get("channel")
	ptzcmd := r.URL.Query().Get("ptzcmd")
	if c := FindChannel(id, channel); c != nil {
		util.ReturnError(0, fmt.Sprintf("control code:%d", c.Control(ptzcmd)), w, r)
	} else {
		util.ReturnError(util.APIErrorNotFound, fmt.Sprintf("device %q channel %q not found", id, channel), w, r)
	}
}

// API_ptz 球机/云台控制
func (c *GB28181Config) API_ptz(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	id := q.Get("id")
	channel := q.Get("channel")
	cmd := q.Get("cmd")   // 命令名称，见 ptz.go name2code 定义
	hs := q.Get("hSpeed") // 水平速度
	vs := q.Get("vSpeed") // 垂直速度
	zs := q.Get("zSpeed") // 缩放速度

	hsN, err := strconv.ParseUint(hs, 10, 8)
	if err != nil {
		util.ReturnError(util.APIErrorQueryParse, "hSpeed parameter is invalid", w, r)
		return
	}
	vsN, err := strconv.ParseUint(vs, 10, 8)
	if err != nil {
		util.ReturnError(util.APIErrorQueryParse, "vSpeed parameter is invalid", w, r)
		return
	}
	zsN, err := strconv.ParseUint(zs, 10, 8)
	if err != nil {
		util.ReturnError(util.APIErrorQueryParse, "zSpeed parameter is invalid", w, r)
		return
	}

	ptzcmd, err := toPtzStrByCmdName(cmd, uint8(hsN), uint8(vsN), uint8(zsN))
	if err != nil {
		util.ReturnError(util.APIErrorQueryParse, err.Error(), w, r)
		return
	}
	if c := FindChannel(id, channel); c != nil {
		code := c.Control(ptzcmd)
		util.ReturnError(code, "device received", w, r)
	} else {
		util.ReturnError(util.APIErrorNotFound, fmt.Sprintf("device %q channel %q not found", id, channel), w, r)
	}
}

// API_invite 从设备拉取视频流
func (c *GB28181Config) API_invite(w http.ResponseWriter, r *http.Request) {
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
		util.ReturnError(util.APIErrorNotFound, fmt.Sprintf("device %q channel %q not found", id, channel), w, r)
	} else if opt.IsLive() && c.State.Load() > 0 {
		util.ReturnError(util.APIErrorQueryParse, "live stream already exists", w, r)
	} else if code, streamPath, err := c.Invite(&opt); err == nil {
		if code == 200 {

			util.ReturnValue(util.GenAddr(streamPath), w, r)
		} else {
			util.ReturnError(util.APIErrorInternal, fmt.Sprintf("invite return code %d", code), w, r)
		}
	} else {
		util.ReturnError(util.APIErrorInternal, err.Error(), w, r)
	}
}

// API_bye 停止拉流
func (c *GB28181Config) API_bye(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	channel := r.URL.Query().Get("channel")
	streamPath := r.URL.Query().Get("streamPath")
	if c := FindChannel(id, channel); c != nil {
		util.ReturnError(0, fmt.Sprintf("bye code:%d", c.Bye(streamPath)), w, r)
	} else {
		util.ReturnError(util.APIErrorNotFound, fmt.Sprintf("device %q channel %q not found", id, channel), w, r)
	}
}

// API_play_pause 暂停播放
func (c *GB28181Config) API_play_pause(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	channel := r.URL.Query().Get("channel")
	streamPath := r.URL.Query().Get("streamPath")
	if c := FindChannel(id, channel); c != nil {
		util.ReturnError(0, fmt.Sprintf("pause code:%d", c.Pause(streamPath)), w, r)
	} else {
		util.ReturnError(util.APIErrorNotFound, fmt.Sprintf("device %q channel %q not found", id, channel), w, r)
	}
}

// API_play_resume 恢复播放
func (c *GB28181Config) API_play_resume(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	channel := r.URL.Query().Get("channel")
	streamPath := r.URL.Query().Get("streamPath")
	if c := FindChannel(id, channel); c != nil {
		util.ReturnError(0, fmt.Sprintf("resume code:%d", c.Resume(streamPath)), w, r)
	} else {
		util.ReturnError(util.APIErrorNotFound, fmt.Sprintf("device %q channel %q not found", id, channel), w, r)
	}
}

// API_play_seek 跳转到播放时间
func (c *GB28181Config) API_play_seek(w http.ResponseWriter, r *http.Request) {
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

// API_play_forward 快进/快退播放
func (c *GB28181Config) API_play_forward(w http.ResponseWriter, r *http.Request) {
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

// API_position 移动位置订阅
func (c *GB28181Config) API_position(w http.ResponseWriter, r *http.Request) {
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

func (c *GB28181Config) API_get_position(w http.ResponseWriter, r *http.Request) {
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

func (c *GB28181Config) API_switch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	d := query.Get("deny")
	if d != "" {
		GB28181Plugin.Debug("get")
		if d == "true" {
			GB28181Plugin.Debug("true")
			Cache.Store("deny", true)
		} else if d == "false" {
			GB28181Plugin.Debug("false")
			Cache.Store("deny", false)
		}
	}
	util.ReturnOK(w, r)
}

// API_snapshot 图像抓拍
func (c *GB28181Config) API_snapshot(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	id := query.Get("id")
	channel := query.Get("channel")
	ch := FindChannel(id, channel)
	if ch == nil {
		util.ReturnError(util.APIErrorNotFound, fmt.Sprintf("device %s  not found", id), w, r)
		return
	}
	code := ch.ImageCaptureConfig()
	util.ReturnError(code, "device received", w, r)
}

// API_file_upload 文件上传
func (c *GB28181Config) API_file_upload(w http.ResponseWriter, req *http.Request) {
	contentType := req.Header.Get("content-type")
	contentLen := req.ContentLength

	//fmt.Printf("upload content-type:%s,content-length:%d", contentType, contentLen)
	if !strings.Contains(contentType, "multipart/form-data") {
		w.Write([]byte("content-type must be multipart/form-data"))
		return
	}
	if contentLen >= 4*1024*1024 { // 10 MB
		w.Write([]byte("file to large,limit 4MB"))
		return
	}

	err := req.ParseMultipartForm(4 * 1024 * 1024)
	if err != nil {
		//http.Error(w, err.Error(), http.StatusInternalServerError)
		w.Write([]byte("ParseMultipartForm error:" + err.Error()))
		return
	}

	if len(req.MultipartForm.File) == 0 {
		w.Write([]byte("not have any file"))
		return
	}

	for name, files := range req.MultipartForm.File {
		//fmt.Printf("req.MultipartForm.File,name=%s", name)

		if len(files) != 1 {
			w.Write([]byte("too many files"))
			return
		}
		if name == "" {
			w.Write([]byte("is not FileData"))
			return
		}

		for _, f := range files {
			handle, err := f.Open()
			if err != nil {
				w.Write([]byte(fmt.Sprintf("unknown error,fileName=%s,fileSize=%d,err:%s", f.Filename, f.Size, err.Error())))
				return
			}

			path := "./uploads/" + f.Filename
			dst, _ := os.Create(path)
			io.Copy(dst, handle)
			dst.Close()
			//fmt.Printf("successful uploaded,fileName=%s,fileSize=%.2f MB,savePath=%s \n", f.Filename, float64(contentLen)/1024/1024, path)

			w.Write([]byte("successful,url=" + url.QueryEscape(f.Filename)))
		}
	}
}

// API_Images 显示图片
func (c *GB28181Config) API_Images(w http.ResponseWriter, r *http.Request) {
	imgPath := r.URL.Query().Get("path")
	if !strings.HasPrefix(imgPath, "uploads") {
		util.ReturnError(util.APIErrorNotFound, "not found", w, r)
		return
	}
	buff, err := os.ReadFile(imgPath)
	if err != nil {
		util.ReturnError(util.APIErrorInternal, err.Error(), w, r)
		return
	}
	w.Write(buff)
}

// API_Capture_List 上传的图片列表
func (c *GB28181Config) API_Capture_List(w http.ResponseWriter, r *http.Request) {

	var (
		page int = 1
		size int = 10
		err  error
	)

	q := r.URL.Query()

	id := q.Get("id")
	channel := q.Get("channel")
	pageStr := q.Get("page")
	sizeStr := q.Get("size")

	if pageStr != "" {
		page, err = strconv.Atoi(pageStr)
		if err != nil {
			GB28181Plugin.Error(err.Error())
			page = 1
		}
		if page == 0 {
			page = 1
		}
	}

	if sizeStr != "" {
		size, err = strconv.Atoi(sizeStr)
		if err != nil {
			GB28181Plugin.Error(err.Error())
			size = 10
		}
		if size == 0 {
			size = 10
		}
	}
	list, total, err := model.CaptureList(GB28181Plugin.DB, id, channel, page, size)
	if err != nil {
		util.ReturnError(util.APIErrorInternal, err.Error(), w, r)
		return
	}
	util.ReturnValue(map[string]interface{}{"list": list, "total": total, "page": page, "size": size}, w, r)
}
