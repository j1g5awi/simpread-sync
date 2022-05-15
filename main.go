package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tidwall/gjson"
	"gopkg.in/gomail.v2"
)

var Version string = "(devel)"
var (
	configFile   string
	port         int
	syncPath     string
	outputPath   string
	markdownPath string
	autoRemove   bool
	smtpHost     string
	smtpPort     int
	smtpUsername string
	smtpPassword string
	mailTitle    string
	receiverMail string
	kindleMail   string
	checkVersion bool
)

var rootCmd = &cobra.Command{
	Use: "simpread-sync",
	Run: func(cmd *cobra.Command, args []string) {
		http.HandleFunc("/verify", verifyHandle)
		http.HandleFunc("/config", configHandle)
		http.HandleFunc("/plain", plainHandle)
		http.HandleFunc("/mail", mailHandle)
		http.HandleFunc("/convert", convertHandle)
		http.HandleFunc("/wkhtmltopdf", wkhtmltopdfHandle)
		http.HandleFunc("/reading/", readingHandle)
		http.HandleFunc("/proxy", proxyHandle)
		http.HandleFunc("/textbundle", textbundleHandle)

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
	rootCmd.Flags().StringVar(&markdownPath, "markdown-path", "", "markdown output path")
	rootCmd.Flags().BoolVar(&autoRemove, "auto-remove", false, "auto remove")
	rootCmd.Flags().StringVar(&smtpHost, "smtp-host", "", "smtp host")
	rootCmd.Flags().IntVar(&smtpPort, "smtp-port", 465, "smtp port")
	rootCmd.Flags().StringVar(&smtpUsername, "smtp-username", "", "smtp username")
	rootCmd.Flags().StringVar(&smtpPassword, "smtp-password", "", "smtp password")
	rootCmd.Flags().StringVar(&mailTitle, "mail-title", "[简悦] - {{title}}", "mail title")
	rootCmd.Flags().StringVar(&receiverMail, "receiver-mail", "", "receiver mail")
	rootCmd.Flags().StringVar(&kindleMail, "kindle-mail", "", "kindle mail")
	rootCmd.Flags().BoolVarP(&checkVersion, "version", "V", false, "check version")

	viper.BindPFlag("port", rootCmd.Flags().Lookup("port"))
	viper.BindPFlag("syncPath", rootCmd.Flags().Lookup("sync-path"))
	viper.BindPFlag("outputPath", rootCmd.Flags().Lookup("output-path"))
	viper.BindPFlag("markdownPath", rootCmd.Flags().Lookup("markdown-path"))
	viper.BindPFlag("autoRemove", rootCmd.Flags().Lookup("auto-remove"))
	viper.BindPFlag("smtpHost", rootCmd.Flags().Lookup("smtp-host"))
	viper.BindPFlag("smtpPort", rootCmd.Flags().Lookup("smtp-port"))
	viper.BindPFlag("smtpUsername", rootCmd.Flags().Lookup("smtp-username"))
	viper.BindPFlag("smtpPassword", rootCmd.Flags().Lookup("smtp-password"))
	viper.BindPFlag("mailTitle", rootCmd.Flags().Lookup("mail-title"))
	viper.BindPFlag("receiverMail", rootCmd.Flags().Lookup("receiver-mail"))
	viper.BindPFlag("kindleMail", rootCmd.Flags().Lookup("kindle-mail"))
}

func initConfig() {
	if checkVersion {
		log.Println("当前版本：", Version)
		if Version == "(devel)" {
			os.Exit(0)
		}
		resp, err := http.Get("https://api.github.com/repos/j1g5awi/simpread-sync/releases/latest")
		if err != nil {
			log.Fatal("检查更新失败：", err)
		}
		defer resp.Body.Close()
		data, _ := ioutil.ReadAll(resp.Body)
		remote := gjson.Get(string(data), "tag_name").String()
		sp := regexp.MustCompile(`v(\d+)\.(\d+)\.(\d+)-?(.+)?`)
		cur := sp.FindStringSubmatch(Version)
		re := sp.FindStringSubmatch(remote)
		for i := 1; i <= 3; i++ {
			curSub, _ := strconv.Atoi(cur[i])
			reSub, _ := strconv.Atoi(re[i])
			if curSub < reSub {
				log.Printf("检测到最新版 %s，请前往 https://github.com/j1g5awi/simpread-sync/releases 下载", remote)
				os.Exit(0)
			} else if curSub > reSub {
				os.Exit(0)
			}
		}
		if cur[4] == "" || re[4] == "" {
			if re[4] == "" && cur[4] != re[4] {
				log.Printf("检测到最新版 %s，请前往 https://github.com/j1g5awi/simpread-sync/releases 下载", remote)
			}
		} else if cur[4] < re[4] {
			log.Printf("检测到最新版 %s，请前往 https://github.com/j1g5awi/simpread-sync/releases 下载", remote)
		}
		os.Exit(0)
	}

	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.SetConfigFile("config.json")
	}

	if err := viper.ReadInConfig(); err == nil {
		log.Println("加载配置文件：", viper.ConfigFileUsed())
	}

	port = viper.GetInt("port")
	syncPath = viper.GetString("syncPath")
	outputPath = viper.GetString("outputPath")
	markdownPath = viper.GetString("markdownPath")
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
	if markdownPath == "" {
		markdownPath = outputPath
	}
}

