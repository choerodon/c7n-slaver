package server

import (
	"github.com/vinkdong/gox/log"
	"net/http"
	"time"
	"fmt"
	"database/sql"
	"os/exec"
	"io/ioutil"
	"encoding/json"
)

type PortRequest struct {
	Ports []int `json:"ports"`
}

type ServerAddr struct {
	Hosts []string `json:"hosts"`
	Ports []int `json:"ports"`
}

type MountPathInfo struct {
	Path string `json:"path"`
	Require string `json:"require"`
}

type StorageStatus struct {
	Success bool `json:"success"`
	Free string  `json:"free"`
}
type MysqlInfo struct {
	MysqlHost string `json:"mysql_host"`
	MysqlPort string `json:"mysql_port"`
	MysqlName string `json:"mysql_name"`
	MysqlPwd  string `json:"mysql_pwd"`
}

type Requst struct {
	Scop string 		`json:"scop"`
	Mysql MysqlInfo 	`json:"mysql_info"`
	DatabaseName string	`json:"database_name"`
	SQL  string 		`json:"sql"`
}

type CommandExec struct {
	CommandLine string `json:"command"`
}

type C7nInfo struct {
	Path string 	`json:"path"`
	Random string 	`json:"random"`
}

type RandomInfo struct {
	Url string 		`json:"url"`
	Random string 	`json:"random"`
}

func (C *CommandExec)ExecuteCommand() (err error)  {
	cmd := exec.Command("sh", "-c", C.CommandLine)
	err = cmd.Run()
	if err != nil {
		log.Error(err.Error())
	}
	return
}

func (M *MysqlInfo)ConnetMySql() (db *sql.DB, err error)  {
	ConnetInfo   := fmt.Sprint( M.MysqlName,":",M.MysqlPwd,"@tcp(",M.MysqlHost,":",M.MysqlPort,")/?charset=utf8&timeout=3s")
	db, err = sql.Open("mysql", ConnetInfo)
	if err != nil {
		log.Errorf("Failed to connect mysql: %s", err)
		return
	}
	err = db.Ping()
	if err != nil {
		log.Errorf("Failed to ping mysql: %s", err)
	}
	return
}
func (R *Requst)Executed(db *sql.DB) (err error) {
	if R.Scop == "database" {
		_,err = db.Exec(R.SQL)
		if err != nil {
			log.Errorf("Failed to execute: %s", err)
		}
	} else {
		_,err = db.Exec(fmt.Sprint("USE ",R.DatabaseName))
		_,err = db.Exec(R.SQL)
		if err != nil {
			log.Errorf("Failed executee: %s", err)
		}
	}
	return
}

func (s *ServerAddr) StartNetCheck() (err error){
	c := &http.Client{
		Timeout: time.Second,
	}
	for _, ip := range s.Hosts {
		for _, port := range s.Ports {
			_, err = c.Get(fmt.Sprint("http://",ip,":",port,"/health"))
			if err != nil {
				log.Error(err)
				return
			}
			log.Info(fmt.Sprint("http://",ip,":",port,"/health"," ok"))
		}
	}
	return
}

func (p *PortRequest) StartServers() error {
	for _, port := range p.Ports {
		s := NewServer(port)
		s.AddHealthHandler()
		startedServer = append(startedServer, s)
		go func() {
			err := s.Start()
			if err != nil {
				log.Error(err)
			}
		}()
	}
	return nil
}
func (c7n *C7nInfo)StartMonitor(s *Server) error  {
	s.ServerMux.HandleFunc(c7n.Path, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		Context := fmt.Sprintf(`{"random":"%s"}`,c7n.Random)
		w.Write([]byte(Context))
	})
	return nil
}
func (r *RandomInfo)CheckRadom() error {
	response, err := http.Get(r.Url)
	defer response.Body.Close()
	if err != nil {
		return err
	}
	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	type RD struct {
		Random string `json:"random"`
	}
	ran := &RD{}
	json.Unmarshal(contents,ran)
	if ran.Random != r.Random {
		return fmt.Errorf("host %s random does not match",r.Url)
	}
	return nil
}
