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
	config "v1.1-fulfiller/config"
)

type S3Session struct {
	Session *session.Session
	Bucket  string
	Name    string
}

var Session = &S3Session{}

func Setup() {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1")},
	)
	if err != nil {
		log.Fatal(err)
	}
	Session.Session = sess
	Session.Bucket = config.S3Config.Bucket
	Session.Name = config.S3Config.LogName
}

func UploadToS3(data []byte) {
	file := bytes.NewReader(data)
	uploader := s3manager.NewUploader(Session.Session)
	fileName := fmt.Sprintf("%s-%v.json", Session.Name, time.Now().UnixNano()/int64(time.Millisecond))
	log.Println("uploading... ", time.Now())
	_, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(config.S3Config.Bucket),
		Key:    aws.String(fileName),
		Body:   file,
	})
	if err != nil {
		log.Panicln(err)
	}
	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(Session.Bucket),
		Key:    aws.String(fmt.Sprintf("%s-%s.json", Session.Name, "current")),
		Body:   file,
	})
	if err != nil {
		log.Panicln(err)
	}
	log.Println("done uploading", time.Now())
}

func GetOrderbook() (data []byte) {

	downloader := s3manager.NewDownloader(Session.Session)
	log.Println("fetching orderbook")

	buf := aws.NewWriteAtBuffer([]byte{})
	_, err := downloader.Download(buf,
		&s3.GetObjectInput{
			Bucket: aws.String(os.Getenv("BUCKET")),
			Key:    aws.String("orderbook-current.json"),
		})
	if err != nil {
		log.Panicln("Unable to download item", err)
		return nil
	}
	data = buf.Bytes()
	return data
}
