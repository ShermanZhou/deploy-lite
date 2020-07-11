package main

import (
	"flag"
	"log"
	"os"
)

var logInfo = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.LUTC)
var logErr = log.New(os.Stderr, "Error: ", log.Ldate|log.Ltime|log.LUTC)

func main() {
	var listen = flag.String("listen", "localhost:8080", "api listen on host:port")
	var packagePath = flag.String("package-path", "deployment_packages", "deployment package pickup folder")
	var scriptPath = flag.String("script-path", "deployment_scripts", "deployment script path, put your script under deployment_scripts/some_name_space/")
	var logPath = flag.String("log-path", "deployment_scripts/log", "where deployment script logs are written and sent to app")
	var tokenFile = flag.String("tokenFile", "token.txt", "tokens one per line, for authentication")
	flag.Parse()

	var err error
	err = ensurePathExists(*scriptPath)
	if err != nil {
		logErr.Fatalln(err)
	}
	err = ensurePathExists(*packagePath)
	if err != nil {
		logErr.Fatalln(err)
	}
	err = ensurePathExists(*logPath)
	if err != nil {
		logErr.Fatalln(err)
	}

	api := Api{}
	apiData := ApiDataCtx{
		PackagePath: *packagePath,
		ScriptPath:  *scriptPath,
		LogPath:     *logPath,
		TokenFile:   *tokenFile,
	}
	srv := api.init(*listen, &apiData)
	logInfo.Printf("listening on %s\n", *listen)
	logErr.Fatalln(srv.ListenAndServe())
}

func ensurePathExists(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.MkdirAll(path, 0770)
	}
	return nil
}
