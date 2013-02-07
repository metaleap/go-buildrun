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
	if strings.Index(filePath, "go-buildrun") >= 0 {
		panic("go-buildrun tool refuses file paths containing 'go-buildrun'...")
	}
	if rawBytes, err := ioutil.ReadFile(filePath); err == nil {
		if tmp := string(rawBytes); strings.HasPrefix(tmp, "package main\n") || strings.HasPrefix(tmp, "package main\r") || strings.Contains(tmp, "\npackage main\n") || strings.Contains(tmp, "\npackage main\r") {
			return true
		}
	} else {
		panic(err)
	}
	return false
}

func processTemplateConsumer(fp string, templates map[string]string, errChan chan error) {
	const begin, end, multSepDef = "//#begin-gt", "//#end-gt", "GT_MULT_SEP:"
	var (
		err                                                        error
		rawBytes                                                   []byte
		parts, repl, mult                                          []string
		posBegin, posEnd, posCrlf                                  int
		orig, cur, curBefore, curLine, curAfter, tmpl, multSep, rp string
	)
	if rawBytes, err = ioutil.ReadFile(fp); err == nil {
		orig = string(rawBytes)
		cur = orig
		if posBegin, posEnd = strings.Index(cur, begin), strings.Index(cur, end); (posBegin > -1) && (posEnd > (posBegin + len(begin))) {
			if posCrlf = posBegin + strings.Index(cur[posBegin:], "\n"); posCrlf > posBegin {
				curBefore, curAfter, curLine = cur[:posBegin], cur[posEnd:], cur[posBegin:posCrlf]
				if parts = strings.Split(curLine, " "); len(parts) > 1 {
					if tmpl = templates[parts[1]]; len(tmpl) == 0 {
						tmpl = fmt.Sprintf("\nTEMPLATE NOT FOUND: %s\n", parts[1])
					}
					startAt := 2
					if strings.HasPrefix(parts[startAt], multSepDef) {
						multSep = strings.Replace(parts[startAt], multSepDef, "", -1)
						startAt++
					}
					if len(multSep) > 0 {
						var (
							multPass, multLoop int
							multOut, multThis  string
							multParts          = map[string][]string{}
						)
						for _, rp = range parts[startAt:] {
							if repl = strings.Split(rp, ":"); len(repl) > 1 {
								multParts[repl[0]] = strings.Split(repl[1], multSep)
								if multLoop == 0 || multLoop > len(multParts[repl[0]]) {
									multLoop = len(multParts[repl[0]])
								}
							}
						}
						for multPass < multLoop {
							multThis = tmpl
							for rp, mult = range multParts {
								multThis = strings.Replace(multThis, "__"+rp+"__", mult[multPass], -1)
							}
							multOut += multThis
							multPass++
						}
						tmpl = multOut
					} else {
						for _, rp = range parts[startAt:] {
							if repl = strings.Split(rp, ":"); len(repl) > 1 {
								tmpl = strings.Replace(tmpl, "__"+repl[0]+"__", repl[1], -1)
							}
						}
					}
					cur = curBefore + curLine + tmpl + curAfter
				}
			}
		}
		if cur != orig {
			err = ioutil.WriteFile(fp, []byte(cur), os.ModePerm)
		}
	}
	errChan <- err
}

func processTemplates(dirPath string) (hasDocGoFile bool) {
	const pkg = "package gt\n"
	var (
		rawBytes       []byte
		tmpl           string
		pos            int
		goFilePaths    []string
		templates      = map[string]string{}
		fileInfos, err = ioutil.ReadDir(dirPath)
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
				if pos = strings.Index(tmpl, pkg); pos >= 0 {
					tmpl = tmpl[len(pkg):]
				}
				templates[fi.Name()] = "\n" + tmpl + "\n"
			} else if strings.ToLower(fi.Name()) == "doc.go" {
				hasDocGoFile = true
			}
		}
	}
	if len(templates) > 0 {
		numFiles := len(goFilePaths)
		errChan := make(chan error)
		for _, fp := range goFilePaths {
			go processTemplateConsumer(fp, templates, errChan)
		}
		for i := 0; i < numFiles; i++ {
			if err = <-errChan; err != nil {
				panic(err)
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
		cmdArgs                  []string
	)
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()
	origFilePath = *flagFilePath
	isMainPkg = checkForMainPackage(origFilePath)
	hasDocGoFile = processTemplates(filepath.Dir(origFilePath))
	if pos := strings.Index(goPath, string(os.PathListSeparator)); pos > 0 {
		goPath = goPath[:pos]
		fmt.Printf("Your GOPATH contains multiple paths, using %v.\n", goPath)
	}
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
				for _, cmdRunPath := range strings.Split(string(rawBytes), "\n") {
					if cmdRunPath = strings.Replace(strings.Trim(cmdRunPath, " \t\r\n"), "$dir", dp, -1); (len(cmdRunPath) > 0) && !(strings.HasPrefix(cmdRunPath, "#") || strings.HasPrefix(cmdRunPath, "//")) {
						log.Printf("RUN: %s\n", cmdRunPath)
						cmdArgs = strings.Split(cmdRunPath, " ")
						rawBytes, err = exec.Command(cmdArgs[0], cmdArgs[1:]...).CombinedOutput()
						if len(rawBytes) > 0 {
							fmt.Printf("%s", string(rawBytes))
						}
						if err != nil {
							log.Printf("ERR: %v", err)
						}
					}
				}
			}
		}
	}
	log.Printf("RUN: go install %s\n", goInstPath)
	rawBytes, err = exec.Command("go", "install", goInstPath).CombinedOutput()
	if len(rawBytes) > 0 {
		allowRun = false
		fmt.Printf("%s\n", trimLines(string(rawBytes), 5))
	}
	if err != nil {
		allowRun = false
		fmt.Printf("%+v\n", err)
	}
	if allowRun && (hasDocGoFile || strings.Contains(goInstPath, "metaleap/go-xsd-pkg")) && (!isMainPkg) && (len(*flagGenDocHtml) > 0) {
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
		log.Printf("RUN: godoc -html=true %s\n", goInstPath)
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
		log.Printf("RUN: %s\n%s\n\n", cmdRunPath, strings.Repeat("_", 25+len(cmdRunPath)))
		rawBytes, err = exec.Command(cmdRunPath).CombinedOutput()
		if len(rawBytes) > 0 {
			fmt.Printf("%s\n", trimLines(string(rawBytes), 10))
		}
		if err != nil {
			log.Printf("ERROR: %+v\n", err)
		}
	}
}
