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
)

func checkForMainPackage(filePath string) bool {
	var err error
	var rawBytes []byte
	var tmp string
	if strings.Index(filePath, "go-buildrun") >= 0 {
		panic("go-buildrun tool cannot build itself!")
	}
	if rawBytes, err = ioutil.ReadFile(filePath); err == nil {
		if tmp = string(rawBytes); strings.HasPrefix(tmp, "package main\n") || strings.HasPrefix(tmp, "package main\r") || strings.Contains(tmp, "\npackage main\n") || strings.Contains(tmp, "\npackage main\r") {
			return true
		}
	} else {
		panic(err)
	}
	return false
}

func processTemplates(dirPath string) (hasDocGoFile bool) {
	const begin, end, pkg = "//#begin-gt", "//#end-gt", "package gt\n"
	var (
		rawBytes                                      []byte
		orig, cur, curBefore, curLine, curAfter, tmpl string
		posBegin, posEnd, posCrlf                     int
		parts, repl, goFilePaths                      []string
		templates                                     = map[string]string{}
		fileInfos, err                                = ioutil.ReadDir(dirPath)
	)
	if err != nil {
		panic(err)
	}
	for _, fi := range fileInfos {
		if !fi.IsDir() {
			if strings.HasSuffix(fi.Name(), ".gt.go") {
				goFilePaths = append(goFilePaths, filepath.Join(dirPath, fi.Name()))
			} else if strings.HasSuffix(fi.Name(), ".gt") {
				if rawBytes, err = ioutil.ReadFile(filepath.Join(dirPath, fi.Name())); err != nil {
					panic(err)
				}
				tmpl = string(rawBytes)
				if posBegin = strings.Index(tmpl, pkg); posBegin >= 0 {
					tmpl = tmpl[len(pkg):]
				}
				templates[fi.Name()] = "\n" + tmpl + "\n"
			} else if strings.ToLower(fi.Name()) == "doc.go" {
				hasDocGoFile = true
			}
		}
	}
	if len(templates) > 0 {
		for _, fp := range goFilePaths {
			if rawBytes, err = ioutil.ReadFile(fp); err != nil {
				panic(err)
			}
			orig = string(rawBytes)
			cur = orig
			if posBegin, posEnd = strings.Index(cur, begin), strings.Index(cur, end); (posBegin > -1) && (posEnd > (posBegin + len(begin))) {
				if posCrlf = posBegin + strings.Index(cur[posBegin:], "\n"); posCrlf > posBegin {
					curBefore, curAfter, curLine = cur[:posBegin], cur[posEnd:], cur[posBegin:posCrlf]
					if parts = strings.Split(curLine, " "); len(parts) > 1 {
						if tmpl = templates[parts[1]]; len(tmpl) == 0 {
							tmpl = fmt.Sprintf("\nTEMPLATE NOT FOUND: %v\n", parts[1])
						}
						for _, rp := range parts[2:] {
							if repl = strings.Split(rp, ":"); len(repl) > 1 {
								tmpl = strings.Replace(tmpl, "__"+repl[0]+"__", repl[1], -1)
							}
						}
						cur = curBefore + curLine + tmpl + curAfter
					}
				}
			}
			if cur != orig {
				ioutil.WriteFile(fp, []byte(cur), os.ModePerm)
			}
		}
	}
	return
}

func trimLines(str string, maxLines int) string {
	return str
	var lines = strings.Split(str, "\n")
	if len(lines) > maxLines {
		lines = lines[0:maxLines]
	}
	return strings.Join(lines, "\n")
}

func main() {
	var (
		startTime                = time.Now()
		pathSep                  = string(os.PathSeparator)
		flagFilePath             = flag.String("f", "", "Full path to current .go source file from which to build (go install) package.")
		flagGenDocHtml           = flag.String("d", "", "Specify a file name such as doc.html to generate single-page package-doc in package directory; omit to not generate this.")
		goInstPath               string
		goPath                   = os.ExpandEnv("$GOPATH")
		isMainPkg, hasDocGoFile  bool
		origFilePath, cmdRunPath string
		rawBytes                 []byte
		err                      error
		allowRun                 = true
		dirFiles                 []os.FileInfo
	)
	runtime.LockOSThread()
	flag.Parse()
	origFilePath = *flagFilePath
	isMainPkg = checkForMainPackage(origFilePath)
	hasDocGoFile = processTemplates(filepath.Dir(origFilePath))
	goInstPath = strings.Replace(origFilePath[len(filepath.Join(goPath, "src")+pathSep):], pathSep, "/", -1)
	goInstPath = goInstPath[0:strings.LastIndex(goInstPath, "/")]
	for dp := filepath.Dir(origFilePath); len(dp) > len(goPath); dp = filepath.Dir(dp) {
		if dirFiles, err = ioutil.ReadDir(dp); err != nil {
			panic(err)
		}
		for _, fi := range dirFiles {
			if (!fi.IsDir()) && strings.HasSuffix(fi.Name(), ".go-buildrun") {
				if rawBytes, err = ioutil.ReadFile(filepath.Join(dp, fi.Name())); err != nil {
					panic(err)
				}
				cmdRunPath = strings.Trim(string(rawBytes), " \t\r\n")
				log.Printf("RUN: %v\n", cmdRunPath)
				if rawBytes, err = exec.Command(cmdRunPath, dp).CombinedOutput(); err != nil {
					panic(err)
				}
				if len(rawBytes) > 0 {
					fmt.Printf("%v", string(rawBytes))
				}
				break
			}
		}
	}
	log.Printf("RUN: go install %v\n", goInstPath)
	rawBytes, err = exec.Command("go", "install", goInstPath).CombinedOutput()
	if len(rawBytes) > 0 {
		allowRun = false
		fmt.Printf("%v\n", trimLines(string(rawBytes), 5))
	}
	if err != nil {
		allowRun = false
		fmt.Printf("%+v\n", err)
	}
	if allowRun && (hasDocGoFile || strings.Contains(goInstPath, "metaleap/go-xsd-pkg")) && (!isMainPkg) && (len(*flagGenDocHtml) > 0) {
		var docTemplate = `<html>
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
		log.Printf("RUN: godoc -html=true %v\n", goInstPath)
		// if rawBytes, err = ioutil.ReadFile(filepath.Join(goPath, "src/github.com/metaleap/go-buildrun/doctemplate.html")); (err == nil) && (len(rawBytes) > 0) {
		// 	docTemplate = string(rawBytes)
		// }
		if rawBytes, err = exec.Command("godoc", "-html=true", goInstPath).CombinedOutput(); err != nil {
			log.Printf("GODOC error: %v\n", err)
		} else if err = ioutil.WriteFile(filepath.Join(filepath.Dir(origFilePath), *flagGenDocHtml), []byte(fmt.Sprintf(docTemplate, goInstPath, goInstPath, string(rawBytes))), os.ModePerm); err != nil {
			log.Printf("DOC file write error: %v\n", err)
		}
	}
	log.Printf("TOTAL BUILD TIME: %v\n", time.Now().Sub(startTime))
	if allowRun && isMainPkg {
		cmdRunPath = filepath.Join(goPath, "bin", goInstPath[strings.LastIndex(goInstPath, "/")+1:])
		log.Printf("RUN: %v\n", cmdRunPath)
		rawBytes, err = exec.Command(cmdRunPath).CombinedOutput()
		if len(rawBytes) > 0 {
			fmt.Printf("%v\n", trimLines(string(rawBytes), 10))
		}
		if err != nil {
			log.Printf("ERROR: %+v\n", err)
		}
	}
}
