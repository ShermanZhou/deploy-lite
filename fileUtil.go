package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type fileUtl struct {
}
type fileEntry struct {
	Name    string
	Created time.Time
}

type fileEntries []fileEntry

func (t fileEntries) Len() int {
	return len(t)
}
func (t fileEntries) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}
func (t fileEntries) Less(i, j int) bool {
	return t[i].Created.Before(t[j].Created)
}

var tokenFileChan = make(chan bool, 1)

func (fu fileUtl) readLogFile(ctx *ApiDataCtx, namespace string, session string) ([]byte, error) {
	logPath := ctx.LogPath
	if filepath.IsAbs(logPath) {
		logPath = filepath.Join(fu.currentExecutablePath(), logPath)
	}
	f := filepath.Join(logPath, fmt.Sprintf("%s-%s.log", namespace, session))
	if _, err := os.Stat(f); os.IsNotExist(err) {
		return nil, err
	}
	return ioutil.ReadFile(f)
}

func (fu fileUtl) readTokenFile(tokenFile string) ([]string, error) {
	if !filepath.IsAbs(tokenFile) {
		tokenFile = filepath.Join(fu.currentExecutablePath(), tokenFile)
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

func (fu fileUtl) currentExecutablePath() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		logErr.Panicln(err.Error())
	}
	return dir
}

func (fu fileUtl) listLogFile(ctx *ApiDataCtx, namespace string) (fileEntries, error) {
	logPath := ctx.LogPath
	if !filepath.IsAbs(logPath) {
		logPath = filepath.Join(fu.currentExecutablePath(), logPath)
	}
	dir, err := os.Open(logPath)
	if err != nil {
		return nil, err
	}
	fileInfos, err := dir.Readdir(0)
	if err != nil {
		return nil, err
	}
	entries := fileEntries{}
	pattern, _ := regexp.Compile(fmt.Sprintf("^%s-\\w+\\.log$", namespace))
	for _, info := range fileInfos {
		if info.IsDir() {
			continue
		}
		name := info.Name()
		if !pattern.MatchString(name) {
			continue
		}
		created := info.ModTime().UTC().Round(time.Second)
		entries = append(entries, fileEntry{
			Name:    strings.TrimSuffix(name, filepath.Ext(name)),
			Created: created,
		})
	}
	sort.Sort(sort.Reverse(entries))
	return entries, nil
}
