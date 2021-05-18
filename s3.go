package main

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

func getOrderbookBytes() (data []byte) {
	data, err := exchange.MarshalJSON()
	if err != nil {
		log.Println(err)
		return
	}
	return data
}

func UploadToS3(data []byte, name string) {
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
	log.Println("done uploading", time.Now())
}

func GetOrderbookS3() (data []byte) {
	sess, _ := AwsGetSession()
	downloader := s3manager.NewDownloader(sess)

	svc := s3.New(sess)
	log.Println("fetching... ")
	resp, err := svc.ListObjectsV2(&s3.ListObjectsV2Input{Bucket: aws.String(os.Getenv("BUCKET")), Prefix: aws.String("orderbook")})
	if err != nil {
		log.Println("Unable to list items in bucket: ", os.Getenv("BUCKET"), err)
		return nil
	}
	s := resp.Contents[len(resp.Contents)-1]
	log.Println(resp.Contents)
	buf := aws.NewWriteAtBuffer([]byte{})
	_, err = downloader.Download(buf,
		&s3.GetObjectInput{
			Bucket: aws.String(os.Getenv("BUCKET")),
			Key:    aws.String(*s.Key),
		})
	if err != nil {
		log.Println("Unable to download item", err)
		return nil
	}
	data = buf.Bytes()
	return data
}
