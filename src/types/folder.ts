export interface Folder {
  id: number;
  uuid: string;
  name: string;
  parent_id: number;
  owner_id: number;
}

export interface File {
  id: number;
  uuid: string;
  name: string;
  extension: string;
  file_size: number;
  parent: string | null;
  created_at: string;
  delete_time: string;
  created_by: number;
}

// FolderItem is a single row from a unified folder+file listing endpoint
// (drive listing, trash listing) — folders and files come back interleaved
// in one deterministically-sorted, paginated list rather than as two
// separately-paginated arrays. Fields that don't apply to a folder row
// (extension, mimeType, checksum, created_by) are simply absent/zero.
export interface FolderItem {
  type: 'folder' | 'file';
  id: number;
  uuid: string;
  name: string;
  extension?: string;
  mimeType?: string;
  file_size: number;
  checksum?: string;
  parent_id?: number;
  owner_id?: number;
  created_by?: number;
  created_at: string;
  deleted_at?: string;
}

export interface Breadcrumb {
  folder_id: string;
  label: string;
}

export interface FolderContents {
  items: FolderItem[];
  breadcrumbs: Breadcrumb[];
  page: number;
  limit: number;
  total: number;
}
