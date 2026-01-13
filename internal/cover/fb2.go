package cover

import (
	"encoding/base64"
	"encoding/xml"
	"io"
	"os"
	"strings"
)

func getFb2Cover(filePath string) ([]byte, string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()

	// FB2 is XML. It can be huge, but the cover is usually near the beginning or the binaries are at the end.
	// We might need to decode the whole struct or stream.
	// Let's try full decode for simplicity first, assuming reasonable size.
	// Optimally we should use a streaming decoder to find the specific binary tag.

	// decoder := xml.NewDecoder(f)

	type Binary struct {
		ID          string `xml:"id,attr"`
		ContentType string `xml:"content-type,attr"`
		Content     string `xml:",chardata"`
	}

	// We look for a binary with id matching the coverpage image ref
	// Or often just the first binary that looks like a cover if schema is loose.
	// Standard: <description><title-info><coverpage><image l:href="#cover.jpg"/></coverpage>...
	// And then <binary id="cover.jpg" ...>...</binary>

	// Let's try to map the interesting parts.

	var fictionBook struct {
		Description struct {
			TitleInfo struct {
				CoverPage struct {
					Image struct {
						Href string `xml:"href,attr"`
					} `xml:"image"`
				} `xml:"coverpage"`
			} `xml:"title-info"`
		} `xml:"description"`
		Binaries []Binary `xml:"binary"`
	}

	// Note: href is often namespaced like xlink:href, decoder might ignore namespace prefix in basic struct field unless specified.
	// We might need to iterate tokens if struct mapping is tricky with namespaces.
	// Let's try reading all into a byte slice and unmarshaling if memory allows.
	// If files are > 50MB this might be bad, but FB2 are usually text.

	// Reset file pointer
	f.Seek(0, 0)
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, "", err
	}

	if err := xml.Unmarshal(data, &fictionBook); err != nil {
		return nil, "", err
	}

	coverID := strings.TrimPrefix(fictionBook.Description.TitleInfo.CoverPage.Image.Href, "#")
	if coverID == "" {
		// Fallback: look for "cover.jpg" or similar
		coverID = "cover.jpg"
	}

	for _, bin := range fictionBook.Binaries {
		if bin.ID == coverID || (coverID == "cover.jpg" && strings.Contains(bin.ID, "cover")) {
			// Found it
			imgData, err := base64.StdEncoding.DecodeString(strings.TrimSpace(bin.Content))
			if err != nil {
				return nil, "", err
			}
			return imgData, bin.ContentType, nil
		}
	}

	// Fallback: return first image binary?
	if len(fictionBook.Binaries) > 0 {
		bin := fictionBook.Binaries[0]
		if strings.HasPrefix(bin.ContentType, "image/") {
			imgData, err := base64.StdEncoding.DecodeString(strings.TrimSpace(bin.Content))
			if err != nil {
				return nil, "", err
			}
			return imgData, bin.ContentType, nil
		}
	}

	return nil, "", io.EOF
}
