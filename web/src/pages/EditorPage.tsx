import { useEffect, useState } from 'react';
import {
  Grid,
  Paper,
  Typography,
  Box,
  Button,
  TextField,
  Chip,
} from '@mui/material';
import { api } from '../api/client';
import { usePromptStore } from '../store';
import type { ComboRegion, ActiveSlice, ComboSlice } from '../types';
import { RegionPanel } from '../components/RegionPanel';
import { ActiveSlices } from '../components/ActiveSlices';

/** 编辑器页面 — 三栏布局：左侧选择器 / 中间预览 / 右侧已选片段 */
export function EditorPage() {
  const [comboTree, setComboTree] = useState<ComboRegion[]>([]);
  const { title, setTitle, regions, getPromptPreview } = usePromptStore();
  const [status, setStatus] = useState('');

  // 初始化加载数据
  useEffect(() => {
    api.getComboTree().then((res) => setComboTree(res.data.regions));
    api.getActivePrompt().then((res) => {
      if (res.data.regions?.length) {
        setTitle(res.data.title || '');
        usePromptStore.getState().setRegions(res.data.regions);
      }
    });
  }, [setTitle]);

  /** 点击左侧片段 -> 添加到对应区域 */
  const handleSliceClick = (region: ComboRegion, slice: ComboSlice) => {
    const existing = regions.find((r) => r.region_id === region.id);
    const activeSlice: ActiveSlice = {
      slice_id: slice.id,
      sort_order: existing ? existing.slices.length : 0,
      custom_text: null,
    };
    if (existing) {
      usePromptStore.getState().addSliceToRegion(region.id, activeSlice);
    } else {
      usePromptStore.getState().addRegion({
        region_id: region.id,
        region_name: region.name,
        sort_order: regions.length,
        slices: [activeSlice],
      });
    }
  };

  /** 保存当前 Prompt 到后端 */
  const handleSave = async () => {
    try {
      await api.updateActivePrompt({ title, regions, updated_at: '' });
      setStatus('已保存');
      setTimeout(() => setStatus(''), 2000);
    } catch {
      setStatus('保存失败');
    }
  };

  const preview = getPromptPreview();

  return (
    <Grid container spacing={2} sx={{ height: 'calc(100vh - 100px)' }}>
      {/* 左侧：区域 + 片段选择器 */}
      <Grid size={3}>
        <Paper sx={{ p: 2, height: '100%', overflow: 'auto' }}>
          <Typography variant="subtitle1" gutterBottom>
            提示词库
          </Typography>
          <RegionPanel regions={comboTree} onSliceClick={handleSliceClick} />
        </Paper>
      </Grid>

      {/* 中间：标题 + 预览 + 保存 */}
      <Grid size={6}>
        <Paper
          sx={{
            p: 2,
            height: '100%',
            display: 'flex',
            flexDirection: 'column',
          }}
        >
          {/* 标题栏 */}
          <Box sx={{ mb: 2, display: 'flex', gap: 1, alignItems: 'center' }}>
            <TextField
              size="small"
              label="标题"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              sx={{ flexGrow: 1 }}
            />
            <Button
              variant="contained"
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

          {/* 预览区域 */}
          <Paper
            variant="outlined"
            sx={{
              flexGrow: 1,
              p: 2,
              bgcolor: 'grey.900',
              fontFamily: 'monospace',
              whiteSpace: 'pre-wrap',
              overflow: 'auto',
            }}
          >
            {preview || '从左侧选择提示词块开始构建...'}
          </Paper>
        </Paper>
      </Grid>

      {/* 右侧：已选片段列表（支持拖拽排序） */}
      <Grid size={3}>
        <Paper sx={{ p: 2, height: '100%', overflow: 'auto' }}>
          <Typography variant="subtitle1" gutterBottom>
            已选片段
          </Typography>
          <ActiveSlices />
        </Paper>
      </Grid>
    </Grid>
  );
}
