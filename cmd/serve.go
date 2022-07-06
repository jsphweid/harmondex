package cmd

import (
	"fmt"
	"time"

	"github.com/bep/debounce"
	"github.com/jsphweid/mir1/chord"
	"github.com/spf13/cobra"
	"gitlab.com/gomidi/midi/v2"
	_ "gitlab.com/gomidi/midi/v2/drivers/rtmididrv" // autoregisters driver
)

func init() {
	rootCmd.AddCommand(serveCmd)
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "serves",
	Long:  `serves`,
	Run: func(cmd *cobra.Command, args []string) {
		serve()
	},
}

func serve() {
	defer midi.CloseDriver()
	in, err := midi.InPort(0)
	if err != nil {
		fmt.Println("can't find VMPK")
		return
	}

	allChunks := loadAllChunks()
	onNotes := make(chord.OnNotes)

	play := func() {
		fmt.Printf("onNotes: %v\n", onNotes)
		chords := findChords(allChunks, onNotes)
		fmt.Printf("chords: %v\n", chords)
	}

	debounced := debounce.New(100 * time.Millisecond)

	stop, err := midi.ListenTo(in, func(msg midi.Message, timestampms int32) {
		var bt []byte
		var ch, key, vel uint8
		switch {
		case msg.GetSysEx(&bt):
			fmt.Printf("got sysex: % X\n", bt)
		case msg.GetNoteStart(&ch, &key, &vel):
			onNotes[key] = true
			debounced(play)
		case msg.GetNoteEnd(&ch, &key):
			delete(onNotes, key)
			debounced(play)
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
