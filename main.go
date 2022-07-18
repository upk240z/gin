package main

import (
	"bytes"
	"crypto/sha256"
	"flag"
	"fmt"
	"github.com/clsung/grcode"
	"github.com/djimenez/iconv-go"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"golang.org/x/text/width"
	"image"
	"io"
	"os"
	"regexp"
	"strings"
	"time"
)

func DecodeQrCode(data []byte) ([]string, error) {
	img, ext, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	hash := sha256.New()
	hash.Write(data)
	file := fmt.Sprintf("tmp/%x.%s", hash.Sum(nil), ext)
	_, statErr := os.Stat(file)
	if statErr != nil {
		os.WriteFile(file, data, 0666)
	}

	results, imgErr := grcode.GetDataFromImage(img)
	if imgErr != nil {
		return nil, imgErr
	}

	return results, nil
}

func ParseQrText(text string) (string, map[string]interface{}) {
	text = width.Narrow.String(text)
	yearPrefix := time.Now().Format("2006")[0:2]

	info := map[string]interface{}{}

	pattern1, _ := regexp.Compile(`^2/-\s+/(\d{5})(\d{4})/(\d+)/(\d+)/`)
	pattern2, _ := regexp.Compile(`2/(.{4})(.{3})(.)(.{4})/`)

	result1 := pattern1.FindAllStringSubmatch(text, -1)
	if result1 != nil {
		info["kata"] = result1[0][1]
		info["rui"] = result1[0][2]
		ymd := result1[0][3]
		ym := result1[0][4]
		info["inspection-fin-date"] = yearPrefix + ymd[0:2] + "-" + ymd[2:4] + "-" + ymd[4:]
		info["first-month"] = yearPrefix + ym[0:2] + "-" + ym[2:]
		text = text[len(result1[0][0]):]
	}

	result2 := pattern2.FindAllStringSubmatch(text, -1)
	if result2 != nil {
		plate := map[string]string{}
		plate["area"] = strings.ReplaceAll(result2[0][1], " ", "")
		plate["class"] = result2[0][2]
		plate["hira"] = result2[0][3]
		plate["number"] = strings.ReplaceAll(result2[0][4], " ", "0")
		info["plate"] = plate
	}

	code := "error"
	if result1 != nil && result2 != nil {
		code = "success"
	} else if result1 != nil || result2 != nil {
		code = "warning"
	}

	return code, info
}

func main() {
	router := gin.Default()

	if _, envErr := os.Stat(".env"); envErr == nil {
		godotenv.Load(".env")
	}

	origins := strings.Split(os.Getenv("CORS_ORIGINS"), ",")

	router.Use(cors.New(cors.Config{
		AllowOrigins: origins,
		AllowMethods: []string{
			"POST",
			"GET",
			"OPTIONS",
		},
		AllowHeaders: []string{
			"Access-Control-Allow-Credentials",
			"Access-Control-Allow-Headers",
			"Content-Type",
			"Content-Length",
			"Accept-Encoding",
			"Authorization",
		},
		AllowCredentials: true,
	}))

	logF, _ := os.OpenFile(
		fmt.Sprintf("./log/gin-%s.log", time.Now().Format("20060102")),
		os.O_APPEND|os.O_CREATE|os.O_RDWR,
		0666,
	)
	gin.DefaultWriter = io.MultiWriter(logF)

	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	router.GET("/check-text", func(c *gin.Context) {
		text, exist := c.GetQuery("text")

		result := false

		if exist {
			sjis, _ := iconv.ConvertString(text, "utf-8", "sjis-win")
			utf8, _ := iconv.ConvertString(sjis, "sjis-win", "utf-8")
			fmt.Println(utf8)
			if text == utf8 {
				result = true
			}
		}

		c.JSON(200, gin.H{
			"result": result,
		})
	})

	router.POST("/qr-img", func(c *gin.Context) {
		data, _ := c.GetRawData()

		output := map[string]interface{}{
			"message": "Success",
		}

		for {
			lines, err := DecodeQrCode(data)
			if err != nil {
				fmt.Println(err)
				output["result"] = "error"
				output["message"] = err.Error()
				break
			}

			if len(lines) == 0 {
				output["result"] = "error"
				output["message"] = "Detection failed"
				break
			}

			code, info := ParseQrText(strings.Join(lines, " "))
			output["result"] = code
			output["info"] = info
			if code == "error" {
				output["message"] = "Detection failed"
			} else if code == "warning" {
				output["message"] = "Some detection failed"
			}

			break
		}

		c.JSON(200, output)
	})

	port := flag.Int("port", 8888, "listen port number")
	flag.Parse()

	fmt.Printf("http://localhost:%d/\n", *port)
	router.Run(fmt.Sprintf(":%d", *port))
}
