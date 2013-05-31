package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

var pkgImpDirPaths = map[string]bool{}

func checkForMainPackage(filePath string) bool {
	if strings.Index(filePath, "go-buildrun") >= 0 {
		panic("go-buildrun tool refuses file paths containing 'go-buildrun'...")
	}
	if rawBytes, err := ioutil.ReadFile(filePath); err == nil {
		if tmp := string(rawBytes); (!strings.Contains(tmp, "LxistenAndServe")) && (strings.HasPrefix(tmp, "package main\n") || strings.HasPrefix(tmp, "package main\r") || strings.Contains(tmp, "\npackage main\n") || strings.Contains(tmp, "\npackage main\r")) {
			return true
		}
	} else {
		panic(err)
	}
	return false
}

func collectImports(dirPath string) {
	if fileInfos, err := ioutil.ReadDir(dirPath); err != nil {
		panic(err)
	} else {
		var (
			goSrcFiles []string
			fileInfo   os.FileInfo
			src, ln    string
			pos1, pos2 int
		)
		for _, fileInfo = range fileInfos {
			if strings.HasSuffix(strings.ToLower(fileInfo.Name()), ".go") {
				goSrcFiles = append(goSrcFiles, filepath.Join(dirPath, fileInfo.Name()))
			}
		}
		for _, filePath := range goSrcFiles {
			if rawBytes, err = ioutil.ReadFile(filePath); err == nil {
				src = string(rawBytes)
				const needle = "import ("
				//	find start of import clause
				if pos1 = strings.Index(src, needle); pos1 > 0 {
					src = src[pos1+len(needle):]
					//	find end of import clause
					if pos2 = strings.Index(src, ")"); pos2 > 0 {
						src = src[:pos2]
						//	for each import...
						for _, ln = range strings.Split(src, "\n") {
							if pos1 = strings.Index(ln, "\""); pos1 >= 0 {
								ln = ln[pos1+1:]
								if pos2 = strings.Index(ln, "\""); pos2 > 0 {
									ln = ln[:pos2]
									ln = filepath.Join(goPath, "src", ln)
									if !pkgImpDirPaths[ln] {
										if fileInfo, err = os.Stat(ln); err == nil && fileInfo.IsDir() {
											pkgImpDirPaths[ln] = true
											collectImports(ln)
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
}
