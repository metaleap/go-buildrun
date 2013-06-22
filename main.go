package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/go-forks/fsnotify"
)

func trimLines(str string, maxLines int) string {
	return str
	var lines = strings.Split(str, "\n")
	if len(lines) > maxLines {
		lines = lines[0:maxLines]
	}
	return strings.Join(lines, "\n")
}

var (
	origDirPath, origFilePath, goPath, goInstPath string
	rawBytes                                      []byte

	instSucceeded = true
)

func runBuiltProgram() {
	cmdRunPath := filepath.Join(goPath, "bin", goInstPath[strings.LastIndex(goInstPath, "/")+1:])
	log.Printf("[RUN]\t%s\t\t(in %s)\n%s\n\n", cmdRunPath, origDirPath, strings.Repeat("_", 28+len(cmdRunPath)))
	var (
		watch *fsnotify.Watcher
		err   error
	)
	if watch, err = fsnotify.NewWatcher(); err == nil {
		defer watch.Close()
		if err = watch.Watch(origDirPath); err == nil {
			for impDirPath, _ := range pkgImpDirPaths {
				if err = watch.Watch(impDirPath); err != nil {
					break
				}
			}
			if err == nil {
				cmd := exec.Command(cmdRunPath)
				cmd.Dir = origDirPath
				cmd.Stderr = os.Stderr
				cmd.Stdout = os.Stdout
				cmd.Stdin = os.Stdin
				go func() {
					for {
						select {
						case evt := <-watch.Event:
							if evt != nil && strings.HasSuffix(strings.ToLower(evt.Name), ".go") && cmd.Process != nil {
								log.Printf("[WATCH]\tchange: %s", evt.Name)
								cmd.Process.Kill()
								break
							}
						case err = <-watch.Error:
							if err != nil {
								log.Printf("[WATCH]\terror: %s", err.Error())
								break
							}
						}
					}
				}()
				err = cmd.Run()
			}
		}
	}
	if err != nil {
		log.Printf("[ERROR]\t%+v\n", err)
	}
}

func runGoDoc(fileName string) {
	docTemplate := `<html>
	<head>
		<title>Package %s</title>
		<meta charset="UTF-8" />
		<meta http-equiv="Content-Type" content="text/html; charset=utf-8" />
		<link type="text/css" rel="stylesheet" href="http://golang.org/doc/style.css" />
		<script type="text/javascript" src="https://ajax.googleapis.com/ajax/libs/jquery/1.8.2/jquery.min.js"></script>
		<script type="text/javascript" src="http://golang.org/doc/godocs.js"></script>
	</head>
	<body>
		<div id="page" class="wide">
		<div class="container">
		<h1>Package %s</h1>
		<div id="nav"></div>
		%s
		</div></div>
	</body>
</html>`
	log.Printf("[RUN]\tgodoc -html=true %s\n", goInstPath)
	// if rawBytes, err = ioutil.ReadFile(filepath.Join(goPath, "src/github.com/metaleap/go-buildrun/doctemplate.html")); (err == nil) && (len(rawBytes) > 0) {
	// 	docTemplate = string(rawBytes)
	// }
	var err error
	if rawBytes, err = exec.Command("godoc", "-html=true", goInstPath).CombinedOutput(); err != nil {
		log.Printf("[GODOC]\terror: %v\n", err)
	} else if err = ioutil.WriteFile(filepath.Join(origDirPath, fileName), []byte(fmt.Sprintf(docTemplate, goInstPath, goInstPath, string(rawBytes))), os.ModePerm); err != nil {
		log.Printf("[DOC]\tfile write error: %v\n", err)
	}
}

func runGoInstall() {
	var err error
	log.Printf("[RUN]\tgo install %s\n", goInstPath)
	rawBytes, err = exec.Command("go", "install", goInstPath).CombinedOutput()
	if len(rawBytes) > 0 {
		instSucceeded = false
		fmt.Printf("%s\n", trimLines(string(rawBytes), 5))
	}
	if err != nil {
		instSucceeded = false
		fmt.Printf("%+v\n", err)
	}
}

