package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/eolseng/wow-markets/companion/internal/updatefeed"
)

func main() {
	if len(os.Args) < 2 {
		fatal(errors.New("usage: updatefeed <generate-key|public-key|generate>"))
	}
	var err error
	switch os.Args[1] {
	case "generate-key":
		err = generateKey(os.Args[2:])
	case "public-key":
		err = printPublicKey(os.Args[2:])
	case "generate":
		err = generate(os.Args[2:])
	default:
		err = fmt.Errorf("unsupported command %q", os.Args[1])
	}
	if err != nil {
		fatal(err)
	}
}

func generateKey(args []string) error {
	flags := flag.NewFlagSet("generate-key", flag.ContinueOnError)
	privatePath := flags.String("private-key", "", "path for the base64 private key")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*privatePath) == "" {
		return errors.New("--private-key is required")
	}
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("generate Ed25519 key: %w", err)
	}
	encoded := base64.StdEncoding.EncodeToString(privateKey) + "\n"
	if err := os.WriteFile(*privatePath, []byte(encoded), 0o600); err != nil {
		return fmt.Errorf("write private key: %w", err)
	}
	fmt.Println(base64.StdEncoding.EncodeToString(publicKey))
	return nil
}

func printPublicKey(args []string) error {
	flags := flag.NewFlagSet("public-key", flag.ContinueOnError)
	privatePath := flags.String("private-key", "", "path to the base64 private key")
	if err := flags.Parse(args); err != nil {
		return err
	}
	privateKey, err := loadPrivateKey(*privatePath)
	if err != nil {
		return err
	}
	publicKey := privateKey.Public().(ed25519.PublicKey)
	fmt.Println(base64.StdEncoding.EncodeToString(publicKey))
	return nil
}

func generate(args []string) error {
	flags := flag.NewFlagSet("generate", flag.ContinueOnError)
	privatePath := flags.String("private-key", "", "path to the base64 private key; defaults to UPDATE_SIGNING_PRIVATE_KEY_BASE64")
	version := flags.String("version", "", "semantic companion version")
	buildVersion := flags.String("build-version", "", "increasing numeric native build version")
	channelValue := flags.String("channel", "", "stable or beta")
	publishedValue := flags.String("published-at", "", "RFC3339 publication time")
	assetsDir := flags.String("assets-dir", "", "directory containing the final companion installers")
	outputDir := flags.String("output-dir", "", "directory for signed appcasts")
	critical := flags.Bool("critical", false, "mark this as a mandatory critical update")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if _, err := updatefeed.ParseVersion(*version); err != nil {
		return err
	}
	if parsed, err := strconv.ParseUint(strings.TrimSpace(*buildVersion), 10, 64); err != nil || parsed == 0 {
		return errors.New("--build-version must be a positive integer")
	}
	channel, err := updatefeed.ParseChannel(*channelValue)
	if err != nil {
		return err
	}
	published, err := time.Parse(time.RFC3339, *publishedValue)
	if err != nil {
		return fmt.Errorf("--published-at must be RFC3339: %w", err)
	}
	if strings.TrimSpace(*assetsDir) == "" || strings.TrimSpace(*outputDir) == "" {
		return errors.New("--assets-dir and --output-dir are required")
	}
	privateKey, err := loadPrivateKey(*privatePath)
	if err != nil {
		return err
	}

	tag := "companion-v" + *version
	releaseBase := "https://github.com/eolseng/wow-markets/releases/download/" + tag
	notesURL := "https://github.com/eolseng/wow-markets/releases/tag/" + tag
	targets := []feedTarget{
		{
			fileName:    "macos-arm64.xml",
			assetName:   "wow-markets-companion-macos-arm64.dmg",
			contentType: "application/x-apple-diskimage",
			os:          "macos",
			arch:        "arm64",
		},
		{
			fileName:    "windows-amd64.xml",
			assetName:   "wow-markets-companion-windows-amd64-setup.exe",
			contentType: "application/vnd.microsoft.portable-executable",
			os:          "windows",
			arch:        "amd64",
		},
	}
	if err := os.MkdirAll(*outputDir, 0o755); err != nil {
		return fmt.Errorf("create appcast output directory: %w", err)
	}
	for _, target := range targets {
		assetPath := filepath.Join(*assetsDir, target.assetName)
		artifact, err := os.ReadFile(assetPath)
		if err != nil {
			return fmt.Errorf("read %s: %w", target.assetName, err)
		}
		content := renderFeed(feedInput{
			Version:      *version,
			BuildVersion: strings.TrimSpace(*buildVersion),
			Title:        "WoW Markets Companion " + *version,
			Description:  "This update is ready to install. Select Version History for full release notes.",
			PublishedAt:  published.UTC().Format(time.RFC1123Z),
			NotesURL:     notesURL,
			AssetURL:     releaseBase + "/" + target.assetName,
			Length:       len(artifact),
			Signature:    base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, artifact)),
			ContentType:  target.contentType,
			OS:           target.os,
			Arch:         target.arch,
			Critical:     *critical,
		})
		signed := appendFeedSignature(content, privateKey)
		fileName := fmt.Sprintf("companion-%s-%s", channel, target.fileName)
		if err := os.WriteFile(filepath.Join(*outputDir, fileName), signed, 0o644); err != nil {
			return fmt.Errorf("write %s appcast: %w", target.os, err)
		}
	}
	return nil
}

