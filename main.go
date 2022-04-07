package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"

	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/valyala/fastjson"
	"gopkg.in/gomail.v2"
)

type Config struct {
	port         int
	syncPath     string
	outputPath   string
	smtpHost     string
	smtpPort     int
	username     string
	password     string
	receiverMail string
	title        string
	kindleMail   string
}

func initConfig(path string) *Config {

	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}

	config := &Config{
		port:         fastjson.GetInt(data, "port"),
		syncPath:     fastjson.GetString(data, "syncPath"),
		outputPath:   fastjson.GetString(data, "outputPath"),
		title:        fastjson.GetString(data, "title"),
		smtpHost:     fastjson.GetString(data, "smtpHost"),
		smtpPort:     fastjson.GetInt(data, "smtpPort"),
		username:     fastjson.GetString(data, "username"),
		password:     fastjson.GetString(data, "password"),
		receiverMail: fastjson.GetString(data, "receiverMail"),
		kindleMail:   fastjson.GetString(data, "kindleMail"),
	}

	if config.port == 0 {
		config.port = 7026
	}
	if config.smtpPort == 0 {
		config.smtpPort = 465
	}
	if config.title == "" {
		config.title = "[简悦] - %s"
	}
	if config.syncPath != "" && config.outputPath == "" {
		config.outputPath = filepath.Join(config.syncPath, "output")
	}

	log.Println("init config")

	return config
}

// 未验证 json 返回 201
// 已验证 json 返回 403
// 这里无论如何都返回成功，有其他用处以后再说
func verifyHandle(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Fatal(err)
	}
	result, err := json.Marshal(struct {
		Code   int    `json:"code"`
		Status string `json:"status"`
	}{
		Code:   403,
		Status: "same",
	})
	if err != nil {
		log.Fatal(err)
	}

	_, err = w.Write(result)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("verify success")
}

// 如果浏览器插件的设置项更改了，它会发一个 key 为 config 的请求，json 返回 200
// 剩余情况下，返回一个 key 为 result 的 json
func configHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	if config.syncPath != "" {
		err := r.ParseForm()
		if err != nil {
			log.Fatal(err)
		}
		if data := r.Form.Get("config"); data != "" {
			err := ioutil.WriteFile(filepath.Join(config.syncPath, "simpread_config.json"), []byte(data), 644)
			if err != nil {
				log.Fatal(err)
			}

			result, err := json.Marshal(struct {
				Status int `json:"status"`
			}{Status: 200})
			if err != nil {
				log.Fatal(err)
			}

			_, err = w.Write(result)
			if err != nil {
				log.Fatal(err)
			}
			log.Println("sync config from browser")
		} else {
			config, err := ioutil.ReadFile(filepath.Join(config.syncPath, "simpread_config.json"))
			if err != nil {
				log.Fatal(err)
			}
			result, err := json.Marshal(struct {
				Status int    `json:"status"`
				Result string `json:"result"`
			}{
				Status: 200,
				Result: string(config),
			})
			if err != nil {
				log.Fatal(err)
			}

			_, err = w.Write(result)
			if err != nil {
				log.Fatal(err)
			}
			log.Println("sync config from local")
		}
	} else {
		result, err := json.Marshal(struct {
			Status string `json:"status"`
		}{
			Status: "error",
		})
		if err != nil {
			log.Fatal(err)
		}

		_, err = w.Write(result)
		if err != nil {
			log.Fatal(err)
		}
		log.Fatal("Please set syncPath first !")
	}
}

func plainHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	err := r.ParseForm()
	if err != nil {
		log.Fatal(err)
	}
	title := r.Form.Get("title")
	content := r.Form.Get("content")

	err = ioutil.WriteFile(filepath.Join(config.outputPath, title), []byte(content), 0644)
	if err != nil {
		log.Fatal(err)
	}

	result, err := json.Marshal(struct {
		Status int `json:"status"`
	}{Status: 200})
	if err != nil {
		log.Fatal(err)
	}

	_, err = w.Write(result)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("save file: %s\n", title)
}

// TODO 发送后删除附件
func mailHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	err := r.ParseForm()
	if err != nil {
		log.Fatal(err)
	}
	title := r.Form.Get("title")
	content := r.Form.Get("content")
	attach := r.Form.Get("attach")

	d := gomail.NewDialer(config.smtpHost, config.smtpPort, config.username, config.password)
	s, err := d.Dial()
	if err != nil {
		panic(err)
	}

	m := gomail.NewMessage()
	m.SetHeader("From", config.username)
	m.SetHeader("To", config.receiverMail)
	m.SetHeader("Subject", fmt.Sprintf(config.title, title))
	m.SetBody("text/html", content)

	var attachPath string
	if attach != "" {
		attachPath = fmt.Sprintf("tmp-%s.%s", title, attach)
		m.Attach(attachPath)
	}

	err = gomail.Send(s, m)
	if err != nil {
		log.Fatal(err)
	}

	if attach != "" {
		os.Remove(attachPath)
	}

	result, err := json.Marshal(struct {
		Status int `json:"status"`
	}{Status: 200})
	if err != nil {
		log.Fatal(err)
	}

	_, err = w.Write(result)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("send mail: %s\n", title)
}

func convertHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	err := r.ParseForm()
	if err != nil {
		log.Fatal(err)
	}
	title := r.Form.Get("title")
	content := r.Form.Get("content")
	in := r.Form.Get("in")   //md
	out := r.Form.Get("out") //epub

	err = ioutil.WriteFile(title+"."+in, []byte(content), 0644)
	if err != nil {
		log.Fatal(err)
	}

	pandoc := "pandoc"
	if runtime.GOOS == "darwin" {
		pandoc = "/usr/local/bin/pandoc"
	}
	cmd := exec.Command(pandoc, title+"."+in, "-o", filepath.Join(config.outputPath, title+"."+out))

	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
	}

	err = cmd.Wait()
	if err != nil {
		log.Fatal(err)
	}

	os.Remove(title + "." + in)

	result, err := json.Marshal(struct {
		Status int `json:"status"`
	}{Status: 200})
	if err != nil {
		log.Fatal(err)
	}

	_, err = w.Write(result)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("convert file: %s\n", title)
}

var config *Config

func main() {
	var path string
	if len(os.Args) < 2 {
		path = "config.json"
	} else {
		path = os.Args[1]
	}
	config = initConfig(path)

	http.HandleFunc("/verify", verifyHandle)
	http.HandleFunc("/config", configHandle)
	http.HandleFunc("/plain", plainHandle)
	http.HandleFunc("/mail", mailHandle)
	http.HandleFunc("/convert", convertHandle)

	err := http.ListenAndServe(fmt.Sprint(":", config.port), nil)
	if err != nil {
		log.Fatal(err)
	}
}
