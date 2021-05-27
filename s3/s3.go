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
)

const name = "orderbook"

func AwsGetSession() (sess *session.Session, err error) {
	sess, err = session.NewSession(&aws.Config{
		Region: aws.String("us-east-1")},
	)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return sess, nil
}

func UploadToS3(data []byte) {
	file := bytes.NewReader(data)
	sess, _ := AwsGetSession()
	uploader := s3manager.NewUploader(sess)
	fileName := fmt.Sprintf("%s-%v.json", name, time.Now().UnixNano()/int64(time.Millisecond))
	log.Println("uploading... ", time.Now())
	_, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(os.Getenv("BUCKET")),
		Key:    aws.String(fileName),
		Body:   file,
	})
	if err != nil {
		log.Println(err)
	}
	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(os.Getenv("BUCKET")),
		Key:    aws.String(fmt.Sprintf("%s-%s.json", name, "current")),
		Body:   file,
	})
	if err != nil {
		log.Println(err)
	}
	log.Println("done uploading", time.Now())
}

func GetOrderbook() (data []byte) {
	sess, _ := AwsGetSession()
	downloader := s3manager.NewDownloader(sess)
	log.Println("fetching... ")
	
	buf := aws.NewWriteAtBuffer([]byte{})
	_, err := downloader.Download(buf,
		&s3.GetObjectInput{
			Bucket: aws.String(os.Getenv("BUCKET")),
			Key:    aws.String("orderbook-current.json"),
		})
	if err != nil {
		log.Println("Unable to download item", err)
		return nil
	}
	data = buf.Bytes()
	return data
}
