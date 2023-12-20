package gb28181

import (
	"fmt"
	"github.com/ghettovoice/gosip/sip"
	"net/http"
)

func ImageCaptureConfig(d *Device, channelID string) error {

	request := d.CreateRequest(sip.MESSAGE)
	contentType := sip.ContentType("Application/MANSCDP+xml")
	request.AppendHeader(&contentType)

	body := BuildImageCaptureConfig(d.sn, 1, 1, channelID, "", "123")
	request.SetBody(body, true)
	GB28181Plugin.Sugar().Debugf("SIP->image capture config:%s", request)
	resp, err := d.SipRequestForResponse(request)
	if err != nil {
		return fmt.Errorf("query error: %s", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("query error, status=%d", resp.StatusCode())
	}

	return nil
}
