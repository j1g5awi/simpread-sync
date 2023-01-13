package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"gopkg.in/gomail.v2"
)

var Version string = "(devel)"
var (
	configFile     string
	port           int
	syncPath       string
	outputPath     string
	enhancedOutput []map[string]string
	autoRemove     bool
	smtpHost       string
	smtpPort       int
	smtpUsername   string
	smtpPassword   string
	mailTitle      string
	receiverMail   string
	kindleMail     string
	version        bool
	uid            string
)

var rootCmd = &cobra.Command{
	Use: "simpread-sync",
	PreRun: func(cmd *cobra.Command, args []string) {
		parseCustomizedFlags(cmd, args)
		parseCustomizedEnv()
		initConfig()
	},
	Run: func(cmd *cobra.Command, args []string) {
		localSync := http.NewServeMux()
		localSync.HandleFunc("/verify", verifyHandle)
		localSync.HandleFunc("/config", configHandle)
		localSync.HandleFunc("/plain", plainHandle)
		localSync.HandleFunc("/mail", mailHandle)
		localSync.HandleFunc("/convert", convertHandle)
		localSync.HandleFunc("/wkhtmltopdf", wkhtmltopdfHandle)
		localSync.HandleFunc("/reading/", readingHandle)
		localSync.HandleFunc("/proxy", proxyHandle)
		localSync.HandleFunc("/textbundle", textbundleHandle)
		localSync.HandleFunc("/notextbundle", notextbundleHandle)
		go func() {
			err := http.ListenAndServe(fmt.Sprint(":", port), localSync)
			if err != nil {
				log.Fatal(err)
			}
		}()

		API := http.NewServeMux()
		API.HandleFunc("/add", APIaddHandle)
		API.HandleFunc("/adds", APIaddsHandle)
		API.HandleFunc("/new", APIaddHandle)
		API.HandleFunc("/webhook", APIaddHandle)
		API.HandleFunc("/reading/", APIreadingHandle)
		API.HandleFunc("/list", APIlistHandle)
		go func() {
			err := http.ListenAndServe(fmt.Sprint(":", 7027), API)
			if err != nil {
				log.Fatal(err)
			}
		}()

		for {
		}
	},
	DisableFlagParsing: true,
}

func parseCustomizedFlags(cmd *cobra.Command, args []string) {
	for i := 0; i < len(args); i++ {
		s := args[i]
		if len(s) > 2 && s[:2] == "--" {
			a := args[i+1:]
			name := s[2:]
			if len(name) == 0 || name[0] == '-' || name[0] == '=' {
				continue
			}
			split := strings.SplitN(name, "=", 2)
			name = split[0]
			if strings.HasSuffix(name, "-path") && name != "sync-path" && name != "output-path" {
				var value string
				if len(split) == 2 {
					value = split[1]
					args = append(args[:i], args[i+1:]...)
					i -= 1
				} else if len(a) > 0 {
					value = a[0]
					args = append(args[:i], args[i+2:]...)
					i -= 2
				}
				enhancedOutput = append(enhancedOutput, map[string]string{
					"extension": strings.Replace(name, "-path", "", 1),
					"path":      value})
			}
		}
	}
	cmd.DisableFlagParsing = false
	err := cmd.ParseFlags(args)
	if err != nil {
		fmt.Println(err)
		cmd.Help()
		fmt.Println()
		os.Exit(0)
	}

	if cmd.Flag("help").Value.String() == "true" {
		cmd.Help()
		fmt.Println()
		os.Exit(0)
	}

	if cmd.Flag("version").Value.String() == "true" {
		checkVersion()
	}
}

func parseCustomizedEnv() {
	for _, env := range os.Environ() {
		split := strings.SplitN(env, "=", 2)
		name := split[0]
		value := split[1]
		if strings.HasPrefix(name, "OUTPUT_PATH_") {
			enhancedOutput = append(enhancedOutput, map[string]string{
				"extension": strings.ToLower(strings.Replace(name, "OUTPUT_PATH_", "", 1)),
				"path":      value})
		}
	}
}

