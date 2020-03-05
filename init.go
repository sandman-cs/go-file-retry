package main

import (
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io/ioutil"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	cache "github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"
)

// Configuration File Opjects
type configuration struct {
	ServerName  string
	SrcDir      string
	DstDir      string
	DeadLtrDir  string
	AppName     string
	AppVer      string
	SysLogSrv   string
	SysLogPort  string
	ChannelSize int
	LogLevel    string
	RetryCount  int
	RetryDelay  time.Duration
}

var (
	conf                                             configuration
	alive, gemail, bemail, dmemail, qemail, dupEmail int

	closeReceived  bool
	retryHashCheck *cache.Cache
)

func init() {

	//Load Default Configuration Values

	conf.AppName = "Go - FileRetry"
	conf.AppVer = "1.0"
	conf.SysLogSrv = "splunk"
	conf.SysLogPort = "514"
	conf.ServerName, _ = os.Hostname()
	conf.ChannelSize = 1024
	conf.LogLevel = "info"
	conf.RetryCount = 1
	conf.RetryDelay = 15

	szTemp := getCurrentExecDirectory()

	if runtime.GOOS == "windows" {
		fmt.Println("Windows OS detected, setting default paths based on this...")
		conf.SrcDir = szTemp + "\\retry\\"
		conf.DstDir = szTemp + "\\work\\"
		conf.DeadLtrDir = szTemp + "\\deadleter\\"

	} else {
		fmt.Println("Unix/Linux OS detected, setting default paths based on this...")
		conf.SrcDir = szTemp + "/retry/"
		conf.DstDir = szTemp + "/work/"
		conf.DeadLtrDir = szTemp + "/deadleter/"
	}

	//Load Configuration Data
	dat, _ := ioutil.ReadFile("config.json")
	err := json.Unmarshal(dat, &conf)
	failOnError(err, "Failed to load config.json")

	createIfNotExist(conf.SrcDir)
	createIfNotExist(conf.DstDir)
	createIfNotExist(conf.DeadLtrDir)

	//fmt.Println("Running Config: ", conf)

	// Log as JSON instead of the default ASCII formatter.
	log.SetFormatter(&log.JSONFormatter{})
	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)
	// Set Log Level to debug, info or warn, system supports debug, info, warn, fatal, panic
	conf.LogLevel = strings.ToLower(conf.LogLevel)
	if conf.LogLevel == "debug" {
		log.SetLevel(log.DebugLevel)
	} else if conf.LogLevel == "warn" {
		log.SetLevel(log.WarnLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	retryHashCheck = cache.New(conf.RetryDelay*time.Minute, conf.RetryDelay+30*time.Minute)

	go healthCheck()

}

func retryHashChecker(matchString string) bool {

	// Create CRC
	crc32InUint32 := crc32.ChecksumIEEE([]byte(matchString))
	crc32InString := strconv.FormatUint(uint64(crc32InUint32), 16)

	// Check for a cache hit first
	count, found := retryHashCheck.Get(crc32InString)

	//Debug code...
	fmt.Println("Count for: ", matchString, " :", count)

	if count != nil {
		if count.(int) < conf.RetryCount {
			found = false
		} else {
			logRetryToSplunk(matchString, count.(int))
		}
		retryHashCheck.SetDefault(crc32InString, count.(int)+1)
	} else {
		retryHashCheck.SetDefault(crc32InString, 1)
	}
	return found
}

func logRetryToSplunk(msg string, count int) {

	re := regexp.MustCompile(`\r?\n`)
	msg = re.ReplaceAllString(msg, " ")
	payload := "{\"app\": \"" + conf.AppName + "\",\"retry_count\":" + strconv.Itoa(count) + ",\"payload\":\"" + msg + "\"}"
	sendUDPMessage(payload)
}
