import { apiRequest } from './request';

export interface UploadedFile {
  fileName: string;
  originalName: string;
  objectKey: string;
  url: string;
  size: number;
  contentType: string;
}

export function uploadSystemFile(file: File, scope = 'general') {
  const formData = new FormData();
  formData.append('file', file);
  return apiRequest<UploadedFile>({
    url: '/system/upload',
    method: 'post',
    params: { scope },
    data: formData,
    headers: {
      'Content-Type': 'multipart/form-data',
    },
  });
}
