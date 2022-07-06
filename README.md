# mir1

You will need a folder in root dir called "lmd_full" that has all the lakh midi data.

`go install`
`mir1 index`

### running the server

You'll need FluidSynth installed. Then run it with a soundfont that you like.
`fluidsynth /path/to/somefont.sf2`
`mir1 server`

### terminology
bucket - bin to put similar data in
chunk - small files around a certain size derived from big files
chunk index - index at top of each chunk file
file number - number that identifies an original midi file


# TODO
 - [ ] come up with better name
 - [ ] use uint32 for tick offset instead of 64??
