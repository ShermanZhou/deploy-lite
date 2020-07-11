package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v2"
)

type Deploy struct {
}

func (Deploy) parseYaml(reader io.Reader) (DeployYAML, error) {
	var err error
	dec := yaml.NewDecoder(reader)
	dep := DeployYAML{}
	err = dec.Decode(&dep)
	return dep, err
}

func (Deploy) excTask(sessionID int64, yamlObject DeployYAML, ctx *ApiDataCtx) {
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

	tokens, err := fileUtl{}.readTokenFile(ctx.TokenFile)
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
			tarPath = filepath.Join(fileUtl{}.currentExecutablePath(), tarPath)
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
			scriptPath = filepath.Join(fileUtl{}.currentExecutablePath(), scriptPath)
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
