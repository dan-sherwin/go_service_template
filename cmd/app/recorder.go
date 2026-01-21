package app

import (
	"fmt"
	"log"
	"os"
	"runtime/trace"

	"scm.dev.dsherwin.net/dsherwin/go_service_template/cmd/app/rpc"
)

type (
	RecorderCommandDef struct {
		DumpRecorder DumpRecorderCommand `cmd:"" name:"dumpRecorder" hidden:"" help:"Dump the flight recorder to a file"`
	}
	DumpRecorderCommand struct {
		File string `arg:"" help:"File to dump recorder to"`
	}
)

var (
	recorder     *trace.FlightRecorder
	dumpRecorder = make(chan string, 1)
)

func init() {
	rpc.RegisterName("Recorder", &DumpRecorderCommand{})
}

func StartRecorder() {
	recorder = trace.NewFlightRecorder(trace.FlightRecorderConfig{})
	if err := recorder.Start(); err != nil {
		log.Printf("Failed to start flight recorder: %v", err)
	}
	startRecorderDumpWatch()
}

func PanicDefer() {
	if r := recover(); r != nil {
		if n, err := dumpRecorderToFile("panic_trace.out"); err != nil {
			log.Println("Failed to create trace file:", err)
		} else {
			log.Printf("Flight recorder trace saved to panic_trace.out (%d bytes)\n", n)
		}
		panic(r)
	}
}

func startRecorderDumpWatch() {
	go func() {
		for {
			recorderFile := <-dumpRecorder
			if _, err := dumpRecorderToFile(recorderFile); err != nil {
				log.Printf("Failed to dump recorder to file %s: %v", recorderFile, err)
			}
		}
	}()
}

func dumpRecorderToFile(filePath string) (int64, error) {
	f, err := os.Create(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to create trace file %s: %w", filePath, err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("Failed to close trace file %s: %v", filePath, err)
		}
	}()
	n, err := recorder.WriteTo(f)
	if err != nil {
		return 0, fmt.Errorf("failed to write trace to file %s: %w", filePath, err)
	}
	return n, nil
}

func (c *DumpRecorderCommand) Run() error {
	err := rpc.Call("Recorder.DumpRecorder", c.File, nil)
	if err != nil {
		fmt.Printf("Error dumping recorder: %s\n", err)
		return err
	}
	fmt.Println("Dump request sent")
	return nil
}

func (c *DumpRecorderCommand) DumpRecorder(file string, _ *struct{}) error {
	dumpRecorder <- file
	return nil
}
