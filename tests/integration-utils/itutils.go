// Package itutils provides utility functions and configuration for integration
// testing with the Deis open source PaaS.

package itutils

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/ThomasRooney/gexpect"
	gson "github.com/bitly/go-simplejson"
	"github.com/deis/deis/tests/utils"
)

// Deis points to the CLI used to run tests.
var Deis = "deis "

// DeisTestConfig allows tests to be repeated against different
// targets, with different example apps, using specific credentials, and so on.
type DeisTestConfig struct {
	AuthKey     string
	Hosts       string
	Domain      string
	SSHKey      string
	ClusterName string
	UserName    string
	Password    string
	Email       string
	ExampleApp  string
	AppName     string
	ProcessNum  string
	ImageID     string
	Version     string
	AppUser     string
}

// randomApp is used for the test run if DEIS_TEST_APP isn't set
var randomApp = GetRandomApp()

// GetGlobalConfig returns a test configuration object.
func GetGlobalConfig() *DeisTestConfig {
	authKey := os.Getenv("DEIS_TEST_AUTH_KEY")
	if authKey == "" {
		authKey = "deis"
	}
	hosts := os.Getenv("DEIS_TEST_HOSTS")
	if hosts == "" {
		hosts = "172.17.8.100"
	}
	domain := os.Getenv("DEIS_TEST_DOMAIN")
	if domain == "" {
		domain = "local.deisapp.com"
	}
	sshKey := os.Getenv("DEIS_TEST_SSH_KEY")
	if sshKey == "" {
		sshKey = "~/.vagrant.d/insecure_private_key"
	}
	exampleApp := os.Getenv("DEIS_TEST_APP")
	if exampleApp == "" {
		exampleApp = randomApp
	}
	var envCfg = DeisTestConfig{
		AuthKey:     authKey,
		Hosts:       hosts,
		Domain:      domain,
		SSHKey:      sshKey,
		ClusterName: "dev",
		UserName:    "test",
		Password:    "asdf1234",
		Email:       "test@test.co.nz",
		ExampleApp:  exampleApp,
		AppName:     "sample",
		ProcessNum:  "2",
		ImageID:     "buildtest",
		Version:     "2",
		AppUser:     "test1",
	}
	return &envCfg
}

// Curl connects to a Deis endpoint to see if the example app is running.
func Curl(t *testing.T, params *DeisTestConfig) {
	url := "http://" + params.AppName + "." + params.Domain
	response, err := http.Get(url)
	if err != nil {
		t.Fatalf("not reachable:\n%v", err)
	}
	body, err := ioutil.ReadAll(response.Body)
	fmt.Println(string(body))
	if !strings.Contains(string(body), "Powered by Deis") {
		t.Fatalf("App not started")
	}
}

// AuthCancel tests whether `deis auth:cancel` destroys a user's account.
func AuthCancel(t *testing.T, params *DeisTestConfig) {
	fmt.Println("deis auth:cancel")
	child, err := gexpect.Spawn(Deis + " auth:cancel")
	if err != nil {
		t.Fatalf("command not started\n%v", err)
	}
	fmt.Println("username:")
	err = child.Expect("username:")
	if err != nil {
		t.Fatalf("expect username failed\n%v", err)
	}
	child.SendLine(params.UserName)
	fmt.Print("password:")
	err = child.Expect("password:")
	if err != nil {
		t.Fatalf("expect password failed\n%v", err)
	}
	child.SendLine(params.Password)
	err = child.ExpectRegex("(y/n)")
	if err != nil {
		t.Fatalf("expect cancel \n%v", err)
	}
	child.SendLine("y")
	err = child.Expect("Account cancelled")
	if err != nil {
		t.Fatalf("command executiuon failed\n%v", err)
	}
	child.Close()

}

