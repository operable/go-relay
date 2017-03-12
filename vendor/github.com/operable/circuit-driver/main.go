package main

import (
	"github.com/operable/circuit-driver/api"
	"os"
)

const (
	ExitBadLogger = 1
	ExitBadRead
	ExitBadExec
	ExitBadWrite
)

func main() {
	decoder := api.WrapDecoder(os.Stdin)
	encoder := api.WrapEncoder(os.Stdout)
	var driver api.BlockingDriver
	for {
		var request api.ExecRequest
		if err := decoder.DecodeRequest(&request); err != nil {
			os.Exit(ExitBadRead)
		}
		if request.GetDie() == true {
			os.Exit(0)
		}
		execResult, err := driver.Run(&request)
		encodeErr := encoder.EncodeResult(&execResult)
		if err != nil {
			os.Exit(ExitBadExec)
		}
		if encodeErr != nil {
			os.Exit(ExitBadWrite)
		}

	}
}
