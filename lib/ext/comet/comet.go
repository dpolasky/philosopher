package comet

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"philosopher/lib/met"
	"philosopher/lib/msg"
	"philosopher/lib/sys"

	"github.com/sirupsen/logrus"
)

// Comet represents the tool configuration
type Comet struct {
	DefaultBin   string
	DefaultParam string
	Win32        string
	Win64        string
	Unix64       string
	WinParam     string
	UnixParam    string
}

// New constructor
func New(temp string) Comet {

	var self Comet

	self.DefaultBin = ""
	self.DefaultParam = ""
	self.Win32 = temp + string(filepath.Separator) + "comet.2019011.win32.exe"
	self.Win64 = temp + string(filepath.Separator) + "comet.2019011.win64.exe"
	self.Unix64 = temp + string(filepath.Separator) + "comet.2019011.linux.exe"
	self.WinParam = temp + string(filepath.Separator) + "comet.params.txt"
	self.UnixParam = temp + string(filepath.Separator) + "comet.params"

	return self
}

// Run is the Comet main entry point
func Run(m met.Data, args []string) met.Data {

	var cmt = New(m.Temp)

	if len(m.Comet.Param) < 1 || !m.Comet.Print && len(args) < 1 {
		msg.Comet(errors.New(""), "error")
	}

	// deploy the binaries
	cmt.Deploy(m.Arch)

	if m.Comet.Print {
		logrus.Info("Printing parameter file")
		sys.CopyFile(cmt.DefaultParam, filepath.Base(cmt.DefaultParam))
		return m
	}

	// collect and store the mz files
	m.Comet.RawFiles = args

	// convert the param file to binary and store it in meta
	var binFile []byte
	paramAbs, _ := filepath.Abs(m.Comet.Param)
	binFile, e := ioutil.ReadFile(paramAbs)
	if e != nil {
		msg.Custom(e, "error")
	}
	m.Comet.ParamFile = binFile

	if !m.Comet.NoIndex {
		var extFlag = true

		// the indexing will help later in case other commands are used for qunatification
		// it will provide easy and fast access to mz data
		for _, i := range args {
			if strings.Contains(i, "mzML") {
				extFlag = false
			}
		}

		if !extFlag {
			//logrus.Info("Indexing spectra: please wait, this can take a few minutes")
			//raw.IndexMz(args)
		} else {
			logrus.Info("mz file format not supported for indexing, skipping the indexing")
		}
	}

	// run comet
	cmt.Execute(args, m.Comet.Param)

	return m
}

// Execute is the main function to execute Comet
func (c *Comet) Execute(cmdArgs []string, param string) {

	par := fmt.Sprintf("-P%s", param)
	args := []string{par}

	for _, i := range cmdArgs {
		file, _ := filepath.Abs(i)
		args = append(args, file)
	}

	run := exec.Command(c.DefaultBin, args...)
	run.Stdout = os.Stdout
	run.Stderr = os.Stderr
	run.Start()
	_ = run.Wait()

}
