package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

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
				curBefore, curAfter, curLine = cur[:posBegin], cur[posEnd:], strings.TrimSpace(cur[posBegin:posCrlf])
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
					cur = curBefore + curLine + "\n" + tmpl + "\n" + curAfter
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
				const pkg = "package "
				if pos = strings.Index(tmpl, pkg); pos >= 0 {
					tmpl = tmpl[len(pkg):]
					tmpl = tmpl[strings.Index(tmpl, "\n")+1:]
				}
				templates[fi.Name()] = "\n" + strings.TrimSpace(tmpl) + "\n"
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
