package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Repo struct {
	ssh   string
	https string
}

var repoMap = map[string]Repo{
	"knative/eventing": {
		ssh: "git@github.com:knative/eventing.git",
		https: "https://github.com/knative/eventing.git",
	},
	"knative/serving": {
		ssh: "git@github.com:knative/serving.git",
		https: "https://github.com/knative/serving.git",
	},
	"cloudevents/sdk-go": {
		ssh: "git@github.com:cloudevents/sdk-go.git",
		https: "https://github.com/cloudevents/sdk-go.git",
	},
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /repos/{org}/{repoName}", func(w http.ResponseWriter, r *http.Request) {
		org := r.PathValue("org")
		repoName := r.PathValue("repoName")
		scheme := r.URL.Query().Get("scheme")

		fmt.Printf("received request with org=%s, repoName=%s, scheme=%s\n", org, repoName, scheme)
		w.Header().Set("Content-Type", "application/json")

		if org == "" {
			data := map[string]string{
				"reason": "missing org in request path /repos/{org}/{repoName}",
			}

			jData, _ := json.Marshal(data)
			w.WriteHeader(http.StatusBadRequest)
			w.Write(jData)
		}
		if repoName == "" {
			data := map[string]string{
				"reason": "missing repoName in request path /repos/{org}/{repoName}",
			}

			jData, _ := json.Marshal(data)
			w.WriteHeader(http.StatusBadRequest)
			w.Write(jData)
		}

		repo, ok := repoMap[fmt.Sprintf("%s/%s", org, repoName)]
		if !ok {
			data := map[string]string{
				"reason": "no matching repo found",
			}

			jData, _ := json.Marshal(data)
			w.WriteHeader(http.StatusNotFound)
			w.Write(jData)
		}

		repoUrl := repo.ssh
		if scheme == "https" {
			repoUrl = repo.https
		}

		data := map[string]string{
			"url": repoUrl,
		}

		jData, _ := json.Marshal(data)
		w.Write(jData)
	})

	http.ListenAndServe(":9090", mux)
}
