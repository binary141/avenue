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
}

export interface Breadcrumb {
  folder_id: string;
  label: string;
}

export interface FolderContents {
  folders: Folder[];
  files: File[];
  breadcrumbs: Breadcrumb[];
}
