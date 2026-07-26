package handlers

import "testing"

func TestContentDispositionAttachmentEscapesQuotes(t *testing.T) {
	got := contentDispositionAttachment(`evil".pdf`)
	want := `attachment; filename="evil\".pdf"`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestContentDispositionAttachmentStripsCRLF(t *testing.T) {
	got := contentDispositionAttachment("evil\r\nX-Injected: yes")
	if got == "" {
		t.Fatal("expected a non-empty header value")
	}
	for _, c := range got {
		if c == '\r' || c == '\n' {
			t.Fatalf("header value contains raw CR/LF: %q", got)
		}
	}
}
