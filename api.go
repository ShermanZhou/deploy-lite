package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	_ "sync/atomic"
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
var sessionID int64 = 10000
var yamlHttpHeader = "text/x-yaml"
var fallbackDeploymentNamespace = "namespace"
var tokenFileChan = make(chan bool, 1)

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
	r.HandleFunc("/status", api.getInfo).Methods(http.MethodGet)

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

	newSessionID := atomic.AddInt64(&sessionID, 1)
	ctx := apiHelperGetApiDataCtx(r)
	yamlObj, err := parseDeployYaml(r.Body)
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
	go executeTask(newSessionID, yamlObj, ctx)
	w.Write([]byte("OK"))
}

func (*Api) getInfo(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
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
func currentExecutablePath() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		logErr.Panicln(err.Error())
	}
	return dir
}
func parseDeployYaml(reader io.Reader) (DeployYAML, error) {
	var err error
	dec := yaml.NewDecoder(reader)
	dep := DeployYAML{}
	err = dec.Decode(&dep)
	return dep, err
}

func executeTask(sessionID int64, yamlObject DeployYAML, ctx *ApiDataCtx) {
	var namespace = yamlObject.Namespace
	if len(namespace) == 0 {
		namespace = fallbackDeploymentNamespace
	}
	logName := fmt.Sprintf("%s-%s.log", namespace, strconv.Itoa(int(sessionID)))
	logFile, err := os.Create(path.Join(ctx.LogPath, logName))
	defer logFile.Close()

	taskInfoLog := log.New(logFile, "INFO: ", log.Ldate|log.Ltime|log.LUTC)
	taskErrLog := log.New(logFile, "ERROR: ", log.Ldate|log.Ltime|log.LUTC)

	if err != nil {
		taskErrLog.Printf("Create log file %s \n", err.Error())
		return
	}

	tokens, err := readTokenFile(ctx.TokenFile)
	if err != nil {
		taskErrLog.Printf("Read token file failed %s \n", err.Error())
		return
	}
	if len(tokens) > 0 {
		tst := yamlObject.AuthToken
		var found bool
		for _, t := range tokens {
			if t == tst {
				found = true
				break
			}
		}
		if !found {
			taskErrLog.Println("Token authentication failed. Quit all tasks.")
			return
		} else {
			taskInfoLog.Println("Token OK, will start tasks.")
		}
	}

	for k, pkg := range yamlObject.Deploy {
		if pkg.Skip {
			taskInfoLog.Printf("Skip task %s \n", k)
			continue
		}
		// todo: namespace validation and authorization
		taskInfoLog.Printf("Start deploy task in yaml: %s \n", k)
		tarPath := path.Join(ctx.PackagePath, pkg.Package)
		if !filepath.IsAbs(tarPath) {
			tarPath = filepath.Join(currentExecutablePath(), tarPath)
		}
		tardir := filepath.Dir(tarPath)
		taskInfoLog.Printf("Step 1: \nStart tar decompress: %s \n", tarPath)
		cmd := exec.Command("tar", "-xf", tarPath, "-C", tardir)
		cmd.Stderr = logFile
		cmd.Stdout = logFile
		err = cmd.Run()
		if err != nil {
			taskErrLog.Printf("tar decompress err %s \n", err.Error())
		} else {
			taskInfoLog.Println("Completed tar decompress.")
		}
		scriptPath := path.Join(ctx.ScriptPath, pkg.Script)
		if !filepath.IsAbs(scriptPath) {
			scriptPath = filepath.Join(currentExecutablePath(), scriptPath)
		}
		// the script is predefined and the tar packages are also predefined.
		taskInfoLog.Printf("Step 2: \nStart deployment script: %s \n", scriptPath)
		cmd = exec.Command("sh", scriptPath)
		cmd.Stderr = logFile
		cmd.Stdout = logFile
		err = cmd.Run()
		if err != nil {
			taskErrLog.Printf("Run script command err %s \n", err.Error())
		} else {
			taskInfoLog.Println("Script completed")
		}
		taskInfoLog.Printf("End of deploy task %s \n", k)
	}
}

func readTokenFile(tokenFile string) ([]string, error) {
	if !filepath.IsAbs(tokenFile) {
		tokenFile = filepath.Join(currentExecutablePath(), tokenFile)
	}

	tokenFileChan <- true
	defer func() {
		<-tokenFileChan
	}()

	content, err := ioutil.ReadFile(tokenFile)
	if err != nil {
		return nil, err
	}
	tokens := []string{}
	for _, rawline := range bytes.Split(content, []byte("\n")) {
		token := string(rawline)
		token = strings.TrimSpace(token)
		if len(token) > 0 {
			tokens = append(tokens, strings.TrimSpace(token))
		}
	}
	return tokens, nil
}
