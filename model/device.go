package model

import (
	"gorm.io/gorm"
	"time"
)

func CreateDevice(db *gorm.DB, version, id, deviceIp, sipIP, mediaIp, DeviceRegisterStatus string) (err error) {
	err = db.Create(&Gb28181Device{
		Version:                          version,
		DeviceID:                         id,
		Name:                             "新增设备",
		Manufacturer:                     "",
		Model:                            "",
		Firmware:                         "",
		Transport:                        "",
		StreamMode:                       "",
		OnLine:                           true,
		RegisterTime:                     time.Now(),
		IP:                               deviceIp,
		CreatedAt:                        time.Now(),
		UpdatedAt:                        time.Now(),
		Port:                             0,
		Expires:                          0,
		SubscribeCycleForCatalog:         0,
		SubscribeCycleForMobilePosition:  0,
		MobilePositionSubmissionInterval: 0,
		SubscribeCycleForAlarm:           0,
		HostAddress:                      "",
		Charset:                          "",
		SsrcCheck:                        false,
		GeoCoordSys:                      "",
		MediaServerID:                    "",
		CustomName:                       "",
		SdpIP:                            sipIP,
		LocalIP:                          mediaIp,
		Password:                         "",
		AsMessageChannel:                 false,
		KeepaliveIntervalTime:            0,
		SwitchPrimarySubStream:           false,
		BroadcastPushAfterAck:            false,
		Status:                           DeviceRegisterStatus,
	}).Error
	return err
}

func UpdateDeviceKeepalive(db *gorm.DB, version, deviceID string) (err error) {
	err = db.Model(&Gb28181Device{}).Where("version=? AND device_id=?", version, deviceID).Update("keepalive_time", time.Now()).Error
	return err
}

func UpdateDeviceInfo(db *gorm.DB, version, deviceID, deviceName, manufacturer, model string) (err error) {
	update := map[string]interface{}{"name": deviceName, "manufacturer": manufacturer, "model": model}
	err = db.Model(&Gb28181Device{}).Where("version=? AND device_id=?", version, deviceID).Omit("updated_at").Updates(update).Error
	return err
}

func DeviceList(db *gorm.DB, version string) (device []Gb28181Device, err error) {
	device = make([]Gb28181Device, 0)
	err = db.Model(&Gb28181Device{}).Where("version=?", version).Find(&device).Error
	return device, err
}

func UpdateDeviceStatus(db *gorm.DB, version, deviceID, status string, online bool) (err error) {
	update := map[string]interface{}{"status": status, "on_line": online}
	err = db.Model(&Gb28181Device{}).Where("version=? AND device_id=?", version, deviceID).Updates(update).Error
	return err
}
