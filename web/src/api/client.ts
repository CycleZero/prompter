import axios from 'axios';
import type {
  Region,
  Slice,
  ActivePrompt,
  Record,
  PaginatedResponse,
  ComboRegion,
  SliceType,
} from '../types';

// 创建 axios 实例，所有请求以 /api 为前缀（通过 Vite 代理转发到后端 7745 端口）
const client = axios.create({ baseURL: '/api' });

export const api = {
  // ===== 区域 (Regions) =====
  /** 获取所有区域 */
  listRegions: () => client.get<Region[]>('/regions'),
  /** 创建新区域 */
  createRegion: (data: { name: string; sort_order: number; description?: string }) =>
    client.post<Region>('/regions', data),
  /** 获取单个区域 */
  getRegion: (id: number) => client.get<Region>(`/regions/${id}`),
  /** 更新区域 */
  updateRegion: (id: number, data: { name?: string; sort_order?: number; description?: string }) =>
    client.put<Region>(`/regions/${id}`, data),
  /** 删除区域 */
  deleteRegion: (id: number) => client.delete(`/regions/${id}`),

  // ===== 片段 (Slices) =====
  /** 获取片段列表，支持按区域和类型筛选 */
  listSlices: (params?: { region_id?: number; type_id?: number }) =>
    client.get<{ list: Slice[]; total: number }>('/slices', { params }),
  /** 创建新片段 */
  createSlice: (data: { content: string; region_ids: number[] }) =>
    client.post<Slice>('/slices', data),
  /** 获取单个片段 */
  getSlice: (id: number) => client.get<Slice>(`/slices/${id}`),
  /** 更新片段内容 */
  updateSlice: (id: number, data: { content?: string }) =>
    client.put<Slice>(`/slices/${id}`, data),
  /** 删除片段 */
  deleteSlice: (id: number) => client.delete(`/slices/${id}`),

  // ===== 活动 Prompt =====
  /** 获取当前活动 Prompt */
  getActivePrompt: () => client.get<ActivePrompt>('/active-prompt'),
  /** 保存活动 Prompt */
  updateActivePrompt: (data: ActivePrompt) => client.put('/active-prompt', data),

  // ===== 记录 (Records) =====
  /** 持久化指定 UUID 对应的记录 */
  persistRecord: (uuid: string) => client.post<Record>(`/records/${uuid}`),
  /** 获取记录分页列表 */
  listRecords: (page?: number, pageSize?: number) =>
    client.get<PaginatedResponse<Record>>('/records', { params: { page, page_size: pageSize } }),
  /** 获取单条记录 */
  getRecord: (id: number) => client.get<Record>(`/records/${id}`),
  /** 删除记录 */
  deleteRecord: (id: number) => client.delete(`/records/${id}`),

  // ===== 组合树 & 片段类型 =====
  /** 获取区域-片段组合树 */
  getComboTree: () => client.get<{ regions: ComboRegion[] }>('/combo/tree'),
  /** 获取片段类型树 */
  getSliceTypes: () => client.get<{ types: SliceType[] }>('/slice-types'),
  /** 按类型 ID 获取片段列表（便捷方法，等价于 listSlices({ type_id })） */
  listSlicesByType: (typeId: number) =>
    client.get<{ list: Slice[]; total: number }>('/slices', { params: { type_id: typeId } }),
};
