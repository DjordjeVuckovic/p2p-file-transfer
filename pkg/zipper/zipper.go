package zipper

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type ZipOptions struct {
	// The path to the file to be zipped
	InputPaths []string
	// The path to the output zip file
	OutputPath string
	// The compression level (0-9)
	CompressionLevel int
	// Whether to include the original file in the zip
	IncludeOriginal bool
	// abort zipping
	Abort chan struct{}
}

const DefaultOutputPath = "output.zip"

func WithPaths(inputPaths []string, outputPath string) OptionFn {
	return func(o *ZipOptions) {
		o.InputPaths = inputPaths
		o.OutputPath = outputPath
	}
}

type OptionFn func(*ZipOptions)

func loadOptions(opts ...OptionFn) *ZipOptions {
	opt := &ZipOptions{
		OutputPath:       DefaultOutputPath,
		CompressionLevel: 0,
		IncludeOriginal:  false,
		Abort:            make(chan struct{}),
	}
	for _, o := range opts {
		o(opt)
	}
	return opt
}

func Zip(opts ...OptionFn) error {
	opt := loadOptions(opts...)
	zipFile, err := os.Create(opt.OutputPath)
	if err != nil {
		return err
	}
	defer func(zipFile *os.File) {
		err := zipFile.Close()
		if err != nil {
			fmt.Printf("Error creating zip archive: %s\n", err)
		}
	}(zipFile)

	zipWriter := zip.NewWriter(zipFile)
	defer func(zipWriter *zip.Writer) {
		err := zipWriter.Close()
		if err != nil {
			fmt.Printf("Error closing zip archive: %s\n", err)
		}
	}(zipWriter)

	processed := make(map[string]bool)

	for _, source := range opt.InputPaths {
		absSource, err := filepath.Abs(source)
		if err != nil {
			fmt.Printf("Warning: Could not get absolute path for %s: %v\n", source, err)
			continue
		}
		info, err := os.Stat(absSource)
		if err != nil {
			fmt.Printf("Warning: Could not stat %s: %v\n", source, err)
			continue
		}
		if info.IsDir() {
			// Walk directory
			baseDir := filepath.Dir(absSource)
			err = filepath.Walk(absSource, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					fmt.Printf("Warning: Error accessing %s: %v\n", path, err)
					return err
				}

				// Skip directories as entries (they'll be created implicitly)
				if info.IsDir() {
					fmt.Printf("Skipping directory %s\n", path)
					return nil
				}

				// Get relative path
				relPath, err := filepath.Rel(baseDir, path)
				if err != nil {
					return err
				}

				// Use forward slashes for consistency in pattern matching
				relPathFwd := filepath.ToSlash(relPath)

				// Check if file is already processed
				if processed[relPathFwd] {
					return nil
				}
				// Mark file as processed
				processed[relPathFwd] = true

				// Add file to zip
				return addFileToZip(zipWriter, path, relPathFwd)
			})
			if err != nil {
				return err
			}
		} else {
			// Single file
			relPath := filepath.Base(absSource)
			relPathFwd := filepath.ToSlash(relPath)

			// Add file to zip
			if !processed[relPathFwd] {
				err = addFileToZip(zipWriter, absSource, relPathFwd)
				if err != nil {
					return err
				}
				processed[relPathFwd] = true
			}
		}
	}

	return nil
}

func addFileToZip(zipWriter *zip.Writer, filePath, zipPath string) error {
	// Check if the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Printf("Warning: File %s does not exist\n", filePath)
		return nil
	}
	// is file?
	if info, err := os.Stat(filePath); err != nil || info.IsDir() {
		fmt.Printf("Warning: %s is not a file\n", filePath)
		return nil
	}
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("Error opening file %s: %s\n", filePath, err)
		return err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			fmt.Printf("Error closing file %s: %s\n", filePath, err)
		}
	}(file)

	// Get file info for header
	info, err := file.Stat()
	if err != nil {
		fmt.Printf("Error getting file info for %s: %s\n", filePath, err)
		return err
	}

	// Create zip header
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		fmt.Printf("Error creating zip header for %s: %s\n", filePath, err)
		return err
	}

	// Set compression
	header.Method = zip.Deflate

	// Set relative path in zip
	header.Name = zipPath

	// Create writer for this file within zip
	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}

	// Copy file content to zip
	_, err = io.Copy(writer, file)
	if err != nil {
		return err
	}

	fmt.Printf("Added: %s\n", zipPath)
	return nil
}
