package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"gopkg.in/yaml.v2"
)

type AppValueCtxKey string

type Api struct {
	R *mux.Router
}

type ApiDataCtx struct {
	PackagePath string
	LogPath     string
	ScriptPath  string
	TokenFile   string
}

type DeployPackage struct {
	Script  string `yaml:"script"`
	Package string `yaml:"package"`
	Skip    bool   `yaml:"skip"`
}
type DeployYAML struct {
	Namespace string                   `yaml:"namespace"`
	AuthToken string                   `yaml:"authToken"`
	Deploy    map[string]DeployPackage `yaml:"deploy"`
}

var apiDataCtxKey AppValueCtxKey = "apiDataCtxKey"
var yamlHttpHeader = "text/x-yaml"
var fallbackDeploymentNamespace = "namespace"
var deployMixInterval = time.Minute * 2

type deployTaskPoolItem struct {
	TaskHash string
	Time     string // unix second
}

var deployTaskPool = sync.Map{}

func (api *Api) init(listen string, apiData *ApiDataCtx) *http.Server {
	api.R = mux.NewRouter()
	srv := &http.Server{
		Handler:      api.R,
		Addr:         listen,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	r := api.R.PathPrefix("/api/v1").Subrouter()
	r.Use(mwApiDataCtxFactor(apiData))
	r.HandleFunc("/deploy", api.deploy).Methods(http.MethodPost)
	r.HandleFunc("/status/{namespaceSession}", api.getInfo).Methods(http.MethodGet)
	return srv

}

func mwApiDataCtxFactor(apiDataCtx *ApiDataCtx) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), apiDataCtxKey, apiDataCtx)
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}

func (*Api) deploy(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	if contentType != yamlHttpHeader {
		httpError(w, 500, "Requires Content-Type to be "+yamlHttpHeader, errors.New(""))
		return
	}

	newSessionID := time.Now().UnixNano()
	ctx := apiHelperGetApiDataCtx(r)
	yamlObj, err := Deploy{}.parseYaml(r.Body)
	defer r.Body.Close()

	if err != nil || yamlObj.Deploy == nil {
		sample := DeployYAML{}
		sample.Deploy = make(map[string]DeployPackage)
		//sample.Namespace = "prod"
		sample.Deploy["front-end-app"] = DeployPackage{
			Package: "front-ui.tar",
			Skip:    false,
			Script:  "deploy-ui.sh",
		}
		sample.Deploy["back-end-app"] = DeployPackage{
			Package: "go-api-backend.tar",
			Skip:    false,
			Script:  "deploy-backend.sh",
		}
		sampleStr, _ := yaml.Marshal(&sample)
		logInfo.Printf("yaml output: %s\n", string(sampleStr))
		httpError(w, 500, fmt.Sprintf("Parse deployment yaml Error.\n sample yaml: \n%s\n", string(sampleStr)), err)
		return
	}
	// prevent same deploy trigger using yaml:
	// won't be err, object parse back
	yamlBytes, _ := yaml.Marshal(yamlObj)
	hashOfYaml := base64.StdEncoding.EncodeToString(getBinaryHash(yamlBytes))
	if result, ok := deployTaskPool.Load(hashOfYaml); ok {
		timeStr := result.(string)
		createdtime, _ := time.Parse(time.RFC3339, string(timeStr))
		if createdtime.Sub(time.Now()) < deployMixInterval {
			httpError(w, 500, "Same deployment has already started", nil)
			return
		}
	}
	deployTaskPool.Store(hashOfYaml, time.Now().Format(time.RFC3339))
	// delete after minInterval
	ticker := time.NewTicker(deployMixInterval + 2*time.Minute)
	go func(key string) {
		<-ticker.C
		deployTaskPool.Delete(key)
	}(hashOfYaml)

	go Deploy{}.excTask(newSessionID, yamlObj, ctx)

	namespace := yamlObj.Namespace
	if len(yamlObj.Namespace) == 0 {
		namespace = fallbackDeploymentNamespace
	}
	io.WriteString(w, fmt.Sprintf("%s-%d", namespace, newSessionID))
}

func getBinaryHash(bytes []byte) []byte {
	hasher := sha256.New()
	hasher.Write(bytes)
	return hasher.Sum(nil)
}

func (*Api) getInfo(w http.ResponseWriter, r *http.Request) {
	routerVars := mux.Vars(r)
	namespaceSession := routerVars["namespaceSession"]
	r.URL.Query().Get("")
	ctx := apiHelperGetApiDataCtx(r)
	splitter := strings.LastIndex(namespaceSession, "-")
	var namespace, sessionId string

	if splitter == -1 {
		namespace = namespaceSession
		sessionId = ""
	} else {
		namespace = namespaceSession[0:splitter]
		sessionId = namespaceSession[splitter+1:]
	}

	if sessionId == "" {
		entries, err := fileUtl{}.listLogFile(ctx, namespace)
		if err != nil {
			httpError(w, 500, "Can not read status", err)
			return
		}
		for _, entry := range entries {
			io.WriteString(w, fmt.Sprintf("%s \t %s\n", entry.Name, entry.Created))
		}
		return
	}

	fcontent, err := fileUtl{}.readLogFile(ctx, namespace, sessionId)
	if err != nil {
		httpError(w, 500, "Can not read status", err)
		return
	}
	w.Header().Add("Content-Type", "text/plain")
	w.Write(fcontent)
}

func apiHelperGetApiDataCtx(r *http.Request) *ApiDataCtx {
	return r.Context().Value(apiDataCtxKey).(*ApiDataCtx)
}

func httpError(w http.ResponseWriter, code int, info string, err error) {
	var errDetails string
	if err != nil {
		errDetails = err.Error()
	}
	logErr.Printf("%s DetailError:%s\n", info, errDetails)
	w.WriteHeader(code)
	w.Write([]byte(info))
}
