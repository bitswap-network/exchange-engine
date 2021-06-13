package s3

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"v1.1-fulfiller/config"
)

type S3Session struct {
	Session *session.Session
	Bucket  string
	Name    string
}

var Session = &S3Session{}

func Setup() {
	log.Println("s3 setup")
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1")},
	)
	if err != nil {
		log.Fatal(err)
	}
	Session.Session = sess
	Session.Bucket = config.S3Config.Bucket
	Session.Name = config.S3Config.LogName
	log.Println("s3 setup complete")
}

func UploadToS3(data []byte) {
	log.Println("uploading... ", time.Now())
	file := bytes.NewReader(data)
	uploader := s3manager.NewUploader(Session.Session)
	fileName := fmt.Sprintf("%s-%v.json", Session.Name, time.Now().UnixNano()/int64(time.Millisecond))

	_, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(config.S3Config.Bucket),
		Key:    aws.String(fileName),
		Body:   file,
	})
	if err != nil {
		log.Panicln(err)
		return
	}
	var backupTail string
	if config.IsTest {
		backupTail = "current-staging"
	} else {
		backupTail = "current"
	}
	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(Session.Bucket),
		Key:    aws.String(fmt.Sprintf("%s-%s.json", Session.Name, backupTail)),
		Body:   file,
	})
	if err != nil {
		log.Panicln(err)
		return
	}
	log.Println("done uploading", time.Now())
	return
}

func GetOrderbook() (data []byte) {

	var backupTail string
	if config.IsTest {
		backupTail = "current-staging"
	} else {
		backupTail = "current"
	}

	downloader := s3manager.NewDownloader(Session.Session)
	log.Println("fetching orderbook: ", fmt.Sprintf("%s-%s.json", Session.Name, backupTail))

	buf := aws.NewWriteAtBuffer([]byte{})
	_, err := downloader.Download(buf,
		&s3.GetObjectInput{
			Bucket: aws.String(os.Getenv("BUCKET")),
			Key:    aws.String(fmt.Sprintf("%s-%s.json", Session.Name, backupTail)),
		})
	if err != nil {
		log.Println("Unable to download item", err)
		return nil
	}
	data = buf.Bytes()
	return data
}
