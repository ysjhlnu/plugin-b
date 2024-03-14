package model

import "time"

type GB28181DeviceChannel struct {
	ID           uint64                       `gorm:"autoIncrement:true;primaryKey;unique;column:id;type:bigint unsigned;not null" json:"id"`
	Version      string                       `gorm:"uniqueIndex:uk_device_device;column:version;type:varchar(255);default:null;comment:'国标版本'" json:"version"` // 国标版本
	CustomName   string                       `gorm:"column:custom_name;type:varchar(255);default:null" json:"custom_name"`
	DeviceID     string                       `gorm:"uniqueIndex:uk_device_device;column:device_id;type:varchar(50);not null;comment:'国标ID'" json:"device_id"` // 国标ID
	OnLine       bool                         `gorm:"column:on_line;type:tinyint(1);default:null;default:0" json:"on_line"`
	RegisterTime time.Time                    `gorm:"column:register_time;type:datetime;default:null;comment:'注册时间'" json:"register_time"` // 注册时间
	Status       string                       `gorm:"column:status;type:varchar(255);default:null" json:"status"`
	CreatedAt    time.Time                    `gorm:"column:created_at;type:datetime;default:null" json:"created_at"`
	UpdatedAt    time.Time                    `gorm:"column:updated_at;type:datetime;default:null" json:"updated_at"`
	ChannelList  []GB28181DeviceChannelDetail `gorm:"-" json:"channel_list"`
}

type GB28181DeviceChannelDetail struct {
	ID         uint64    `gorm:"autoIncrement:true;primaryKey;unique;column:id;type:bigint unsigned;not null" json:"id"`
	Version    string    `gorm:"column:version;type:varchar(255);default:null;comment:'国标版本'" json:"version"` // 国标版本
	DeviceID   string    `gorm:"uniqueIndex:uk_wvp_device_channel_unique_device_channel;column:device_id;type:varchar(50);not null" json:"device_id"`
	ChannelID  string    `gorm:"uniqueIndex:uk_wvp_device_channel_unique_device_channel;column:channel_id;type:varchar(50);not null" json:"channel_id"`
	CustomName string    `gorm:"column:custom_name;type:varchar(255);default:null" json:"custom_name"`
	Status     string    `gorm:"column:status;type:varchar(255);default:null;default:0;comment:'设备状态（必选）'" json:"status"`               // 设备状态（必选）
	ParentID   string    `gorm:"column:parent_id;type:varchar(50);default:null;comment:'父设备/区域/系统 ID（ 可必选，有父设备需要填写）'" json:"parent_id"` // 父设备/区域/系统 ID（ 可必选，有父设备需要填写）
	CreateTime time.Time `gorm:"column:create_time;type:datetime;not null" json:"create_time"`
	UpdateTime time.Time `gorm:"column:update_time;type:datetime;not null" json:"update_time"`
}
