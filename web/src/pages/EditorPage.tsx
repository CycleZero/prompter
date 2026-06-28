import { useEffect, useState } from 'react';
import {
  Box,
  Button,
  TextField,
  Paper,
  Typography,
  IconButton,
  Drawer,
  Chip,
  Card,
  CardHeader,
  CardContent,
} from '@mui/material';
import { Menu as MenuIcon, Save, DragIndicator } from '@mui/icons-material';
import {
  DndContext,
  closestCenter,
  PointerSensor,
  useSensor,
  useSensors,
} from '@dnd-kit/core';
import type { DragEndEvent } from '@dnd-kit/core';
import {
  SortableContext,
  rectSortingStrategy,
  useSortable,
} from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import { api } from '../api/client';
import { usePromptStore } from '../store';
import type {
  ComboRegion,
  ComboSlice,
  ActiveSlice,
  ActivePromptRegion,
} from '../types';
import { RegionPanel } from '../components/RegionPanel';

/** 编辑器页面 — 主区域为可拖拽排序的 Region 卡片，Slice 以 Chip 形式展示，底部为 Prompt 预览 */
export function EditorPage() {
  const [comboTree, setComboTree] = useState<ComboRegion[]>([]);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const { title, setTitle, regions, getPromptPreview } = usePromptStore();
  const [status, setStatus] = useState('');

  // 初始化加载组合树和活动 Prompt 数据
  useEffect(() => {
    api.getComboTree().then((res) => setComboTree(res.data.regions));
    api.getActivePrompt().then((res) => {
      if (res.data.regions?.length) {
        setTitle(res.data.title || '');
        usePromptStore.getState().setRegions(res.data.regions);
      }
    });
  }, [setTitle]);

  /** 点击提示词库中的片段 -> 添加到对应区域 */
  const handleSliceClick = (region: ComboRegion, slice: ComboSlice) => {
    const existing = regions.find((r) => r.region_id === region.id);
    const s: ActiveSlice = {
      slice_id: slice.id,
      sort_order: existing ? existing.slices.length : 0,
      custom_text: null,
    };
    if (existing) {
      usePromptStore.getState().addSliceToRegion(region.id, s);
    } else {
      usePromptStore.getState().addRegion({
        region_id: region.id,
        region_name: region.name,
        sort_order: regions.length,
        slices: [s],
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

  // 拖拽传感器：需要移动 5px 后才触发拖拽，避免误触
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
  );

  /** Region 卡片拖拽排序结束回调 */
  const handleRegionDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (!over || active.id === over.id) return;
    const oldIdx = regions.findIndex(
      (r) => `region-${r.region_id}` === active.id,
    );
    const newIdx = regions.findIndex(
      (r) => `region-${r.region_id}` === over.id,
    );
    if (oldIdx !== -1 && newIdx !== -1) {
      usePromptStore.getState().moveRegion(oldIdx, newIdx);
    }
  };

  return (
    <Box sx={{ height: 'calc(100vh - 96px)', display: 'flex', flexDirection: 'column' }}>
      {/* 顶部工具栏 */}
      <Box
        sx={{
          display: 'flex',
          gap: 1,
          mb: 2,
          alignItems: 'center',
          flexShrink: 0,
        }}
      >
        <IconButton onClick={() => setDrawerOpen(true)}>
          <MenuIcon />
        </IconButton>
        <TextField
          size="small"
          label="标题"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          sx={{ flexGrow: 1 }}
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

      {/* 主区域：可拖拽排序的 Region 卡片列表 */}
      <Box sx={{ flexGrow: 1, overflow: 'auto', pb: 1 }}>
        <DndContext
          sensors={sensors}
          collisionDetection={closestCenter}
          onDragEnd={handleRegionDragEnd}
        >
          <SortableContext
            items={regions.map((r) => `region-${r.region_id}`)}
            strategy={rectSortingStrategy}
          >
            {regions.map((region) => (
              <SortableRegionCard key={region.region_id} region={region} />
            ))}
          </SortableContext>
        </DndContext>

        {/* 添加 Region 占位卡片，点击打开提示词库抽屉 */}
        <Card
          sx={{
            mt: 1,
            borderStyle: 'dashed',
            cursor: 'pointer',
            opacity: 0.6,
            '&:hover': { opacity: 1 },
          }}
          onClick={() => setDrawerOpen(true)}
        >
          <CardContent sx={{ textAlign: 'center', py: 1 }}>
            <Typography variant="body2" color="text.secondary">
              + 从提示词库添加 Region 片段
            </Typography>
          </CardContent>
        </Card>
      </Box>

      {/* 底部 Prompt 预览栏 */}
      <Paper
        variant="outlined"
        sx={{
          p: 1.5,
          flexShrink: 0,
          bgcolor: 'grey.900',
          fontFamily: 'monospace',
          fontSize: '0.85rem',
          maxHeight: 80,
          overflow: 'auto',
        }}
      >
        📝 {preview || '从提示词库选择片段开始构建...'}
      </Paper>

      {/* 左侧抽屉：提示词库 */}
      <Drawer open={drawerOpen} onClose={() => setDrawerOpen(false)}>
        <Box sx={{ width: 300, p: 2 }}>
          <Typography variant="h6" gutterBottom>
            提示词库
          </Typography>
          <RegionPanel
            regions={comboTree}
            onSliceClick={(r, s) => {
              handleSliceClick(r, s);
            }}
          />
        </Box>
      </Drawer>
    </Box>
  );
}

/** 可拖拽排序的 Region 卡片组件 */
function SortableRegionCard({ region }: { region: ActivePromptRegion }) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
  } = useSortable({ id: `region-${region.region_id}` });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
  };

  const removeRegion = usePromptStore((s) => s.removeRegion);
  const moveSlice = usePromptStore((s) => s.moveSlice);
  const removeSlice = usePromptStore((s) => s.removeSliceFromRegion);

  // 内层拖拽传感器：需要移动 3px 后触发，比外层更灵敏
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 3 } }),
  );

  /** Slice Chip 拖拽排序结束回调 */
  const handleSliceDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (!over || active.id === over.id) return;
    const oldIdx = region.slices.findIndex(
      (s) => `slice-${s.slice_id}` === active.id,
    );
    const newIdx = region.slices.findIndex(
      (s) => `slice-${s.slice_id}` === over.id,
    );
    if (oldIdx !== -1 && newIdx !== -1) {
      moveSlice(region.region_id, oldIdx, newIdx);
    }
  };

  return (
    <Card ref={setNodeRef} style={style} sx={{ mb: 1 }}>
      <CardHeader
        title={region.region_name}
        slotProps={{ title: { variant: 'subtitle2' } }}
        action={
          <Box sx={{ display: 'flex', alignItems: 'center' }}>
            {/* Region 拖拽手柄 */}
            <IconButton
              size="small"
              {...attributes}
              {...listeners}
              sx={{ cursor: 'grab' }}
            >
              <DragIndicator fontSize="small" />
            </IconButton>
            {/* 删除 Region 按钮 */}
            <IconButton
              size="small"
              onClick={() => removeRegion(region.region_id)}
            >
              <Chip
                label="×"
                size="small"
                sx={{ cursor: 'pointer' }}
              />
            </IconButton>
          </Box>
        }
        sx={{ py: 0.5, px: 1 }}
      />
      <CardContent sx={{ pt: 0, pb: 1, px: 1, '&:last-child': { pb: 1 } }}>
        <DndContext
          sensors={sensors}
          collisionDetection={closestCenter}
          onDragEnd={handleSliceDragEnd}
        >
          <SortableContext
            items={region.slices.map((s) => `slice-${s.slice_id}`)}
            strategy={rectSortingStrategy}
          >
            <Box
              sx={{
                display: 'flex',
                flexWrap: 'wrap',
                gap: 0.5,
                alignItems: 'center',
              }}
            >
              {region.slices.map((slice) => (
                <SortableSliceChip
                  key={slice.slice_id}
                  regionId={region.region_id}
                  slice={slice}
                  onRemove={removeSlice}
                />
              ))}
            </Box>
          </SortableContext>
        </DndContext>
      </CardContent>
    </Card>
  );
}

/** 可拖拽排序的 Slice Chip 组件 — 以标签形式展示 */
function SortableSliceChip({
  regionId,
  slice,
  onRemove,
}: {
  regionId: number;
  slice: ActiveSlice;
  onRemove: (regionId: number, sliceId: number) => void;
}) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
  } = useSortable({ id: `slice-${slice.slice_id}` });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
  };

  // 显示标签文本：优先使用自定义文本，否则显示片段 ID
  const label = slice.custom_text ?? `#${slice.slice_id}`;

  return (
    <Box
      ref={setNodeRef}
      style={style}
      {...attributes}
      {...listeners}
      sx={{ cursor: 'grab' }}
    >
      <Chip
        label={label}
        size="small"
        variant="filled"
        color="primary"
        onDelete={() => onRemove(regionId, slice.slice_id)}
        sx={{
          '& .MuiChip-label': {
            maxWidth: 120,
            overflow: 'hidden',
            textOverflow: 'ellipsis',
          },
        }}
      />
    </Box>
  );
}
