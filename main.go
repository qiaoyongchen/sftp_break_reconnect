package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

var SSH_HOST *string
var SSH_USER *string
var SSH_PASSWORD *string
var REMOTE_FILE_NAME *string
var LOCAL_FILE_NAME *string
var PER_SECONDS_SHOW_SPEED *string
var TIMEOUT *string

func main() {

	// recover
	defer func() {
		if err := recover(); err != nil {
			buf := new(bytes.Buffer)
			fmt.Println()
			fmt.Println(">>>>>>", err, "<<<<<<")
			fmt.Println()
			for i := 1; ; i++ {
				pc, file, line, ok := runtime.Caller(i)
				if !ok {
					break
				}
				fmt.Fprintf(buf, "file:%s ( line: %d, pc: 0x%x )\n", file, line, pc)
			}

			fmt.Println(buf.String())
			fmt.Println()
			fmt.Println()
		}
	}()

	SSH_HOST = flag.String("h", "", "remote host")
	SSH_USER = flag.String("u", "", "remote ssh user")
	SSH_PASSWORD = flag.String("p", "", "remote ssh user's password")
	REMOTE_FILE_NAME = flag.String("rf", "", "remote file (abs file path)")
	LOCAL_FILE_NAME = flag.String("lf", "", "local file (abs file path)")
	PER_SECONDS_SHOW_SPEED = flag.String("secs", "1", "per seconds show speed of progress")
	TIMEOUT = flag.String("to", "10", "timeout for ssh service")

	flag.Parse()

	perSecs, perSecsErr := strconv.Atoi(*PER_SECONDS_SHOW_SPEED)
	if perSecsErr != nil {
		panic(fmt.Sprintf("can't parse PER_SECONDS_SHOW_SPEED, %e", perSecsErr))
	}

	timeout, timeOutErr := strconv.Atoi(*TIMEOUT)
	if timeOutErr != nil {
		panic(fmt.Sprintf("can't parse TIMEOUT, %e", timeOutErr))
	}

	// debug info: configration
	fmt.Println("SSH_HOST:", *SSH_HOST, "SSH_USER:", *SSH_USER, "SSH_PASSWORD:",
		*SSH_PASSWORD, "remoteFileName:", *REMOTE_FILE_NAME, "localFileName:", *LOCAL_FILE_NAME)

	if *SSH_HOST == "" || *SSH_USER == "" || *SSH_PASSWORD == "" || *REMOTE_FILE_NAME == "" || *LOCAL_FILE_NAME == "" {
		panic("parameter error")
	}

	sshConfig := &ssh.ClientConfig{
		User: *SSH_USER,
		Auth: []ssh.AuthMethod{
			ssh.Password(*SSH_PASSWORD),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		ClientVersion:   "",
		Timeout:         time.Duration(timeout) * time.Second,
	}

	sshClient, err := ssh.Dial("tcp", *SSH_HOST, sshConfig)
	if err != nil {
		panic(err)
	}
	defer sshClient.Close()

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		panic(err)
	}
	defer sftpClient.Close()

	// remote
	remoteFile, err := sftpClient.Open(*REMOTE_FILE_NAME)
	if err != nil {
		panic(err)
	}
	defer remoteFile.Close()

	// local
	var localFile *os.File
	var localFileErr error
	_, localFileStatErr := os.Stat(*LOCAL_FILE_NAME)

	if localFileStatErr == nil {
		// already exist

		localFile, localFileErr = os.OpenFile(*LOCAL_FILE_NAME, os.O_RDWR|os.O_APPEND, os.ModeAppend|os.ModePerm)
		if localFileErr != nil {
			panic(localFileErr)
		}

		// seek position
		rstat, rstaterr := remoteFile.Stat()
		if rstaterr != nil {
			panic(rstaterr)
		}

		lstat, lstaterr := localFile.Stat()
		if lstaterr != nil {
			panic(lstaterr)
		}

		if rstat.Size() == lstat.Size() {
			fmt.Println("下载已完成")
			os.Exit(0)
		}

		// seek
		fmt.Println("本地文件:", *LOCAL_FILE_NAME, "已存在, 偏移位置为: ", lstat.Size(), "(", formatFileSize(lstat.Size()), "), 从该位置继续下载 ...")
		remoteFile.Seek(lstat.Size(), io.SeekStart)

	} else if os.IsNotExist(localFileStatErr) {
		// not exist

		localFile, localFileErr = os.Create(*LOCAL_FILE_NAME)
		if localFileErr != nil {
			panic(localFileErr)
		}
	} else {
		// file system error

		panic(localFileStatErr)
	}
	defer localFile.Close()

	rstat, _ := remoteFile.Stat()
	rsize := rstat.Size()

	// show speed of progress
	go func() {
		for {
			time.Sleep(time.Second * time.Duration(perSecs))
			lstat, _ := localFile.Stat()
			lsize := lstat.Size()
			fmt.Println("总大小:", formatFileSize(rsize), ", 已下载:", formatFileSize(lsize), "进度:", fmt.Sprintf("%.2f", (float64(lsize)*100/float64(rsize)))+"%")
		}
	}()

	// transfer begin
	_, copyErr := io.Copy(localFile, remoteFile)
	if copyErr != nil {
		panic(copyErr)
	}

	// transfer end
	fmt.Println("下载完成")
}

func formatFileSize(s int64) (size string) {
	if s < 1024 {
		return fmt.Sprintf("%.2fB", float64(s)/float64(1))
	} else if s < (1024 * 1024) {
		return fmt.Sprintf("%.2fKB", float64(s)/float64(1024))
	} else if s < (1024 * 1024 * 1024) {
		return fmt.Sprintf("%.2fMB", float64(s)/float64(1024*1024))
	} else if s < (1024 * 1024 * 1024 * 1024) {
		return fmt.Sprintf("%.2fGB", float64(s)/float64(1024*1024*1024))
	} else if s < (1024 * 1024 * 1024 * 1024 * 1024) {
		return fmt.Sprintf("%.2fTB", float64(s)/float64(1024*1024*1024*1024))
	} else {
		return fmt.Sprintf("%.2fEB", float64(s)/float64(1024*1024*1024*1024*1024))
	}
}
