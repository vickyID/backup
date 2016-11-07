package main 

import(
	"fmt"
	"errors"
	"log"
	"strings"
	"encoding/json"
	"flag"

	"github.com/matryer/filedb"
	
)


type path struct {
	Path string
	Hash string
}


func (p path) String() string {
	return fmt.Sprintf("%s [%s]", p.Path,p.Hash)
}


//usage 
// backup -db=/path/to/db add {path} [paths ......]
// backup -db=/path/to/db remove {path} [paths ......]
// backup -db=/path/to/db list 




func main() {

	var fatalErr error
	defer func(){
		if fatalErr != nil {
			flag.PrintDefaults()
			log.Fatalln(fatalErr)
		}
	}()

	var (
		dbpath = flag.String("db","./backupdata","path to database directory")
	)

	flag.Parse()
	args := flag.Args() 

	//we check if the user has provided any command line arguments if not return an usage error
	if len(args) < 1 {
		fatalErr = errors.New("invalid usage; must specify command")
		return 
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

	//depending on the command provided, switch and execute the function 
	switch strings.ToLower(args[0]) {
	case "list":
		var path path 
		col.ForEach(func(i int, data []byte) bool {
			json.Unmarshal(data, &path)
			if err != nil {
				fatalErr = err 
				return false 
			}
			fmt.Printf("=%s\n",path)
			return false 

		})
	case "add":
		if len(args[1:]) == 0 { 
			fatalErr = errors.New("You must specify paths to be added")
			return 
		}
		for _, p := range args[1:] {
			path := &path{Path:p, Hash:"Not Yet Archived"}
			if err := col.InsertJSON(path); err != nil {
				fatalErr = err 
				return
			}
			fmt.Printf("+%s\n",path)
		}
	case "remove":
		var path path 
		col.RemoveEach(func(i int, data[]byte)(bool,bool){
			err := json.Unmarshal(data,&path)
			if err != nil {
				fatalErr = err 
				return false, true
			}
			for _, p := range args[1:]{
				if path.Path == p {
					fmt.Printf("-%s\n",path)
					return true,false 
				}
			}
			return false,false 

		})
	}
}