package model

import "gorm.io/gorm"

func CaptureList(db *gorm.DB, id, channel string, page, size int) (list []Gb28181Capture, total int64, err error) {
	list = make([]Gb28181Capture, 0)
	query := db.Model(&Gb28181Capture{})
	if id != "" {
		query.Where("device_id=?", id)
	}
	if channel != "" {
		query.Where("channel_id=?", channel)
	}
	err = query.Count(&total).Limit(size).Offset((page - 1) * size).Find(&list).Error
	return list, total, err
}
