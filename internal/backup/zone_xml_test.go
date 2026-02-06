//go:build linux
// +build linux

package backup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lazyfirewall/internal/firewalld"
)

func TestParseZoneXML_XXE(t *testing.T) {
	tests := []struct {
		name    string
		xml     string
		wantErr bool
	}{
		{
			name: "xxe external entity",
			xml: `<?xml version="1.0"?>
<!DOCTYPE zone [
  <!ENTITY xxe SYSTEM "file:///etc/passwd">
]>
<zone>
  <short>&xxe;</short>
</zone>`,
			wantErr: true,
		},
		{
			name: "xml bomb entity expansion",
			xml: `<?xml version="1.0"?>
<!DOCTYPE zone [
  <!ENTITY a "A">
  <!ENTITY b "&a;&a;&a;&a;">
  <!ENTITY c "&b;&b;&b;&b;">
]>
<zone><short>&c;</short></zone>`,
			wantErr: true,
		},
		{
			name: "valid xml",
			xml: `<?xml version="1.0"?>
<zone>
  <short>Test</short>
  <service name="ssh"/>
</zone>`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseZoneXML([]byte(tt.xml))
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseZoneXML() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseZoneXML_SizeLimit(t *testing.T) {
	body := strings.Repeat("a", maxParsedXMLSize+1)
	xml := `<?xml version="1.0"?><zone><short>` + body + `</short></zone>`
	_, err := ParseZoneXML([]byte(xml))
	if err == nil {
		t.Fatalf("expected size-limit error")
	}
}

func TestParseZoneXMLFile_SizeLimit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "big.xml")
	data := strings.Repeat("a", maxXMLFileSize+1)
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := ParseZoneXMLFile(path)
	if err == nil {
		t.Fatalf("expected ParseZoneXMLFile size-limit error")
	}
}

func TestMarshalZoneXML_Nil(t *testing.T) {
	_, err := MarshalZoneXML(nil)
	if err == nil {
		t.Fatalf("expected error for nil zone")
	}
}

func TestWriteZoneXMLFile(t *testing.T) {
	tempDir := t.TempDir()
	oldConfig := zoneConfigDir
	zoneConfigDir = tempDir
	t.Cleanup(func() { zoneConfigDir = oldConfig })

	z := &firewalld.Zone{
		Name:       "public",
		Short:      "Public",
		Services:   []string{"ssh"},
		Masquerade: true,
	}

	path, err := WriteZoneXMLFile("public", z)
	if err != nil {
		t.Fatalf("WriteZoneXMLFile() error = %v", err)
	}
	if path != filepath.Join(tempDir, "public.xml") {
		t.Fatalf("path = %q, want %q", path, filepath.Join(tempDir, "public.xml"))
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file %q to exist: %v", path, err)
	}
}

func TestWriteZoneXMLFile_InvalidZone(t *testing.T) {
	z := &firewalld.Zone{Name: "bad"}
	_, err := WriteZoneXMLFile("../bad", z)
	if err == nil {
		t.Fatalf("expected invalid zone error")
	}
}

func TestMarshalAndParseZoneXML_RoundTrip(t *testing.T) {
	orig := &firewalld.Zone{
		Target:      "DROP",
		Short:       "Public",
		Description: "Test zone",
		Services:    []string{"ssh", "http"},
		Ports: []firewalld.Port{
			{Port: "22", Protocol: "tcp"},
			{Port: "53", Protocol: "udp"},
		},
		Interfaces: []string{"eth0"},
		Sources:    []string{"10.0.0.0/24", "mac:aa:bb:cc:dd:ee:ff", "ipset:blocklist"},
		IcmpBlocks: []string{"echo-request"},
		Masquerade: true,
		IcmpInvert: true,
	}

	data, err := MarshalZoneXML(orig)
	if err != nil {
		t.Fatalf("MarshalZoneXML() error = %v", err)
	}

	parsed, err := ParseZoneXML(data)
	if err != nil {
		t.Fatalf("ParseZoneXML() error = %v", err)
	}

	if parsed.Target != orig.Target || parsed.Short != orig.Short || parsed.Description != orig.Description {
		t.Fatalf("basic fields mismatch: parsed=%+v orig=%+v", parsed, orig)
	}
	if len(parsed.Services) != len(orig.Services) || len(parsed.Ports) != len(orig.Ports) || len(parsed.Sources) != len(orig.Sources) {
		t.Fatalf("collection lengths mismatch: parsed=%+v orig=%+v", parsed, orig)
	}
	if !parsed.Masquerade || !parsed.IcmpInvert {
		t.Fatalf("boolean fields mismatch: parsed masquerade=%v invert=%v", parsed.Masquerade, parsed.IcmpInvert)
	}
}
