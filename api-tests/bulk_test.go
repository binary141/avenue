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

func TestBulkMove(t *testing.T) {
	client, h := authedClient(t)

	destName := uniqueName("bulk-move-dest")
	srcName := uniqueName("bulk-move-src")
	fileName := uniqueName("bulk-move-file") + ".txt"

	if err := client.CreateFolder(h, destName, ""); err != nil {
		t.Fatalf("CreateFolder (dest): %v", err)
	}
	if err := client.CreateFolder(h, srcName, ""); err != nil {
		t.Fatalf("CreateFolder (src): %v", err)
	}
	if err := client.UploadFile(h, fileName, strings.NewReader("bulk move test content"), ""); err != nil {
		t.Fatalf("UploadFile: %v", err)
	}

	root := listRoot(t, client, h)
	dest := findItem(root, "folder", destName)
	src := findItem(root, "folder", srcName)
	file := findItem(root, "file", fileName)
	if dest == nil || src == nil || file == nil {
		t.Fatalf("expected dest, src, and file in root listing, got dest=%v src=%v file=%v", dest, src, file)
	}

	if _, err := client.BulkMove(h, sdk.BulkMoveRequest{
		FileIDs:   []string{file.UUID},
		FolderIDs: []string{src.UUID},
		Parent:    dest.UUID,
	}); err != nil {
		t.Fatalf("BulkMove: %v", err)
	}

	root = listRoot(t, client, h)
	if findItem(root, "folder", srcName) != nil || findItem(root, "file", fileName) != nil {
		t.Fatalf("expected moved items gone from root listing after bulk move")
	}

	contents, err := client.ListFolderContents(h, dest.UUID, 1, 200)
	if err != nil {
		t.Fatalf("ListFolderContents (dest): %v", err)
	}
	if findItem(contents.Items, "folder", srcName) == nil || findItem(contents.Items, "file", fileName) == nil {
		t.Fatalf("expected moved items inside destination folder")
	}

	// Moving a folder into itself (or a descendant) must be rejected.
	if _, err := client.BulkMove(h, sdk.BulkMoveRequest{
		FolderIDs: []string{dest.UUID},
		Parent:    dest.UUID,
	}); err == nil {
		t.Fatalf("expected BulkMove into self to fail")
	}

	// Clean up.
	if _, err := client.DeleteFolder(h, dest.UUID); err != nil {
		t.Fatalf("DeleteFolder (cleanup): %v", err)
	}
	if err := client.PurgeFolder(h, dest.UUID); err != nil {
		t.Fatalf("PurgeFolder (cleanup): %v", err)
	}
}
