package utils

import (
	"io/ioutil"
	"log"
	"net/http"
)

func GetPublicIP() string {
	url := "https://api.ipify.org?format=text"
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalln("Failed to initialize agent (1x01)")
	}
	defer resp.Body.Close()
	ip, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln("Failed to initialize agent (1x02)")
	}
	return string(ip)
}
