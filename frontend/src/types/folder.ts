export interface Folder {
  folder_id: string;
  name: string;
  parent: string | null;
  owner_id: number;
}

export interface File {
  id: string;
  name: string;
  extension: string;
  file_size: number;
  parent: string | null;
  created_at: string;
  delete_time: string;
}

export interface FolderContents {
  folders: Folder[];
  files: File[];
}

