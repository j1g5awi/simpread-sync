package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/gomail.v2"
)

var (
	configFile   string
	port         int
	syncPath     string
	outputPath   string
	smtpHost     string
	smtpPort     int
	smtpUsername string
	smtpPassword string
	mailTitle    string
	receiverMail string
	kindleMail   string
)

var rootCmd = &cobra.Command{
	Use: "simpread-sync",
	Run: func(cmd *cobra.Command, args []string) {
		http.HandleFunc("/verify", verifyHandle)
		http.HandleFunc("/config", configHandle)
		http.HandleFunc("/plain", plainHandle)
		http.HandleFunc("/mail", mailHandle)
		http.HandleFunc("/convert", convertHandle)
		http.HandleFunc("/reading/", readingHandle)
		http.HandleFunc("/proxy", proxyHandle)

		err := http.ListenAndServe(fmt.Sprint(":", port), nil)
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.Flags().StringVarP(&configFile, "config", "c", "", "config file")
	rootCmd.Flags().IntVarP(&port, "port", "p", 7026, "port")
	rootCmd.Flags().StringVar(&syncPath, "sync-path", "", "sync path")
	rootCmd.Flags().StringVar(&outputPath, "output-path", "", "output path")
	rootCmd.Flags().StringVar(&smtpHost, "smtp-host", "", "smtp host")
	rootCmd.Flags().IntVar(&smtpPort, "smtp-port", 465, "smtp port")
	rootCmd.Flags().StringVar(&smtpUsername, "smtp-username", "", "smtp username")
	rootCmd.Flags().StringVar(&smtpPassword, "smtp-password", "", "smtp password")
	rootCmd.Flags().StringVar(&mailTitle, "mail-title", "[简悦] - {{ title }}", "mail title")
	rootCmd.Flags().StringVar(&receiverMail, "receiver-mail", "", "receiver mail")
	rootCmd.Flags().StringVar(&kindleMail, "kindle-mail", "", "kindle mail")

	viper.BindPFlag("port", rootCmd.Flags().Lookup("port"))
	viper.BindPFlag("syncPath", rootCmd.Flags().Lookup("sync-path"))
	viper.BindPFlag("outputPath", rootCmd.Flags().Lookup("output-path"))
	viper.BindPFlag("smtpHost", rootCmd.Flags().Lookup("smtp-host"))
	viper.BindPFlag("smtpPort", rootCmd.Flags().Lookup("smtp-port"))
	viper.BindPFlag("smtpUsername", rootCmd.Flags().Lookup("smtp-username"))
	viper.BindPFlag("smtpPassword", rootCmd.Flags().Lookup("smtp-password"))
	viper.BindPFlag("mailTitle", rootCmd.Flags().Lookup("mail-title"))
	viper.BindPFlag("receiverMail", rootCmd.Flags().Lookup("receiver-mail"))
	viper.BindPFlag("kindleMail", rootCmd.Flags().Lookup("kindle-mail"))
}

func initConfig() {
	if configFile != "" {
		viper.SetConfigFile(configFile)
	}

	if err := viper.ReadInConfig(); err == nil {
		log.Println("加载配置文件：", viper.ConfigFileUsed())
	}

	port = viper.GetInt("port")
	syncPath = viper.GetString("syncPath")
	outputPath = viper.GetString("outputPath")
	smtpHost = viper.GetString("smtpHost")
	smtpPort = viper.GetInt("smtpPort")
	smtpUsername = viper.GetString("smtpUsername")
	smtpPassword = viper.GetString("smtpPassword")
	mailTitle = viper.GetString("mailTitle")
	receiverMail = viper.GetString("receiverMail")
	kindleMail = viper.GetString("kindleMail")

	if syncPath == "" {
		log.Fatal("未读取到 syncPath！")
	}
	if outputPath == "" {
		outputPath = filepath.Join(syncPath, "output")
	}
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
	if syncPath != "" {
		err := r.ParseForm()
		if err != nil {
			log.Fatal(err)
		}
		if data := r.Form.Get("config"); data != "" {
			err := ioutil.WriteFile(filepath.Join(syncPath, "simpread_config.json"), []byte(data), 0644)
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
			config, err := ioutil.ReadFile(filepath.Join(syncPath, "simpread_config.json"))
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

	err = ioutil.WriteFile(filepath.Join(outputPath, title), []byte(content), 0644)
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

func mailHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	err := r.ParseForm()
	if err != nil {
		log.Fatal(err)
	}
	// 这里偷懒直接替换文本
	title := strings.ReplaceAll(mailTitle, "{{ title }}", r.Form.Get("title"))

	content := r.Form.Get("content")
	attach := r.Form.Get("attach")

	d := gomail.NewDialer(smtpHost, smtpPort, smtpUsername, smtpPassword)
	s, err := d.Dial()
	if err != nil {
		panic(err)
	}

	m := gomail.NewMessage()
	m.SetHeader("From", smtpUsername)
	m.SetHeader("To", receiverMail)
	m.SetHeader("Subject", title)
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

	err = ioutil.WriteFile("tmp-"+title+"."+in, []byte(content), 0644)
	if err != nil {
		log.Fatal(err)
	}

	pandoc := "pandoc"
	if runtime.GOOS == "darwin" {
		pandoc = "/usr/local/bin/pandoc"
	}
	cmd := exec.Command(pandoc, "tmp-"+title+"."+in, "-o", filepath.Join(outputPath, title+"."+out))

	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
	}

	err = cmd.Wait()
	if err != nil {
		log.Fatal(err)
	}

	os.Remove("tmp-" + title + "." + in)

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

func readingHandle(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Fatal(err)
	}

	var files []string
	fileInfo, err := ioutil.ReadDir(outputPath)
	if err != nil {
		log.Fatal(err)
	}
	for _, file := range fileInfo {
		if !file.IsDir() {
			files = append(files, file.Name())
		}
	}

	var result []byte
	if r.RequestURI == "/reading/index" {
		w.Header().Set("content-type", "application/json")
		result, err = json.Marshal(struct {
			Files []string `json:"files"`
		}{Files: files})
		if err != nil {
			log.Fatal(err)
		}
	} else {
		id := strings.Replace(r.URL.Path, "/reading/", "", 1)
		if err != nil {
			log.Fatal(err)
		}

		query := r.URL.Query().Get("title")
		suffix := r.Header.Get("type")
		if suffix == "" {
			suffix = ".html"
		}

		var title string
		for _, file := range files {
			if (strings.HasPrefix(file, id+"-") &&
				strings.HasSuffix(file, suffix) &&
				!strings.Contains(file, "@annote")) ||
				file == id+suffix ||
				file == query+suffix {
				title = file
				break
			}
		}

		if title != "" {
			result, err = ioutil.ReadFile(filepath.Join(outputPath, title))
			if err != nil {
				log.Fatal(err)
			}
		} else {
			w.Header().Set("content-type", "application/json")
			result, err = json.Marshal(struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			}{Code: 404, Message: "没有找到对应的内容"})
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	_, err = w.Write(result)
	if err != nil {
		log.Fatal(err)
	}
}

func proxyHandle(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	resp, err := http.Get(url)
	if err != nil {
		log.Println("proxy error: ", err)
		return
	}
	defer resp.Body.Close()

	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		log.Println("proxy error: ", err)
		return
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
