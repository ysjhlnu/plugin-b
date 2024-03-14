package gb28181

import (
	"errors"
	"fmt"
)

var (
	name2code = map[string]uint8{
		"stop":      0,
		"right":     1,
		"left":      2,
		"down":      4,
		"downright": 5,
		"downleft":  6,
		"up":        8,
		"upright":   9,
		"upleft":    10,
		"zoomin":    16,
		"zoomout":   32,
	}

	command2code = map[string]uint8{
		"STOP":         0,
		"RIGHT":        1,
		"LEFT":         2,
		"DOWN":         4,
		"RIGHT_DOWN":   5,
		"LEFT_DOWN":    6,
		"UP":           8,
		"RIGHT_UP":     9,
		"LEFT_UP":      10,
		"IRIS_ENLARGE": 16,
		"IRIS_REDUCE":  32,
		"ADD_PRESET":   129,
		"GOTO_PRESET":  130,
		"DEL_PRESET":   131,
		"LIST_PRESET":  0,
	}
)

func toPtzStrByCmdName(cmdName string, horizontalSpeed, verticalSpeed, zoomSpeed uint8) (string, error) {
	c, err := toPtzCode(cmdName)
	if err != nil {
		return "", err
	}
	return toPtzStr(c, horizontalSpeed, verticalSpeed, zoomSpeed), nil
}

func toPtzStrByCmdName2(cmdName string, horizontalSpeed, verticalSpeed, zoomSpeed uint8) (string, error) {
	c, ok := command2code[cmdName]
	if !ok {
		return "", errors.New("")
	}
	return toPtzStr(c, horizontalSpeed, verticalSpeed, zoomSpeed), nil
}

func toPtzStr(cmdCode, horizontalSpeed, verticalSpeed, zoomSpeed uint8) string {
	checkCode := uint16(0xA5+0x0F+0x01+cmdCode+horizontalSpeed+verticalSpeed+(zoomSpeed&0xF0)) % 0x100

	return fmt.Sprintf("A50F01%02X%02X%02X%01X0%02X",
		cmdCode,
		horizontalSpeed,
		verticalSpeed,
		zoomSpeed>>4, // 根据 GB28181 协议，zoom 只取 4 bit
		checkCode,
	)
}

func toPtzCode(cmd string) (uint8, error) {
	if code, ok := name2code[cmd]; ok {
		return code, nil
	} else {
		return 0, fmt.Errorf("invalid ptz cmd %q", cmd)
	}
}
