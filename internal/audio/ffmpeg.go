package audio

import (
	"os/exec"
	"strings"
)

// FFmpeg provides a fluent API for building ffmpeg commands
// Ported from abb_ia project
type FFmpeg struct {
	input  ffmpegInput
	output ffmpegOutput
	params ffmpegParams
	cmd    *exec.Cmd
}

type ffmpegInput struct {
	fileNames []string
	args      string
}

type ffmpegOutput struct {
	fileName string
	args     string
}

type ffmpegParams struct {
	args string
}

// NewFFmpeg creates a new FFmpeg command builder
func NewFFmpeg() *FFmpeg {
	return &FFmpeg{
		input:  ffmpegInput{},
		output: ffmpegOutput{},
		params: ffmpegParams{},
		cmd:    nil,
	}
}

// Input adds an input file with optional arguments
func (f *FFmpeg) Input(fileName string, args string) *FFmpeg {
	f.input.fileNames = append(f.input.fileNames, fileName)
	f.input.args += args
	return f
}

// Output sets the output file with optional arguments
func (f *FFmpeg) Output(fileName string, args string) *FFmpeg {
	f.output.fileName = fileName
	f.output.args = args
	return f
}

// Params adds additional parameters
func (f *FFmpeg) Params(args string) *FFmpeg {
	f.params.args += " " + args
	return f
}

// SendProgressTo adds progress reporting to a URL
func (f *FFmpeg) SendProgressTo(url string) *FFmpeg {
	f.params.args += " -progress " + url
	return f
}

// Overwrite sets whether to overwrite output files
func (f *FFmpeg) Overwrite(b bool) *FFmpeg {
	if b {
		f.params.args += " -y"
	}
	return f
}

// Run executes the ffmpeg command
func (f *FFmpeg) Run() (string, error) {
	args := newArgs().
		appendArgs(f.params.args).
		appendArgs(f.input.args)
	for _, fileName := range f.input.fileNames {
		args = args.appendArgs("-i").appendFileName(fileName)
	}
	args = args.appendArgs(f.output.args).appendFileName(f.output.fileName)
	f.cmd = exec.Command("ffmpeg", args.toSlice()...)
	out, err := f.cmd.CombinedOutput()
	if err != nil {
		return string(out), err
	}
	return string(out), nil
}

// Kill terminates the ffmpeg process
func (f *FFmpeg) Kill() error {
	if f.cmd != nil && f.cmd.Process != nil {
		return f.cmd.Process.Kill()
	}
	return nil
}

// args helper for building command arguments
type args struct {
	items []string
}

func newArgs() *args {
	return &args{}
}

func (a *args) appendFileName(arg string) *args {
	a.items = append(a.items, arg)
	return a
}

func (a *args) appendArgs(arg ...string) *args {
	for _, ar := range arg {
		ars := strings.Fields(ar)
		a.items = append(a.items, ars...)
	}
	return a
}

func (a *args) toSlice() []string {
	return a.items
}