func runGoVet() {
	var err error
	log.Printf("[RUN]\tgo vet %s", goInstPath)
	if rawBytes, err = exec.Command("go", "vet", goInstPath).CombinedOutput(); err != nil {
		log.Printf("[GOVET]\terror: %v\n", err)
	} else if len(rawBytes) > 0 {
		fmt.Printf("%s\n", string(rawBytes))
	}
}

func runPrebuildCommands() {
	var (
		err      error
		dirFiles []os.FileInfo
		cmdArgs  []string
	)
	for dp := origDirPath; len(dp) > len(goPath); dp = filepath.Dir(dp) {
		if dirFiles, err = ioutil.ReadDir(dp); err != nil {
			panic(err)
		}
		for _, fi := range dirFiles {
			if (!fi.IsDir()) && (strings.HasSuffix(fi.Name(), ".go-prebuild") || strings.HasSuffix(fi.Name(), ".go-buildrun")) {
				if rawBytes, err = ioutil.ReadFile(filepath.Join(dp, fi.Name())); err != nil {
					panic(err)
				}
				for _, cmdRunPath := range strings.Split(string(rawBytes), "\n") {
					if cmdRunPath = os.ExpandEnv(strings.Replace(strings.Trim(cmdRunPath, " \t\r\n"), "$dir", dp, -1)); (len(cmdRunPath) > 0) && !(strings.HasPrefix(cmdRunPath, "#") || strings.HasPrefix(cmdRunPath, "//")) {
						log.Printf("[RUN]\t%s\n", cmdRunPath)
						cmdArgs = strings.Split(cmdRunPath, " ")
						if cmdArgs[0] == "start" && runtime.GOOS == "windows" {
							cmdArgs = append([]string{"cmd", "/C"}, cmdArgs...)
						}
						rawBytes, err = exec.Command(cmdArgs[0], cmdArgs[1:]...).CombinedOutput()
						if len(rawBytes) > 0 {
							fmt.Printf("%s", string(rawBytes))
						}
						if err != nil {
							log.Printf("[ERR]\t%v", err)
						}
					}
				}
			}
		}
	}
}

func main() {
	var (
		startTime      = time.Now()
		flagFilePath   = flag.String("f", "", "Full path to current .go source file from which to build (go install) package.")
		flagGenDocHtml = flag.String("d", "", "Specify a file name such as doc.html to generate single-page package-doc in package directory; omit to not generate this.")
		flagVet        = flag.Bool("v", false, "run go vet?")
	)
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()
	origFilePath = *flagFilePath
	origDirPath = filepath.Dir(origFilePath)
	goPath = os.ExpandEnv("$GOPATH")
	if pos := strings.Index(goPath, string(os.PathListSeparator)); pos > 0 {
		goPath = goPath[:pos]
		fmt.Printf("Your GOPATH contains multiple paths, using %v.\n", goPath)
	}
	collectImports(origDirPath)
	isMainPkg, hasDocGoFile := checkForMainPackage(origFilePath), processTemplates(origDirPath)
	pathSep := string(os.PathSeparator)
	goInstPath = strings.Replace(origFilePath[len(filepath.Join(goPath, "src")+pathSep):], pathSep, "/", -1)
	goInstPath = goInstPath[0:strings.LastIndex(goInstPath, "/")]
	//	STEP 1. run pre-build commands
	runPrebuildCommands()
	//	STEP 2. go install
	runGoInstall()
	//	STEP 3. go doc
	if instSucceeded && (hasDocGoFile || strings.Contains(goInstPath, "metaleap/go-xsd-pkg")) && (!isMainPkg) && (len(*flagGenDocHtml) > 0) {
		runGoDoc(*flagGenDocHtml)
	}
	//	STEP 4. go vet
	if *flagVet {
		runGoVet()
	}
	log.Printf("TOTAL BUILD TIME: %v\n", time.Now().Sub(startTime))
	//	STEP 5. "go run"
	if instSucceeded && isMainPkg {
		runBuiltProgram()
	}
}
