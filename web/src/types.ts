// 区域（Prompt 段落分组）
export interface Region {
  id: number;
  name: string;
  sort_order: number;
  description: string;
  created_at: string;
  updated_at: string;
}

// 提示词片段
export interface Slice {
  id: number;
  content: string;
  translated_content: string;
  origin_language: string;
  target_language: string;
  created_at: string;
  updated_at: string;
}

// 活动 Prompt 中的单个片段
export interface ActiveSlice {
  slice_id: number;
  content: string;
  translated_content: string;
  custom_text: string | null;
  sort_order: number;
}

// 活动 Prompt 中的区域
export interface ActivePromptRegion {
  region_id: number;
  region_name: string;
  sort_order: number;
  slices: ActiveSlice[];
}

// 活动 Prompt（当前编辑中的 Prompt）
export interface ActivePrompt {
  title: string;
  regions: ActivePromptRegion[];
  updated_at: string;
}

// 记录中的片段
export interface RecordSlice {
  slice_id: number;
  content: string;
  custom_text: string | null;
  sort_order: number;
}

// 记录中的区域
export interface RecordRegion {
  region_id: number;
  region_name: string;
  sort_order: number;
  slices: RecordSlice[];
}

// 持久化记录
export interface Record {
  id: number;
  external_id: string;
  title: string;
  full_content: string;
  regions: RecordRegion[];
  created_at: string;
  updated_at: string;
}

// 组合树中的区域（含片段列表）
export interface ComboRegion {
  id: number;
  name: string;
  sort_order: number;
  description: string;
  slices: ComboSlice[];
}

// 组合树中的片段
export interface ComboSlice {
  id: number;
  content: string;
  translated_content: string;
  origin_language: string;
  target_language: string;
  sort_order: number;
}

// 片段类型树节点
export interface SliceType {
  id: number;
  name: string;
  parent_id: number | null;
  sort_order: number;
  children: SliceType[];
}

// 分页响应
export interface PaginatedResponse<T> {
  list: T[];
  total: number;
  page: number;
}
