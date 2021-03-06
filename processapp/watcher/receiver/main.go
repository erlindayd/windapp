package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	c3to10 "eaciit/wfdemo-git/processapp/threeextractor/convert3secto10min/lib"
	"github.com/eaciit/database/base"
	"github.com/eaciit/orm"
	"github.com/fsnotify/fsnotify"
	// "github.com/metakeule/fmtdate"

	//dc "eaciit/wfdemo-git/processapp/threeextractor/dataconversion"
	. "eaciit/wfdemo-git/processapp/watcher/controllers"
	"archive/tar"
	"archive/zip"
	"time"
	"bytes"
	tk "github.com/eaciit/toolkit"
)

const (
	NOK = "NOK"
	OK  = "OK"
	// TIME_FORMAT_STR = "DDMMYYhhmmss"
)

type Command struct {
	Action  string
	Command string
	Success string
	Fail    string
}
type Configuration struct {
	Draft    string
	Process  string
	Fail     string
	Success  string
	Errors   string
	Archive  string
	Commands []Command
}

var (
	conn base.IConnection
	ctx  *orm.DataContext
	conf Configuration
	mux  = &sync.Mutex{}
	pathSep = string(os.PathSeparator)
)

func main() {
	
	fmt.Println(".."+pathSep+"conf"+pathSep+"receiver-config.json")
	
	file, _ := os.Open(".."+pathSep+"conf"+pathSep+"receiver-config.json")
	decoder := json.NewDecoder(file)
	err := decoder.Decode(&conf)

	if err != nil {
		panic(err)
	}

	// now, _ := time.Parse("2006-1-2 15:4:05", "2016-10-22 23:57:30")
	// tenMinInfo := GenTenMinuteInfo(now)

	// log.Printf("%#v \n", tenMinInfo)

	runtime.GOMAXPROCS(runtime.NumCPU())
	log.Println("Starting the app..\n")

	log.Println()
	log.Printf("Draft: %v\n", conf.Draft)
	log.Printf("Process: %v\n", conf.Process)
	log.Printf("Fail: %v\n", conf.Fail)
	log.Printf("Success: %v\n", conf.Success)
	log.Printf("Errors: %v\n", conf.Errors)
	log.Println()

	/*db, _ := PrepareConnection()
	base := new(BaseController)
	base.Ctx = orm.New(db)
	defer base.Ctx.Close()

	log.Println("xxx")

	new(GenTenFromThreeSecond).Generate(base, "sample.csv")*/

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				// log.Println("event:", event)
				if event.Op&fsnotify.Create == fsnotify.Create {					
					go processFile(event.Name, conf.Commands)
				}
			case err := <-watcher.Errors:
				log.Println("error:", err)
			}
		}
	}()

	// watch draft
	err = watcher.Add(conf.Draft)
	if err != nil {
		log.Fatal(err)
	}

	// watch process
	err = watcher.Add(conf.Process)
	if err != nil {
		log.Fatal(err)
	}

	/*// watch fail
	err = watcher.Add("/Users/frezadev/Documents/watch/fail")
	if err != nil {
		log.Fatal(err)
	}

	// watch success
	err = watcher.Add("/Users/frezadev/Documents/watch/success")
	if err != nil {
		log.Fatal(err)
	}*/
	<-done
}

func processFile(filePath string, com []Command) {
	// dt := time.Now()
	// dtStr := fmtdate.Format(TIME_FORMAT_STR, dt)
	fmt.Println("Process")
	for true{
		byteOut,err:=runCMD("lsof "+filePath)
		if err!=nil{
			log.Print("Gagal")
		}
		if len(byteOut)==0{
			break
		}else{
			fmt.Println(string(byteOut))
			time.Sleep(200*time.Millisecond)
		}
		
	}
	file := strings.Split(filePath, pathSep)
	fileName := file[len(file)-1]

	if strings.Contains(fileName, ".csv") {
		time.Sleep(100*time.Millisecond)
		log.Printf("Proccess file: %v \n", filePath)
		var action Command

		if strings.Contains(filePath, conf.Draft) {
			action = com[0]
		} else if strings.Contains(filePath, conf.Process) {
			action = com[1]
		}

		// log.Printf("action: %v \n", action)

		next := action.Action
	done:

		for {
			log.Printf("\n\nnext: %v \n", next)

			if next == "DONE" {
				break done
			} else {
				for _, act := range com {
					//fmt.Println("PPPP")
					if act.Action == next {
						next = run(act, fileName)
						
						break
					}
				}
			}
		}

		log.Printf("DONE for file: %v\n", filePath)
	} else if strings.Contains(fileName, ".tar") {
		time.Sleep(100*time.Millisecond)
		untar(filePath, conf.Draft)
		_, _ = runCMD(fmt.Sprintf("mv %v %v", filePath, conf.Archive))
	} else if strings.Contains(fileName, ".zip") {
		time.Sleep(100*time.Millisecond)
		e:=unzip(filePath, conf.Draft)
		if e!=nil{
			panic(e)
		}
		fmt.Println("Unzip Done")
		_, _ = runCMD(fmt.Sprintf("mv %v %v", filePath, conf.Archive))
	}
}