func init() {
	rootCmd.Flags().StringVarP(&configFile, "config", "c", "", "config file")
	rootCmd.Flags().IntVarP(&port, "port", "p", 7026, "port")
	rootCmd.Flags().StringVar(&syncPath, "sync-path", "", "sync path")
	rootCmd.Flags().StringVar(&outputPath, "output-path", "", "output path")
	rootCmd.Flags().BoolVar(&autoRemove, "auto-remove", false, "auto remove")
	rootCmd.Flags().StringVar(&smtpHost, "smtp-host", "", "smtp host")
	rootCmd.Flags().IntVar(&smtpPort, "smtp-port", 465, "smtp port")
	rootCmd.Flags().StringVar(&smtpUsername, "smtp-username", "", "smtp username")
	rootCmd.Flags().StringVar(&smtpPassword, "smtp-password", "", "smtp password")
	rootCmd.Flags().StringVar(&mailTitle, "mail-title", "[简悦] - {{title}}", "mail title")
	rootCmd.Flags().StringVar(&receiverMail, "receiver-mail", "", "receiver mail")
	rootCmd.Flags().StringVar(&kindleMail, "kindle-mail", "", "kindle mail")
	rootCmd.Flags().BoolVarP(&version, "version", "V", false, "check version")
	rootCmd.Flags().StringVarP(&uid, "uid", "u", "", "user id")

	viper.BindPFlag("port", rootCmd.Flags().Lookup("port"))
	viper.BindPFlag("syncPath", rootCmd.Flags().Lookup("sync-path"))
	viper.BindPFlag("outputPath", rootCmd.Flags().Lookup("output-path"))
	viper.BindPFlag("autoRemove", rootCmd.Flags().Lookup("auto-remove"))
	viper.BindPFlag("smtpHost", rootCmd.Flags().Lookup("smtp-host"))
	viper.BindPFlag("smtpPort", rootCmd.Flags().Lookup("smtp-port"))
	viper.BindPFlag("smtpUsername", rootCmd.Flags().Lookup("smtp-username"))
	viper.BindPFlag("smtpPassword", rootCmd.Flags().Lookup("smtp-password"))
	viper.BindPFlag("mailTitle", rootCmd.Flags().Lookup("mail-title"))
	viper.BindPFlag("receiverMail", rootCmd.Flags().Lookup("receiver-mail"))
	viper.BindPFlag("kindleMail", rootCmd.Flags().Lookup("kindle-mail"))
	viper.BindPFlag("uid", rootCmd.Flags().Lookup("uid"))

	viper.BindEnv("port", "LISTEN_PORT")
	viper.BindEnv("syncPath", "SYNC_PATH")
	viper.BindEnv("outputPath", "OUTPUT_PATH")
	viper.BindEnv("autoRemove", "AUTO_REMOVE")
	viper.BindEnv("smtpHost", "SMTP_HOST")
	viper.BindEnv("smtpPort", "SMTP_PORT")
	viper.BindEnv("smtpUsername", "SMTP_USERNAME")
	viper.BindEnv("smtpPassword", "SMTP_PASSWORD")
	viper.BindEnv("mailTitle", "MAIL_TITLE")
	viper.BindEnv("receiverMail", "MAIL_RECEIVER")
	viper.BindEnv("kindleMail", "MAIL_KINDLE")
	viper.BindEnv("uid", "UID")
}

