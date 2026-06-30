import { useEffect, useState } from 'react';
import {
  Box,
  Button,
  TextField,
  Paper,
  Typography,
  IconButton,
  Chip,
} from '@mui/material';
import { Save, Close } from '@mui/icons-material';
import { api } from '../api/client';
import { usePromptStore } from '../store';
import type { ActiveSlice, SliceType, Slice, SearchSlice } from '../types';
import { RegionPanel } from '../components/RegionPanel';

/** 编辑器页面 — 单页布局：当前 Prompt 区域 + 提示词库 + 底部预览栏 */
export function EditorPage() {
  // 提示词类型树（分类数据）
  const [sliceTypes, setSliceTypes] = useState<SliceType[]>([]);
  // 保存状态提示
  const [status, setStatus] = useState('');
  // Zustand 全局状态
  const { title, setTitle, regions, getPromptPreview } = usePromptStore();

  // 初始化：加载片段类型树和活动 Prompt 数据
  useEffect(() => {
    api.getSliceTypes().then((res) => setSliceTypes(res.data.types));
    api.getActivePrompt().then((res) => {
      if (res.data.regions?.length) {
        setTitle(res.data.title || '');
        usePromptStore.getState().setRegions(res.data.regions);
      }
    });
  }, [setTitle]);

  /** 点击提示词库中的标签 → 自动创建或添加到对应区域 */
  const handleSliceClick = (typeName: string, slice: Slice | SearchSlice) => {
    const currentRegions = usePromptStore.getState().regions;
    // SearchSlice 与 Slice 均有 translated_content，但为安全起见使用类型断言
    const fullSlice = slice as { translated_content?: string };
    const existing = currentRegions.find((r) => r.region_name === typeName);

    const activeSlice: ActiveSlice = {
      slice_id: slice.id,
      content: slice.content,
      translated_content: fullSlice.translated_content ?? '',
      sort_order: existing ? existing.slices.length : 0,
      custom_text: null,
    };

    if (existing) {
      usePromptStore.getState().addSliceToRegion(existing.region_id, activeSlice);
    } else {
      // 自动创建新 Region
      usePromptStore.getState().addRegion({
        region_id: Date.now(),
        region_name: typeName,
        sort_order: currentRegions.length,
        slices: [activeSlice],
      });
    }
  };

  /** 从区域中移除指定片段 */
  const handleRemoveSlice = (regionId: number, sliceId: number) => {
    usePromptStore.getState().removeSliceFromRegion(regionId, sliceId);
  };

  /** 删除整个区域 */
  const handleRemoveRegion = (regionId: number) => {
    usePromptStore.getState().removeRegion(regionId);
  };

  /** 保存当前 Prompt 到后端 */
  const handleSave = async () => {
    try {
      const currentRegions = usePromptStore.getState().regions;
      const currentTitle = usePromptStore.getState().title;
      await api.updateActivePrompt({ title: currentTitle, regions: currentRegions, updated_at: '' });
      setStatus('已保存');
      setTimeout(() => setStatus(''), 2000);
    } catch {
      setStatus('保存失败');
    }
  };

  const preview = getPromptPreview();

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', flexGrow: 1, minHeight: 0 }}>
      {/* 标题栏：Prompt 标题 + 保存按钮 + 状态提示 */}
      <Box sx={{ p: 2, display: 'flex', gap: 1, alignItems: 'center', flexShrink: 0 }}>
        <TextField
          size="small"
          label="Prompt 标题"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          sx={{ flexGrow: 1 }}
          slotProps={{
            input: {
              sx: { bgcolor: '#f8f9fa' },
            },
          }}
        />
        <Button
          variant="contained"
          startIcon={<Save />}
          onClick={handleSave}
          disabled={regions.length === 0}
        >
          保存
        </Button>
        {status && (
          <Chip
            label={status}
            color={status === '已保存' ? 'success' : 'error'}
            size="small"
          />
        )}
      </Box>

      {/* 主内容区域：当前 Prompt + 提示词库（双区域，可滚动） */}
      <Box sx={{ flexGrow: 1, overflow: 'auto', px: 2, pb: 1 }}>
        {/* 区域 1：当前 Prompt — 已选片段按 Region 分组展示 */}
        <Paper sx={{ p: 2, mb: 2, bgcolor: '#fafbfc' }} variant="outlined">
          <Typography variant="subtitle1" sx={{ mb: 1.5, fontWeight: 600 }}>
            当前 Prompt
          </Typography>
          {regions.length === 0 ? (
            <Typography color="text.secondary" variant="body2">
              点击下方标签库中的标签开始构建 Prompt
            </Typography>
          ) : (
            <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 2 }}>
              {regions.map((region) => (
                <Box
                  key={region.region_id}
                  sx={{
                    border: 1,
                    borderColor: 'divider',
                    borderRadius: 1,
                    p: 1.5,
                    minWidth: 200,
                    bgcolor: 'white',
                  }}
                >
                  {/* Region 标题行 */}
                  <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
                    <Typography
                      variant="caption"
                      sx={{ fontWeight: 600, color: 'text.secondary', flexGrow: 1 }}
                    >
                      {region.region_name}
                    </Typography>
                    <IconButton
                      size="small"
                      onClick={() => handleRemoveRegion(region.region_id)}
                      sx={{ p: 0, ml: 0.5 }}
                    >
                      <Close fontSize="inherit" />
                    </IconButton>
                  </Box>
                  {/* Region 内的标签列表 */}
                  <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                    {region.slices.map((slice) => (
                      <Chip
                        key={slice.slice_id}
                        label={slice.custom_text ?? slice.content ?? `#${slice.slice_id}`}
                        size="small"
                        color="primary"
                        variant="filled"
                        onDelete={() => handleRemoveSlice(region.region_id, slice.slice_id)}
                        sx={{ fontSize: '0.75rem' }}
                      />
                    ))}
                  </Box>
                </Box>
              ))}
            </Box>
          )}
        </Paper>

        {/* 区域 2：提示词库 — 搜索框 + 分类标签树 */}
        <Paper sx={{ p: 2, bgcolor: '#fafbfc' }} variant="outlined">
          <Typography variant="subtitle1" sx={{ mb: 1.5, fontWeight: 600 }}>
            提示词库
          </Typography>
          <RegionPanel types={sliceTypes} onSliceClick={handleSliceClick} />
        </Paper>
      </Box>

      {/* 底部预览栏：拼接后的完整 Prompt 文本 */}
      <Paper
        sx={{
          p: 1.5,
          flexShrink: 0,
          borderTop: 1,
          borderColor: 'divider',
          bgcolor: '#f8f9fa',
          fontFamily: 'monospace',
          fontSize: '0.8rem',
          color: 'text.secondary',
        }}
        variant="outlined"
      >
        📝 {preview || '从提示词库中选择标签开始构建...'}
      </Paper>
    </Box>
  );
}
