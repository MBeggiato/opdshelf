package cover

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"mime"
	"path"
	"path/filepath"
	"strings"
)

func getEpubCover(filePath string) ([]byte, string, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, "", err
	}
	defer r.Close()

	// 1. Find META-INF/container.xml to locate full-path
	containerFile, err := r.Open("META-INF/container.xml")
	if err != nil {
		return nil, "", err
	}
	defer containerFile.Close()

	var container struct {
		Rootfiles struct {
			Rootfile []struct {
				FullPath string `xml:"full-path,attr"`
			} `xml:"rootfile"`
		} `xml:"rootfiles"`
	}

	if err := xml.NewDecoder(containerFile).Decode(&container); err != nil {
		return nil, "", err
	}

	if len(container.Rootfiles.Rootfile) == 0 {
		return nil, "", errors.New("no rootfile found in container.xml")
	}

	opfPath := container.Rootfiles.Rootfile[0].FullPath

	// 2. Open OPF file
	opfFile, err := findFileInZip(r, opfPath)
	if err != nil {
		return nil, "", err
	}

	opfContent, err := io.ReadAll(opfFile)
	opfFile.Close()
	if err != nil {
		return nil, "", err
	}

	// 3. Parse OPF to find manifest and metadata
	var opf struct {
		Metadata struct {
			Meta []struct {
				Name    string `xml:"name,attr"`
				Content string `xml:"content,attr"`
			} `xml:"meta"`
		} `xml:"metadata"`
		Manifest struct {
			Item []struct {
				ID         string `xml:"id,attr"`
				Href       string `xml:"href,attr"`
				MediaType  string `xml:"media-type,attr"`
				Properties string `xml:"properties,attr"`
			} `xml:"item"`
		} `xml:"manifest"`
	}

	if err := xml.Unmarshal(opfContent, &opf); err != nil {
		return nil, "", err
	}

	var coverHref string
	var coverMediaType string

	// Method A: ID referenced by meta name="cover"
	var coverID string
	for _, meta := range opf.Metadata.Meta {
		if meta.Name == "cover" {
			coverID = meta.Content
			break
		}
	}

	if coverID != "" {
		for _, item := range opf.Manifest.Item {
			if item.ID == coverID {
				coverHref = item.Href
				coverMediaType = item.MediaType
				break
			}
		}
	}

	// Method B: item with properties="cover-image"
	if coverHref == "" {
		for _, item := range opf.Manifest.Item {
			if strings.Contains(item.Properties, "cover-image") {
				coverHref = item.Href
				coverMediaType = item.MediaType
				break
			}
		}
	}

	if coverHref == "" {
		return nil, "", errors.New("cover not found in opf")
	}

	// Resolve path relative to OPF file
	coverPath := path.Join(path.Dir(opfPath), coverHref)

	// 4. Extract cover image
	imgFile, err := findFileInZip(r, coverPath)
	if err != nil {
		// Try unescaped path just in case
		// coverPath might contain URL encoded chars? usually zip paths are raw.
		return nil, "", fmt.Errorf("could not find cover image at %s: %w", coverPath, err)
	}
	defer imgFile.Close()

	imgData, err := io.ReadAll(imgFile)
	if err != nil {
		return nil, "", err
	}

	if coverMediaType == "" {
		coverMediaType = mime.TypeByExtension(filepath.Ext(coverHref))
		if coverMediaType == "" {
			coverMediaType = "application/octet-stream"
		}
	}

	return imgData, coverMediaType, nil
}

func findFileInZip(r *zip.ReadCloser, name string) (io.ReadCloser, error) {
	for _, f := range r.File {
		if f.Name == name {
			return f.Open()
		}
	}
	return nil, errors.New("file not found in zip: " + name)
}
