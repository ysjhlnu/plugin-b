package model

import (
	"gorm.io/gorm"
	"time"
)

func CreateDeviceChannel(db *gorm.DB, version, deviceID, channelID, name, manufacturer, model, owner, civilCode, address, parentID, status string,
	safetyWay, registerWay, secrecy, parental int) (err error) {
	err = db.Create(&Gb28181DeviceChannel{
		Version:     version,
		ChannelID:   channelID,
		Name:        name,
		Manufacture: manufacturer,
		Model:       model,
		Owner:       owner,
		CivilCode:   civilCode,
		Address:     address,
		ParentID:    parentID,
		SafetyWay:   safetyWay,
		RegisterWay: registerWay,
		Secrecy:     secrecy,
		Status:      status,
		DeviceID:    deviceID,
		Parental:    parental,
		CreateTime:  time.Now(),
		UpdateTime:  time.Now(),
	}).Error
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
