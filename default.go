package main

import (
	"fmt"
	"net"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
)

//checkError function
func checkError(err error) {
	if err != nil {
		log.WithFields(log.Fields{
			"app":    conf.AppName,
			"ver":    conf.AppVer,
			"server": conf.ServerName,
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
	checkError(err)
	if err == nil {

		LocalAddr, err := net.ResolveUDPAddr("udp", ":0")
		checkError(err)

		Conn, err := net.DialUDP("udp", LocalAddr, ServerAddr)
		checkError(err)

		defer Conn.Close()
		buf := []byte(msg)
		if _, err := Conn.Write(buf); err != nil {
			checkError(err)
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

//sendMetrics to stdout
func sendMetrics(good int, bad int, del int) {

	log.WithFields(log.Fields{
		"app":    conf.AppName,
		"ver":    conf.AppVer,
		"server": conf.ServerName,
		"good":   good,
		"bad":    bad,
		"del":    del,
		"dup":    dupEmail,
	}).Info("metrics")
}

func healthCheck() {
	sendMessage("Starting health check thread...")

	for {
		payload := "{\"app\": \"" + conf.AppName + "\",\"good_email\":" + strconv.Itoa(gemail) + ",\"bad_email\":" + strconv.Itoa(bemail) + ",\"del_email\":" + strconv.Itoa(dmemail) + ",\"dup_email\":" + strconv.Itoa(dupEmail) + "}"
		sendUDPMessage(payload)

		alive = 0
		gemail = 0
		bemail = 0
		dmemail = 0
		dupEmail = 0

		time.Sleep(60 * time.Second)
	}
}
