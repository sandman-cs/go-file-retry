package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go workLoop()

	go func() {
		sig := <-sigs
		sendMessage("Exiting Program, recieved: ")
		sendMessage(fmt.Sprintln(sig))

		//Add Code here to send message to kill all input threads.
		closeReceived = true
		sendMessage("Closing input threads...")
		time.Sleep(time.Second * 2)
		os.Exit(0)
	}()

	for {
		time.Sleep(time.Second * 1)
		reader := bufio.NewReader(os.Stdin)
		text, _ := reader.ReadString('\n')
		fmt.Println(text)
	}

}

func workLoop() {
	defer func() {
		if r := recover(); r != nil {
			sendMessage(fmt.Sprintf("Recovered from error: %s ", r))
		}
	}()
	for {
		//This is debug code...
		fmt.Println("Checking for work at: ", conf.SrcDir, " ...")

		files, err := ioutil.ReadDir(conf.SrcDir)
		checkError(err, "Error getting list of files in source directory...")
		time.Sleep(500 * time.Millisecond)
		for _, file := range files {
			sFileName := conf.SrcDir + file.Name()

			//This is debug code...
			fmt.Println("Checking: ", sFileName)

			if info, err := os.Stat(sFileName); err == nil && !info.IsDir() {
				if info.Size() < 10 {
					go sendMessage(fmt.Sprintf("Zero byte file: %s", sFileName))
					fmt.Println("Zero byte file: ", sFileName)
					err = os.Remove(sFileName)
					//checkError(err)
				} else {
					if !retryHashChecker(sFileName) {
						err = moveFile(sFileName, conf.DstDir+file.Name())
					} else {
						err = moveFile(sFileName, conf.DeadLtrDir+file.Name())
					}
					//checkError(err)
				}
				checkError(err, "Error processing file in source directory...")
			}
		}

		time.Sleep(conf.RetryDelay * time.Minute)
		if closeReceived {
			sendMessage(fmt.Sprint("Closing input thread..."))
			return
		}
	}
}
