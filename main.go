package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"os"
	"os/exec"
	"strings"
	"time"
)

func checkForMainPackage (filePath string) bool {
	var err error
	var rawBytes []byte
	var tmp string
	if strings.Index(filePath, "go-buildrun") >= 0 { panic("go-buildrun tool cannot build itself!") }
	if rawBytes, err = ioutil.ReadFile(filePath); err == nil {
		if tmp = string(rawBytes); strings.HasPrefix(tmp, "package main\n") || strings.HasPrefix(tmp, "package main\r") { return true }
	} else {
		panic(err)
	}
	return false
}

func processTemplates (dirPath string) {
	const begin, end, pkg = "//#begin-gt", "//#end-gt", "package gt\n"
	var (
		rawBytes []byte
		orig, cur, curBefore, curLine, curAfter, tmpl string
		posBegin, posEnd, posCrlf int
		parts, repl, goFilePaths []string
		templates = map[string]string {}
		fileInfos, err = ioutil.ReadDir(dirPath)
	)
	if err != nil { panic(err) }
	for _, fi := range fileInfos {
		if !fi.IsDir() {
			if strings.HasSuffix(fi.Name(), ".gt.go") {
				goFilePaths = append(goFilePaths, filepath.Join(dirPath, fi.Name()))
			} else if strings.HasSuffix(fi.Name(), ".gt") {
				if rawBytes, err = ioutil.ReadFile(filepath.Join(dirPath, fi.Name())); err != nil { panic(err) }
				tmpl = string(rawBytes)
				if posBegin = strings.Index(tmpl, pkg); posBegin >= 0 { tmpl = tmpl[len(pkg) :] }
				templates[fi.Name()] = "\n" + tmpl + "\n"
			}
		}
	}
	if len(templates) > 0 {
		for _, fp := range goFilePaths {
			if rawBytes, err = ioutil.ReadFile(fp); err != nil { panic(err) }
			orig = string(rawBytes); cur = orig
			if posBegin, posEnd = strings.Index(cur, begin), strings.Index(cur, end); (posBegin > -1) && (posEnd > (posBegin + len(begin))) {
				if posCrlf = posBegin + strings.Index(cur[posBegin :], "\n"); posCrlf > posBegin {
					curBefore, curAfter, curLine = cur[: posBegin], cur[posEnd :], cur[posBegin : posCrlf]
					if parts = strings.Split(curLine, " "); len(parts) > 1 {
						if tmpl = templates[parts[1]]; len(tmpl) == 0 { tmpl = fmt.Sprintf("\nTEMPLATE NOT FOUND: %v\n", parts[1]) }
						for _, rp := range parts[2 :] {
							if repl = strings.Split(rp, ":"); len(repl) > 1 {
								tmpl = strings.Replace(tmpl, "__" + repl[0] + "__", repl[1], -1)
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
}

func trimLines (str string, maxLines int) string {
	return str
	var lines = strings.Split(str, "\n")
	if len(lines) > maxLines { lines = lines[0 : maxLines] }
	return strings.Join(lines, "\n")
}

func main () {
	var (
		startTime = time.Now()
		pathSep = string(os.PathSeparator)
		flagFilePath = flag.String("f", "", "file: current .go source file from which to build")
		goInstPath string
		goPath = os.ExpandEnv("$GOPATH")
		isMainPkg bool
		origFilePath, cmdRunPath string
		rawBytes []byte
		err error
		allowRun = true
		dirFiles []os.FileInfo
	)
	flag.Parse()
	origFilePath = *flagFilePath
	isMainPkg = checkForMainPackage(origFilePath)
	processTemplates(filepath.Dir(origFilePath))
	goInstPath = strings.Replace(origFilePath [len(filepath.Join(goPath, "src") + pathSep) : ], pathSep, "/", -1)
	goInstPath = goInstPath [0 : strings.LastIndex(goInstPath, "/")]
	for dp := filepath.Dir(origFilePath); len(dp) > len(goPath); dp = filepath.Dir(dp) {
		if dirFiles, err = ioutil.ReadDir(dp); err != nil { panic(err) }
		for _, fi := range dirFiles {
			if (!fi.IsDir()) && strings.HasSuffix(fi.Name(), ".go-buildrun") {
				if rawBytes, err = ioutil.ReadFile(filepath.Join(dp, fi.Name())); err != nil { panic(err) }
				if rawBytes, err = exec.Command(strings.Trim(string(rawBytes), " \t\r\n"), dp).CombinedOutput(); err != nil { panic(err) }
				if len(rawBytes) > 0 {
					fmt.Printf("%v", string(rawBytes))
				}
				break
			}
		}
	}
	rawBytes, err = exec.Command("go", "install", goInstPath).CombinedOutput()
	if len(rawBytes) > 0 {
		allowRun = false
		fmt.Printf("%v\n", trimLines(string(rawBytes), 5))
	}
	if err != nil {
		allowRun = false
		fmt.Printf("%+v\n", err)
	}
	fmt.Printf("TOTAL BUILD TIME: %v\n", time.Now().Sub(startTime))
	if (allowRun && isMainPkg) {
		cmdRunPath = filepath.Join(goPath, "bin", goInstPath [strings.LastIndex(goInstPath, "/") + 1 : ])
		rawBytes, err = exec.Command(cmdRunPath).CombinedOutput()
		if len(rawBytes) > 0 { fmt.Printf("%v\n", trimLines(string(rawBytes), 10)) }
		if err != nil { fmt.Printf("%+v\n", err) }
	}
}
