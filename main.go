package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

var SSH_HOST *string
var SSH_USER *string
var SSH_PASSWORD *string
var remoteFileName *string
var localFileName *string

func main() {
	SSH_HOST = flag.String("h", "", "remote host")
	SSH_USER = flag.String("u", "", "remote ssh user")
	SSH_PASSWORD = flag.String("p", "", "remote ssh user's password")
	remoteFileName = flag.String("rf", "", "remote file (abs file path)")
	localFileName = flag.String("lf", "", "local file (abs file path)")
	flag.Parse()

	// debug info: configration
	fmt.Println("SSH_HOST:", *SSH_HOST, "SSH_USER:", *SSH_USER, "SSH_PASSWORD:",
		*SSH_PASSWORD, "remoteFileName:", *remoteFileName, "localFileName:", *localFileName)

	if *SSH_HOST == "" || *SSH_USER == "" || *SSH_PASSWORD == "" || *remoteFileName == "" || *localFileName == "" {
		panic("parameter error")
	}

	sshConfig := &ssh.ClientConfig{
		User: *SSH_USER,
		Auth: []ssh.AuthMethod{
			ssh.Password(*SSH_PASSWORD),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		ClientVersion:   "",
		Timeout:         10 * time.Second,
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
	remoteFile, err := sftpClient.Open(*remoteFileName)
	if err != nil {
		panic(err)
	}
	defer remoteFile.Close()

	// local
	var localFile *os.File
	var localFileErr error
	_, localFileStatErr := os.Stat(*localFileName)

	// already exist
	if localFileStatErr == nil {
		localFile, localFileErr = os.OpenFile(*localFileName, os.O_RDWR|os.O_APPEND, os.ModeAppend|os.ModePerm)
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
		fmt.Println("本地文件:", *localFileName, "已存在, 偏移位置为: ", lstat.Size(), "(", formatFileSize(lstat.Size()), "), 从该位置继续下载 ...")
		remoteFile.Seek(lstat.Size(), io.SeekStart)

		// not exist
	} else if os.IsNotExist(localFileStatErr) {
		localFile, localFileErr = os.Create(*localFileName)
		if localFileErr != nil {
			panic(localFileErr)
		}
	} else {
		panic(localFileStatErr)
	}
	defer localFile.Close()

	go func() {
		for {
			time.Sleep(time.Second)
			lstat, _ := localFile.Stat()
			rstat, _ := remoteFile.Stat()
			lsize := lstat.Size()
			rsize := rstat.Size()

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
