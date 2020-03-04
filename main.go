package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

type emailPayload struct {
	payload []string
	attach  bool
}

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

func main() {

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		sendMessage("Exiting Program, recieved: ")
		sendMessage(fmt.Sprintln(sig))

		//Add Code here to send message to kill all input threads.
		closeReceived = true
		sendMessage("Closing input threads...")
		time.Sleep(time.Second * 5)
		//Add Code here to wait for email in buffers to be sent before finishing close.
		go func() {
			for {
				if gemail == 0 {
					time.Sleep(time.Second * 5)
					sendMessage("Decrementing Send Thread Count.")

				}
			}
		}()
		sendMessage("Waiting for send treads to finish...")
		os.Exit(0)
	}()

	for {
		time.Sleep(time.Second * 1)
		reader := bufio.NewReader(os.Stdin)
		text, _ := reader.ReadString('\n')
		fmt.Println(text)
	}

}

func workLoop(workDir string, chanID int) {
	defer func() {
		if r := recover(); r != nil {
			sendMessage(fmt.Sprintf("Recovered from error: %s ", r))
		}
	}()
	for {
		alive++
		files, err := ioutil.ReadDir(workDir)
		checkError(err)
		time.Sleep(500 * time.Millisecond)
		for _, file := range files {
			alive++
			sFileName := fmt.Sprintf("%s\\%s", workDir, file.Name())
			if info, err := os.Stat(sFileName); err == nil && !info.IsDir() {
				if info.Size() < 10 {
					go sendMessage(fmt.Sprintf("Zero byte file: %s", sFileName))
					fmt.Println("Zero byte file: ", sFileName)
					err = os.Remove(sFileName)
					checkError(err)
				} else {
					dat, err := ioutil.ReadFile(sFileName)
					checkError(err)
					if err == nil {
						sLine := stringToLines(string(dat))
						if !strings.Contains(sLine[0], "@") {
							sendMessage(fmt.Sprintf("No Valid email found in \"%s\", deleting %s", sLine[0], sFileName))
							//err := os.Remove(sFileName)
							//checkError(err)
							bemail++
						} else if emailHashChecker(string(dat)) {
							dupEmail++
							sendMessage(fmt.Sprintf("Duplicate found in \"%s\", deleting %s", sLine[0], sFileName))
							//err := os.Remove(sFileName)
							//checkError(err)
						} else {
							var tEmail emailPayload
							tEmail.payload = sLine
							sendDebugMessage(fmt.Sprintf("Sending %s to %s", sFileName, sLine[0]))
							//Decision Point for email type...
							if strings.Contains(sFileName, ".dtt") {
								tEmail.attach = true
							}

							//err := os.Remove(sFileName)
							//checkError(err)
							qemail++
						}
						err := os.Remove(sFileName)
						checkError(err)
					}
				}
			}
		}
		//time.Sleep(500 * time.Millisecond)
		if closeReceived {
			sendMessage(fmt.Sprint("Closing input thread ID: ", chanID))
			return
		}
	}
}

func healthCheck() {
	sendMessage("Starting health check thread...")

	for {
		payload := "{\"app\": \"emailer\",\"good_email\":" + strconv.Itoa(gemail) + ",\"bad_email\":" + strconv.Itoa(bemail) + ",\"del_email\":" + strconv.Itoa(dmemail) + ",\"dup_email\":" + strconv.Itoa(dupEmail) + "}"
		sendUDPMessage(payload)

		alive = 0
		gemail = 0
		bemail = 0
		dmemail = 0
		dupEmail = 0

		time.Sleep(60 * time.Second)
	}
}

func getXMLAttach(rawXML string) []string {

	var attach []string

	for {

		offset := strings.Index(rawXML, "<attachment>") + 12
		if offset > 12 {
			rawXML = rawXML[offset:]
			attach = append(attach, rawXML[:strings.Index(rawXML, "</attachment>")])
		} else {
			break
		}

	}
	return attach
}
