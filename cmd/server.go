package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"gitlab.com/gomidi/midi/v2"
	_ "gitlab.com/gomidi/midi/v2/drivers/rtmididrv" // autoregisters driver
)

func init() {
	rootCmd.AddCommand(serverCmd)
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Runs server",
	Long:  `Runs server`,
	Run: func(cmd *cobra.Command, args []string) {
		startServer()
	},
}

func loadMainState() {
	// var chunks []Chunk
}

func startServer() {
	defer midi.CloseDriver()
	in, err := midi.InPort(0)
	if err != nil {
		fmt.Println("can't find VMPK")
		return
	}

	loadMainState()

	// load our main state
	// manage current midi notes
	// retrieve
	// concurrency, also, debounce

	stop, err := midi.ListenTo(in, func(msg midi.Message, timestampms int32) {
		var bt []byte
		var ch, key, vel uint8
		switch {
		case msg.GetSysEx(&bt):
			fmt.Printf("got sysex: % X\n", bt)
		case msg.GetNoteStart(&ch, &key, &vel):
			fmt.Printf("starting note %s on channel %v with velocity %v\n", midi.Note(key), ch, vel)
		case msg.GetNoteEnd(&ch, &key):
			fmt.Printf("ending note %s on channel %v\n", midi.Note(key), ch)
		default:
			// ignore
		}
	}, midi.UseSysEx())

	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
		return
	}

	time.Sleep(time.Second * 5000) // lol
	stop()
}
