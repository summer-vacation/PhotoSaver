/*
	backend https server:
		upload file
		download file 
*/
package main

import(
	"context"
	"os"
	"io"
	"log"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/tencentyun/cos-go-sdk-v5"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"PhotoSaver/src/config"
)

var (
	cfg = pflag.StringP("config", "c", "", "config file path")
)

func main() {
	pflag.Parse()
	if err:= config.Init(*cfg); err != nil {
		panic(err)
	}

	urlParsed, _ := url.Parse("http://<bucketname-appid>.cos.<>.myqcloud.com")
	baseUrl := &cos.BaseURL{BucketURL: urlParsed}
	cosClient := cos.NewClient(baseUrl, &http.Client{
		Transport: &cos.AuthorizationTransport{
            SecretID: "COS_SECRETID",
            SecretKey: "COS_SECRETKEY",  
        },
	})

	gin.SetMode(viper.GetString("runmode"))
	gin.DisableConsoleColor()
	f, _ := os.Create("gin.log")
	
	gin.DefaultWriter = io.MultiWriter(f)

	router := gin.New()

	// accept upload request and save file to COS
	router.POST("/upload", func(c *gin.Context){
		name := c.PostForm("name")
		form, err := c.MultipartForm()
		if err!=nil{
			c.String(http.StatusBadRequest, fmt.Sprintf("Upload file request form error: %s", err.Error()))
			return
		}
		files := form.File["files"]

		for _, file := range files {
			filename := filepath.Base(file.Filename)
			if err := c.SaveUploadedFile(file, filename); err != nil {
				c.String(http.StatusBadRequest, fmt.Sprintf("Upload file to server error: %s", err.Error()))
				return
			}
			src, err := file.Open()
			if err != nil {
				c.String(http.StatusBadRequest, fmt.Sprintf("Upload file open error: %s", err.Error()))
			}
			
			opt := &cos.ObjectPutOptions{
				ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{
					ContentLength: int(file.Size),
				},
			}
			// 先写同步的，测试ok再改协程

			rsp, err := cosClient.Object.Put(context.Background(), filename, src, opt)
			if err != nil{
				c.String(http.StatusBadRequest, fmt.Sprintf("Upload file to COS error: %s", err.Error()))
			}
			if rsp.StatusCode != 200 {
				c.String(http.StatusBadRequest, fmt.Sprintf("Upload file to COS error, retCode: %d", rsp.StatusCode))
			}
			log.Printf("file :%s, upload to cos ok!", filename)
		}
		c.String(http.StatusOK, fmt.Sprintf("user %s,Upload file to COS ok!",name))
	})
	router.Run(":8080")
}