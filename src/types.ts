export interface Book {
  title: string;
  filename: string;
  size: number;
  mimeType: string;
  lastUpdated: Date;
  // Helpers for sorting/filtering
  simpleMime?: string;
}

export type SortMode = 'name-asc' | 'name-desc' | 'date-asc' | 'date-desc';

export interface TemplateData {
  books: Book[];
  baseUrl: string;
  currentTime: string;
  sortMode: SortMode;
}
