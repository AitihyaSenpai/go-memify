package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/Nadim147c/fang"
	"github.com/spf13/cobra"
)

var (
	padding int     = 200
	ratio   float64 = 4
	font    string  = "Impact"
	top     string
	bottom  string
	left    string
	right   string
)

func init() {
	flag := cmd.Flags()
	flag.IntVarP(&padding, "padding", "p", padding, "Padding around the text")
	flag.Float64VarP(&ratio, "ratio", "a", ratio, "Aspect ratio of text image")
	flag.StringVarP(&font, "font", "f", font, "Text font to use")
	flag.StringVarP(&top, "top", "t", "", "Add text to the top")
	flag.StringVarP(&bottom, "bottom", "b", "", "Add text to the bottom")
	flag.StringVarP(&left, "left", "l", "", "Add text to the left")
	flag.StringVarP(&right, "right", "r", "", "Add text to the right")
	cmd.MarkFlagsMutuallyExclusive("top", "bottom", "left", "right")
}

func first(texts ...string) (string, error) {
	for _, s := range texts {
		if s != "" {
			return s, nil
		}
	}
	return "", errors.New("Please provide a valid text")
}

var cmd = &cobra.Command{
	Use: "memify [--flags] <path/to/video>",
	Example: `
	# Create a meme with top text
	memify --top "My funny meme!" ~/funny-meme-clip.mp4
	`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		text, err := first(top, bottom, left, right)
		if err != nil {
			return nil
		}

		const size = 1000

		buf := bytes.NewBuffer(nil)
		magickCmd := exec.CommandContext(
			cmd.Context(),
			"magick",
			"-size", fmt.Sprintf("%.0fx%d", size*ratio, size),
			"xc:white",
			"-gravity", "center",
			"-font", font,
			"-fill", "black",
			"caption:"+text,
			"-colorspace", "sRGB",
			"-composite",
			"png:-", // write output to stdout as PNG
		)
		magickCmd.Stdout = buf
		magickCmd.Stderr = os.Stderr

		if err := magickCmd.Run(); err != nil {
			return err
		}

		output := asciiFileName(text) + "-meme.mp4"

		filterComplex := fmt.Sprintf("[1:v]pad=iw+%d:ih+%d:%d:%d:white[filterdText];\n", padding*2, padding*2, padding, padding)
		switch text {
		case top:
			filterComplex += `
			[0:v]scale=1000:-2[video];
			[filterdText]scale=1000:-2[text];
			[text][video]vstack=inputs=2[out];
			[out]format=yuv420p
			`
		case bottom:
			filterComplex += `
			[0:v]scale=1000:-2[video];
			[filterdText]scale=1000:-2[text];
			[video][text]vstack=inputs=2[out];
			[out]format=yuv420p
			`
		case left:
			filterComplex += `
			[0:v]scale=-2:1000[video];
			[filterdText]scale=-2:1000[text];
			[text][video]hstack=inputs=2[out];
			[out]format=yuv420p
			`
		case right:
			filterComplex += `
			[0:v]scale=-2:1000[video];
			[filterdText]scale=-2:1000[text];
			[video][text]hstack=inputs=2[out];
			[out]format=yuv420p
			`
		}

		ffmpegCmd := exec.CommandContext(
			cmd.Context(),
			"ffmpeg",
			"-i", args[0],
			"-i", "pipe:0",
			"-filter_complex", filterComplex,
			"-map", "0:a?",
			"-map_metadata", "-1",
			"-c:v", "libx264",
			"-crf", "23",
			"-preset", "veryfast",
			"-y", output,
		)

		ffmpegCmd.Stderr = os.Stderr
		ffmpegCmd.Stdout = os.Stdout
		ffmpegCmd.Stdin = buf

		return ffmpegCmd.Run()
	},
}

func main() {
	err := fang.Execute(
		context.Background(),
		cmd,
		fang.WithFlagTypes(),
		fang.WithNotifySignal(syscall.SIGINT, syscall.SIGTERM),
		fang.WithShorthandPadding(),
		fang.WithoutCompletions(),
	)
	if err != nil {
		os.Exit(1)
	}
}
