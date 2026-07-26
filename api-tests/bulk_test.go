package apitests

import (
	"strings"
	"testing"

	"avenue/backend/sdk"
)

func TestBulkDeleteAndRestore(t *testing.T) {
	client, h := authedClient(t)

	folderName := uniqueName("bulk-folder")
	fileName := uniqueName("bulk-file") + ".txt"

	if err := client.CreateFolder(h, folderName, ""); err != nil {
		t.Fatalf("CreateFolder: %v", err)
	}
	if err := client.UploadFile(h, fileName, strings.NewReader("bulk test content"), ""); err != nil {
		t.Fatalf("UploadFile: %v", err)
	}

	root := listRoot(t, client, h)
	folder := findItem(root, "folder", folderName)
	file := findItem(root, "file", fileName)
	if folder == nil || file == nil {
		t.Fatalf("expected both test items in root listing, got folder=%v file=%v", folder, file)
	}

	if _, err := client.BulkDelete(h, sdk.BulkDeleteRequest{
		FileIDs:   []string{file.UUID},
		FolderIDs: []string{folder.UUID},
	}); err != nil {
		t.Fatalf("BulkDelete: %v", err)
	}

	root = listRoot(t, client, h)
	if findItem(root, "folder", folderName) != nil || findItem(root, "file", fileName) != nil {
		t.Fatalf("expected both test items gone from root listing after bulk delete")
	}

	trash, err := client.ListTrash(h, 1, 200)
	if err != nil {
		t.Fatalf("ListTrash: %v", err)
	}
	if findItem(trash.Items, "folder", folderName) == nil || findItem(trash.Items, "file", fileName) == nil {
		t.Fatalf("expected both test items in trash listing after bulk delete")
	}

	if _, err := client.BulkRestore(h, sdk.BulkRestoreRequest{
		FileIDs:   []string{file.UUID},
		FolderIDs: []string{folder.UUID},
	}); err != nil {
		t.Fatalf("BulkRestore: %v", err)
	}

	root = listRoot(t, client, h)
	folder = findItem(root, "folder", folderName)
	file = findItem(root, "file", fileName)
	if folder == nil || file == nil {
		t.Fatalf("expected both test items back in root listing after bulk restore")
	}

	// Clean up.
	if _, err := client.DeleteFolder(h, folder.UUID); err != nil {
		t.Fatalf("DeleteFolder (cleanup): %v", err)
	}
	if err := client.PurgeFolder(h, folder.UUID); err != nil {
		t.Fatalf("PurgeFolder (cleanup): %v", err)
	}
	if err := client.DeleteFile(h, file.UUID); err != nil {
		t.Fatalf("DeleteFile (cleanup): %v", err)
	}
	if err := client.PurgeFile(h, file.UUID); err != nil {
		t.Fatalf("PurgeFile (cleanup): %v", err)
	}
}
