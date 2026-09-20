// Sharing a file or folder with another user of the server (as opposed to a
// public link). 'viewer' can look and download; 'editor' can also upload,
// create, rename, and delete things inside a shared folder.
export type ShareRole = 'viewer' | 'editor';

// One grant on a file or folder, as listed to that item's owner.
export interface UserShare {
  id: number;
  resource_type: 'file' | 'folder';
  resource_id: string;
  grantee_id: number;
  grantee_email: string;
  grantee_first_name: string;
  grantee_last_name: string;
  role: ShareRole;
  created_at: string;
}

// A file or folder another user shared with the caller.
export interface SharedWithMeItem {
  share_id: number;
  type: 'file' | 'folder';
  uuid: string;
  name: string;
  extension?: string;
  mimeType?: string;
  file_size: number;
  role: ShareRole;
  shared_by_id: number;
  shared_by_email: string;
  shared_at: string;
}
