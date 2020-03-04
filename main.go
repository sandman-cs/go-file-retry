package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net"
	"net/mail"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/scorredoira/email"
)

type emailPayload struct {
	payload []string
	attach  bool
}

var (
	conf                                             configuration
	alive, gemail, bemail, dmemail, qemail, dupEmail int
	emailPub                                         []chan emailPayload
	closeReceived                                    bool
)

//checkError function
func checkError(err error) {
	if err != nil {
		log.WithFields(log.Fields{
			"app":    conf.AppName,
			"ver":    conf.AppVer,
			"Inst":   conf.InstanceID,
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
		"Inst":   conf.InstanceID,
		"server": conf.ServerName,
	}).Info(msg)
}

//sendMessage to udp listener
func sendDebugMessage(msg string) {

	log.WithFields(log.Fields{
		"app":    conf.AppName,
		"ver":    conf.AppVer,
		"Inst":   conf.InstanceID,
		"server": conf.ServerName,
	}).Debug(msg)
}

//sendMessage to udp listener
func sendWarnMessage(msg string) {

	log.WithFields(log.Fields{
		"app":    conf.AppName,
		"ver":    conf.AppVer,
		"Inst":   conf.InstanceID,
		"server": conf.ServerName,
	}).Warn(msg)
}

//sendMetrics to stdout
func sendMetrics(good int, bad int, del int) {

	log.WithFields(log.Fields{
		"app":    conf.AppName,
		"ver":    conf.AppVer,
		"Inst":   conf.InstanceID,
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
		UnRegister(conf.DbServer, conf.DbUser, conf.DbPwd, conf.HealthID, conf.InstanceID)

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
					wg.Done()
				}
			}
		}()
		sendMessage("Waiting for send treads to finish...")
		wg.Wait()
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
						} else if stringContains(sLine[0], conf.DelMatch) {
							sendMessage(fmt.Sprintf("Match found in \"%s\", deleting %s", sLine[0], sFileName))
							//err := os.Remove(sFileName)
							//checkError(err)
							dmemail++
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
							emailPub[chanID] <- tEmail
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

func sendEmailChannel(input chan emailPayload, srv emailSrv) {
	defer func() {
		if r := recover(); r != nil {
			sendMessage(fmt.Sprintf("Recovered from error: %s ", r))
		}
	}()

	var szTemp emailPayload

	sendMessage(fmt.Sprint("Connecting to Server: ", srv))
	wg.Add(1)
	for {
		szTemp = <-input
		if szTemp.attach {
			sendEmailWithAttach(szTemp.payload, srv)
		} else {
			sendEmailNoAttach(szTemp.payload, srv)
		}
		gemail++
	}
}

func sendEmailWithAttach(payload []string, srv emailSrv) {
	defer func() {
		if r := recover(); r != nil {
			sendMessage(fmt.Sprintf("Recovered from error: %s ", r))
		}
	}()

	// compose the message
	body := ""
	for i := 4; i < len(payload); i++ {
		body += payload[i] + "\n"
	}
	m := email.NewMessage(payload[2], body)
	m.From = mail.Address{Name: payload[1], Address: payload[1]}
	m.To = strings.Split(payload[0], ";")

	// add attachments
	if strings.Contains(payload[3], "<attachlist>") {
		attchList := getXMLAttach(payload[3])
		for _, attach := range attchList {
			fi, e := os.Stat(attach)
			checkError(e)
			if e == nil && fi.Size() <= conf.AttachSize {
				if err := m.Attach(attach); err != nil {
					checkError(err)
				}
			}

			if fi.Size() > conf.AttachSize {
				err := os.Remove(attach)
				checkError(err)
				sendMessage(fmt.Sprint("Error: attachment too large: ", fi.Name()))
			}
		}
	} else {
		fi, e := os.Stat(payload[3])

		if e == nil && fi.Size() <= conf.AttachSize {
			if err := m.Attach(payload[3]); err != nil {
				log.Println(err)
			}
		}

		if fi.Size() > conf.AttachSize {
			err := os.Remove(payload[3])
			checkError(err)
			sendMessage(fmt.Sprint("Error: attachment too large: ", fi.Name()))

		}

	}

	if err := email.Send(srv.EmailSrv+":"+srv.EmailPrt, nil, m); err != nil {
		log.Println(err)
	}
}

func sendEmailNoAttach(payload []string, srv emailSrv) {
	// compose the message
	body := ""
	for i := 3; i < len(payload); i++ {
		body += payload[i] + "\n"
	}
	m := email.NewMessage(payload[2], body)
	m.From = mail.Address{Name: payload[1], Address: payload[1]}
	m.To = strings.Split(payload[0], ";")

	if err := email.Send(srv.EmailSrv+":"+srv.EmailPrt, nil, m); err != nil {
		checkError(err)
	}
}

func stringToLines(s string) []string {
	var lines []string

	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		sendMessage(fmt.Sprintln("Error reading standard input:", err))
	}
	return lines
}

// stringContains checkes the srcString for any matches in the
// list, which is an array of strings.
func stringContains(a string, list []string) bool {
	for _, b := range list {
		if strings.Contains(a, b) {
			return true
		}
	}
	return false
}

func healthCheck() {
	sendMessage("Starting health check thread...")
	register := true
	for {
		if register {
			HealthRegister(conf.DbServer, conf.DbUser, conf.DbPwd, conf.HealthID, conf.InstanceID, conf.ServerName, conf.AppName+" "+conf.AppVer)
			register = false
		} else if alive > 0 {
			HealthCheck(conf.DbServer, conf.DbUser, conf.DbPwd, conf.HealthID, conf.InstanceID)
		} else {
			sendMessage("Failed Health Check...")
		}
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
