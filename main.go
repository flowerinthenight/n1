package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
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

func uploadFileToEndpoint(url string, file string) ([]byte, string, error) {
	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)
	fileWriter, err := bodyWriter.CreateFormFile("uploadfile", file)
	if err != nil {
		traceln(err)
		return nil, "", err
	}

	fh, err := os.Open(file)
	if err != nil {
		traceln(err)
		return nil, "", err
	}

	_, err = io.Copy(fileWriter, fh)
	if err != nil {
		traceln(err)
		return nil, "", err
	}

	contentType := bodyWriter.FormDataContentType()
	bodyWriter.Close()
	resp, err := http.Post(url, contentType, bodyBuf)
	if err != nil {
		traceln(err)
		return nil, "", err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	return body, resp.Status, err
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

	url := `http://` + host + `:8080/api/v1/update/runner`
	body, status, err := uploadFileToEndpoint(url, file)
	if err != nil {
		traceln(err)
		return err
	}

	traceln("[" + host + "]")
	traceln(status)
	traceln(string(body))
	return nil
}

func updateConf(host string, file string) error {
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

	url := `http://` + host + `:8080/api/v1/update/conf`
	body, status, err := uploadFileToEndpoint(url, file)
	if err != nil {
		traceln(err)
		return err
	}

	traceln("[" + host + "]")
	traceln(status)
	traceln(string(body))
	return nil
}

func sendGetOctetStream(url string, data string) ([]byte, string, error) {
	// Use byte as payload to accommodate all sorts of file naming weirdness.
	var payload = []byte(data)
	client := &http.Client{}
	r, _ := http.NewRequest("GET", url, bytes.NewBuffer(payload))
	r.Header.Add("Content-Type", "application/octet-stream")
	resp, err := client.Do(r)
	if err != nil {
		traceln(err)
		return nil, "", err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		traceln(err)
		return nil, "", err
	}

	return body, resp.Status, nil
}

func sendExecCommand(host string, cmd string, outFile string) error {
	url := `http://` + host + `:8080/api/v1/exec`
	body, status, err := sendGetOctetStream(url, cmd)
	if err != nil {
		traceln(err)
		return err
	}

	traceln(status)
	traceln(string(body))
	if outFile != "" {
		err := ioutil.WriteFile(outFile, body, 0644)
		if err != nil {
			traceln(err)
			return err
		}
	}

	return nil
}

func sendFileStats(host string, fileList string, outFile string) error {
	url := `http://` + host + `:8080/api/v1/filestat`
	body, status, err := sendGetOctetStream(url, fileList)
	if err != nil {
		traceln(err)
		return err
	}

	traceln(status)
	traceln(string(body))
	if outFile != "" {
		err := ioutil.WriteFile(outFile, body, 0644)
		if err != nil {
			traceln(err)
			return err
		}
	}

	return nil
}

func sendReadFile(host string, file string, outFile string) error {
	url := `http://` + host + `:8080/api/v1/readfile`
	body, status, err := sendGetOctetStream(url, file)
	if err != nil {
		traceln(err)
		return err
	}

	traceln(status)
	traceln(string(body))
	if outFile != "" {
		err := ioutil.WriteFile(outFile, body, 0644)
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
			ArgsUsage: "[self|runner|conf]",
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
					case "conf":
						hosts := strings.Split(c.String("hosts"), ",")
						for _, host := range hosts {
							updateConf(host, c.String("file"))
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
					Name:  "host",
					Value: "localhost",
					Usage: "target `host`",
				},
				cli.StringFlag{
					Name:  "out",
					Value: "",
					Usage: "write output to `file`",
				},
			},
			Action: func(c *cli.Context) error {
				if !c.IsSet("cmd") {
					traceln("Flag 'cmd' not set.")
					return nil
				}

				// Todo: support list of hosts as target.
				return sendExecCommand(c.String("host"), c.String("cmd"), c.String("out"))
			},
		},
		{
			Name:  "stat",
			Usage: "get file stats",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "files",
					Value: "",
					Usage: "comma-separated file list",
				},
				cli.StringFlag{
					Name:  "host",
					Value: "localhost",
					Usage: "target `host`",
				},
				cli.StringFlag{
					Name:  "out",
					Value: "",
					Usage: "write output to `file`",
				},
			},
			Action: func(c *cli.Context) error {
				if !c.IsSet("files") {
					traceln("Flag 'files' not set.")
					return fmt.Errorf("Flag 'files' not set.")
				}

				// Todo: support list of hosts as target.
				return sendFileStats(c.String("host"), c.String("files"), c.String("out"))
			},
		},
		{
			Name:  "read",
			Usage: "read a file",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "file",
					Value: "",
					Usage: "file to read",
				},
				cli.StringFlag{
					Name:  "host",
					Value: "localhost",
					Usage: "target `host`",
				},
				cli.StringFlag{
					Name:  "out",
					Value: "",
					Usage: "write output to `file`",
				},
			},
			Action: func(c *cli.Context) error {
				if !c.IsSet("file") {
					traceln("Flag 'file' not set.")
					return fmt.Errorf("Flag 'file' not set.")
				}

				// Todo: support list of hosts as target.
				return sendReadFile(c.String("host"), c.String("file"), c.String("out"))
			},
		},
		{
			Name:  "version",
			Usage: "get 'holly' version",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "host",
					Value: "localhost",
					Usage: "target `host`",
				},
			},
			Action: func(c *cli.Context) error {
				client := &http.Client{}
				r, _ := http.NewRequest("GET", `http://`+c.String("host")+`:8080/api/v1/version`, nil)
				resp, err := client.Do(r)
				if err != nil {
					traceln(err)
					return err
				}

				defer resp.Body.Close()
				body, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					traceln(err)
					return err
				}

				traceln(resp.Status)
				traceln(string(body))
				return nil
			},
		},
	}

	app.Run(os.Args)
}
