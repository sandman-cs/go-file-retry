package main

import (
	"fmt"
	"net"

	log "github.com/sirupsen/logrus"
)

//checkError function
func checkError(err error, txt string) {
	if err != nil {
		log.WithFields(log.Fields{
			"app":    conf.AppName,
			"ver":    conf.AppVer,
			"server": conf.ServerName,
			"msg":    txt,
		}).Error(fmt.Sprint(err))
	}
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
		panic(fmt.Sprintf("%s: %s", msg, err))
	}
}

func sendUDPMessage(msg string) {
	ServerAddr, err := net.ResolveUDPAddr("udp", conf.SysLogSrv+":"+conf.SysLogPort)
	checkError(err, "Error resolving syslog server address...")
	if err == nil {

		LocalAddr, err := net.ResolveUDPAddr("udp", ":0")
		checkError(err, "Error creating socket to send UDP message...")

		Conn, err := net.DialUDP("udp", LocalAddr, ServerAddr)
		checkError(err, "Error connecting too syslog destination...")

		defer Conn.Close()
		buf := []byte(msg)
		if _, err := Conn.Write(buf); err != nil {
			checkError(err, "Error sending data too syslog destination...")
		}
	}
}

//sendMessage to udp listener
func sendMessage(msg string) {

	log.WithFields(log.Fields{
		"app":    conf.AppName,
		"ver":    conf.AppVer,
		"server": conf.ServerName,
	}).Info(msg)
}

//sendMessage to udp listener
func sendDebugMessage(msg string) {

	log.WithFields(log.Fields{
		"app":    conf.AppName,
		"ver":    conf.AppVer,
		"server": conf.ServerName,
	}).Debug(msg)
}

//sendMessage to udp listener
func sendWarnMessage(msg string) {

	log.WithFields(log.Fields{
		"app":    conf.AppName,
		"ver":    conf.AppVer,
		"server": conf.ServerName,
	}).Warn(msg)
}
