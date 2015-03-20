package pointer

import (
	"fmt"
	"github.com/cheggaaa/pb"
	"github.com/hawser/git-hawser/hawser"
	"github.com/technoweenie/go-contentaddressable"
	"io"
	"os"
)

func Smudge(writer io.Writer, ptr *Pointer, workingfile string, cb hawser.CopyCallback) error {
	mediafile, err := hawser.LocalMediaPath(ptr.Oid)
	if err != nil {
		return err
	}

	var wErr *hawser.WrappedError
	if stat, statErr := os.Stat(mediafile); statErr != nil || stat == nil {
		wErr = downloadFile(writer, ptr, workingfile, mediafile, cb)
	} else {
		wErr = readLocalFile(writer, ptr, mediafile, cb)
	}

	if wErr != nil {
		return &SmudgeError{ptr.Oid, mediafile, wErr}
	} else {
		return nil
	}
}

func downloadFile(writer io.Writer, ptr *Pointer, workingfile, mediafile string, cb hawser.CopyCallback) *hawser.WrappedError {
	reader, size, wErr := hawser.Download(mediafile)
	if reader != nil {
		defer reader.Close()
	}

	if wErr != nil {
		wErr.Errorf("Error downloading %s.", mediafile)
		return wErr
	}

	if ptr.Size == 0 {
		ptr.Size = size
	}

	mediaFile, err := contentaddressable.NewFile(mediafile)
	if err != nil {
		return hawser.Errorf(err, "Error opening media file buffer.")
	}

	bar := pb.New64(size)
	bar.SetUnits(pb.U_BYTES)
	bar.Output = os.Stderr
	bar.Start()

	_, err = hawser.CopyWithCallback(mediaFile, bar.NewProxyReader(reader), ptr.Size, cb)
	if err == nil {
		err = mediaFile.Accept()
	}
	mediaFile.Close()

	fmt.Fprintf(os.Stderr, "\nDownloaded %s\n", workingfile)

	if err != nil {
		return hawser.Errorf(err, "Error buffering media file.")
	}

	return readLocalFile(writer, ptr, mediafile, nil)
}

func readLocalFile(writer io.Writer, ptr *Pointer, mediafile string, cb hawser.CopyCallback) *hawser.WrappedError {
	reader, err := os.Open(mediafile)
	if err != nil {
		return hawser.Errorf(err, "Error opening media file.")
	}
	defer reader.Close()

	if ptr.Size == 0 {
		if stat, _ := os.Stat(mediafile); stat != nil {
			ptr.Size = stat.Size()
		}
	}

	_, err = hawser.CopyWithCallback(writer, reader, ptr.Size, cb)
	return hawser.Errorf(err, "Error reading from media file.")
}

type SmudgeError struct {
	Oid      string
	Filename string
	*hawser.WrappedError
}
