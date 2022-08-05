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
 - [ ] come up with better name
 - [ ] profile everything in indexing and see what can be improved
 - [ ] use map
 - [ ] validate results
 - [ ] why are the midi notes not stopping in playback on FE
 - [ ] caching?
 - [ ] 30ms just getting results
 - [ ] profile... 300ms response now is not good


### Reducing junk chords effort

Fixing the number of BS chords being created.
Initially, any time a note on/off event happened, it triggered a new chord to be saved.
This created roughly 261,397,621 per 180k files. 43 chunks files about 70MB each.

After the change, we were able to cut out 35% of the chords and chunk files
169,834,360

### Run Tests

Run all:
 - `go test -tags="e2e" ./...`
Run unit:
 - `go test  ./...`

### Run Actions Locally
 - `brew install act`
 - `act` or on M1 (`--container-architecture linux/amd64`)