func untar(tarball, target string) error {
	reader, err := os.Open(tarball)
	if err != nil {
		return err
	}
	defer reader.Close()
	tarReader := tar.NewReader(reader)

	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		} else if err != nil {
			log.Printf("ERR: %#v \n", err.Error())
			return err
		}

		path := filepath.Join(target, header.Name)
		info := header.FileInfo()
		// log.Printf("path: %#v \n", path)
		// log.Printf("info: %#v \n", info)
		if strings.Contains(info.Name(), ".csv") {
			if info.IsDir() {
				if err = os.MkdirAll(path, info.Mode()); err != nil {
					return err
				}
				continue
			}

			file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
			if err != nil {
				return err
			}
			defer file.Close()
			_, err = io.Copy(file, tarReader)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func unzip(archive, target string) error {
	fmt.Println("Unzip",archive,target)
	reader, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(target, 0755); err != nil {
		return err
	}

	for _, file := range reader.File {
		if strings.Contains(file.Name, ".csv") {

			path := filepath.Join(target, file.Name)
			fileReader, err := file.Open()

			if err != nil {
				log.Printf("ERR: %#v \n", err.Error())
				return err
			}
			defer fileReader.Close()

			targetFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
			if err != nil {
				return err
			}
			defer targetFile.Close()

			if _, err := io.Copy(targetFile, fileReader); err != nil {
				return err
			}
		}
	}

	reader.Close()
	return nil
}

func runCMD(cmdStr string) (out []byte, err error) {
	if pathSep=="\\"{
		cmdStr=strings.Replace(cmdStr,"\\","/",-1)
		
	}
	
	fmt.Println("sh", "-c", cmdStr)
	var errBuff bytes.Buffer
	cmd := exec.Command("sh", "-c", cmdStr)
	cmd.Stderr = &errBuff
	//cmd.Run()
	//cmd.Path = os.Getenv("Path")
	//fmt.Println(cmd.Path)
	out, err = cmd.Output()	
	if err!=nil{
		fmt.Println(errBuff.String())
	}
	return
}

func run(action Command, file string) (next string) {
	cmdStr := ""
	runCommand := true
	log.Printf("run: %v | %v \n", action, file)
	// mux.Lock()

	if action.Action == "COPY_TO_PROCESS" {
		cmdStr = fmt.Sprintf(action.Command, conf.Draft+pathSep+file, conf.Process+pathSep+file)
	} else if action.Action == "COPY_TO_SUCCESS" {
		runCommand = doProcess(conf.Process + pathSep + file)
		
		cmdStr = fmt.Sprintf(action.Command, conf.Process+pathSep+file, conf.Success+pathSep+file)
	} else if action.Action == "COPY_TO_FAIL" {
		cmdStr = fmt.Sprintf(action.Command, conf.Process+pathSep+file, conf.Fail+pathSep+file)
	}
	//log.Printf("cmdstr: %v \n", cmdStr)
	// mux.Unlock()

	if runCommand {
		out, err := runCMD(cmdStr)

		if out != nil {
			log.Printf("%v \n", out)
		}

		if err != nil {
			log.Printf("result: %v %s\n%s", err.Error(),cmdStr,string(out))
			next = action.Success
		} else {
			next = action.Success
		}
	} else {
		log.Println("DONE")
		next = action.Fail
	}

	// log.Printf("next: %v \n", next)

	return
}

func doProcess(file string) (success bool) {
	log.Printf("doProcess: %v \n", file)
	db, e := PrepareConnection()
	if e != nil {
		log.Printf("ERROR on Process: %v\n", e.Error())
		success = false
	} else {
		//start := time.Now()
		base := new(BaseController)
		base.Ctx = orm.New(db)
		defer base.Ctx.Close()

		anFile := strings.Split(file, pathSep)
		fileName := anFile[len(anFile)-1]

		muxDo := &sync.Mutex{}

		muxDo.Lock()
		errorLine := new(ConvScadaTreeSecs).Generate(base, file)
		WriteWatcherErrors(errorLine, fileName, conf.Errors)
		log.Println("ConvScadaTreeSecs: DONE")
		muxDo.Unlock()
		log.Println("Begin Converting ",file)
		err := c3to10.Generate(tk.M{}.Set("selector","file").Set("file",fileName))
		if err != nil {
			tk.Println(err)
		} else {
			tk.Println(">> DONE <<")
		}

		// errorLine = tk.M{}
		// errorLine = new(GenTenFromThreeSecond).Generate(base, fileName)
		// WriteWatcherErrors(errorLine, fileName+"-ten", conf.Errors)
		// log.Println("GenTenFromThreeSecond: DONE")
		
		/*
		muxDo.Lock()
		errorLine = tk.M{}
		conv := dc.NewDataConversion(base.Ctx)
		errorLine = conv.Generate(fileName)
		// errorLine = new(GenTenFromThreeSecond).Generate(base, fileName)
		WriteWatcherErrors(errorLine, fileName+"-ten", conf.Errors)
		log.Printf("GenTenFromThreeSecond: %v DONE | in %v seconds \n", file, time.Now().Sub(start).Seconds())
		success = true
		muxDo.Unlock()*/
	}

	return
}
