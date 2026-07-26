package apitests

import (
	"strings"
	"testing"
)

func TestFileLifecycle(t *testing.T) {
	client, h := authedClient(t)
	name := uniqueName("file") + ".txt"

	if err := client.UploadFile(h, name, strings.NewReader("hello from api-tests"), ""); err != nil {
		t.Fatalf("UploadFile: %v", err)
	}

	file := findItem(listRoot(t, client, h), "file", name)
	if file == nil {
		t.Fatalf("uploaded file %q not found in root listing", name)
	}
	if file.FileSize == 0 {
		t.Errorf("expected uploaded file to have a non-zero size, got %d", file.FileSize)
	}

	// Rename.
	newName := uniqueName("file") + "-renamed.txt"
	if err := client.UpdateFileName(h, file.UUID, newName); err != nil {
		t.Fatalf("UpdateFileName: %v", err)
	}
	if findItem(listRoot(t, client, h), "file", newName) == nil {
		t.Fatalf("renamed file %q not found in root listing", newName)
	}

	// Trash it.
	if err := client.DeleteFile(h, file.UUID); err != nil {
		t.Fatalf("DeleteFile: %v", err)
	}
	if findItem(listRoot(t, client, h), "file", newName) != nil {
		t.Fatalf("trashed file %q still present in root listing", newName)
	}

	trash, err := client.ListTrash(h, 1, 200)
	if err != nil {
		t.Fatalf("ListTrash: %v", err)
	}
	if findItem(trash.Items, "file", newName) == nil {
		t.Fatalf("trashed file %q not found in trash listing", newName)
	}

	// Restore it.
	if err := client.RestoreFile(h, file.UUID); err != nil {
		t.Fatalf("RestoreFile: %v", err)
	}
	if findItem(listRoot(t, client, h), "file", newName) == nil {
		t.Fatalf("restored file %q not found in root listing", newName)
	}

	// Clean up: trash + permanently purge.
	if err := client.DeleteFile(h, file.UUID); err != nil {
		t.Fatalf("DeleteFile (cleanup): %v", err)
	}
	if err := client.PurgeFile(h, file.UUID); err != nil {
		t.Fatalf("PurgeFile (cleanup): %v", err)
	}
}
