package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"

	"github.com/drone/go-github/github"
	"github.com/pelletier/go-toml"
)

const (
	DEFAULT_PORT            = 3457
	DEFAULT_CONFIG_LOCATION = "config.toml"
)

var config *toml.TomlTree

// main loads the config and starts listening on the specified port for github
// sent web hooks
func main() {
	var err error

	// Get the location of the config file (relative to the binary)
	configLocation := flag.String(
		"config",
		DEFAULT_CONFIG_LOCATION,
		"location of toml config file")
	flag.Parse()

	// Load the config file (config needs to be toml)
	config, err = toml.LoadFile(*configLocation)

	if err != nil {
		panic("No config file... cannot continue")
	}

	// Get the port to listen on
	portStr := config.Get("server.port").(string)
	port, _ := strconv.ParseInt(portStr, 10, 64)

	if port == 0 || port < 0 {
		port = DEFAULT_PORT
	}

	//a := config.Get("commands.develop").([]interface{})
	//fmt.Printf("a %+v\n", a)

	//for _, x := range a {
	//fmt.Printf("XXXXXXXXXXXXXXX\n\n %+v\n", x)
	//}

	fmt.Printf("-------------- starting to listen on port %d", port)
	http.Handle("/hook", http.HandlerFunc(getHook))
	err = http.ListenAndServe(fmt.Sprintf(":%d", port), nil)

	if err != nil {
		panic("Can't listen or serve!!" + err.Error())
	}
}

// getHook handle the incoming JSON request from github
func getHook(w http.ResponseWriter, req *http.Request) {
	// Get the JSON and put it into a hook struct
	decoder := json.NewDecoder(req.Body)
	var h github.PostReceiveHook
	err := decoder.Decode(&h)

	if err != nil {
		fmt.Println("ERROR: ", err.Error())
		fmt.Fprint(w, "No JSON... what? ("+err.Error()+")")

		return
	}

	// If there is a branch, run the commands
	if len(h.Branch()) > 0 {
		runCommands(h.Branch())
	}

	fmt.Fprint(w, "OK"+h.Ref)
}

// runCommands based on the branch that was in the hook
func runCommands(branch string) {
	c := config.Get("commands." + branch)

	if c == nil {
		fmt.Println("No commands to run")
		return
	}

	commands := c.([]interface{})

	log := ""

	for i, c := range commands {
		co := c.(string)
		split := strings.Split(co, " ")
		cmd := split[0]

		var out []byte
		var err error

		//	out, err = exec.Command("ls", "/tmp").Output()
		switch len(split) {
		case 0:
			// Nothing to do
		case 1:
			out, err = exec.Command(cmd).Output()
		case 2:
			out, err = exec.Command(string(cmd), string(split[1])).Output()
		default:
			s := split[1:]
			out, err = exec.Command(string(cmd), s...).Output()
		}

		if err != nil {
			fmt.Printf("err %+v\n", err)
		}
		log += fmt.Sprintf("%d. Run: %s\n", i, c)
		log += fmt.Sprintf("%d. Result: %s\n", i, out)
	}

	fmt.Println(log)
}
