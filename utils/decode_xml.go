package utils

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"golang.org/x/net/html/charset"
)

func DecodeXML(data []byte, v any) error {
	decoder := xml.NewDecoder(bytes.NewReader([]byte(data)))
	decoder.CharsetReader = charset.NewReaderLabel
	if err := decoder.Decode(v); err != nil {
		fmt.Println(err)
		err = DecodeGbk(v, data)
		if err != nil {
			return err
		}
		return err
	}
	//if err := xml.Unmarshal(data, v); err != nil {
	//	return err
	//}
	return nil
}