type feedTarget struct {
	fileName    string
	assetName   string
	contentType string
	os          string
	arch        string
}

type feedInput struct {
	Version      string
	BuildVersion string
	Title        string
	Description  string
	PublishedAt  string
	NotesURL     string
	AssetURL     string
	Length       int
	Signature    string
	ContentType  string
	OS           string
	Arch         string
	Critical     bool
}

func renderFeed(input feedInput) []byte {
	critical := ""
	if input.Critical {
		critical = "      <sparkle:criticalUpdate />\n"
	}
	return []byte(fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<rss version="2.0" xmlns:sparkle="http://www.andymatuschak.org/xml-namespaces/sparkle" xmlns:wow="https://wowmarkets.app/xml-namespaces/update">
  <channel>
    <title>WoW Markets Companion updates</title>
    <item>
      <title>%s</title>
      <pubDate>%s</pubDate>
      <sparkle:version>%s</sparkle:version>
      <sparkle:shortVersionString>%s</sparkle:shortVersionString>
      <description>%s</description>
      <sparkle:fullReleaseNotesLink>%s</sparkle:fullReleaseNotesLink>
%s      <enclosure url="%s" length="%d" type="%s" sparkle:version="%s" sparkle:edSignature="%s" sparkle:os="%s" wow:arch="%s" />
    </item>
  </channel>
</rss>
`, escape(input.Title), escape(input.PublishedAt), escape(input.Version),
		escape(input.Version), escape(input.Description), escape(input.NotesURL), critical, escape(input.AssetURL),
		input.Length, escape(input.ContentType), escape(input.BuildVersion), escape(input.Signature),
		escape(input.OS), escape(input.Arch)))
}

func appendFeedSignature(content []byte, privateKey ed25519.PrivateKey) []byte {
	signature := base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, content))
	block := fmt.Sprintf("<!-- sparkle-signatures:\nedSignature: %s\nlength: %d\n-->\n", signature, len(content))
	return append(append([]byte(nil), content...), []byte(block)...)
}

func escape(value string) string {
	var output strings.Builder
	_ = xml.EscapeText(&output, []byte(value))
	return output.String()
}

func loadPrivateKey(path string) (ed25519.PrivateKey, error) {
	var value []byte
	var err error
	if strings.TrimSpace(path) != "" {
		value, err = os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read update private key: %w", err)
		}
	} else {
		value = []byte(os.Getenv("UPDATE_SIGNING_PRIVATE_KEY_BASE64"))
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(value)))
	if err != nil || len(decoded) != ed25519.PrivateKeySize {
		return nil, errors.New("update private key must be a base64-encoded 64-byte Ed25519 private key")
	}
	return ed25519.PrivateKey(decoded), nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
