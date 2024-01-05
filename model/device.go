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

func CreateDeviceChannel(db *gorm.DB, create *Gb28181DeviceChannel) (err error) {
	err = db.Create(&create).Error
	return err
}

func UpdateDeviceChannelStatus(db *gorm.DB, version, deviceID, channelID, deviceStatus string) (err error) {
	err = db.Model(&Gb28181DeviceChannel{}).
		Where("version=? AND device_id=? AND channel_id=?", version, deviceID, channelID).
		Update("status", deviceStatus).Error
	return err
}

func DeleteDeviceChannel(db *gorm.DB, version, deviceID, channelID string) (err error) {
	err = db.Model(&Gb28181DeviceChannel{}).
		Where("version=? AND device_id=? AND channel_id=?", version, deviceID, channelID).
		Delete(nil).Error
	return err
}

// UpdateDeviceChannelPosition 更新设备通道GPS坐标
func UpdateDeviceChannelPosition(db *gorm.DB, version, deviceID, channelID, lng, lat string) (err error) {
	update := map[string]interface{}{"gps_time": time.Now(), "longitude": lng, "latitude": lat}
	err = db.Model(&Gb28181DeviceChannel{}).
		Where("version=? AND device_id=? AND channel_id=?", version, deviceID, channelID).
		Updates(update).Error
	return err
}

func UpdateDeviceChannel(db *gorm.DB, version, deviceID, channelID string, update Gb28181DeviceChannel) (err error) {
	err = db.Model(&Gb28181DeviceChannel{}).
		Where("version=? AND device_id=? AND channel_id=?", version, deviceID, channelID).Updates(update).Error
	return err
}

func DeviceChannelList(db *gorm.DB, version, deviceID string) (channelList []Gb28181DeviceChannel, err error) {
	channelList = make([]Gb28181DeviceChannel, 0)
	err = db.Model(&Gb28181DeviceChannel{}).
		Where("version=? AND device_id=?", version, deviceID).Find(&channelList).Error
	return channelList, err
}
