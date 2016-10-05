package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	// "net/url"
	"os"
	"regexp"
	"runtime"
	"strings"

	"github.com/urfave/cli"
)

const internalVersion = "1.0"

func traceln(v ...interface{}) {
	pc, _, _, _ := runtime.Caller(1)
	fn := runtime.FuncForPC(pc)
	fno := regexp.MustCompile(`^.*\.(.*)$`)
	fnName := fno.ReplaceAllString(fn.Name(), "$1")
	m := fmt.Sprintln(v...)
	log.Print("["+fnName+"] ", m)
}

func updateService(host string, file string, reboot bool) error {
	if file == "" {
		err := fmt.Errorf("No file provided. See --file flag for more info.")
		traceln(err)
		return err
	}

	if host == "" {
		err := fmt.Errorf("No host/ip provided. See --hosts flag for more info.")
		traceln(err)
		return err
	}

	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)
	fileWriter, err := bodyWriter.CreateFormFile("uploadfile", file)
	if err != nil {
		traceln(err)
		return err
	}

	fh, err := os.Open(file)
	if err != nil {
		traceln(err)
		return err
	}

	_, err = io.Copy(fileWriter, fh)
	if err != nil {
		traceln(err)
		return err
	}

	contentType := bodyWriter.FormDataContentType()
	bodyWriter.Close()
	url := `http://` + host + `:8080/api/v1/update/self`
	if !reboot {
		url = url + `?reboot=false`
	}

	resp, err := http.Post(url, contentType, bodyBuf)
	if err != nil {
		traceln(err)
		return err
	}

	defer resp.Body.Close()
	resp_body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		traceln(err)
		return err
	}

	traceln("[" + host + "]")
	traceln(resp.Status)
	traceln(string(resp_body))

	return nil
}

func updateRunner(host string, file string) error {
	if file == "" {
		err := fmt.Errorf("No file provided. See --file flag for more info.")
		traceln(err)
		return err
	}

	if host == "" {
		err := fmt.Errorf("No host/ip provided. See --hosts flag for more info.")
		traceln(err)
		return err
	}

	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)
	fileWriter, err := bodyWriter.CreateFormFile("uploadfile", file)
	if err != nil {
		traceln(err)
		return err
	}

	fh, err := os.Open(file)
	if err != nil {
		traceln(err)
		return err
	}

	_, err = io.Copy(fileWriter, fh)
	if err != nil {
		traceln(err)
		return err
	}

	contentType := bodyWriter.FormDataContentType()
	bodyWriter.Close()
	url := `http://` + host + `:8080/api/v1/update/runner`
	resp, err := http.Post(url, contentType, bodyBuf)
	if err != nil {
		traceln(err)
		return err
	}

	defer resp.Body.Close()
	resp_body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		traceln(err)
		return err
	}

	traceln("[" + host + "]")
	traceln(resp.Status)
	traceln(string(resp_body))
	return nil
}

func sendExecCommand(targetUrl string, cmd string, outFile string) error {
	var json = []byte(`{"cmd":"` + cmd + `"}`)
	client := &http.Client{}
	r, _ := http.NewRequest("POST", targetUrl, bytes.NewBuffer(json))
	r.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(r)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	traceln("Response status:", resp.Status)
	resp_body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		traceln(err)
		return err
	}

	traceln(string(resp_body))
	if outFile != "" {
		err := ioutil.WriteFile(outFile, resp_body, 0644)
		if err != nil {
			traceln(err)
			return err
		}
	}

	return nil
}

func main() {
	app := cli.NewApp()
	app.Name = "n1"
	app.Usage = "Client interface for `holly` service."
	app.Version = internalVersion
	app.Copyright = "(c) 2016 Chew Esmero."
	app.Commands = []cli.Command{
		{
			Name:  "update",
			Usage: "update holly module(s)",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "file",
					Value: "",
					Usage: "new `file` to upload",
				},
				cli.StringFlag{
					Name:  "hosts",
					Value: "localhost",
					Usage: "list of target `host(s)`, separated by ','",
				},
				cli.BoolFlag{
					Name:  "reboot",
					Usage: "should reboot after update (default: true)",
				},
			},
			ArgsUsage: "[self|runner]",
			Action: func(c *cli.Context) error {
				if c.NArg() > 0 {
					switch c.Args().Get(0) {
					case "self":
						hosts := strings.Split(c.String("hosts"), ",")
						for _, host := range hosts {
							reboot := true
							if c.IsSet("reboot") && c.Bool("reboot") == false {
								reboot = false
							}

							updateService(host, c.String("file"), reboot)
						}
					case "runner":
						hosts := strings.Split(c.String("hosts"), ",")
						for _, host := range hosts {
							updateRunner(host, c.String("file"))
						}
					default:
						traceln("Valid argument is either 'self' or 'runner' or none.")
						return nil
					}
				} else {
					traceln("Not yet supported.")
				}

				return nil
			},
		},
		{
			Name:  "exec",
			Usage: "remote execute command",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "cmd",
					Value: "",
					Usage: "`command` to execute",
				},
				cli.StringFlag{
					Name:  "url",
					Value: "",
					Usage: "target `url`",
				},
				cli.StringFlag{
					Name:  "out",
					Value: "",
					Usage: "write output to `file`",
				},
			},
			Action: func(c *cli.Context) error {
				if c.IsSet("url") && c.IsSet("cmd") {
					err := sendExecCommand(c.String("url"), c.String("cmd"), c.String("out"))
					if err != nil {
						traceln(err)
					}
					return err
				} else {
					traceln("Flags 'url' and/or 'cmd' not set.")
				}
				return nil
			},
		},
	}

	app.Run(os.Args)
}
