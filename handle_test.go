package b

import (
	"crypto/md5"
	"fmt"
	"testing"
)

func getDigest(method, raw string) string {
	switch method {
	case "MD5":
		return fmt.Sprintf("%x", md5.Sum([]byte(raw)))
	default: //如果没有算法，默认使用MD5
		return fmt.Sprintf("%x", md5.Sum([]byte(raw)))
	}
}

func verifyStr(username, passwd, realm, nonce, uri string) string {

	//1、将 username,realm,password 依次组合获取 1 个字符串，并用算法加密的到密文 r1
	s1 := fmt.Sprintf("%s:%s:%s", username, realm, passwd)
	r1 := getDigest("MD5", s1)
	//2、将 method，即REGISTER ,uri 依次组合获取 1 个字符串，并对这个字符串使用算法 加密得到密文 r2
	s2 := fmt.Sprintf("REGISTER:%s", uri)
	r2 := getDigest("MD5", s2)

	if r1 == "" || r2 == "" {
		BPlugin.Error("Authorization algorithm wrong")
		return ""
	}
	//3、将密文 1，nonce 和密文 2 依次组合获取 1 个字符串，并对这个字符串使用算法加密，获得密文 r3，即Response
	s3 := fmt.Sprintf("%s:%s:%s", r1, nonce, r2)
	r3 := getDigest("MD5", s3)
	//4、计算服务端和客户端上报的是否相等
	return r3
}

func TestVerify(t *testing.T) {
	//res := verifyStr("340200000013205162", "123456", "192.168.1.166", "12669864027800072351804054206864", "sip:340200000020000001@192.168.1.166:15060")
	res := verifyStr("340200000020000164", "123456", "340200000", "72886024655987732175261001231075", "sip:340200000020000164@183.67.31.178:15060")
	t.Log(res)
}