// 未验证 json 返回 201
// 已验证 json 返回 403
// 这里无论如何都返回成功，有其他用处以后再说
func verifyHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	err := r.ParseForm()
	if err != nil {
		log.Println(err)
		return
	}
	result, err := json.Marshal(struct {
		Code   int    `json:"code"`
		Status string `json:"status"`
	}{
		Code:   403,
		Status: "same",
	})
	if err != nil {
		log.Println(err)
		return
	}

	_, err = w.Write(result)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("verify success")
}

// 规避标准库大小限制
func myParseForm(r *http.Request) error {
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	vs, err := url.ParseQuery(string(b))
	if err != nil {
		return err
	}
	r.Form = make(url.Values)
	for k, vs := range vs {
		r.Form[k] = append(r.Form[k], vs...)
	}
	return nil
}

var etag string
var unrdist map[int]struct{}

// 如果浏览器插件的设置项更改了，它会发一个 key 为 config 的请求，json 返回 200
// 剩余情况下，返回一个 key 为 result 的 json
func configHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	if syncPath != "" {
		err := myParseForm(r)
		if err != nil {
			log.Println(err)
			return
		}

		if data := r.Form.Get("config"); data != "" {
			err := ioutil.WriteFile(filepath.Join(syncPath, "simpread_config.json"), []byte(data), 0644)
			if err != nil {
				log.Println(err)
				return
			}

			if autoRemove {
				newUnrdist := make(map[int]struct{}, len(gjson.Get(data, "unrdist").Array()))
				for _, unrd := range gjson.Get(data, "unrdist").Array() {
					newUnrdist[int(unrd.Get("idx").Int())] = struct{}{}
				}
				var toDelete int
				for idx := range unrdist {
					if _, ok := newUnrdist[idx]; !ok {
						toDelete = idx
						break
					}
				}
				unrdist = newUnrdist
				fileInfo, err := ioutil.ReadDir(outputPath)
				if err != nil {
					log.Println(err)
					return
				}
				for _, file := range fileInfo {
					if !file.IsDir() && strings.HasPrefix(file.Name(), fmt.Sprint(toDelete, "-")) {
						err := os.Remove(filepath.Join(outputPath, file.Name()))
						if err != nil {
							log.Println(err)
						}
					}
				}
			}

			result, err := json.Marshal(struct {
				Status int `json:"status"`
			}{Status: 200})
			if err != nil {
				log.Println(err)
				return
			}

			_, err = w.Write(result)
			if err != nil {
				log.Println(err)
				return
			}
			log.Println("sync config from browser")
		} else {
			config, err := ioutil.ReadFile(filepath.Join(syncPath, "simpread_config.json"))
			if err != nil {
				log.Println(err)
				return
			}

			hash := md5.Sum(config)
			etag = hex.EncodeToString(hash[:])
			if r.Header.Get("If-None-Match") == etag {
				w.WriteHeader(http.StatusNotModified)
				return
			}

			w.Header().Set("Etag", etag)

			result, err := json.Marshal(struct {
				Status int    `json:"status"`
				Result string `json:"result"`
			}{
				Status: 200,
				Result: string(config),
			})
			if err != nil {
				log.Println(err)
				return
			}

			_, err = w.Write(result)
			if err != nil {
				log.Println(err)
				return
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
			log.Println(err)
			return
		}

		_, err = w.Write(result)
		if err != nil {
			log.Println(err)
			return
		}
	}
}

func plainHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	err := myParseForm(r)
	if err != nil {
		log.Println(err)
		return
	}

	title := r.Form.Get("title")
	content := r.Form.Get("content")

	var filePath string
	if strings.HasSuffix(title, ".md") && !strings.HasPrefix(title, "tmp-") {
		filePath = filepath.Join(markdownPath, title)
	} else {
		filePath = filepath.Join(outputPath, title)
	}
	err = ioutil.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		log.Println(err)
		return
	}

	result, err := json.Marshal(struct {
		Status int `json:"status"`
	}{Status: 200})
	if err != nil {
		log.Println(err)
		return
	}

	_, err = w.Write(result)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("save file:", title)
}

func mailHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	err := r.ParseForm()
	if err != nil {
		log.Println(err)
		return
	}
	// 这里偷懒直接替换文本
	title := strings.ReplaceAll(mailTitle, "{{title}}", r.Form.Get("title"))

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
		log.Println(err)
		return
	}

	if attach != "" {
		os.Remove(attachPath)
	}

	result, err := json.Marshal(struct {
		Status int `json:"status"`
	}{Status: 200})
	if err != nil {
		log.Println(err)
		return
	}

	_, err = w.Write(result)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("send mail:", title)
}

func convertHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	err := r.ParseForm()
	if err != nil {
		log.Println(err)
		return
	}
	title := r.Form.Get("title")
	content := r.Form.Get("content")
	in := r.Form.Get("in")   //md
	out := r.Form.Get("out") //epub

	err = ioutil.WriteFile("tmp-"+title+"."+in, []byte(content), 0644)
	if err != nil {
		log.Println(err)
		return
	}

	pandoc := "pandoc"
	if runtime.GOOS == "darwin" {
		pandoc = "/usr/local/bin/pandoc"
	}
	cmd := exec.Command(pandoc, "tmp-"+title+"."+in, "-o", filepath.Join(outputPath, title+"."+out))

	err = cmd.Run()
	if err != nil {
		log.Println(err)
		return
	}

	os.Remove("tmp-" + title + "." + in)

	result, err := json.Marshal(struct {
		Status int `json:"status"`
	}{Status: 200})
	if err != nil {
		log.Println(err)
		return
	}

	_, err = w.Write(result)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("convert file:", title)
}

func wkhtmltopdfHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	err := r.ParseForm()
	if err != nil {
		log.Println(err)
		return
	}
	title := r.Form.Get("title")
	content := r.Form.Get("content")
	params := strings.Split(r.Form.Get("params"), " ")
	root := r.Form.Get("root")

	err = ioutil.WriteFile("tmp-"+title+".html", []byte(content), 0644)
	if err != nil {
		log.Println(err)
		return
	}

	if root == "" {
		root = "wkhtmltopdf"
	}
	params = append(params, "tmp-"+title+".html", filepath.Join(outputPath, title+".pdf"))
	cmd := exec.Command(root, params...)

	err = cmd.Run()
	if err != nil {
		log.Println(err)
		return
	}

	os.Remove("tmp-" + title + ".html")

	result, err := json.Marshal(struct {
		Status int `json:"status"`
	}{Status: 200})
	if err != nil {
		log.Println(err)
		return
	}

	_, err = w.Write(result)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("wkhtmltopdf:", title)
}

func readingHandle(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Println(err)
		return
	}

	var files []string
	fileInfo, err := ioutil.ReadDir(outputPath)
	if err != nil {
		log.Println(err)
		return
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
			log.Println(err)
			return
		}
		log.Println("reading index")
	} else {
		id := strings.Replace(r.URL.Path, "/reading/", "", 1)
		if err != nil {
			log.Println(err)
			return
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
				log.Println(err)
				return
			}
			log.Println("reading file:", title)
		} else {
			w.Header().Set("content-type", "application/json")
			result, err = json.Marshal(struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			}{Code: 404, Message: "没有找到对应的内容"})
			if err != nil {
				log.Println(err)
				return
			}
		}
	}

	_, err = w.Write(result)
	if err != nil {
		log.Println(err)
		return
	}
}

var matchImage = regexp.MustCompile(`(?i)\!\[(\S+)?\]\(http(s)?:\/\/[^)]+\)`)
var matchReplace = regexp.MustCompile(`^!\[(\S+)?\]\(|\)$`)

func textbundleHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	err := r.ParseForm()
	if err != nil {
		log.Println(err)
		return
	}
	title := r.Form.Get("title")
	content := r.Form.Get("content")
	images := matchImage.FindAllString(content, -1)

	filePath := filepath.Join(markdownPath, title+".textbundle")

	err = os.Mkdir(filePath, 0755)
	if err != nil {
		log.Println(err)
		return
	}
	err = os.Mkdir(filepath.Join(filePath, "assets"), 0755)
	if err != nil {
		log.Println(err)
		return
	}

	for i, image := range images {
		content = strings.Replace(content, image, fmt.Sprint("![](assets/", i, ".png)"), 1)
		go func(i int, image string) {
			image = matchReplace.ReplaceAllString(image, "")

			resp, err := http.Get(image)
			if err != nil {
				log.Println(err)
				return
			}
			defer resp.Body.Close()

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Println(err)
				return
			}

			err = ioutil.WriteFile(filepath.Join(filePath, "assets", fmt.Sprint(i, ".png")), body, 0644)
			if err != nil {
				log.Println(err)
				return
			}
		}(i, image)
	}

	err = ioutil.WriteFile(filepath.Join(filePath, "info.json"), []byte("[object Object]"), 0644)
	if err != nil {
		log.Println(err)
		return
	}

	err = ioutil.WriteFile(filepath.Join(filePath, "text.markdown"), []byte(content), 0644)
	if err != nil {
		log.Println(err)
		return
	}

	result, err := json.Marshal(struct {
		Status int `json:"status"`
	}{Status: 200})
	if err != nil {
		log.Println(err)
		return
	}

	_, err = w.Write(result)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("save textbundle:", title)
}

func proxyHandle(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	resp, err := http.Get(url)
	if err != nil {
		log.Println("proxy error:", err)
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
		log.Println("proxy error:", err)
		return
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
