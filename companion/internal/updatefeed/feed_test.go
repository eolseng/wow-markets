package updatefeed

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"testing"
)

func TestSignedFeedSelectsNewerMatchingTarget(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	artifact := []byte("signed installer")
	artifactSignature := base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, artifact))
	content := []byte(fmt.Sprintf(`<?xml version="1.0"?>
<rss version="2.0" xmlns:sparkle="%s" xmlns:wow="%s"><channel>
<item><title>1.0.1</title><sparkle:version>1.0.1</sparkle:version><enclosure url="https://github.com/eolseng/wow-markets/releases/download/companion-v1.0.1/wow-markets-companion-macos-arm64.dmg" length="%d" sparkle:edSignature="%s" sparkle:os="macos" wow:arch="arm64" /></item>
<item><title>1.1.0</title><sparkle:version>1.1.0</sparkle:version><enclosure url="https://github.com/eolseng/wow-markets/releases/download/companion-v1.1.0/wow-markets-companion-windows-amd64-setup.exe" length="%d" sparkle:edSignature="%s" sparkle:os="windows" wow:arch="amd64" /></item>
</channel></rss>
`, sparkleNamespace, wowNamespace, len(artifact), artifactSignature, len(artifact), artifactSignature))
	payload := signTestFeed(content, privateKey)
	releases, err := ParseSigned(payload, publicKey)
	if err != nil {
		t.Fatalf("ParseSigned() error = %v", err)
	}
	selected, err := Select(releases, "1.0.0", TargetMacOSARM64)
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}
	if selected == nil || selected.Version != "1.0.1" {
		t.Fatalf("Select() = %#v, want macOS 1.0.1", selected)
	}
	if err := VerifyArtifact(artifact, selected.Signature, publicKey); err != nil {
		t.Fatalf("VerifyArtifact() error = %v", err)
	}
}

func TestSignedFeedRejectsTamperingAndWrongLength(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	content := []byte(`<?xml version="1.0"?><rss><channel><item /></channel></rss>`)
	payload := signTestFeed(content, privateKey)
	payload[10] ^= 1
	if _, err := ParseSigned(payload, publicKey); err == nil {
		t.Fatal("ParseSigned() accepted a modified appcast")
	}

	payload = signTestFeed(content, privateKey)
	payload = []byte(string(payload[:len(content)]) + fmt.Sprintf("<!-- sparkle-signatures:\nedSignature: %s\nlength: 1\n-->\n", base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, content))))
	if _, err := ParseSigned(payload, publicKey); err == nil {
		t.Fatal("ParseSigned() accepted a mismatched signed length")
	}
}

func TestSelectRejectsDowngradesAndWrongArchitectures(t *testing.T) {
	releases := []Release{
		{Version: "0.9.9", OS: "macos", Arch: "arm64"},
		{Version: "1.1.0", OS: "macos", Arch: "amd64"},
		{Version: "1.2.0", OS: "windows", Arch: "amd64"},
	}
	selected, err := Select(releases, "1.0.0", TargetMacOSARM64)
	if err != nil {
		t.Fatal(err)
	}
	if selected != nil {
		t.Fatalf("Select() = %#v, want no compatible upgrade", selected)
	}
}

func TestSemanticVersionOrdering(t *testing.T) {
	ordered := []string{"1.0.0-alpha", "1.0.0-alpha.1", "1.0.0-beta", "1.0.0-rc.1", "1.0.0", "1.0.1"}
	for index := 1; index < len(ordered); index++ {
		left, err := ParseVersion(ordered[index-1])
		if err != nil {
			t.Fatal(err)
		}
		right, err := ParseVersion(ordered[index])
		if err != nil {
			t.Fatal(err)
		}
		if left.Compare(right) >= 0 {
			t.Fatalf("%s should sort before %s", ordered[index-1], ordered[index])
		}
	}
}

func signTestFeed(content []byte, privateKey ed25519.PrivateKey) []byte {
	signature := base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, content))
	return []byte(string(content) + fmt.Sprintf("<!-- sparkle-signatures:\nedSignature: %s\nlength: %d\n-->\n", signature, len(content)))
}
