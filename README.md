# harmondex

`go install`
`harmondex index path/to/src/files`
`harmondex serve path/to/src/files`

### running the server

You'll need FluidSynth installed. Then run it with a soundfont that you like.
`fluidsynth /path/to/somefont.sf2`
`harmondex server`

### terminology
bucket - bin to put similar data in
chunk - small files around a certain size derived from big files
chunk index - index at top of each chunk file
file number - number that identifies an original midi file


# TODO
 - [ ] return paginated results with page info
 - [ ] partial chords might be accidentally prioritized (there is no look ahead)
 - [ ] come up with better name
 - [ ] profile everything in indexing and see what can be improved
 - [ ] use map
 - [ ] validate results
 - [ ] why are the midi notes not stopping in playback on FE
 - [ ] caching?
 - [ ] 30ms just getting results
 - [ ] profile... 300ms response now is not good
 - [ ] write some e2e test?
 - [ ] run tests on CI?
