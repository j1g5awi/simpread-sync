package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

//未验证 json 返回 403
//已验证 json 返回 201
func verifyHandle(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
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

// 如果浏览器插件的设置项更改了，它会发一个 key 为 config 的 urlencoded 请求，json 返回 200
// 如果本地文件更改了，返回一个 key 为 result 的 json
func configHandle(w http.ResponseWriter, r *http.Request, path string) {
	w.Header().Set("content-type", "application/json")

	r.ParseForm()
	if config := r.Form.Get("config"); config != "" {
		err := ioutil.WriteFile(path, []byte(config), 644)
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
		config, err := ioutil.ReadFile(path)
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
}

func main() {
	path := os.Args[1]
	if path == "" {
		path = "simpread_config.json"
	}
	log.Println("config file path:", path)

	http.HandleFunc("/verify", verifyHandle)
	http.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) { configHandle(w, r, path) })

	err := http.ListenAndServe(":7026", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
