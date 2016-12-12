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
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/urfave/cli"
)

const (
	name            = "n1"
	internalVersion = "1.0"
	usage           = "Client interface for 'holly' service."
	copyright       = "(c) 2016 Chew Esmero."
)

func traceln(v ...interface{}) {
	pc, _, _, _ := runtime.Caller(1)
	fn := runtime.FuncForPC(pc)
	fno := regexp.MustCompile(`^.*\.(.*)$`)
	fnName := fno.ReplaceAllString(fn.Name(), "$1")
	m := fmt.Sprintln(v...)
	log.Print("["+fnName+"] ", m)
}

// Use http methods for the 'method' argument.
// Returns the body, response status, and error.
func httpOctetStream(method, url, data string) ([]byte, string, error) {
	// Use byte as payload to accommodate all sorts of file naming weirdness.
	var payload = []byte(data)
	client := &http.Client{}
	r, _ := http.NewRequest(method, url, bytes.NewBuffer(payload))
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

// Returns the body, response status, and error.
func uploadFileToEndpoint(url string, file string) ([]byte, string, error) {
	bodyBuf := &bytes.Buffer{}
	var rs string
	bodyWriter := multipart.NewWriter(bodyBuf)
	fileWriter, err := bodyWriter.CreateFormFile("uploadfile", file)
	if err != nil {
		traceln(err)
		return nil, rs, err
	}

	fh, err := os.Open(file)
	if err != nil {
		traceln(err)
		return nil, rs, err
	}

	_, err = io.Copy(fileWriter, fh)
	if err != nil {
		traceln(err)
		return nil, rs, err
	}

	contentType := bodyWriter.FormDataContentType()
	bodyWriter.Close()
	resp, err := http.Post(url, contentType, bodyBuf)
	if err != nil {
		traceln(err)
		return nil, rs, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	return body, resp.Status, err
}

// Upload some file to some location (generic upload).
func uploadFileGeneric(host string, file string, path string) error {
	if host == "" {
		err := fmt.Errorf("No host/ip provided. See --hosts flag for more info.")
		traceln(err)
		return err
	}

	if file == "" {
		err := fmt.Errorf("No file provided. See --file flag for more info.")
		traceln(err)
		return err
	}

	url := `http://` + host + `:8080/api/v1/upload`
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

	loc, err := bodyWriter.CreateFormField("path")
	if err != nil {
		traceln(err)
		return err
	}

	_, err = loc.Write([]byte(path))
	if err != nil {
		traceln(err)
		return err
	}

	contentType := bodyWriter.FormDataContentType()
	bodyWriter.Close()
	resp, err := http.Post(url, contentType, bodyBuf)
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

	str := fmt.Sprintf("%s", body)
	traceln(str)
	return nil
}

func updateService(host string, file string, reboot bool) error {
	if host == "" {
		err := fmt.Errorf("No host/ip provided. See --hosts flag for more info.")
		traceln(err)
		return err
	}

	if file == "" {
		err := fmt.Errorf("No file provided. See --file flag for more info.")
		traceln(err)
		return err
	}

	url := `http://` + host + `:8080/api/v1/update/self`
	if !reboot {
		// We reboot the target system by default.
		url = url + `?reboot=false`
	}

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

func updateRunner(host string, file string) error {
	if host == "" {
		err := fmt.Errorf("No host/ip provided. See --hosts flag for more info.")
		traceln(err)
		return err
	}

	upfile := file
	if file == "" {
		// If no file provided, we download the runner to tempdir.
		traceln("tempdir:", os.TempDir())
		f, err := downloadRunner(os.TempDir(), "")
		if err != nil {
			traceln(err)
			return err
		}

		upfile = os.TempDir() + `\` + f
	}

	url := `http://` + host + `:8080/api/v1/update/runner`
	body, status, err := uploadFileToEndpoint(url, upfile)
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
	if host == "" {
		err := fmt.Errorf("No host/ip provided. See --hosts flag for more info.")
		traceln(err)
		return err
	}

	if file == "" {
		err := fmt.Errorf("No file provided. See --file flag for more info.")
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

func sendExecCommand(host, cmd, outFile string, interactive, wait bool, waitms int) error {
	url := `http://` + host + `:8080/api/v1/exec`
	if interactive {
		url = url + `?interactive=true`
		shouldWait := "true"
		if !wait {
			shouldWait = "false"
		}

		url = url + `&wait=` + shouldWait
		url = url + `&waitms=` + fmt.Sprintf("%d", waitms)
	}

	body, status, err := httpOctetStream("GET", url, cmd)
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
	body, status, err := httpOctetStream("GET", url, fileList)
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
	body, status, err := httpOctetStream("GET", url, file)
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

// Returns the filename when download succeeds.
func downloadRunner(targetDir string, fileUrl string) (string, error) {
	if targetDir == "" {
		traceln("Please provide a target directory.")
		return "", nil
	}

	url := fileUrl
	if url == "" {
		// Default to 64bit runner.
		url = `https://gitlab-ci-multi-runner-downloads.s3.amazonaws.com/latest/binaries/gitlab-ci-multi-runner-windows-amd64.exe`
	}

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	_, f := filepath.Split(url)
	if len(f) == 0 {
		traceln("Cannot determine filename from url.")
		return "", nil
	}

	fp := targetDir + `\` + f
	traceln("target:", fp)
	out, err := os.Create(fp)
	if err != nil {
		return "", err
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}

	defer out.Close()
	return f, nil
}

func main() {
	app := cli.NewApp()
	app.Name = name
	app.Usage = usage
	app.Version = internalVersion
	app.Copyright = copyright
	app.Commands = []cli.Command{
		{
			Name:  "runner",
			Usage: "download gitlab runner",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "dir",
					Value: "",
					Usage: "target directory",
				},
				cli.StringFlag{
					Name:  "url",
					Value: "",
					Usage: "file url to download (default: 64bit runner)",
				},
			},
			Action: func(c *cli.Context) error {
				_, err := downloadRunner(c.String("dir"), c.String("url"))
				return err
			},
		},
		{
			Name:  "update",
			Usage: "update 'holly' module(s)",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "file",
					Value: "",
					Usage: "`file` to upload ([runner] option: download latest x64 when empty)",
				},
				cli.StringFlag{
					Name:  "hosts",
					Value: "localhost",
					Usage: "list of target `host(s)`, separated by ','",
				},
				cli.BoolFlag{
					Name:  "reboot",
					Usage: "should reboot after update (default: true for [self] option)",
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
			Name:  "upload",
			Usage: "update file to 'holly'",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "file",
					Value: "",
					Usage: "new `file` to upload",
				},
				cli.StringFlag{
					Name:  "path",
					Value: "root",
					Usage: "file destination path",
				},
				cli.StringFlag{
					Name:  "host",
					Value: "localhost",
					Usage: "upload destination host",
				},
			},
			Action: func(c *cli.Context) error {
				if !c.IsSet("file") {
					traceln("Flag 'file' not set.")
					return nil
				}

				// Todo: support list of hosts as target, and list of file-path pairs.
				return uploadFileGeneric(c.String("host"), c.String("file"), c.String("path"))
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
				cli.BoolFlag{
					Name:  "interactive",
					Usage: "run as interactive (default: false)",
				},
				cli.BoolFlag{
					Name:  "wait",
					Usage: "wait for cmd to exit (default: true)",
				},
				cli.IntFlag{
					Name:  "waitms",
					Value: 5000,
					Usage: "wait `timeout` in ms",
				},
			},
			Action: func(c *cli.Context) error {
				if !c.IsSet("cmd") {
					traceln("Flag 'cmd' not set.")
					return nil
				}

				interactive := false
				wait := true
				if c.IsSet("interactive") {
					interactive = c.Bool("interactive")
				}

				if c.IsSet("wait") {
					wait = c.Bool("wait")
				}

				// Todo: support list of hosts as target.
				return sendExecCommand(c.String("host"), c.String("cmd"), c.String("out"), interactive, wait, c.Int("waitms"))
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
