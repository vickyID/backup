package main 

import (
	"errors"
	"flag"
	"time"
	"os"
	"os/signal"
	"encoding/json"
	"log"
	"syscall"
	"fmt"
	"github.com/vickyID/backup"
	"github.com/matryer/filedb"

)

type path struct {
	Path string
	Hash string
}


func (p path) String() string {
	return fmt.Sprintf("%s [%s]", p.Path,p.Hash)
}


func main() {
	var fatalErr error 
	defer func(){
		log.Fatalln(fatalErr)
	}()

	var (
		interval = flag.Duration("interval",5*time.Second, "Interval Between Checks")
		dbpath = flag.String("db","./db","path to database directory")
		archive = flag.String("archive","archive","path to archive location")
	)
	flag.Parse()

	m := &backup.Monitor{
		Destination: *archive,
		Archiver: backup.ZIP,
		Paths: make(map[string]string),
	}


	//open a connection to the local database and defer the closing of the connection 
	db, err := filedb.Dial(*dbpath)
	if err != nil {
		fatalErr = err 
		return 
	}
	defer db.Close() 

	//Get an iterator to iterate over the contents of the database and defer the closing of the iterator
	col, err := db.C("paths")
	if err != nil {
		fatalErr = err 
		return 
	}


	var path path
	col.ForEach(func(_ int, data []byte) bool {
		if err := json.Unmarshal(data,&path); err != nil {
			fatalErr = err 
			return true
		}
		m.Paths[path.Path] = path.Hash
		return false // carry on 
	})

	if fatalErr != nil {
		return
	}

	if len(m.Paths) < 1 {
		fatalErr = errors.New("No paths - use backup tool to add atleas one")
		return 
	}

	check(m,col)

	//important information learning
	signalChan := make(chan os.Signal,1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <- signalChan:
			fmt.Println("Stopping......")
			return
		case <-time.After(*interval):
			check(m,col)
		}
	}
}

func check(m *backup.Monitor, col *filedb.C){
	log.Println("Checking.......")
	counter , err := m.Now()
	if err != nil {
		log.Fatalln("Failed to backup:", err)
	}

	if counter > 0 { 
		log.Printf(" Archived %d directories\n",counter)
		var path path 
		col.SelectEach(func(_ int, data []byte) (bool, []byte, bool) {
			if err := json.Unmarshal(data,&path); err != nil {
				log.Println("Failed to inmarshal data {skipping}: ", err)
				return true, data, false
			}
			path.Hash, _ = m.Paths[path.Path]
			newdata, err := json.Marshal(&path)
			if err != nil {
				log.Println("Failed to marshal data {skipping}: ", err)
				return true, data, false
			}
			return true,newdata,false 
		})
	} 	else {
		log.Println("No Changes")
	}
}