func checkVersion() {
	log.Println("当前版本：", Version)
	if Version == "(devel)" {
		os.Exit(0)
	}
	resp, err := http.Get("https://api.github.com/repos/j1g5awi/simpread-sync/releases/latest")
	if err != nil {
		log.Fatal("检查更新失败：", err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
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

func initConfig() {
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
	enhancedOutputInterface := viper.Get("enhancedOutput")
	if enhancedOutputInterface, ok := enhancedOutputInterface.([]interface{}); ok {
		for _, i := range enhancedOutputInterface {
			tmpMap := map[string]string{}
			for k, v := range i.(map[string]interface{}) {
				tmpMap[k] = v.(string)
			}
			enhancedOutput = append(enhancedOutput, tmpMap)
		}
	}
	autoRemove = viper.GetBool("autoRemove")
	smtpHost = viper.GetString("smtpHost")
	smtpPort = viper.GetInt("smtpPort")
	smtpUsername = viper.GetString("smtpUsername")
	smtpPassword = viper.GetString("smtpPassword")
	mailTitle = viper.GetString("mailTitle")
	receiverMail = viper.GetString("receiverMail")
	kindleMail = viper.GetString("kindleMail")
	uid = viper.GetString("uid")

	if syncPath == "" {
		log.Fatal("未读取到 syncPath！")
	}
	if outputPath == "" {
		outputPath = filepath.Join(syncPath, "output")
	}
	os.MkdirAll(outputPath, 0755)

	config, err := os.ReadFile(filepath.Join(syncPath, "simpread_config.json"))
	if err != nil {
		log.Println(err)
		return
	}
	unrdist = make(map[int]struct{}, len(gjson.GetBytes(config, "unrdist").Array()))
	for _, unrd := range gjson.GetBytes(config, "unrdist").Array() {
		unrdist[int(unrd.Get("idx").Int())] = struct{}{}
	}
}

// 本地未存储 uid 返回 {"code": 201}
// 本地 uid 与 header 中的 uid 一致 json 返回 {"code":403,"status":"same"}
// 本地 uid 与 header 中的 uid 不一致 json 返回 {"code":403,"status":"uid"}
func verifyHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	err := r.ParseForm()
	if err != nil {
		log.Println(err)
		return
	}
	var result []byte
	if uid != "" && r.Header.Get("uid") == uid {
		result, err = json.Marshal(struct {
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
	} else if uid != "" {
		result, err = json.Marshal(struct {
			Code   int    `json:"code"`
			Status string `json:"status"`
		}{
			Code:   403,
			Status: "uid",
		})
		if err != nil {
			log.Println(err)
			return
		}
	} else if r.Header.Get("uid") != "" {
		uid = r.Header.Get("uid")
		result, err = json.Marshal(struct {
			Code int `json:"code"`
		}{
			Code: 201,
		})
		if err != nil {
			log.Println(err)
			return
		}
		log.Println("verify success")
	}
	_, err = w.Write(result)
	if err != nil {
		log.Println(err)
		return
	}
}

func checkUid(w http.ResponseWriter, r *http.Request) error {
	if uid != "" && r.Header.Get("uid") == uid {
		return nil
	}
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	result, err := json.Marshal(struct {
		Code   int    `json:"code"`
		Status string `json:"status"`
	}{
		Code:   401,
		Status: "uid",
	})
	if err != nil {
		log.Println(err)
		return err
	}
	_, err = w.Write(result)
	if err != nil {
		log.Println(err)
		return err
	}
	return errors.New("uid error")
}

func getOutputPaths(extension string) []string {
	return getOutputPathsWithPath(extension, outputPath)
}

func getOutputPathsWithPath(extension, path string) []string {
	outputPaths := []string{}
	for _, i := range enhancedOutput {
		if extension == i["extension"] {
			path := i["path"]
			if path == "" {
				path = filepath.Join(outputPath, extension)
			}
			os.MkdirAll(path, 0755)
			outputPaths = append(outputPaths, path)
		}
	}
	if len(outputPaths) == 0 {
		outputPaths = append(outputPaths, path)
	}
	return outputPaths
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
// 本来还有个检测 syncPath 是否配置，但命令行启动就检测过了
// 请求压根没带 uid（因为 config 里有？）
func configHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	if syncPath != "" {
		err := myParseForm(r)
		if err != nil {
			log.Println(err)
			return
		}

		if data := r.Form.Get("config"); data != "" {
			err := os.WriteFile(filepath.Join(syncPath, "simpread_config.json"), []byte(data), 0644)
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
				fileInfo, err := os.ReadDir(outputPath)
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
			config, err := os.ReadFile(filepath.Join(syncPath, "simpread_config.json"))
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

// 校验 uid
func plainHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	if err := checkUid(w, r); err != nil {
		return
	} else {
		err := myParseForm(r)
		if err != nil {
			log.Println(err)
			return
		}

		title := r.Form.Get("title")
		content := r.Form.Get("content")

		suffix := path.Ext(title)[1:]
		if strings.HasPrefix(title, "tmp-") {
			suffix = "tmp"
		}
		for _, path := range getOutputPaths(suffix) {
			err = os.WriteFile(filepath.Join(path, title), []byte(content), 0644)
			if err != nil {
				log.Println(err)
				continue //TODO 错误处理
			}
		}

		result, err := json.Marshal(struct {
			Status int `json:"status"`
		}{Status: 200})
		if err != nil {
			log.Println(err)
			return
		}
		log.Println("save file:", title)

		_, err = w.Write(result)
		if err != nil {
			log.Println(err)
			return
		}
	}
}

// 校验 uid
func mailHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	if err := checkUid(w, r); err != nil {
		return
	} else {
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
			attachPath = filepath.Join(syncPath, fmt.Sprintf("tmp-%s.%s", title, attach))
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
}

// 校验 uid
func convertHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	if err := checkUid(w, r); err != nil {
		return
	} else {
		err := r.ParseForm()
		if err != nil {
			log.Println(err)
			return
		}
		title := r.Form.Get("title")
		content := r.Form.Get("content")
		in := r.Form.Get("in")   //md
		out := r.Form.Get("out") //epub

		tmpFilePath := filepath.Join(syncPath, fmt.Sprintf("tmp-%s.%s", title, in))
		err = os.WriteFile(tmpFilePath, []byte(content), 0644)
		if err != nil {
			log.Println(err)
			return
		}
		pandoc := "pandoc"
		if runtime.GOOS == "darwin" {
			pandoc = "/usr/local/bin/pandoc"
		}
		//TODO 并发
		for _, path := range getOutputPaths(out) {
			cmd := exec.Command(pandoc, tmpFilePath, "-o", filepath.Join(path, title+"."+out))
			err = cmd.Run()
			if err != nil {
				log.Println(err)
				continue //TODO 错误处理
			}
		}

		os.Remove(tmpFilePath)

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
}

// 不校验 uid
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

	tmpFilePath := filepath.Join(syncPath, fmt.Sprintf("tmp-%s.html", title))
	err = os.WriteFile(tmpFilePath, []byte(content), 0644)
	if err != nil {
		log.Println(err)
		return
	}

	if root == "" {
		root = "wkhtmltopdf"
	}
	// TODO 并发
	for _, path := range getOutputPaths("pdf") {
		cmd := exec.Command(root, append(params, tmpFilePath, filepath.Join(path, title+".pdf"))...)

		err = cmd.Run()
		if err != nil {
			log.Println(err)
			continue //TODO 错误处理
		}
	}

	os.Remove(tmpFilePath)

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

// 请求压根没带 uid
func readingHandle(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Println(err)
		return
	}

	var files []string
	fileInfo, err := os.ReadDir(outputPath)
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
			result, err = os.ReadFile(filepath.Join(outputPath, title))
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

// 校验 uid
func textbundleHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	if err := checkUid(w, r); err != nil {
		return
	} else {
		err := r.ParseForm()
		if err != nil {
			log.Println(err)
			return
		}
		title := r.Form.Get("title")
		content := r.Form.Get("content")
		images := matchImage.FindAllString(content, -1)
		// TODO 提升性能
		for _, path := range getOutputPaths("textbundle") {
			filePath := filepath.Join(path, title+".textbundle")

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

					body, err := io.ReadAll(resp.Body)
					if err != nil {
						log.Println(err)
						return
					}

					err = os.WriteFile(filepath.Join(filePath, "assets", fmt.Sprint(i, ".png")), body, 0644)
					if err != nil {
						log.Println(err)
						return
					}
				}(i, image)
			}

			err = os.WriteFile(filepath.Join(filePath, "info.json"), []byte(`{"transient":true,"type":"net.daringfireball.markdown","creatorIdentifier":"pro.simpread","version":2}`), 0644)
			if err != nil {
				log.Println(err)
				return
			}

			err = os.WriteFile(filepath.Join(filePath, "text.markdown"), []byte(content), 0644)
			if err != nil {
				log.Println(err)
				return
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
		log.Println("save textbundle:", title)
	}
}

// 校验 uid
// 懒得精简代码，复制粘贴
func notextbundleHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	if err := checkUid(w, r); err != nil {
		return
	} else {
		err := r.ParseForm()
		if err != nil {
			log.Println(err)
			return
		}
		title := r.Form.Get("title")
		content := r.Form.Get("content")
		path := r.Form.Get("path")
		if path == "" {
			path = outputPath
		}
		images := matchImage.FindAllString(content, -1)
		// TODO 提升性能
		for _, path := range getOutputPathsWithPath("assets", path) {
			filePath := filepath.Join(path, title)

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

					body, err := io.ReadAll(resp.Body)
					if err != nil {
						log.Println(err)
						return
					}

					err = os.WriteFile(filepath.Join(filePath, "assets", fmt.Sprint(i, ".png")), body, 0644)
					if err != nil {
						log.Println(err)
						return
					}
				}(i, image)
			}

			err = os.WriteFile(filepath.Join(filePath, fmt.Sprint(title, ".md")), []byte(content), 0644)
			if err != nil {
				log.Println(err)
				return
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
		log.Println("save notextbundle:", title)
	}
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

func APIaddHandle(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Println(err)
		return
	}
	url := r.Form.Get("url")
	title := r.Form.Get("title")
	desc := r.Form.Get("desc")
	tags := strings.Split(r.Form.Get("tags"), ",")
	note := r.Form.Get("note")

	config, err := os.ReadFile(filepath.Join(syncPath, "simpread_config.json"))
	if err != nil {
		log.Println(err)
		return
	}
	idx := int(gjson.GetBytes(config, "unrdist.#.idx|0").Int()) + 1
	unrdist[idx] = struct{}{}
	tmp := gjson.GetBytes(config, "unrdist|@reverse").String()
	tmp, err = sjson.Set(tmp, "-1", map[string]interface{}{
		"create":  time.Now().Format("2006年01月02日 15:04:05"), //2022年10月14日 19:59:58
		"desc":    desc,
		"favicon": "",
		"idx":     idx,
		"img":     "",
		"note":    note,
		"tags":    tags,
		"title":   title,
		"url":     url})
	tmp = gjson.Get(tmp, "@this|@reverse").Raw
	config, err = sjson.SetRawBytes(config, "unrdist", []byte(tmp))
	err = os.WriteFile(filepath.Join(syncPath, "simpread_config.json"), config, 0644)
	if err != nil {
		log.Println(err)
		return
	}
	result, err := json.Marshal(struct {
		Code int `json:"code"`
	}{Code: 201})
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

func APIaddsHandle(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Println(err)
		return
	}
	urls := strings.Split(r.Form.Get("urls"), ";;;")
	titles := strings.Split(r.Form.Get("titles"), ";;;")
	tags := strings.Split(r.Form.Get("tags"), ",")

	config, err := os.ReadFile(filepath.Join(syncPath, "simpread_config.json"))
	if err != nil {
		log.Println(err)
		return
	}
	idx := int(gjson.GetBytes(config, "unrdist.#.idx|0").Int()) + 1
	tmp := gjson.GetBytes(config, "unrdist|@reverse").String()
	for i, url := range urls {
		unrdist[idx] = struct{}{}
		tmp, err = sjson.Set(tmp, "-1", map[string]interface{}{
			"create":  time.Now().Format("2006年01月02日 15:04:05"), //2022年10月14日 19:59:58
			"desc":    "",
			"favicon": "",
			"idx":     idx,
			"img":     "",
			"note":    "",
			"tags":    tags,
			"title":   titles[i],
			"url":     url})
		idx += 1
	}
	tmp = gjson.Get(tmp, "@this|@reverse").Raw
	config, err = sjson.SetRawBytes(config, "unrdist", []byte(tmp))
	err = os.WriteFile(filepath.Join(syncPath, "simpread_config.json"), config, 0644)
	if err != nil {
		log.Println(err)
		return
	}
	result, err := json.Marshal(struct {
		Code int `json:"code"`
	}{Code: 201})
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

func APIreadingHandle(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Println(err)
		return
	}

	var files []map[string]string
	fileInfo, err := os.ReadDir(outputPath)
	if err != nil {
		log.Println(err)
		return
	}
	for _, file := range fileInfo {
		if !file.IsDir() {
			fileinfo, _ := file.Info()
			files = append(files, map[string]string{
				"title":  file.Name(),
				"create": fileinfo.ModTime().Format("Mon, 02 Jan 2006 15:04:05 MST")})
		}
	}

	var result []byte
	query := r.Form.Get("title")
	if query == "index" {
		w.Header().Set("content-type", "application/json")
		result, err = json.Marshal(struct {
			Data []map[string]string `json:"data"`
		}{Data: files})
		if err != nil {
			log.Println(err)
			return
		}
		log.Println("API reading index")
	} else {
		id := strings.Replace(r.URL.Path, "/reading/", "", 1)
		if err != nil {
			log.Println(err)
			return
		}
		suffix := ".html"
		var title string
		for _, file := range files {
			if (strings.HasPrefix(file["title"], id+"-") &&
				strings.HasSuffix(file["title"], suffix) &&
				!strings.Contains(file["title"], "@annote")) ||
				file["title"] == id+suffix ||
				file["title"] == query+suffix {
				title = file["title"]
				break
			}
		}

		if title != "" {
			result, err = os.ReadFile(filepath.Join(outputPath, title))
			if err != nil {
				log.Println(err)
				return
			}
			log.Println("API reading file:", title)
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

func APIlistHandle(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Println(err)
		return
	}

	filter := r.Form.Get("filter")
	value := r.Form.Get("value")
	var result []byte
	switch filter {
	case "all":
		config, err := os.ReadFile(filepath.Join(syncPath, "simpread_config.json"))
		if err != nil {
			log.Println(err)
			return
		}
		unrdist := gjson.GetBytes(config, "unrdist").Array()
		tmp := `{"data": []}`
		for i := 0; i < 20 && i < len(unrdist); i++ {
			tmp, _ = sjson.SetRaw(tmp, "data.-1", unrdist[i].Raw)
		}
		result = []byte(tmp)
		w.Header().Set("content-type", "application/json")
	case "daily":
		config, err := os.ReadFile(filepath.Join(syncPath, "simpread_config.json"))
		if err != nil {
			log.Println(err)
			return
		}
		unrdist := gjson.GetBytes(config, "unrdist").Array()
		now := time.Now()
		tmp := `{"data": []}`
		for _, unrd := range unrdist {
			create, _ := time.Parse("2006年01月02日 15:04:05", unrd.Get("create").String())
			if create.Year() == now.Year() && create.Month() == now.Month() &&
				create.Day() == now.Day() {
				tmp, _ = sjson.SetRaw(tmp, "data.-1", unrd.Raw)
			}
		}
		result = []byte(tmp)
		w.Header().Set("content-type", "application/json")
	case "dr":
		config, err := os.ReadFile(filepath.Join(syncPath, "simpread_config.json"))
		if err != nil {
			log.Println(err)
			return
		}
		tmp := `{"data": []}`
		for _, unrd := range gjson.GetBytes(config, `unrdist.#(tags.#(=="dr"))`).Array() {
			tmp, _ = sjson.SetRaw(tmp, "data.-1", unrd.Raw)
		}
		result = []byte(tmp)
		w.Header().Set("content-type", "application/json")
	case "reading":
		var files []map[string]string
		fileInfo, err := os.ReadDir(outputPath)
		if err != nil {
			log.Println(err)
			return
		}
		for _, file := range fileInfo {
			if !file.IsDir() {
				fileinfo, _ := file.Info()
				files = append(files, map[string]string{
					"title":  file.Name(),
					"create": fileinfo.ModTime().Format("Mon, 02 Jan 2006 15:04:05 MST")})
			}
		}
		w.Header().Set("content-type", "application/json")
		result, err = json.Marshal(struct {
			Data []map[string]string `json:"data"`
		}{Data: files})
		if err != nil {
			log.Println(err)
			return
		}
		log.Println("API reading index")
	case "tag":
		config, err := os.ReadFile(filepath.Join(syncPath, "simpread_config.json"))
		if err != nil {
			log.Println(err)
			return
		}
		tmp := `{"data": []}`
		for _, unrd := range gjson.GetBytes(config, fmt.Sprintf(`unrdist.#(tags.#(=="%s"))`, value)).
			Array() {
			tmp, _ = sjson.SetRaw(tmp, "data.-1", unrd.Raw)
		}
		result = []byte(tmp)
		w.Header().Set("content-type", "application/json")
	case "search":
		config, err := os.ReadFile(filepath.Join(syncPath, "simpread_config.json"))
		if err != nil {
			log.Println(err)
			return
		}
		unrdist := gjson.GetBytes(config, "unrdist").Array()
		tmp := `{"data": []}`
		for _, unrd := range unrdist {
			if strings.Contains(unrd.Get("title").String(), value) ||
				strings.Contains(unrd.Get("desc").String(), value) ||
				strings.Contains(unrd.Get("note").String(), value) {
				tmp, _ = sjson.SetRaw(tmp, "data.-1", unrd.Raw)
			}
		}
		result = []byte(tmp)
	}
	_, err = w.Write(result)
	if err != nil {
		log.Println(err)
		return
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
