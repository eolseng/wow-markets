package updatefeed

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

const (
	sparkleNamespace = "http://www.andymatuschak.org/xml-namespaces/sparkle"
	wowNamespace     = "https://wowmarkets.app/xml-namespaces/update"
	signaturePrefix  = "<!-- sparkle-signatures:\n"
	signatureSuffix  = "-->"
)

type Channel string

const (
	ChannelStable Channel = "stable"
	ChannelBeta   Channel = "beta"
)

func ParseChannel(value string) (Channel, error) {
	switch Channel(strings.ToLower(strings.TrimSpace(value))) {
	case ChannelStable:
		return ChannelStable, nil
	case ChannelBeta:
		return ChannelBeta, nil
	default:
		return "", fmt.Errorf("unsupported update channel %q", value)
	}
}

type Target struct {
	OS   string
	Arch string
}

var (
	TargetMacOSARM64   = Target{OS: "macos", Arch: "arm64"}
	TargetWindowsAMD64 = Target{OS: "windows", Arch: "amd64"}
)

type Release struct {
	Version     string
	Title       string
	PublishedAt string
	NotesURL    string
	URL         string
	Length      int64
	Signature   string
	OS          string
	Arch        string
	Mandatory   bool
}

type document struct {
	Channel struct {
		Items []item `xml:"item"`
	} `xml:"channel"`
}

type item struct {
	Title       string    `xml:"title"`
	PublishedAt string    `xml:"pubDate"`
	Version     string    `xml:"http://www.andymatuschak.org/xml-namespaces/sparkle version"`
	NotesURL    string    `xml:"http://www.andymatuschak.org/xml-namespaces/sparkle releaseNotesLink"`
	Mandatory   *struct{} `xml:"http://www.andymatuschak.org/xml-namespaces/sparkle criticalUpdate"`
	Enclosure   enclosure `xml:"enclosure"`
}

type enclosure struct {
	URL       string `xml:"url,attr"`
	Length    string `xml:"length,attr"`
	Signature string `xml:"http://www.andymatuschak.org/xml-namespaces/sparkle edSignature,attr"`
	OS        string `xml:"http://www.andymatuschak.org/xml-namespaces/sparkle os,attr"`
	Arch      string `xml:"https://wowmarkets.app/xml-namespaces/update arch,attr"`
}

func ParseSigned(payload []byte, publicKey ed25519.PublicKey) ([]Release, error) {
	content, signature, err := splitSignedContent(payload)
	if err != nil {
		return nil, err
	}
	if len(publicKey) != ed25519.PublicKeySize {
		return nil, errors.New("update public key has the wrong length")
	}
	if !ed25519.Verify(publicKey, content, signature) {
		return nil, errors.New("appcast signature is invalid")
	}

	var parsed document
	decoder := xml.NewDecoder(bytes.NewReader(content))
	if err := decoder.Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode signed appcast: %w", err)
	}
	if len(parsed.Channel.Items) == 0 {
		return nil, errors.New("signed appcast contains no releases")
	}

	releases := make([]Release, 0, len(parsed.Channel.Items))
	for _, candidate := range parsed.Channel.Items {
		length, err := strconv.ParseInt(strings.TrimSpace(candidate.Enclosure.Length), 10, 64)
		if err != nil || length <= 0 {
			return nil, fmt.Errorf("release %q has an invalid enclosure length", candidate.Version)
		}
		if _, err := ParseVersion(candidate.Version); err != nil {
			return nil, fmt.Errorf("release version: %w", err)
		}
		if _, err := url.ParseRequestURI(candidate.Enclosure.URL); err != nil {
			return nil, fmt.Errorf("release %q has an invalid enclosure URL: %w", candidate.Version, err)
		}
		artifactSignature, err := base64.StdEncoding.DecodeString(strings.TrimSpace(candidate.Enclosure.Signature))
		if err != nil || len(artifactSignature) != ed25519.SignatureSize {
			return nil, fmt.Errorf("release %q has an invalid enclosure signature", candidate.Version)
		}
		releases = append(releases, Release{
			Version:     strings.TrimSpace(candidate.Version),
			Title:       strings.TrimSpace(candidate.Title),
			PublishedAt: strings.TrimSpace(candidate.PublishedAt),
			NotesURL:    strings.TrimSpace(candidate.NotesURL),
			URL:         strings.TrimSpace(candidate.Enclosure.URL),
			Length:      length,
			Signature:   strings.TrimSpace(candidate.Enclosure.Signature),
			OS:          strings.TrimSpace(candidate.Enclosure.OS),
			Arch:        strings.TrimSpace(candidate.Enclosure.Arch),
			Mandatory:   candidate.Mandatory != nil,
		})
	}
	return releases, nil
}

func Select(releases []Release, current string, target Target) (*Release, error) {
	currentVersion, err := ParseVersion(current)
	if err != nil {
		return nil, fmt.Errorf("current version: %w", err)
	}
	var selected *Release
	var selectedVersion Version
	for index := range releases {
		candidate := &releases[index]
		if candidate.OS != target.OS || candidate.Arch != target.Arch {
			continue
		}
		version, err := ParseVersion(candidate.Version)
		if err != nil {
			return nil, err
		}
		if version.Compare(currentVersion) <= 0 {
			continue
		}
		if selected == nil || version.Compare(selectedVersion) > 0 {
			selected = candidate
			selectedVersion = version
		}
	}
	return selected, nil
}

func VerifyArtifact(payload []byte, signatureBase64 string, publicKey ed25519.PublicKey) error {
	signature, err := base64.StdEncoding.DecodeString(strings.TrimSpace(signatureBase64))
	if err != nil || len(signature) != ed25519.SignatureSize {
		return errors.New("artifact signature is malformed")
	}
	if !ed25519.Verify(publicKey, payload, signature) {
		return errors.New("artifact signature is invalid")
	}
	return nil
}

func splitSignedContent(payload []byte) ([]byte, []byte, error) {
	index := bytes.LastIndex(payload, []byte(signaturePrefix))
	if index < 0 {
		return nil, nil, errors.New("appcast is unsigned")
	}
	block := payload[index+len(signaturePrefix):]
	suffix := bytes.Index(block, []byte(signatureSuffix))
	if suffix < 0 {
		return nil, nil, errors.New("appcast signature block is incomplete")
	}
	content := payload[:index]
	var signatureText string
	var declaredLength int
	for _, line := range strings.Split(string(block[:suffix]), "\n") {
		name, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		switch strings.TrimSpace(name) {
		case "edSignature":
			signatureText = strings.TrimSpace(value)
		case "length":
			parsed, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				return nil, nil, errors.New("appcast signature length is invalid")
			}
			declaredLength = parsed
		}
	}
	if declaredLength != len(content) {
		return nil, nil, fmt.Errorf("appcast signed length is %d, got %d bytes", declaredLength, len(content))
	}
	signature, err := base64.StdEncoding.DecodeString(signatureText)
	if err != nil || len(signature) != ed25519.SignatureSize {
		return nil, nil, errors.New("appcast signature is malformed")
	}
	return content, signature, nil
}