// CheckList executes a command and optionally tests whether its output contains
// a given string.
func CheckList(t *testing.T, params interface{}, cmd, contain string, notflag bool) {
	var cmdBuf bytes.Buffer
	tmpl := template.Must(template.New("cmd").Parse(cmd))
	if err := tmpl.Execute(&cmdBuf, params); err != nil {
		t.Fatal(err)
	}
	cmdString := cmdBuf.String()
	fmt.Println(cmdString)
	var cmdl *exec.Cmd
	if strings.Contains(cmd, "cat") {
		cmdl = exec.Command("sh", "-c", cmdString)
	} else {
		cmdl = exec.Command("sh", "-c", Deis+cmdString)
	}
	if stdout, _, err := utils.RunCommandWithStdoutStderr(cmdl); err == nil {
		if strings.Contains(stdout.String(), contain) == notflag {
			t.Fatal(err)
		}
	} else {
		t.Fatal(err)
	}
}

// Execute takes command string and parameters required to execute the command,
// a failflag to check whether the command is expected to fail, and an expect
// string to check whether the command has failed according to failflag.
//
// If failflag is true and the command failed, check the stdout and stderr for
// the expect string.
func Execute(t *testing.T, cmd string, params interface{}, failFlag bool, expect string) {
	var cmdBuf bytes.Buffer
	tmpl := template.Must(template.New("cmd").Parse(cmd))
	if err := tmpl.Execute(&cmdBuf, params); err != nil {
		t.Fatal(err)
	}
	cmdString := cmdBuf.String()
	fmt.Println(cmdString)
	var cmdl *exec.Cmd
	if strings.Contains(cmd, "git") {
		cmdl = exec.Command("sh", "-c", cmdString)
	} else {
		cmdl = exec.Command("sh", "-c", Deis+cmdString)
	}

	switch failFlag {
	case true:
		if stdout, stderr, err := utils.RunCommandWithStdoutStderr(cmdl); err != nil {
			if strings.Contains(stdout.String(), expect) || strings.Contains(stderr.String(), expect) {
				fmt.Println("(Error expected...ok)")
			} else {
				t.Fatalf("Failed:\n%v", err)
			}
		} else {
			if strings.Contains(stdout.String(), expect) || strings.Contains(stderr.String(), expect) {
				fmt.Println("(Error expected...ok)" + expect)
			} else {
				t.Fatalf("Failed:\n%v", err)
			}
		}
	case false:
		if _, _, err := utils.RunCommandWithStdoutStderr(cmdl); err != nil {
			t.Fatalf("Failed:\n%v", err)
		} else {
			fmt.Println("ok")
		}
	}
}

// AppsDestroyTest destroys a Deis app and checks that it was successful.
func AppsDestroyTest(t *testing.T, params *DeisTestConfig) {
	cmd := GetCommand("apps", "destroy")
	if err := utils.Chdir(params.ExampleApp); err != nil {
		t.Fatalf("Failed:\n%v", err)
	}
	Execute(t, cmd, params, false, "")
	if err := utils.Chdir(".."); err != nil {
		t.Fatalf("Failed:\n%v", err)
	}
	if err := utils.Rmdir(params.ExampleApp); err != nil {
		t.Fatalf("Failed:\n%v", err)
	}
}

// GetCommand fetches the given command by type and name from a JSON resource.
func GetCommand(cmdtype, cmd string) string {
	js, _ := gson.NewJson(utils.GetFileBytes("testconfig.json"))
	command, _ := js.Get("commands").Get(cmdtype).Get(cmd).String()
	return command
}

// GetRandomApp returns a known working example app at random for testing.
func GetRandomApp() string {
	rand.Seed(int64(time.Now().Unix()))
	apps := []string{
		"example-clojure-ring",
		// "example-dart",
		"example-dockerfile-python",
		"example-go",
		"example-java-jetty",
		"example-nodejs-express",
		// "example-php",
		"example-play",
		"example-python-django",
		"example-python-flask",
		"example-ruby-sinatra",
		"example-scala",
	}
	return apps[rand.Intn(len(apps))]
}
