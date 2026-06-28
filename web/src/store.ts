import { create } from 'zustand';
import type { ActivePromptRegion, ActiveSlice } from './types';

// Prompt 编辑器全局状态
interface PromptState {
  // 状态
  title: string;
  regions: ActivePromptRegion[];

  // 基础操作
  setTitle: (title: string) => void;
  setRegions: (regions: ActivePromptRegion[]) => void;

  // 片段操作
  /** 向指定区域添加片段 */
  addSliceToRegion: (regionId: number, slice: ActiveSlice) => void;
  /** 从指定区域移除片段 */
  removeSliceFromRegion: (regionId: number, sliceId: number) => void;
  /** 更新片段的自定义文本 */
  updateSliceCustomText: (regionId: number, sliceId: number, text: string | null) => void;
  /** 拖动片段到新位置 */
  moveSlice: (regionId: number, fromIndex: number, toIndex: number) => void;

  // 区域操作
  /** 拖动区域到新位置 */
  moveRegion: (fromIndex: number, toIndex: number) => void;
  /** 移除区域 */
  removeRegion: (regionId: number) => void;
  /** 添加区域 */
  addRegion: (region: ActivePromptRegion) => void;

  // 计算属性
  /** 获取拼接后的 Prompt 预览文本 */
  getPromptPreview: () => string;
}

export const usePromptStore = create<PromptState>((set, get) => ({
  title: '',
  regions: [],

  setTitle: (title) => set({ title }),
  setRegions: (regions) => set({ regions }),

  // --- 片段操作 ---

  addSliceToRegion: (regionId, slice) =>
    set((state) => ({
      regions: state.regions.map((r) =>
        r.region_id === regionId
          ? { ...r, slices: [...r.slices, slice] }
          : r,
      ),
    })),

  removeSliceFromRegion: (regionId, sliceId) =>
    set((state) => ({
      regions: state.regions.map((r) =>
        r.region_id === regionId
          ? { ...r, slices: r.slices.filter((s) => s.slice_id !== sliceId) }
          : r,
      ),
    })),

  updateSliceCustomText: (regionId, sliceId, text) =>
    set((state) => ({
      regions: state.regions.map((r) =>
        r.region_id === regionId
          ? {
              ...r,
              slices: r.slices.map((s) =>
                s.slice_id === sliceId ? { ...s, custom_text: text } : s,
              ),
            }
          : r,
      ),
    })),

  moveSlice: (regionId, fromIndex, toIndex) =>
    set((state) => ({
      regions: state.regions.map((r) => {
        if (r.region_id !== regionId) return r;
        const slices = [...r.slices];
        const [removed] = slices.splice(fromIndex, 1);
        slices.splice(toIndex, 0, removed);
        // 更新 sort_order 以反映新位置
        return { ...r, slices: slices.map((s, i) => ({ ...s, sort_order: i })) };
      }),
    })),

  // --- 区域操作 ---

  moveRegion: (fromIndex, toIndex) =>
    set((state) => {
      const regions = [...state.regions];
      const [removed] = regions.splice(fromIndex, 1);
      regions.splice(toIndex, 0, removed);
      return { regions: regions.map((r, i) => ({ ...r, sort_order: i })) };
    }),

  removeRegion: (regionId) =>
    set((state) => ({
      regions: state.regions.filter((r) => r.region_id !== regionId),
    })),

  addRegion: (region) =>
    set((state) => ({
      regions: [
        ...state.regions,
        { ...region, sort_order: state.regions.length },
      ],
    })),

  // --- 计算属性 ---

  getPromptPreview: () => {
    const { regions } = get();
    const sorted = [...regions].sort((a, b) => a.sort_order - b.sort_order);
    const parts: string[] = [];
    for (const r of sorted) {
      const sortedSlices = [...r.slices].sort((a, b) => a.sort_order - b.sort_order);
      for (const s of sortedSlices) {
        parts.push(s.custom_text ?? s.content);
      }
    }
    return parts.join(', ');
  },
}));
